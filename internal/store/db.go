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

	return err
}

// ============ Account CRUD ============

func (s *Store) ListAccounts() ([]model.Account, error) {
	rows, err := s.db.Query(`SELECT id, user_id, name, platform, code, auto_start, farm_interval, friend_interval, enable_steal, force_lowest, auto_use_fertilizer, auto_buy_fertilizer, fertilizer_target_count, fertilizer_buy_daily_limit, created_at, updated_at FROM accounts ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []model.Account
	for rows.Next() {
		var a model.Account
		var autoStart, enableSteal, forceLowest, autoUseFert, autoBuyFert int
		if err := rows.Scan(&a.ID, &a.UserID, &a.Name, &a.Platform, &a.Code, &autoStart, &a.FarmInterval, &a.FriendInterval, &enableSteal, &forceLowest, &autoUseFert, &autoBuyFert, &a.FertilizerTargetCount, &a.FertilizerBuyDailyLimit, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		a.AutoStart = autoStart == 1
		a.EnableSteal = enableSteal == 1
		a.ForceLowest = forceLowest == 1
		a.AutoUseFertilizer = autoUseFert == 1
		a.AutoBuyFertilizer = autoBuyFert == 1
		accounts = append(accounts, a)
	}
	return accounts, nil
}

func (s *Store) ListAccountsByUserID(userID int64) ([]model.Account, error) {
	rows, err := s.db.Query(`SELECT id, user_id, name, platform, code, auto_start, farm_interval, friend_interval, enable_steal, force_lowest, auto_use_fertilizer, auto_buy_fertilizer, fertilizer_target_count, fertilizer_buy_daily_limit, created_at, updated_at FROM accounts WHERE user_id = ? ORDER BY id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []model.Account
	for rows.Next() {
		var a model.Account
		var autoStart, enableSteal, forceLowest, autoUseFert, autoBuyFert int
		if err := rows.Scan(&a.ID, &a.UserID, &a.Name, &a.Platform, &a.Code, &autoStart, &a.FarmInterval, &a.FriendInterval, &enableSteal, &forceLowest, &autoUseFert, &autoBuyFert, &a.FertilizerTargetCount, &a.FertilizerBuyDailyLimit, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		a.AutoStart = autoStart == 1
		a.EnableSteal = enableSteal == 1
		a.ForceLowest = forceLowest == 1
		a.AutoUseFertilizer = autoUseFert == 1
		a.AutoBuyFertilizer = autoBuyFert == 1
		accounts = append(accounts, a)
	}
	return accounts, nil
}

func (s *Store) GetAccount(id int64) (*model.Account, error) {
	var a model.Account
	var autoStart, enableSteal, forceLowest, autoUseFert, autoBuyFert int
	err := s.db.QueryRow(`SELECT id, user_id, name, platform, code, auto_start, farm_interval, friend_interval, enable_steal, force_lowest, auto_use_fertilizer, auto_buy_fertilizer, fertilizer_target_count, fertilizer_buy_daily_limit, created_at, updated_at FROM accounts WHERE id = ?`, id).
		Scan(&a.ID, &a.UserID, &a.Name, &a.Platform, &a.Code, &autoStart, &a.FarmInterval, &a.FriendInterval, &enableSteal, &forceLowest, &autoUseFert, &autoBuyFert, &a.FertilizerTargetCount, &a.FertilizerBuyDailyLimit, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	a.AutoStart = autoStart == 1
	a.EnableSteal = enableSteal == 1
	a.ForceLowest = forceLowest == 1
	a.AutoUseFertilizer = autoUseFert == 1
	a.AutoBuyFertilizer = autoBuyFert == 1
	return &a, nil
}

func (s *Store) CreateAccount(a *model.Account) error {
	now := time.Now()
	a.CreatedAt = now
	a.UpdatedAt = now
	res, err := s.db.Exec(`INSERT INTO accounts (user_id, name, platform, code, auto_start, farm_interval, friend_interval, enable_steal, force_lowest, auto_use_fertilizer, auto_buy_fertilizer, fertilizer_target_count, fertilizer_buy_daily_limit, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.UserID, a.Name, a.Platform, a.Code, boolToInt(a.AutoStart), a.FarmInterval, a.FriendInterval, boolToInt(a.EnableSteal), boolToInt(a.ForceLowest), boolToInt(a.AutoUseFertilizer), boolToInt(a.AutoBuyFertilizer), a.FertilizerTargetCount, a.FertilizerBuyDailyLimit, now, now)
	if err != nil {
		return err
	}
	a.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) UpdateAccount(a *model.Account) error {
	a.UpdatedAt = time.Now()
	_, err := s.db.Exec(`UPDATE accounts SET name=?, platform=?, code=?, auto_start=?, farm_interval=?, friend_interval=?, enable_steal=?, force_lowest=?, auto_use_fertilizer=?, auto_buy_fertilizer=?, fertilizer_target_count=?, fertilizer_buy_daily_limit=?, updated_at=? WHERE id=?`,
		a.Name, a.Platform, a.Code, boolToInt(a.AutoStart), a.FarmInterval, a.FriendInterval, boolToInt(a.EnableSteal), boolToInt(a.ForceLowest), boolToInt(a.AutoUseFertilizer), boolToInt(a.AutoBuyFertilizer), a.FertilizerTargetCount, a.FertilizerBuyDailyLimit, a.UpdatedAt, a.ID)
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
