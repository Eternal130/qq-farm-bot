package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"qq-farm-bot/internal/model"
)

type Store struct {
	db *sql.DB
}

func New(dbPath string) (*Store, error) {
	os.MkdirAll(filepath.Dir(dbPath), 0755)
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

// Column list shared by all account queries
const accountColumns = `id, user_id, name, platform, code, auto_start,
	farm_interval, friend_interval, enable_steal, force_lowest,
	enable_harvest, enable_plant, enable_sell, enable_weed, enable_bug, enable_water,
	enable_remove_dead, enable_upgrade_land, enable_help_friend, enable_claim_task,
	plant_crop_id, sell_crop_ids, steal_crop_ids,
	auto_use_fertilizer, auto_buy_fertilizer, fertilizer_target_count, fertilizer_buy_daily_limit,
	enable_anti_detection,
	prefer_bag_seeds,
	planting_strategy,
	api_key,
	created_at, updated_at`

func (s *Store) migrate() error {
	ddl := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		is_admin INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS accounts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL DEFAULT 1,
		name TEXT NOT NULL DEFAULT '',
		platform TEXT NOT NULL DEFAULT 'qq',
		code TEXT NOT NULL DEFAULT '',
		auto_start INTEGER NOT NULL DEFAULT 0,
		farm_interval INTEGER NOT NULL DEFAULT 10,
		friend_interval INTEGER NOT NULL DEFAULT 10,
		enable_steal INTEGER NOT NULL DEFAULT 1,
		force_lowest INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		account_id INTEGER NOT NULL,
		tag TEXT NOT NULL DEFAULT '',
		message TEXT NOT NULL DEFAULT '',
		level TEXT NOT NULL DEFAULT 'info',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_logs_account ON logs(account_id, created_at DESC);
	`
	_, err := s.db.Exec(ddl)

	// Migration: add user_id column if not exists (for existing databases)
	_, _ = s.db.Exec(`ALTER TABLE accounts ADD COLUMN user_id INTEGER NOT NULL DEFAULT 1`)
	_, _ = s.db.Exec(`UPDATE accounts SET user_id = 1 WHERE user_id = 0 OR user_id IS NULL`)
	// Migration: add fertilizer columns
	_, _ = s.db.Exec(`ALTER TABLE accounts ADD COLUMN auto_use_fertilizer INTEGER NOT NULL DEFAULT 0`)
	_, _ = s.db.Exec(`ALTER TABLE accounts ADD COLUMN auto_buy_fertilizer INTEGER NOT NULL DEFAULT 0`)
	_, _ = s.db.Exec(`ALTER TABLE accounts ADD COLUMN fertilizer_target_count INTEGER NOT NULL DEFAULT 0`)
	_, _ = s.db.Exec(`ALTER TABLE accounts ADD COLUMN fertilizer_buy_daily_limit INTEGER NOT NULL DEFAULT 0`)

	// Migration: add farm automation toggles (default 1 = enabled for backward compatibility)
	_, _ = s.db.Exec(`ALTER TABLE accounts ADD COLUMN enable_harvest INTEGER NOT NULL DEFAULT 1`)
	_, _ = s.db.Exec(`ALTER TABLE accounts ADD COLUMN enable_plant INTEGER NOT NULL DEFAULT 1`)
	_, _ = s.db.Exec(`ALTER TABLE accounts ADD COLUMN enable_sell INTEGER NOT NULL DEFAULT 1`)
	_, _ = s.db.Exec(`ALTER TABLE accounts ADD COLUMN enable_weed INTEGER NOT NULL DEFAULT 1`)
	_, _ = s.db.Exec(`ALTER TABLE accounts ADD COLUMN enable_bug INTEGER NOT NULL DEFAULT 1`)
	_, _ = s.db.Exec(`ALTER TABLE accounts ADD COLUMN enable_water INTEGER NOT NULL DEFAULT 1`)
	_, _ = s.db.Exec(`ALTER TABLE accounts ADD COLUMN enable_remove_dead INTEGER NOT NULL DEFAULT 1`)
	_, _ = s.db.Exec(`ALTER TABLE accounts ADD COLUMN enable_upgrade_land INTEGER NOT NULL DEFAULT 1`)
	_, _ = s.db.Exec(`ALTER TABLE accounts ADD COLUMN enable_help_friend INTEGER NOT NULL DEFAULT 1`)
	_, _ = s.db.Exec(`ALTER TABLE accounts ADD COLUMN enable_claim_task INTEGER NOT NULL DEFAULT 1`)

	// Migration: add crop selection & filtering columns
	_, _ = s.db.Exec(`ALTER TABLE accounts ADD COLUMN plant_crop_id INTEGER NOT NULL DEFAULT 0`)
	_, _ = s.db.Exec(`ALTER TABLE accounts ADD COLUMN sell_crop_ids TEXT NOT NULL DEFAULT ''`)
	_, _ = s.db.Exec(`ALTER TABLE accounts ADD COLUMN steal_crop_ids TEXT NOT NULL DEFAULT ''`)
	// Migration: add anti-detection column
	_, _ = s.db.Exec(`ALTER TABLE accounts ADD COLUMN enable_anti_detection INTEGER NOT NULL DEFAULT 0`)
	// Migration: add per-account API key column
_, _ = s.db.Exec(`ALTER TABLE accounts ADD COLUMN api_key TEXT NOT NULL DEFAULT ''
`)
	// Migration: op_stats table for operation statistics tracking
	_, _ = s.db.Exec(`CREATE TABLE IF NOT EXISTS op_stats (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		account_id INTEGER NOT NULL,
		op_type TEXT NOT NULL,
		count INTEGER NOT NULL DEFAULT 0,
		gold_delta INTEGER NOT NULL DEFAULT 0,
		exp_delta INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	_, _ = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_op_stats_account_time ON op_stats(account_id, created_at)`)
	_, _ = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_op_stats_type ON op_stats(account_id, op_type, created_at)`)
	// Migration: add detail column to op_stats
	_, _ = s.db.Exec(`ALTER TABLE op_stats ADD COLUMN detail TEXT NOT NULL DEFAULT ''`)
	// Migration: add prefer_bag_seeds column
	_, _ = s.db.Exec(`ALTER TABLE accounts ADD COLUMN prefer_bag_seeds INTEGER NOT NULL DEFAULT 0`)
	// Migration: add planting_strategy column (JSON-encoded composable rules)
	_, _ = s.db.Exec(`ALTER TABLE accounts ADD COLUMN planting_strategy TEXT NOT NULL DEFAULT ''`)

	return err
}

// scanAccount scans a single account row into a model.Account struct.
func scanAccount(scanner interface {
	Scan(dest ...interface{}) error
}) (*model.Account, error) {
	var a model.Account
	var autoStart, enableSteal, forceLowest int
	var enableHarvest, enablePlant, enableSell, enableWeed, enableBug, enableWater int
	var enableRemoveDead, enableUpgradeLand, enableHelpFriend, enableClaimTask int
	var autoUseFert, autoBuyFert, enableAntiDetection, preferBagSeeds int

	if err := scanner.Scan(
		&a.ID, &a.UserID, &a.Name, &a.Platform, &a.Code, &autoStart,
		&a.FarmInterval, &a.FriendInterval, &enableSteal, &forceLowest,
		&enableHarvest, &enablePlant, &enableSell, &enableWeed, &enableBug, &enableWater,
		&enableRemoveDead, &enableUpgradeLand, &enableHelpFriend, &enableClaimTask,
		&a.PlantCropID, &a.SellCropIDs, &a.StealCropIDs,
		&autoUseFert, &autoBuyFert, &a.FertilizerTargetCount, &a.FertilizerBuyDailyLimit,
		&enableAntiDetection,
		&preferBagSeeds,
		&a.PlantingStrategy,
		&a.APIKey,
		&a.CreatedAt, &a.UpdatedAt,
	); err != nil {
		return nil, err
	}

	a.AutoStart = autoStart == 1
	a.EnableSteal = enableSteal == 1
	a.ForceLowest = forceLowest == 1
	a.EnableHarvest = enableHarvest == 1
	a.EnablePlant = enablePlant == 1
	a.EnableSell = enableSell == 1
	a.EnableWeed = enableWeed == 1
	a.EnableBug = enableBug == 1
	a.EnableWater = enableWater == 1
	a.EnableRemoveDead = enableRemoveDead == 1
	a.EnableUpgradeLand = enableUpgradeLand == 1
	a.EnableHelpFriend = enableHelpFriend == 1
	a.EnableClaimTask = enableClaimTask == 1
	a.AutoUseFertilizer = autoUseFert == 1
	a.AutoBuyFertilizer = autoBuyFert == 1
	a.EnableAntiDetection = enableAntiDetection == 1
	a.PreferBagSeeds = preferBagSeeds == 1

	return &a, nil
}

// ============ Account CRUD ============

func (s *Store) ListAccounts() ([]model.Account, error) {
	rows, err := s.db.Query(`SELECT ` + accountColumns + ` FROM accounts ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []model.Account
	for rows.Next() {
		a, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, *a)
	}
	return accounts, nil
}

func (s *Store) ListAccountsByUserID(userID int64) ([]model.Account, error) {
	rows, err := s.db.Query(`SELECT `+accountColumns+` FROM accounts WHERE user_id = ? ORDER BY id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []model.Account
	for rows.Next() {
		a, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, *a)
	}
	return accounts, nil
}

func (s *Store) GetAccount(id int64) (*model.Account, error) {
	row := s.db.QueryRow(`SELECT `+accountColumns+` FROM accounts WHERE id = ?`, id)
	return scanAccount(row)
}

func (s *Store) GetAccountByName(name string) (*model.Account, error) {
	row := s.db.QueryRow(`SELECT `+accountColumns+` FROM accounts WHERE name = ? LIMIT 1`, name)
	return scanAccount(row)
}

func (s *Store) GetAccountByAPIKey(apiKey string) (*model.Account, error) {
	row := s.db.QueryRow(`SELECT `+accountColumns+` FROM accounts WHERE api_key = ? AND api_key != '' LIMIT 1`, apiKey)
	return scanAccount(row)
}

func (s *Store) CreateAccount(a *model.Account) error {
	now := time.Now()
	a.CreatedAt = now
	a.UpdatedAt = now
	res, err := s.db.Exec(`INSERT INTO accounts (
		user_id, name, platform, code, auto_start,
		farm_interval, friend_interval, enable_steal, force_lowest,
		enable_harvest, enable_plant, enable_sell, enable_weed, enable_bug, enable_water,
		enable_remove_dead, enable_upgrade_land, enable_help_friend, enable_claim_task,
		plant_crop_id, sell_crop_ids, steal_crop_ids,
		auto_use_fertilizer, auto_buy_fertilizer, fertilizer_target_count, fertilizer_buy_daily_limit,
		enable_anti_detection,
		prefer_bag_seeds,
		planting_strategy,
		api_key,
		created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.UserID, a.Name, a.Platform, a.Code, boolToInt(a.AutoStart),
		a.FarmInterval, a.FriendInterval, boolToInt(a.EnableSteal), boolToInt(a.ForceLowest),
		boolToInt(a.EnableHarvest), boolToInt(a.EnablePlant), boolToInt(a.EnableSell),
		boolToInt(a.EnableWeed), boolToInt(a.EnableBug), boolToInt(a.EnableWater),
		boolToInt(a.EnableRemoveDead), boolToInt(a.EnableUpgradeLand),
		boolToInt(a.EnableHelpFriend), boolToInt(a.EnableClaimTask),
		a.PlantCropID, a.SellCropIDs, a.StealCropIDs,
		boolToInt(a.AutoUseFertilizer), boolToInt(a.AutoBuyFertilizer),
		a.FertilizerTargetCount, a.FertilizerBuyDailyLimit,
		boolToInt(a.EnableAntiDetection),
		boolToInt(a.PreferBagSeeds),
		a.PlantingStrategy,
		a.APIKey,
		now, now)
	if err != nil {
		return err
	}
	a.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) UpdateAccount(a *model.Account) error {
	a.UpdatedAt = time.Now()
	_, err := s.db.Exec(`UPDATE accounts SET
		name=?, platform=?, code=?, auto_start=?,
		farm_interval=?, friend_interval=?, enable_steal=?, force_lowest=?,
		enable_harvest=?, enable_plant=?, enable_sell=?, enable_weed=?, enable_bug=?, enable_water=?,
		enable_remove_dead=?, enable_upgrade_land=?, enable_help_friend=?, enable_claim_task=?,
		plant_crop_id=?, sell_crop_ids=?, steal_crop_ids=?,
		auto_use_fertilizer=?, auto_buy_fertilizer=?, fertilizer_target_count=?, fertilizer_buy_daily_limit=?,
		enable_anti_detection=?,
		prefer_bag_seeds=?,
		planting_strategy=?,
		api_key=?,
		updated_at=?
	WHERE id=?`,
		a.Name, a.Platform, a.Code, boolToInt(a.AutoStart),
		a.FarmInterval, a.FriendInterval, boolToInt(a.EnableSteal), boolToInt(a.ForceLowest),
		boolToInt(a.EnableHarvest), boolToInt(a.EnablePlant), boolToInt(a.EnableSell),
		boolToInt(a.EnableWeed), boolToInt(a.EnableBug), boolToInt(a.EnableWater),
		boolToInt(a.EnableRemoveDead), boolToInt(a.EnableUpgradeLand),
		boolToInt(a.EnableHelpFriend), boolToInt(a.EnableClaimTask),
		a.PlantCropID, a.SellCropIDs, a.StealCropIDs,
		boolToInt(a.AutoUseFertilizer), boolToInt(a.AutoBuyFertilizer),
		a.FertilizerTargetCount, a.FertilizerBuyDailyLimit,
		boolToInt(a.EnableAntiDetection),
		boolToInt(a.PreferBagSeeds),
		a.PlantingStrategy,
		a.APIKey,
		a.UpdatedAt, a.ID)
	return err
}

// UpdateAccountName updates only the display name of an account.
// Used by the bot to persist the name obtained from the game server after login.
func (s *Store) UpdateAccountName(id int64, name string) error {
	_, err := s.db.Exec(`UPDATE accounts SET name=?, updated_at=? WHERE id=?`, name, time.Now(), id)
	return err
}

func (s *Store) DeleteAccount(id int64) error {
	_, err := s.db.Exec(`DELETE FROM accounts WHERE id = ?`, id)
	if err != nil {
		return err
	}
	_, _ = s.db.Exec(`DELETE FROM logs WHERE account_id = ?`, id)
	return nil
}

// ============ Log ============

func (s *Store) AddLog(entry *model.LogEntry) error {
	entry.CreatedAt = time.Now()
	res, err := s.db.Exec(`INSERT INTO logs (account_id, tag, message, level, created_at) VALUES (?, ?, ?, ?, ?)`,
		entry.AccountID, entry.Tag, entry.Message, entry.Level, entry.CreatedAt)
	if err != nil {
		return err
	}
	entry.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) GetLogs(accountID int64, limit int, beforeID int64) ([]model.LogEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := `SELECT id, account_id, tag, message, level, created_at FROM logs WHERE account_id = ?`
	args := []interface{}{accountID}
	if beforeID > 0 {
		query += ` AND id < ?`
		args = append(args, beforeID)
	}
	query += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []model.LogEntry
	for rows.Next() {
		var l model.LogEntry
		if err := rows.Scan(&l.ID, &l.AccountID, &l.Tag, &l.Message, &l.Level, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}

func (s *Store) CleanOldLogs(days int) error {
	cutoff := time.Now().AddDate(0, 0, -days)
	_, err := s.db.Exec(`DELETE FROM logs WHERE created_at < ?`, cutoff)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ============ User CRUD ============

func (s *Store) CreateUser(u *model.User) error {
	now := time.Now()
	u.CreatedAt = now
	res, err := s.db.Exec(`INSERT INTO users (username, password_hash, is_admin, created_at) VALUES (?, ?, ?, ?)`,
		u.Username, u.PasswordHash, boolToInt(u.IsAdmin), now)
	if err != nil {
		return err
	}
	u.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) GetUserByID(id int64) (*model.User, error) {
	var u model.User
	var isAdmin int
	err := s.db.QueryRow(`SELECT id, username, password_hash, is_admin, created_at FROM users WHERE id = ?`, id).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &isAdmin, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	u.IsAdmin = isAdmin == 1
	return &u, nil
}

func (s *Store) GetUserByUsername(username string) (*model.User, error) {
	var u model.User
	var isAdmin int
	err := s.db.QueryRow(`SELECT id, username, password_hash, is_admin, created_at FROM users WHERE username = ?`, username).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &isAdmin, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	u.IsAdmin = isAdmin == 1
	return &u, nil
}

func (s *Store) UserExists(username string) (bool, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM users WHERE username = ?`, username).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Store) HasAnyUser() (bool, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ============ Operation Stats ============

// AddOpStat inserts a single operation statistics record.
func (s *Store) AddOpStat(r *model.OpRecord) error {
	r.CreatedAt = time.Now()
	_, err := s.db.Exec(
		`INSERT INTO op_stats (account_id, op_type, count, gold_delta, exp_delta, detail, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.AccountID, r.OpType, r.Count, r.GoldDelta, r.ExpDelta, r.Detail, r.CreatedAt)
	return err
}

// GetOpStats returns aggregated operation statistics for an account.
// granularity: "hour", "day", "week", "all"
// from/to: optional time range filters (zero time means no filter)
func (s *Store) GetOpStats(accountID int64, granularity string, from, to time.Time) ([]model.AggregatedStats, error) {
	var periodExpr string
	switch granularity {
	case "hour":
		periodExpr = `strftime('%Y-%m-%d %H:00', created_at)`
	case "day":
		periodExpr = `strftime('%Y-%m-%d', created_at)`
	case "week":
		periodExpr = `strftime('%Y-W%W', created_at)`
	default:
		periodExpr = `'all'`
	}

	query := `SELECT ` + periodExpr + ` as period, op_type, SUM(count) as total_count,
		SUM(CASE WHEN gold_delta > 0 THEN gold_delta ELSE 0 END) as gold_in,
		SUM(CASE WHEN gold_delta < 0 THEN -gold_delta ELSE 0 END) as gold_out,
		SUM(CASE WHEN exp_delta > 0 THEN exp_delta ELSE 0 END) as exp_gained
		FROM op_stats WHERE account_id = ?`
	args := []interface{}{accountID}

	if !from.IsZero() {
		query += ` AND created_at >= ?`
		args = append(args, from)
	}
	if !to.IsZero() {
		query += ` AND created_at <= ?`
		args = append(args, to)
	}

	query += ` GROUP BY period, op_type ORDER BY period ASC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Aggregate rows into per-period buckets
	bucketMap := make(map[string]*model.AggregatedStats)
	var orderedPeriods []string

	for rows.Next() {
		var period, opType string
		var totalCount, goldIn, goldOut, expGained int64
		if err := rows.Scan(&period, &opType, &totalCount, &goldIn, &goldOut, &expGained); err != nil {
			return nil, err
		}
		bucket, ok := bucketMap[period]
		if !ok {
			bucket = &model.AggregatedStats{
				Period:   period,
				OpCounts: make(map[string]int64),
			}
			bucketMap[period] = bucket
			orderedPeriods = append(orderedPeriods, period)
		}
		bucket.OpCounts[opType] = totalCount
		bucket.GoldIn += goldIn
		bucket.GoldOut += goldOut
		bucket.ExpGained += expGained
	}

	result := make([]model.AggregatedStats, 0, len(orderedPeriods))
	for _, p := range orderedPeriods {
		result = append(result, *bucketMap[p])
	}
	return result, nil
}

// GetOpStatsSummary returns overall totals for an account (no time grouping).
func (s *Store) GetOpStatsSummary(accountID int64) (map[string]int64, int64, int64, int64, error) {
	rows, err := s.db.Query(
		`SELECT op_type, SUM(count), SUM(CASE WHEN gold_delta > 0 THEN gold_delta ELSE 0 END),
		SUM(CASE WHEN gold_delta < 0 THEN -gold_delta ELSE 0 END),
		SUM(CASE WHEN exp_delta > 0 THEN exp_delta ELSE 0 END)
		FROM op_stats WHERE account_id = ? GROUP BY op_type`, accountID)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	defer rows.Close()

	counts := make(map[string]int64)
	var totalGoldIn, totalGoldOut, totalExp int64
	for rows.Next() {
		var opType string
		var count, goldIn, goldOut, expGained int64
		if err := rows.Scan(&opType, &count, &goldIn, &goldOut, &expGained); err != nil {
			return nil, 0, 0, 0, err
		}
		counts[opType] = count
		totalGoldIn += goldIn
		totalGoldOut += goldOut
		totalExp += expGained
	}
	return counts, totalGoldIn, totalGoldOut, totalExp, nil
}

// CleanOldOpStats removes operation stats older than the given number of days.
func (s *Store) CleanOldOpStats(days int) error {
	cutoff := time.Now().AddDate(0, 0, -days)
	_, err := s.db.Exec(`DELETE FROM op_stats WHERE created_at < ?`, cutoff)
	return err
}

// ============ Data Summary Queries ============

// DataSummaryTotals holds the top-level summary numbers for the data summary page.
type DataSummaryTotals struct {
	TotalHarvestCount int64 `json:"total_harvest_count"`
	TotalHarvestGold  int64 `json:"total_harvest_gold"`
	TotalStealCount   int64 `json:"total_steal_count"`
	TotalStealGold    int64 `json:"total_steal_gold"`
}

// GetDataSummaryTotals returns aggregated harvest/steal totals for an account within a time range.
func (s *Store) GetDataSummaryTotals(accountID int64, since time.Time) (*DataSummaryTotals, error) {
	var t DataSummaryTotals
	err := s.db.QueryRow(`
		SELECT
			COALESCE(SUM(CASE WHEN op_type='harvest' THEN count ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN op_type='sell' THEN gold_delta ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN op_type='steal' THEN count ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN op_type='steal' THEN gold_delta ELSE 0 END), 0)
		FROM op_stats WHERE account_id = ? AND created_at >= ?`,
		accountID, since).Scan(&t.TotalHarvestCount, &t.TotalHarvestGold, &t.TotalStealCount, &t.TotalStealGold)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// HourlyTrendRow represents one hour's harvest/steal data.
type HourlyTrendRow struct {
	Hour         string `json:"hour"`
	HarvestCount int64  `json:"harvest_count"`
	HarvestGold  int64  `json:"harvest_gold"`
	StealCount   int64  `json:"steal_count"`
	StealGold    int64  `json:"steal_gold"`
}

// GetHourlyTrend returns per-hour harvest/steal data for the last N hours.
func (s *Store) GetHourlyTrend(accountID int64, since time.Time) ([]HourlyTrendRow, error) {
	rows, err := s.db.Query(`
		SELECT
			strftime('%Y-%m-%d %H:00', created_at) AS hour,
			COALESCE(SUM(CASE WHEN op_type='harvest' THEN count ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN op_type='sell' THEN gold_delta ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN op_type='steal' THEN count ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN op_type='steal' THEN gold_delta ELSE 0 END), 0)
		FROM op_stats
		WHERE account_id = ? AND created_at >= ?
		  AND op_type IN ('harvest','sell','steal')
		GROUP BY hour ORDER BY hour ASC`,
		accountID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []HourlyTrendRow
	for rows.Next() {
		var r HourlyTrendRow
		if err := rows.Scan(&r.Hour, &r.HarvestCount, &r.HarvestGold, &r.StealCount, &r.StealGold); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, nil
}

// CropBreakdownRow represents one crop's contribution to revenue.
type CropBreakdownRow struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
	Gold  int64  `json:"gold"`
}

// GetCropBreakdown returns crop-level sell breakdown by parsing the detail field.
// detail format: "白萝卜x10, 核桃x5"
func (s *Store) GetCropBreakdown(accountID int64, since time.Time) ([]CropBreakdownRow, error) {
	rows, err := s.db.Query(`
		SELECT detail, SUM(count) AS total_count, SUM(gold_delta) AS total_gold
		FROM op_stats
		WHERE account_id = ? AND created_at >= ? AND op_type = 'sell' AND detail != ''
		GROUP BY detail ORDER BY total_gold DESC`,
		accountID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []CropBreakdownRow
	for rows.Next() {
		var detail string
		var count, gold int64
		if err := rows.Scan(&detail, &count, &gold); err != nil {
			return nil, err
		}
		result = append(result, CropBreakdownRow{Name: detail, Count: count, Gold: gold})
	}
	return result, nil
}

// StealRankingRow represents one friend's steal stats.
type StealRankingRow struct {
	FriendName string `json:"friend_name"`
	StealCount int64  `json:"steal_count"`
	StealGold  int64  `json:"steal_gold"`
}

// GetStealRanking returns friends ranked by steal count.
func (s *Store) GetStealRanking(accountID int64, since time.Time) ([]StealRankingRow, error) {
	rows, err := s.db.Query(`
		SELECT detail, SUM(count) AS total_count, SUM(gold_delta) AS total_gold
		FROM op_stats
		WHERE account_id = ? AND created_at >= ? AND op_type = 'steal' AND detail != ''
		GROUP BY detail ORDER BY total_count DESC
		LIMIT 20`,
		accountID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []StealRankingRow
	for rows.Next() {
		var r StealRankingRow
		if err := rows.Scan(&r.FriendName, &r.StealCount, &r.StealGold); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, nil
}

// DailySummaryRow represents one day's aggregated stats.
type DailySummaryRow struct {
	Date         string `json:"date"`
	HarvestCount int64  `json:"harvest_count"`
	HarvestGold  int64  `json:"harvest_gold"`
	StealCount   int64  `json:"steal_count"`
	StealGold    int64  `json:"steal_gold"`
	TotalGold    int64  `json:"total_gold"`
}

// GetDailySummary returns per-day summary for the last N days.
func (s *Store) GetDailySummary(accountID int64, since time.Time) ([]DailySummaryRow, error) {
	rows, err := s.db.Query(`
		SELECT
			strftime('%Y-%m-%d', created_at) AS day,
			COALESCE(SUM(CASE WHEN op_type='harvest' THEN count ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN op_type='sell' THEN gold_delta ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN op_type='steal' THEN count ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN op_type='steal' THEN gold_delta ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN gold_delta > 0 THEN gold_delta ELSE 0 END), 0)
		FROM op_stats
		WHERE account_id = ? AND created_at >= ?
		GROUP BY day ORDER BY day DESC`,
		accountID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []DailySummaryRow
	for rows.Next() {
		var r DailySummaryRow
		if err := rows.Scan(&r.Date, &r.HarvestCount, &r.HarvestGold, &r.StealCount, &r.StealGold, &r.TotalGold); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, nil
}
