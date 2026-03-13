package bot

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"qq-farm-bot/internal/model"
	"qq-farm-bot/internal/store"
)

// BotConfig holds the runtime configuration for a bot instance.
type BotConfig struct {
	Platform                string
	Code                    string
	ServerURL               string
	ClientVersion           string
	FarmInterval            int // seconds
	FriendInterval          int // seconds
	EnableSteal             bool
	ForceLowest             bool
	AutoUseFertilizer       bool
	AutoBuyFertilizer       bool
	FertilizerTargetCount   int
	FertilizerBuyDailyLimit int

	// Farm automation toggles
	EnableHarvest     bool
	EnablePlant       bool
	EnableSell        bool
	EnableWeed        bool
	EnableBug         bool
	EnableWater       bool
	EnableRemoveDead  bool
	EnableUpgradeLand bool
	EnableHelpFriend  bool
	EnableClaimTask   bool

	// Crop selection & filtering
	PlantCropID  int    // specific crop to plant (0 = auto)
	SellCropIDs  string // comma-separated crop IDs to sell (empty = all)
	StealCropIDs string // comma-separated crop IDs to steal (empty = all)
	// Planting preference
	PreferBagSeeds bool // prioritize planting seeds from bag
	// Anti-detection
	EnableAntiDetection bool
	// Planting strategy
	PlantingStrategy string
	// Debug
	EnableDebugLog bool
}

const (
	reconnectBackoffInit    = 2 * time.Second
	reconnectBackoffMax     = 60 * time.Second
	maxLoginTimeoutAttempts = 3
)

// connectError wraps a connection/login failure with the disconnect reason
// so the watchdog can apply differentiated retry strategies.
type connectError struct {
	reason DisconnectReason
	err    error
}

func (e *connectError) Error() string { return e.err.Error() }
func (e *connectError) Unwrap() error { return e.err }

// Instance represents a running bot for a single game account.
type Instance struct {
	mu      sync.RWMutex
	account *model.Account
	config  *BotConfig
	net     *Network
	logger  *Logger
	store   *store.Store
	crypto  *Crypto
	stats   *BotStats
	lands   *LandCache
	sc      *StatsCollector
	running bool
	startAt time.Time
	err     string

	stopCh chan struct{} // signals watchdog to stop
}

func NewInstance(account *model.Account, serverURL, clientVersion string, s *store.Store, crypto *Crypto) *Instance {
	cfg := &BotConfig{
		Platform:                account.Platform,
		Code:                    account.Code,
		ServerURL:               serverURL,
		ClientVersion:           clientVersion,
		FarmInterval:            account.FarmInterval,
		FriendInterval:          account.FriendInterval,
		EnableSteal:             account.EnableSteal,
		ForceLowest:             account.ForceLowest,
		AutoUseFertilizer:       account.AutoUseFertilizer,
		AutoBuyFertilizer:       account.AutoBuyFertilizer,
		FertilizerTargetCount:   account.FertilizerTargetCount,
		FertilizerBuyDailyLimit: account.FertilizerBuyDailyLimit,

		// Farm automation toggles
		EnableHarvest:     account.EnableHarvest,
		EnablePlant:       account.EnablePlant,
		EnableSell:        account.EnableSell,
		EnableWeed:        account.EnableWeed,
		EnableBug:         account.EnableBug,
		EnableWater:       account.EnableWater,
		EnableRemoveDead:  account.EnableRemoveDead,
		EnableUpgradeLand: account.EnableUpgradeLand,
		EnableHelpFriend:  account.EnableHelpFriend,
		EnableClaimTask:   account.EnableClaimTask,

		// Crop selection & filtering
		PlantCropID:      account.PlantCropID,
		SellCropIDs:      account.SellCropIDs,
		StealCropIDs:     account.StealCropIDs,
		PreferBagSeeds:   account.PreferBagSeeds,
		PlantingStrategy: account.PlantingStrategy,

		EnableAntiDetection: account.EnableAntiDetection,
		EnableDebugLog:      account.EnableDebugLog,
	}
	if cfg.FarmInterval < 1 {
		cfg.FarmInterval = 10
	}
	if cfg.FriendInterval < 1 {
		cfg.FriendInterval = 10
	}

	logger := NewLogger(account.ID, s)
	logger.SetDebug(cfg.EnableDebugLog)

	return &Instance{
		account: account,
		config:  cfg,
		logger:  logger,
		store:   s,
		stats:   &BotStats{},
		lands:   NewLandCache(),
		crypto:  crypto,
		sc:      NewStatsCollector(account.ID, s),
	}
}

