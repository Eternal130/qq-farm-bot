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
	Platform       string
	Code           string
	ServerURL      string
	ClientVersion  string
	FarmInterval   int // seconds
	FriendInterval int // seconds
	EnableSteal    bool
	ForceLowest    bool
	AutoUseFertilizer      bool
	AutoBuyFertilizer      bool
	FertilizerTargetCount  int
	FertilizerBuyDailyLimit int
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
	stats   *BotStats
	lands   *LandCache
	running bool
	startAt time.Time
	err     string

	stopCh chan struct{} // signals watchdog to stop
}
func NewInstance(account *model.Account, serverURL, clientVersion string, s *store.Store) *Instance {
	cfg := &BotConfig{
		Platform:       account.Platform,
		Code:           account.Code,
		ServerURL:      serverURL,
		ClientVersion:  clientVersion,
		FarmInterval:   account.FarmInterval,
		FriendInterval: account.FriendInterval,
		EnableSteal:    account.EnableSteal,
		ForceLowest:    account.ForceLowest,
		AutoUseFertilizer:      account.AutoUseFertilizer,
		AutoBuyFertilizer:      account.AutoBuyFertilizer,
		FertilizerTargetCount:  account.FertilizerTargetCount,
		FertilizerBuyDailyLimit: account.FertilizerBuyDailyLimit,
	}
	if cfg.FarmInterval < 1 {
		cfg.FarmInterval = 10
	}
	if cfg.FriendInterval < 1 {
		cfg.FriendInterval = 10
	}

	return &Instance{
		account: account,
		config:  cfg,
		logger:  NewLogger(account.ID, s),
		stats:   &BotStats{},
		lands:   NewLandCache(),
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
	net := NewNetwork(inst.logger)

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

	// Start heartbeat
	net.StartHeartbeat(inst.config.ClientVersion, 25*time.Second)

	// Start workers
	farm := NewFarmWorker(net, inst.logger, inst.config, inst.lands)
	go farm.RunLoop()

	friend := NewFriendWorker(net, inst.logger, inst.config, inst.stats)
	go friend.RunLoop()

	task := NewTaskWorker(net, inst.logger)
	go task.RunLoop()

	warehouse := NewWarehouseWorker(net, inst.logger)
	go warehouse.RunLoop()

	fertilizer := NewFertilizerWorker(net, inst.logger, inst.config)
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

	if inst.running && inst.net != nil {
		gid, level, exp, gold, name := inst.net.state.Get()
		s.GID = gid
		s.Name = name
		s.Level = level
		s.Exp = exp
		s.Gold = gold
		startAt := inst.startAt
		s.StartedAt = &startAt

		// Calculate level up estimation from crop harvest data
		gc := GetGameConfig()
		if gc != nil {
			if nextExp, hasNext := gc.GetNextLevelExp(int(level)); hasNext {
				s.NextLevelExp = nextExp
				s.ExpToNextLevel = nextExp - exp
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

// estimateLevelUp calculates expected exp rate and hours to next level based on
// current crop data: harvest times, crop exp, and growth cycle times.
// Multi-season crops produce multiple harvests per planting cycle, which is factored
// into both the exp rate and the discrete harvest event timeline.
func (inst *Instance) estimateLevelUp(expToNextLevel int64) (expRatePerHour float64, hoursToNextLevel float64) {
	if inst.lands == nil || expToNextLevel <= 0 {
		return 0, 0
	}

	harvestInfos := inst.lands.GetHarvestInfo()
	if len(harvestInfos) == 0 {
		return 0, 0
	}

	gc := GetGameConfig()
	nowSec := time.Now().Unix()

	// Calculate total exp per minute rate from all planted lands
	var totalExpPerMin float64

	// Build discrete harvest events
	type harvestEvent struct {
		timeSec int64
		exp     int64
	}
	var events []harvestEvent

	for _, h := range harvestInfos {
		// Apply land exp bonus: server value is pct*100, e.g. 1000 = 10%
		adjustedExp := float64(h.CropExp) * (10000 + float64(h.ExpBonusPct)) / 10000.0
		if adjustedExp <= 0 {
			continue
		}

		// Determine season count and season 2 grow time for this crop
		seasons := 1
		var season2GrowSec int64
		if gc != nil && h.CropID > 0 {
			seasons = gc.GetPlantSeasons(int(h.CropID))
			if seasons >= 2 {
				if pd := gc.GetPlantPhaseData(int(h.CropID)); pd != nil && pd.Season2GrowTime > 0 {
					// Apply time reduction buff to season 2 grow time
					s2Base := int64(pd.Season2GrowTime)
					if h.TimeReducePct > 0 {
						s2Base = s2Base * (10000 - h.TimeReducePct) / 10000
					}
					// Subtract fert reduction for season 2 (max phase skipped)
					s2Fert := int64(pd.Season2MaxPhase)
					if h.TimeReducePct > 0 {
						s2Fert = s2Fert * (10000 - h.TimeReducePct) / 10000
					}
					season2GrowSec = s2Base - s2Fert
					if season2GrowSec < 1 {
						season2GrowSec = 1
					}
				}
			}
		}

		// Steady-state rate: total exp across all seasons / full cycle time
		if h.CycleTimeSec > 0 {
			totalCycleExp := adjustedExp * float64(seasons)
			totalCycleSec := float64(h.CycleTimeSec)
			if seasons >= 2 && season2GrowSec > 0 {
				totalCycleSec += float64(season2GrowSec)
			}
			expPerMin := totalCycleExp / (totalCycleSec / 60.0)
			totalExpPerMin += expPerMin
		}

		// Discrete harvest events for this land
		currentSeason := h.Season
		if currentSeason < 1 {
			currentSeason = 1
		}

		if h.IsMature {
			// Already mature — will be harvested very soon
			events = append(events, harvestEvent{timeSec: nowSec, exp: int64(adjustedExp)})
			// If season 1 mature on a multi-season crop, season 2 harvest follows
			if currentSeason <= 1 && seasons >= 2 && season2GrowSec > 0 {
				events = append(events, harvestEvent{timeSec: nowSec + season2GrowSec, exp: int64(adjustedExp)})
			}
		} else if h.IsGrowing && h.MatureTimeSec > nowSec {
			events = append(events, harvestEvent{timeSec: h.MatureTimeSec, exp: int64(adjustedExp)})
			// If growing in season 1 on a multi-season crop, season 2 harvest follows
			if currentSeason <= 1 && seasons >= 2 && season2GrowSec > 0 {
				events = append(events, harvestEvent{timeSec: h.MatureTimeSec + season2GrowSec, exp: int64(adjustedExp)})
			}
		}
	}

	if totalExpPerMin <= 0 {
		return 0, 0
	}

	expRatePerHour = totalExpPerMin * 60

	// Sort harvest events chronologically
	sort.Slice(events, func(i, j int) bool {
		return events[i].timeSec < events[j].timeSec
	})

	// Walk through harvest events — check if any batch triggers level-up
	remaining := expToNextLevel
	lastEventTime := nowSec
	for _, e := range events {
		remaining -= e.exp
		if remaining <= 0 {
			// Level up happens at this harvest
			secsUntil := e.timeSec - nowSec
			if secsUntil < 0 {
				secsUntil = 0
			}
			hoursToNextLevel = float64(secsUntil) / 3600.0
			return
		}
		lastEventTime = e.timeSec
	}

	// Current harvests not enough — use steady-state rate for the remainder
	additionalSecs := float64(remaining) / totalExpPerMin * 60
	totalSecs := float64(lastEventTime-nowSec) + additionalSecs
	if totalSecs < 0 {
		totalSecs = 0
	}
	hoursToNextLevel = totalSecs / 3600.0
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