func (inst *Instance) Start() error {
	inst.mu.Lock()
	if inst.running {
		inst.mu.Unlock()
		return fmt.Errorf("bot already running")
	}
	inst.stopCh = make(chan struct{})
	inst.mu.Unlock()

	if err := inst.connectAndRun(); err != nil {
		return err
	}

	// Start watchdog for auto-reconnection
	go inst.watchdog()

	return nil
}

// connectAndRun creates a new Network, connects, logs in, and starts all workers.
func (inst *Instance) connectAndRun() error {
	net := NewNetwork(inst.logger, inst.crypto)

	// Connect
	inst.logger.Infof("启动", "正在连接 %s 平台...", inst.config.Platform)
	if err := net.Connect(inst.config.ServerURL, inst.config.Platform, inst.config.ClientVersion, inst.config.Code); err != nil {
		inst.mu.Lock()
		inst.err = err.Error()
		inst.mu.Unlock()
		return fmt.Errorf("connect: %w", err)
	}

	if err := net.Login(inst.config.ClientVersion); err != nil {
		reason := net.GetDisconnectReason()
		net.Close()
		inst.mu.Lock()
		inst.err = err.Error()
		inst.mu.Unlock()
		return &connectError{reason: reason, err: fmt.Errorf("login: %w", err)}
	}

	inst.mu.Lock()
	inst.net = net
	inst.running = true
	inst.startAt = time.Now()
	inst.err = ""
	inst.mu.Unlock()

	// After login, persist account name from game server to database
	_, _, _, _, loginName := net.state.Get()
	if loginName != "" && inst.store != nil {
		inst.mu.Lock()
		needsUpdate := loginName != inst.account.Name
		if needsUpdate {
			inst.account.Name = loginName
		}
		inst.mu.Unlock()
		if needsUpdate {
			inst.store.UpdateAccountName(inst.account.ID, loginName)
		}
	}

	// Start heartbeat
	net.StartHeartbeat(inst.config.ClientVersion, 25*time.Second)

	// Start workers
	farm := NewFarmWorker(net, inst.logger, inst.config, inst.lands, inst.sc)
	go farm.RunLoop()

	friend := NewFriendWorker(net, inst.logger, inst.config, inst.stats, inst.sc)
	go friend.RunLoop()

	task := NewTaskWorker(net, inst.logger, inst.config, inst.sc)
	go task.RunLoop()

	warehouse := NewWarehouseWorker(net, inst.logger, inst.config, inst.sc)
	go warehouse.RunLoop()

	fertilizer := NewFertilizerWorker(net, inst.logger, inst.config, inst.sc)
	go fertilizer.RunLoop()

	return nil
}

func (inst *Instance) watchdog() {
	backoff := reconnectBackoffInit
	loginTimeoutCount := 0

	for {
		inst.mu.RLock()
		net := inst.net
		inst.mu.RUnlock()

		if net == nil {
			return
		}

		select {
		case <-net.Done():
		case <-inst.stopCh:
			return
		}

		reason := net.GetDisconnectReason()
		inst.mu.Lock()
		inst.running = false
		inst.mu.Unlock()

		if !reason.Retryable() {
			inst.logger.Warnf("系统", "连接断开 (reason=%s)，不再重连", reason)
			inst.mu.Lock()
			inst.err = fmt.Sprintf("断开: %s", reason)
			inst.mu.Unlock()
			return
		}

		if reason == DisconnectLoginTimeout {
			loginTimeoutCount++
			if loginTimeoutCount >= maxLoginTimeoutAttempts {
				inst.logger.Warnf("系统", "登录超时累计 %d 次，停止重连", loginTimeoutCount)
				inst.mu.Lock()
				inst.err = fmt.Sprintf("登录超时达上限 (%d/%d)", loginTimeoutCount, maxLoginTimeoutAttempts)
				inst.mu.Unlock()
				return
			}
		}

		inst.logger.Warnf("系统", "连接断开 (reason=%s)，%v 后尝试重连...", reason, backoff)

		// Reconnect loop: retry with exponential backoff until success or stop.
		for {
			select {
			case <-time.After(backoff):
			case <-inst.stopCh:
				inst.logger.Info("系统", "Bot 已停止")
				return
			}

			err := inst.connectAndRun()
			if err == nil {
				inst.logger.Infof("重连", "成功")
				backoff = reconnectBackoffInit
				loginTimeoutCount = 0
				break
			}

			// Check if reconnection failed due to login timeout.
			var ce *connectError
			if errors.As(err, &ce) && ce.reason == DisconnectLoginTimeout {
				loginTimeoutCount++
				if loginTimeoutCount >= maxLoginTimeoutAttempts {
					inst.logger.Warnf("系统", "登录超时累计 %d 次，停止重连", loginTimeoutCount)
					inst.mu.Lock()
					inst.err = fmt.Sprintf("登录超时达上限 (%d/%d)", loginTimeoutCount, maxLoginTimeoutAttempts)
					inst.mu.Unlock()
					return
				}
			}

			inst.logger.Warnf("重连", "失败: %v", err)
			backoff *= 2
			if backoff > reconnectBackoffMax {
				backoff = reconnectBackoffMax
			}
		}
	}
}

func (inst *Instance) Stop() {
	inst.mu.Lock()
	defer inst.mu.Unlock()

	// Signal watchdog to stop
	if inst.stopCh != nil {
		select {
		case <-inst.stopCh:
			// already closed
		default:
			close(inst.stopCh)
		}
	}

	if inst.net != nil {
		inst.net.Close()
	}
	inst.running = false
}

func (inst *Instance) Status() *model.BotStatus {
	inst.mu.Lock()
	defer inst.mu.Unlock()

	s := &model.BotStatus{
		AccountID: inst.account.ID,
		Running:   inst.running,
		Platform:  inst.config.Platform,
		Error:     inst.err,
	}

	// Read state from net even when stopped — net object is closed but not nil'd,
	// so state persists after disconnect/stop.
	if inst.net != nil {
		gid, level, exp, gold, name := inst.net.state.Get()
		s.GID = gid
		s.Name = name
		s.Level = level
		s.Exp = exp
		s.Gold = gold
	}

	if !inst.startAt.IsZero() {
		startAt := inst.startAt
		s.StartedAt = &startAt
	}

	// Calculate level up estimation only when running
	if inst.running && s.Level > 0 {
		gc := GetGameConfig()
		if gc != nil {
			if nextExp, hasNext := gc.GetNextLevelExp(int(s.Level)); hasNext {
				s.NextLevelExp = nextExp
				s.ExpToNextLevel = nextExp - s.Exp
				if s.ExpToNextLevel < 0 {
					s.ExpToNextLevel = 0
				}
				s.ExpRatePerHour, s.HoursToNextLevel = inst.estimateLevelUp(s.ExpToNextLevel)
			}
		}
	}

	if inst.stats != nil {
		s.TotalSteal = inst.stats.TotalSteal
		s.TotalHelp = inst.stats.TotalHelp
		s.FriendsCount = inst.stats.FriendsCount
	}

	if inst.lands != nil {
		totalLands, unlockedLands, landStatuses := inst.lands.Get()
		s.TotalLands = totalLands
		s.UnlockedLands = unlockedLands
		s.Lands = landStatuses
	}

	return s
}

// effectiveGrowSec computes growth time after applying land time-reduction buff
// and subtracting fertilizer skip time (longest-phase optimization).
func effectiveGrowSec(baseSec, fertReduceSec int, timeReducePct int64) int64 {
	base := int64(baseSec)
	if timeReducePct > 0 {
		base = base * (10000 - timeReducePct) / 10000
	}
	fert := int64(fertReduceSec)
	if timeReducePct > 0 {
		fert = fert * (10000 - timeReducePct) / 10000
	}
	eff := base - fert
	if eff < 1 {
		eff = 1
	}
	return eff
}

// resolveStrategySeed determines which seed the bot would plant based on the
// current planting configuration (explicit crop ID, strategy rules, or defaults).
// Uses only static GameConfig data — no network calls required.
// Returns nil if no suitable seed can be determined.
func (inst *Instance) resolveStrategySeed(gc *GameConfig) *SeedYieldRow {
	if gc == nil {
		return nil
	}

	_, level, _, _, _ := inst.net.state.Get()
	yieldRows := gc.GetSeedYieldRows()
	if len(yieldRows) == 0 {
		return nil
	}

	var available []SeedYieldRow
	for _, yr := range yieldRows {
		if yr.RequiredLevel <= int(level) && yr.GrowTimeSec > 0 {
			available = append(available, yr)
		}
	}
	if len(available) == 0 {
		return nil
	}

	// 1. Explicit crop ID override
	if inst.config.PlantCropID > 0 {
		targetSeedID := gc.GetSeedIDForCrop(inst.config.PlantCropID)
		if targetSeedID > 0 {
			for i, yr := range available {
				if yr.SeedID == targetSeedID {
					return &available[i]
				}
			}
		}
	}

	// 2. Strategy-based selection
	strategy := ParsePlantingStrategy(inst.config.PlantingStrategy)
	if strategy != nil {
		if strategy.Mode == StrategyModeFastestLevelUp {
			// fastest_levelup does per-round optimization; approximate with best exp efficiency
			best := &available[0]
			for i := 1; i < len(available); i++ {
				if available[i].FarmExpPerHourNormal > best.FarmExpPerHourNormal {
					best = &available[i]
				}
			}
			return best
		}

		var candidates []SeedCandidate
		for _, yr := range available {
			sc := SeedCandidate{
				SeedID:             yr.SeedID,
				Name:               yr.Name,
				RequiredLevel:      yr.RequiredLevel,
				Price:              yr.Price,
				ExpPerHarvest:      yr.ExpHarvest,
				Seasons:            yr.Seasons,
				GrowTimeSec:        yr.GrowTimeSec,
				ExpEfficiency:      yr.FarmExpPerHourNormal,
				GrowTimeNormalFert: yr.GrowTimeNormalFert,
			}
			if yr.Price > 0 {
				sc.GoldEfficiency = float64(yr.ExpHarvest*yr.Seasons) / float64(yr.Price)
			}
			candidates = append(candidates, sc)
		}

		result := ApplyStrategy(strategy, candidates)
		if len(result) > 0 {
			for i, yr := range available {
				if yr.SeedID == result[0].SeedID {
					return &available[i]
				}
			}
		}
	}

	// 3. ForceLowest: pick lowest required level seed
	if inst.config.ForceLowest {
		best := &available[0]
		for i := 1; i < len(available); i++ {
			if available[i].RequiredLevel < best.RequiredLevel ||
				(available[i].RequiredLevel == best.RequiredLevel && available[i].Price < best.Price) {
				best = &available[i]
			}
		}
		return best
	}

	// 4. Default: best exp efficiency (matches findBestSeed fallback)
	best := &available[0]
	for i := 1; i < len(available); i++ {
		if available[i].FarmExpPerHourNormal > best.FarmExpPerHourNormal {
			best = &available[i]
		}
	}
	return best
}

// estimateLevelUp calculates expected exp rate and hours to next level using a
// time-series simulation. It builds discrete harvest events from currently
// growing crops, then simulates future planting cycles using the configured
// planting strategy to produce an accurate level-up timeline.
func (inst *Instance) estimateLevelUp(expToNextLevel int64) (expRatePerHour float64, hoursToNextLevel float64) {
	if inst.lands == nil || expToNextLevel <= 0 {
		return 0, 0
	}

	harvestInfos := inst.lands.GetHarvestInfo()
	gc := GetGameConfig()
	nowSec := time.Now().Unix()

	// --- Phase 1: Build events from current crops + track per-land free times ---

	type harvestEvent struct {
		timeSec int64
		exp     int64
	}
	type landState struct {
		landID        int64
		freeTimeSec   int64 // when this land becomes available for replanting
		expBonusPct   int64
		timeReducePct int64
	}

	var events []harvestEvent
	var landStates []landState
	var totalExpPerMin float64

	for _, h := range harvestInfos {
		adjustedExp := float64(h.CropExp) * (10000 + float64(h.ExpBonusPct)) / 10000.0
		if adjustedExp <= 0 {
			continue
		}

		seasons := 1
		var season2GrowSec int64
		if gc != nil && h.CropID > 0 {
			seasons = gc.GetPlantSeasons(int(h.CropID))
			if seasons >= 2 {
				if pd := gc.GetPlantPhaseData(int(h.CropID)); pd != nil && pd.Season2GrowTime > 0 {
					season2GrowSec = effectiveGrowSec(pd.Season2GrowTime, pd.Season2MaxPhase, h.TimeReducePct)
				}
			}
		}

		if h.CycleTimeSec > 0 {
			totalCycleExp := adjustedExp * float64(seasons)
			totalCycleSec := float64(h.CycleTimeSec)
			if seasons >= 2 && season2GrowSec > 0 {
				totalCycleSec += float64(season2GrowSec)
			}
			totalExpPerMin += totalCycleExp / (totalCycleSec / 60.0)
		}

		currentSeason := h.Season
		if currentSeason < 1 {
			currentSeason = 1
		}

		var lastHarvestTime int64

		if h.IsMature {
			events = append(events, harvestEvent{timeSec: nowSec, exp: int64(adjustedExp)})
			lastHarvestTime = nowSec
			if currentSeason <= 1 && seasons >= 2 && season2GrowSec > 0 {
				s2Time := nowSec + season2GrowSec
				events = append(events, harvestEvent{timeSec: s2Time, exp: int64(adjustedExp)})
				lastHarvestTime = s2Time
			}
		} else if h.IsGrowing && h.MatureTimeSec > nowSec {
			events = append(events, harvestEvent{timeSec: h.MatureTimeSec, exp: int64(adjustedExp)})
			lastHarvestTime = h.MatureTimeSec
			if currentSeason <= 1 && seasons >= 2 && season2GrowSec > 0 {
				s2Time := h.MatureTimeSec + season2GrowSec
				events = append(events, harvestEvent{timeSec: s2Time, exp: int64(adjustedExp)})
				lastHarvestTime = s2Time
			}
		}

		if lastHarvestTime > 0 {
			landStates = append(landStates, landState{
				landID:        h.LandID,
				freeTimeSec:   lastHarvestTime,
				expBonusPct:   h.ExpBonusPct,
				timeReducePct: h.TimeReducePct,
			})
		}
	}

	// --- Phase 2: Include empty/idle lands in the simulation ---
	_, _, landStatuses := inst.lands.Get()
	trackedLands := make(map[int64]bool, len(landStates))
	for _, ls := range landStates {
		trackedLands[ls.landID] = true
	}
	for _, ls := range landStatuses {
		if !ls.Unlocked || trackedLands[ls.ID] || ls.CropID > 0 {
			continue
		}
		// Skip slave lands occupied by a master's crop
		if ls.MasterLandID > 0 && ls.MasterLandID != ls.ID && trackedLands[ls.MasterLandID] {
			continue
		}
		landStates = append(landStates, landState{
			landID:        ls.ID,
			freeTimeSec:   nowSec,
			expBonusPct:   ls.ExpBonusPct,
			timeReducePct: ls.TimeReducePct,
		})
	}

	if len(landStates) == 0 {
		return 0, 0
	}

	expRatePerHour = totalExpPerMin * 60

	// --- Phase 3: Resolve strategy seed for future planting cycles ---
	strategySeed := inst.resolveStrategySeed(gc)

	// --- Phase 4: Simulate future planting cycles using the strategy seed ---
	if strategySeed != nil {
		s1GrowTimeSec := strategySeed.GrowTimeSec
		s1FertReduceSec := strategySeed.NormalFertReduceSec
		s2GrowTimeSec := strategySeed.Season2GrowTimeSec
		s2FertReduceSec := strategySeed.Season2FertReduceSec
		seedSeasons := strategySeed.Seasons
		seedExp := strategySeed.ExpHarvest

		maxSimTime := nowSec + 30*24*3600 // cap simulation at 30 days
		maxCycles := 50

		// Track cumulative exp for early termination
		var totalSimExp int64
		for _, e := range events {
			totalSimExp += e.exp
		}

		for cycle := 0; cycle < maxCycles && totalSimExp < expToNextLevel; cycle++ {
			for i := range landStates {
				ls := &landStates[i]
				if ls.freeTimeSec >= maxSimTime {
					continue
				}

				s1Eff := effectiveGrowSec(s1GrowTimeSec, s1FertReduceSec, ls.timeReducePct)
				adjExp := int64(seedExp) * (10000 + ls.expBonusPct) / 10000
				if adjExp <= 0 {
					adjExp = int64(seedExp)
				}

				harvest1Time := ls.freeTimeSec + s1Eff
				events = append(events, harvestEvent{timeSec: harvest1Time, exp: adjExp})
				totalSimExp += adjExp

				lastHarvest := harvest1Time
				if seedSeasons >= 2 && s2GrowTimeSec > 0 {
					s2Eff := effectiveGrowSec(s2GrowTimeSec, s2FertReduceSec, ls.timeReducePct)
					harvest2Time := harvest1Time + s2Eff
					events = append(events, harvestEvent{timeSec: harvest2Time, exp: adjExp})
					totalSimExp += adjExp
					lastHarvest = harvest2Time
				}

				ls.freeTimeSec = lastHarvest
			}
		}

		var stratExpPerMin float64
		for _, ls := range landStates {
			adjExp := float64(seedExp) * (10000 + float64(ls.expBonusPct)) / 10000.0
			s1Eff := effectiveGrowSec(s1GrowTimeSec, s1FertReduceSec, ls.timeReducePct)
			totalExp := adjExp * float64(seedSeasons)
			totalTime := float64(s1Eff)
			if seedSeasons >= 2 && s2GrowTimeSec > 0 {
				s2Eff := effectiveGrowSec(s2GrowTimeSec, s2FertReduceSec, ls.timeReducePct)
				totalTime += float64(s2Eff)
			}
			if totalTime > 0 {
				stratExpPerMin += totalExp / (totalTime / 60.0)
			}
		}
		if stratExpPerMin > 0 {
			expRatePerHour = stratExpPerMin * 60
		}
	}

	// --- Phase 5: Walk sorted events to find the level-up moment ---
	sort.Slice(events, func(i, j int) bool {
		return events[i].timeSec < events[j].timeSec
	})

	remaining := expToNextLevel
	lastEventTime := nowSec
	for _, e := range events {
		remaining -= e.exp
		if remaining <= 0 {
			secsUntil := e.timeSec - nowSec
			if secsUntil < 0 {
				secsUntil = 0
			}
			hoursToNextLevel = float64(secsUntil) / 3600.0
			return
		}
		lastEventTime = e.timeSec
	}

	// --- Phase 6: Fallback — use steady-state rate for any remaining exp ---
	if expRatePerHour > 0 {
		additionalSecs := float64(remaining) / (expRatePerHour / 3600.0)
		totalSecs := float64(lastEventTime-nowSec) + additionalSecs
		if totalSecs < 0 {
			totalSecs = 0
		}
		hoursToNextLevel = totalSecs / 3600.0
	}
	return
}

func (inst *Instance) Logger() *Logger {
	return inst.logger
}

func (inst *Instance) IsRunning() bool {
	inst.mu.RLock()
	defer inst.mu.RUnlock()
	return inst.running
}

// UpdateConfig applies updated account settings to the running bot config.
// Workers read config fields via the shared pointer each loop iteration,
// so updated values take effect on the next cycle automatically.
func (inst *Instance) UpdateConfig(account *model.Account) {
	inst.mu.Lock()
	defer inst.mu.Unlock()

	if inst.config == nil {
		return
	}

	inst.config.FarmInterval = account.FarmInterval
	if inst.config.FarmInterval < 1 {
		inst.config.FarmInterval = 10
	}
	inst.config.FriendInterval = account.FriendInterval
	if inst.config.FriendInterval < 1 {
		inst.config.FriendInterval = 10
	}

	inst.config.EnableSteal = account.EnableSteal
	inst.config.ForceLowest = account.ForceLowest
	inst.config.AutoUseFertilizer = account.AutoUseFertilizer
	inst.config.AutoBuyFertilizer = account.AutoBuyFertilizer
	inst.config.FertilizerTargetCount = account.FertilizerTargetCount
	inst.config.FertilizerBuyDailyLimit = account.FertilizerBuyDailyLimit

	inst.config.EnableHarvest = account.EnableHarvest
	inst.config.EnablePlant = account.EnablePlant
	inst.config.EnableSell = account.EnableSell
	inst.config.EnableWeed = account.EnableWeed
	inst.config.EnableBug = account.EnableBug
	inst.config.EnableWater = account.EnableWater
	inst.config.EnableRemoveDead = account.EnableRemoveDead
	inst.config.EnableUpgradeLand = account.EnableUpgradeLand
	inst.config.EnableHelpFriend = account.EnableHelpFriend
	inst.config.EnableClaimTask = account.EnableClaimTask

	inst.config.PlantCropID = account.PlantCropID
	inst.config.PlantingStrategy = account.PlantingStrategy
	inst.config.SellCropIDs = account.SellCropIDs
	inst.config.StealCropIDs = account.StealCropIDs
	inst.config.PreferBagSeeds = account.PreferBagSeeds

	inst.config.EnableAntiDetection = account.EnableAntiDetection

	inst.config.EnableDebugLog = account.EnableDebugLog
	if inst.logger != nil {
		inst.logger.SetDebug(account.EnableDebugLog)
	}
}
