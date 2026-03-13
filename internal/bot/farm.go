package bot

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"

	"qq-farm-bot/internal/model"

	"qq-farm-bot/proto/itempb"
	"qq-farm-bot/proto/plantpb"
	"qq-farm-bot/proto/shoppb"
)

const normalFertilizerID = 1011

// FarmWorker handles all farm automation logic.
type FarmWorker struct {
	net                *Network
	logger             *Logger
	cfg                *BotConfig
	gc                 *GameConfig
	lands              *LandCache
	sc                 *StatsCollector
	fertilized         map[int64]bool // tracks lands we've already fertilized this grow cycle
	reservedForBigSeed map[int64]bool // lands reserved for 2×2 seed planting
}

// shopSeedCandidate represents an available seed from the shop with its level requirement.
type shopSeedCandidate struct {
	goods         *shoppb.GoodsInfo
	requiredLevel int64
}

func NewFarmWorker(net *Network, logger *Logger, cfg *BotConfig, lands *LandCache, sc *StatsCollector) *FarmWorker {
	return &FarmWorker{
		net:                net,
		logger:             logger,
		cfg:                cfg,
		gc:                 GetGameConfig(),
		lands:              lands,
		sc:                 sc,
		fertilized:         make(map[int64]bool),
		reservedForBigSeed: make(map[int64]bool),
	}
}

// RunLoop runs the farm check loop until context is cancelled.
func (f *FarmWorker) RunLoop() {
	// Initial delay: add jitter if anti-detection enabled
	initDelay := 2 * time.Second
	if f.cfg.EnableAntiDetection {
		initDelay = time.Duration(1+rand.Intn(3)) * time.Second
	}
	select {
	case <-time.After(initDelay):
	case <-f.net.ctx.Done():
		return
	}

	for {
		f.checkFarm()
		waitTime := time.Duration(f.cfg.FarmInterval) * time.Second
		if f.cfg.EnableAntiDetection {
			// Add ±30% random jitter to the interval
			base := float64(f.cfg.FarmInterval)
			jitter := base * (0.7 + rand.Float64()*0.6) // 0.7x ~ 1.3x
			waitTime = time.Duration(jitter * float64(time.Second))
		}
		select {
		case <-time.After(waitTime):
		case <-f.net.ctx.Done():
			return
		}
	}
}

func (f *FarmWorker) checkFarm() {
	landsReply, err := f.net.AllLands()
	if err != nil {
		f.logger.Warnf("巡田", "检查失败: %v", err)
		return
	}
	if len(landsReply.Lands) == 0 {
		return
	}

	lands := landsReply.Lands

	unlockedNew, upgradedNew := 0, 0
	if f.cfg.EnableUpgradeLand {
		unlockedNew, upgradedNew = f.autoUnlockAndUpgrade(lands)
		if unlockedNew > 0 || upgradedNew > 0 {
			landsReply, err = f.net.AllLands()
			if err != nil {
				f.logger.Warnf("巡田", "重新获取土地失败: %v", err)
				return
			}
			lands = landsReply.Lands
		}
	}

	status := f.analyzeLands(lands)
	landMap := buildLandMap(lands)

	f.logger.Debugf("巡田", "fertilized缓存: %v", f.fertilized)

	fertilized := f.checkAndFertilize(lands)
	unlockedCount := 0
	for _, land := range lands {
		if land.Unlocked {
			unlockedCount++
		}
	}

	// Re-fetch lands after fertilization so the cache reflects post-fertilize phases
	if fertilized > 0 {
		if freshReply, err := f.net.AllLands(); err == nil {
			lands = freshReply.Lands
		}
	}

	// Update land cache for dashboard display
	f.updateLandCache(lands)

	// Build status summary
	var parts []string
	if len(status.harvestable) > 0 {
		parts = append(parts, fmt.Sprintf("收:%d", len(status.harvestable)))
	}
	if len(status.needWeed) > 0 {
		parts = append(parts, fmt.Sprintf("草:%d", len(status.needWeed)))
	}
	if len(status.needBug) > 0 {
		parts = append(parts, fmt.Sprintf("虫:%d", len(status.needBug)))
	}
	if len(status.needWater) > 0 {
		parts = append(parts, fmt.Sprintf("水:%d", len(status.needWater)))
	}
	if len(status.dead) > 0 {
		parts = append(parts, fmt.Sprintf("枯:%d", len(status.dead)))
	}
	if len(status.empty) > 0 {
		reservedCount := 0
		for _, id := range status.empty {
			if f.reservedForBigSeed[id] {
				reservedCount++
			}
		}
		if reservedCount > 0 {
			parts = append(parts, fmt.Sprintf("空:%d(留:%d)", len(status.empty), reservedCount))
		} else {
			parts = append(parts, fmt.Sprintf("空:%d", len(status.empty)))
		}
	}
	parts = append(parts, fmt.Sprintf("长:%d", len(status.growing)))

	hasWork := len(status.harvestable) > 0 || len(status.needWeed) > 0 || len(status.needBug) > 0 ||
		len(status.needWater) > 0 || len(status.dead) > 0 || len(status.empty) > 0

	var actions []string

	if unlockedNew > 0 {
		actions = append(actions, fmt.Sprintf("解锁%d", unlockedNew))
	}
	if upgradedNew > 0 {
		actions = append(actions, fmt.Sprintf("升级%d", upgradedNew))
	}
	if unlockedNew > 0 || upgradedNew > 0 {
		hasWork = true
	}

	// Batch operations: weed, bug, water (respect config toggles)
	if f.cfg.EnableWeed && len(status.needWeed) > 0 {
		f.logger.Infof("除草", "需除草 %d 块: %s", len(status.needWeed), f.descLands(status.needWeed, landMap))
		if err := f.weedOut(status.needWeed); err == nil {
			actions = append(actions, fmt.Sprintf("除草%d", len(status.needWeed)))
			f.sc.RecordSimple(model.OpWeed, int64(len(status.needWeed)))
		}
	}
	if f.cfg.EnableBug && len(status.needBug) > 0 {
		f.logger.Infof("除虫", "需除虫 %d 块: %s", len(status.needBug), f.descLands(status.needBug, landMap))
		if err := f.insecticide(status.needBug); err == nil {
			actions = append(actions, fmt.Sprintf("除虫%d", len(status.needBug)))
			f.sc.RecordSimple(model.OpBug, int64(len(status.needBug)))
		}
	}
	if f.cfg.EnableWater && len(status.needWater) > 0 {
		f.logger.Infof("浇水", "需浇水 %d 块: %s", len(status.needWater), f.descLands(status.needWater, landMap))
		if err := f.waterLand(status.needWater); err == nil {
			actions = append(actions, fmt.Sprintf("浇水%d", len(status.needWater)))
			f.sc.RecordSimple(model.OpWater, int64(len(status.needWater)))
		}
	}

	if f.cfg.EnableHarvest && len(status.harvestable) > 0 {
		for _, id := range status.harvestable {
			if land, ok := landMap[id]; ok && land.Plant != nil {
				cropName := f.gc.GetPlantName(int(land.Plant.Id))
				totalSeasons := f.gc.GetPlantSeasons(int(land.Plant.Id))
				f.logger.Debugf("收获", "地#%d %s 季=%d/%d 准备收获",
					id, cropName, land.Plant.GetSeason(), totalSeasons)
			}
		}
		f.logger.Infof("收获", "成熟 %d 块: %s", len(status.harvestable), f.descLands(status.harvestable, landMap))
		if err := f.harvest(status.harvestable); err == nil {
			actions = append(actions, fmt.Sprintf("收获%d", len(status.harvestable)))
			f.sc.RecordSimple(model.OpHarvest, int64(len(status.harvestable)))
			for _, id := range status.harvestable {
				delete(f.fertilized, id)
			}
			if freshReply, err := f.net.AllLands(); err == nil {
				freshLandMap := buildLandMap(freshReply.Lands)
				for _, id := range status.harvestable {
					if land, ok := freshLandMap[id]; ok && land.Plant != nil && len(land.Plant.Phases) > 0 {
						cropName := f.gc.GetPlantName(int(land.Plant.Id))
						nowSec := time.Now().Unix()
						cp := getCurrentPhase(land.Plant.Phases, nowSec)
						phaseName := "?"
						if cp != nil {
							phaseName = getPhaseName(cp)
						}
						f.logger.Debugf("收获", "地#%d %s 收后状态: 季=%d Phase=%d(%s) phaseId=%d 阶段数=%d",
							id, cropName, land.Plant.GetSeason(),
							cp.Phase, phaseName, cp.PhaseId,
							len(land.Plant.Phases))
					} else {
						f.logger.Debugf("收获", "地#%d 收后: 已空/枯萎", id)
					}
				}
				freshStatus := f.analyzeLands(freshReply.Lands)
				status.dead = freshStatus.dead
				status.empty = freshStatus.empty
			}
		}
	}

	// Remove only dead/withered plants + plant on empty lands (respect config toggles)
	allDead := []int64{}
	allEmpty := status.empty
	if f.cfg.EnableRemoveDead {
		allDead = append(allDead, status.dead...)
	}
	if f.cfg.EnablePlant && (len(allDead) > 0 || len(allEmpty) > 0) {
		f.autoPlant(allDead, allEmpty, unlockedCount, lands)
		actions = append(actions, fmt.Sprintf("种植%d", len(allDead)+len(allEmpty)))
	}

	if hasWork {
		actionStr := ""
		if len(actions) > 0 {
			actionStr = " → " + strings.Join(actions, "/")
		}
		f.logger.Infof("农场", "[%s]%s", strings.Join(parts, " "), actionStr)
	}
}

// confPhaseTypeNames maps the game's confPhaseType values (phase_id field)
// to Chinese display names, matching the client's confPhaseTypeStr mapping.
var confPhaseTypeNames = map[int64]string{
	0: "无", 1: "种子", 2: "幼株", 3: "幼枝", 4: "幼芽", 5: "幼苗",
	6: "发芽", 7: "长枝", 8: "小叶子", 9: "秧苗", 10: "开花",
	11: "小叶", 12: "大叶子", 13: "成株", 14: "伸长", 15: "幼穗",
	16: "卷心", 17: "分叶", 18: "大叶", 19: "成熟", 20: "初熟",
	21: "花蕾", 22: "结果", 23: "成树", 24: "盛开", 25: "枯萎",
	26: "发菌", 27: "出菇", 28: "幼菇", 29: "幼蕾", 30: "芝蕾",
	31: "含苞", 32: "幼芝", 33: "初放", 34: "莲座期", 35: "成株期",
	36: "已成熟", 37: "成苗",
}

// phaseNames maps PlantPhase enum values to fallback display names,
// used only when phase_id is not available (zero value).
var phaseNames = map[int32]string{
	0: "未知",
	1: "种子",
	2: "发芽",
	3: "小叶",
	4: "大叶",
	5: "开花",
	6: "成熟",
	7: "枯萎",
}

func formatDuration(seconds int) string {
	if seconds <= 0 {
		return "0秒"
	}
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	if hours > 0 && minutes > 0 {
		return fmt.Sprintf("%d小时%d分", hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%d小时", hours)
	}
	if minutes > 0 {
		return fmt.Sprintf("%d分钟", minutes)
	}
	return fmt.Sprintf("%d秒", seconds%60)
}

func getPhaseName(phase *plantpb.PlantPhaseInfo) string {
	if phase.PhaseId != 0 {
		if name, ok := confPhaseTypeNames[phase.PhaseId]; ok {
			return name
		}
		return fmt.Sprintf("阶段%d", phase.PhaseId)
	}
	if name, ok := phaseNames[phase.Phase]; ok {
		return name
	}
	return fmt.Sprintf("阶段%d", phase.Phase)
}

func (f *FarmWorker) descLands(landIDs []int64, landMap map[int64]*plantpb.LandInfo) string {
	var parts []string
	for _, id := range landIDs {
		if land, ok := landMap[id]; ok && land.Plant != nil {
			cropName := f.gc.GetPlantName(int(land.Plant.Id))
			parts = append(parts, fmt.Sprintf("#%d(%s)", id, cropName))
		} else {
			parts = append(parts, fmt.Sprintf("#%d", id))
		}
	}
	return strings.Join(parts, " ")
}

func (f *FarmWorker) updateLandCache(lands []*plantpb.LandInfo) {
	if f.lands == nil {
		return
	}
	nowSec := time.Now().Unix()
	totalLands := len(lands)
	unlockedCount := 0
	var statuses []model.LandStatus
	var harvestInfos []LandHarvestInfo
	landMap := buildLandMap(lands)
	for _, land := range lands {
		ls := model.LandStatus{
			ID:           land.Id,
			Level:        land.Level,
			MaxLevel:     land.MaxLevel,
			Unlocked:     land.Unlocked,
			CouldUpgrade: land.CouldUpgrade,
			CouldUnlock:  land.CouldUnlock,
			MasterLandID: land.MasterLandId,
		}
		if land.Unlocked {
			unlockedCount++
		}
		// Populate land buff data
		if buff := land.GetBuff(); buff != nil {
			ls.ExpBonusPct = buff.PlantExpBonus
			ls.TimeReducePct = buff.PlantingTimeReduction
			ls.YieldBonusPct = buff.PlantYieldBonus
		}
		if land.Plant != nil && len(land.Plant.Phases) > 0 && !isOccupiedSlaveLand(land, landMap) {
			ls.CropID = land.Plant.Id
			ls.CropName = f.gc.GetPlantName(int(land.Plant.Id))
			ls.Season = land.Plant.GetSeason()
			ls.GrowSec = land.Plant.GrowSec
			ls.CropExp = f.gc.GetPlantExp(int(land.Plant.Id))
			ls.PlantSize = f.gc.GetPlantSize(int(land.Plant.Id))
			ls.DryNum = land.Plant.DryNum
			ls.StoleNum = land.Plant.StoleNum
			ls.FruitNum = land.Plant.FruitNum
			ls.LeftFruitNum = land.Plant.LeftFruitNum
			ls.Stealable = land.Plant.Stealable
			ls.FertTimesLeft = land.Plant.LeftInorcFertTimes
			ls.HasWeeds = len(land.Plant.WeedOwners) > 0
			ls.HasInsects = len(land.Plant.InsectOwners) > 0

			currentPhase := getCurrentPhase(land.Plant.Phases, nowSec)
			if currentPhase != nil {
				if currentPhase.PhaseId != 0 {
					if name, ok := confPhaseTypeNames[currentPhase.PhaseId]; ok {
						ls.Phase = name
					} else {
						ls.Phase = fmt.Sprintf("阶段%d", currentPhase.PhaseId)
					}
				} else if name, ok := phaseNames[currentPhase.Phase]; ok {
					ls.Phase = name
				} else {
					ls.Phase = fmt.Sprintf("阶段%d", currentPhase.Phase)
				}
				// Check for weeds/insects from phase timing
				if !ls.HasWeeds && currentPhase.WeedsTime > 0 && toTimeSec(currentPhase.WeedsTime) <= nowSec {
					ls.HasWeeds = true
				}
				if !ls.HasInsects && currentPhase.InsectTime > 0 && toTimeSec(currentPhase.InsectTime) <= nowSec {
					ls.HasInsects = true
				}
			}

			matureTime := getMatureTimeSec(land.Plant.Phases)
			plantTime := getPlantStartTimeSec(land.Plant.Phases)
			if matureTime > 0 {
				ls.MatureTimeSec = matureTime
			}
			if matureTime > 0 && plantTime > 0 && matureTime > plantTime {
				ls.CycleTimeSec = matureTime - plantTime
				hi := LandHarvestInfo{
					LandID:        land.Id,
					CropID:        land.Plant.Id,
					Season:        land.Plant.GetSeason(),
					CropExp:       f.gc.GetPlantExp(int(land.Plant.Id)),
					CycleTimeSec:  matureTime - plantTime,
					MatureTimeSec: matureTime,
				}
				if buff := land.GetBuff(); buff != nil {
					hi.ExpBonusPct = buff.PlantExpBonus
					hi.TimeReducePct = buff.PlantingTimeReduction
					hi.YieldBonusPct = buff.PlantYieldBonus
				}
				if currentPhase != nil {
					switch plantpb.PlantPhase(currentPhase.Phase) {
					case plantpb.PlantPhase_MATURE:
						hi.IsMature = true
					case plantpb.PlantPhase_DEAD:
						// skip dead plants
					default:
						hi.IsGrowing = true
					}
				}
				if hi.CropExp > 0 && (hi.IsMature || hi.IsGrowing) {
					harvestInfos = append(harvestInfos, hi)
				}
			}
		}
		statuses = append(statuses, ls)
	}
	f.lands.Update(totalLands, unlockedCount, statuses, harvestInfos)
}

type landStatus struct {
	harvestable []int64
	needWater   []int64
	needWeed    []int64
	needBug     []int64
	growing     []int64
	empty       []int64
	dead        []int64
}

func (f *FarmWorker) analyzeLands(lands []*plantpb.LandInfo) *landStatus {
	s := &landStatus{}
	nowSec := time.Now().Unix()
	landMap := buildLandMap(lands)

	for _, land := range lands {
		id := land.Id
		if !land.Unlocked {
			continue
		}
		if isOccupiedSlaveLand(land, landMap) {
			continue
		}
		plant := land.Plant
		if plant == nil || len(plant.Phases) == 0 {
			s.empty = append(s.empty, id)
			continue
		}

		phase := getCurrentPhase(plant.Phases, nowSec)
		if phase == nil {
			s.empty = append(s.empty, id)
			continue
		}

		cropName := f.gc.GetPlantName(int(plant.Id))
		phaseName := getPhaseName(phase)

		switch plantpb.PlantPhase(phase.Phase) {
		case plantpb.PlantPhase_DEAD:
			s.dead = append(s.dead, id)
			f.logger.Debugf("分析", "地#%d %s Phase=%d(%s) → 枯萎", id, cropName, phase.Phase, phaseName)
		case plantpb.PlantPhase_MATURE:
			s.harvestable = append(s.harvestable, id)
			f.logger.Debugf("分析", "地#%d %s Phase=%d(%s) 季=%d → 成熟",
				id, cropName, phase.Phase, phaseName, plant.GetSeason())
		default:
			if plant.DryNum > 0 || (phase.DryTime > 0 && toTimeSec(phase.DryTime) <= nowSec) {
				s.needWater = append(s.needWater, id)
			}
			if len(plant.WeedOwners) > 0 || (phase.WeedsTime > 0 && toTimeSec(phase.WeedsTime) <= nowSec) {
				s.needWeed = append(s.needWeed, id)
			}
			if len(plant.InsectOwners) > 0 || (phase.InsectTime > 0 && toTimeSec(phase.InsectTime) <= nowSec) {
				s.needBug = append(s.needBug, id)
			}
			s.growing = append(s.growing, id)
			f.logger.Debugf("分析", "地#%d %s Phase=%d(%s) phaseId=%d 季=%d 阶段数=%d → 生长中",
				id, cropName, phase.Phase, phaseName, phase.PhaseId,
				plant.GetSeason(), len(plant.Phases))
		}
	}
	return s
}

// checkAndFertilize examines growing plants and fertilizes them when they're in their longest phase.
// Normal fertilizer skips the current phase, so applying it during the longest phase saves the most time.
func (f *FarmWorker) checkAndFertilize(lands []*plantpb.LandInfo) int {
	nowSec := time.Now().Unix()
	fertilizeCount := 0
	landMap := buildLandMap(lands)

	for _, land := range lands {
		if !land.Unlocked || land.Plant == nil || len(land.Plant.Phases) == 0 {
			continue
		}
		if isOccupiedSlaveLand(land, landMap) {
			continue
		}

		plant := land.Plant
		landID := land.Id
		cropName := f.gc.GetPlantName(int(plant.Id))

		if f.fertilized[landID] {
			f.logger.Debugf("施肥", "地#%d %s 本轮已施肥, 跳过", landID, cropName)
			continue
		}

		currentPhase := getCurrentPhase(plant.Phases, nowSec)
		if currentPhase == nil {
			f.logger.Debugf("施肥", "地#%d %s 无当前阶段, 跳过", landID, cropName)
			continue
		}

		phase := plantpb.PlantPhase(currentPhase.Phase)
		phaseName := getPhaseName(currentPhase)
		if phase == plantpb.PlantPhase_MATURE || phase == plantpb.PlantPhase_DEAD || phase == plantpb.PlantPhase_PHASE_UNKNOWN {
			f.logger.Debugf("施肥", "地#%d %s Phase=%d(%s) phaseId=%d → 非生长阶段, 跳过",
				landID, cropName, currentPhase.Phase, phaseName, currentPhase.PhaseId)
			continue
		}

		pd := f.gc.GetPlantPhaseData(int(plant.Id))
		if pd == nil {
			f.logger.Debugf("施肥", "地#%d %s 无阶段配置数据, 跳过", landID, cropName)
			continue
		}

		season := plant.GetSeason()
		totalSeasons := f.gc.GetPlantSeasons(int(plant.Id))
		if season < 1 {
			season = 1
		}

		if plant.LeftInorcFertTimes <= 0 {
			f.logger.Debugf("施肥", "地#%d %s 剩余施肥次数=%d → 跳过", landID, cropName, plant.LeftInorcFertTimes)
			continue
		}

		// Derive phase index from remaining growth phases.
		// The server removes past phases from plant.Phases, so:
		//   phaseIdx = totalPhases - remainingGrowthPhases
		// This avoids relying on the Phase enum value (which is always GERMINATION=2 for growing).
		remainingGrowth := 0
		for _, p := range plant.Phases {
			pp := plantpb.PlantPhase(p.Phase)
			if pp != plantpb.PlantPhase_MATURE && pp != plantpb.PlantPhase_DEAD && pp != plantpb.PlantPhase_PHASE_UNKNOWN {
				remainingGrowth++
			}
		}

		var phaseDurations []int
		var maxDuration int
		var allEqual bool

		if season >= 2 && pd.Season2Phases != nil {
			phaseDurations = pd.Season2Phases
			maxDuration = pd.Season2MaxPhase
			allEqual = pd.Season2AllEqual
		} else {
			phaseDurations = pd.PhaseDurations
			maxDuration = pd.MaxPhaseDuration
			allEqual = pd.AllPhasesEqual
		}

		phaseIdx := len(phaseDurations) - remainingGrowth

		f.logger.Debugf("施肥", "地#%d %s 季=%d/%d Phase=%d(%s) phaseId=%d 剩余生长阶段=%d 总阶段=%d phaseIdx=%d fertLeft=%d durations=%v",
			landID, cropName, season, totalSeasons,
			currentPhase.Phase, phaseName, currentPhase.PhaseId,
			remainingGrowth, len(phaseDurations), phaseIdx,
			plant.LeftInorcFertTimes, phaseDurations)

		if phaseIdx < 0 || phaseIdx >= len(phaseDurations) {
			f.logger.Debugf("施肥", "地#%d %s phaseIdx=%d 越界(len=%d, 剩余=%d) → 跳过",
				landID, cropName, phaseIdx, len(phaseDurations), remainingGrowth)
			continue
		}

		currentDuration := phaseDurations[phaseIdx]
		if allEqual || currentDuration >= maxDuration {
			f.logger.Debugf("施肥", "地#%d %s 当前时长=%d max=%d allEqual=%v → 执行施肥",
				landID, cropName, currentDuration, maxDuration, allEqual)
			if f.fertilizeSingle(landID) {
				f.fertilized[landID] = true
				fertilizeCount++
				timeSaved := formatDuration(currentDuration)
				f.logger.Infof("施肥", "地#%d %s [%s阶段] 跳过%s", landID, cropName, phaseName, timeSaved)
			} else {
				f.logger.Debugf("施肥", "地#%d %s 施肥请求失败(服务器拒绝)", landID, cropName)
			}
		} else {
			f.logger.Debugf("施肥", "地#%d %s 当前时长=%d < max=%d → 等待最长阶段",
				landID, cropName, currentDuration, maxDuration)
		}
	}

	if fertilizeCount > 0 {
		f.logger.Infof("施肥", "本轮共施肥 %d 块地", fertilizeCount)
		f.sc.RecordSimple(model.OpFertilize, int64(fertilizeCount))
	}
	return fertilizeCount
}

func (f *FarmWorker) fertilizeSingle(landID int64) bool {
	req := &plantpb.FertilizeRequest{LandIds: []int64{landID}, FertilizerId: normalFertilizerID}
	body, _ := proto.Marshal(req)
	if _, err := f.net.SendRequest("gamepb.plantpb.PlantService", "Fertilize", body); err != nil {
		f.logger.Debugf("施肥", "地#%d 请求失败: %v", landID, err)
		return false
	}
	time.Sleep(50 * time.Millisecond)
	return true
}

func getCurrentPhase(phases []*plantpb.PlantPhaseInfo, nowSec int64) *plantpb.PlantPhaseInfo {
	if len(phases) == 0 {
		return nil
	}
	var best *plantpb.PlantPhaseInfo
	var bestBT int64
	for _, p := range phases {
		bt := toTimeSec(p.BeginTime)
		if bt >= bestBT && bt <= nowSec {
			bestBT = bt
			best = p
		}
	}
	if best != nil {
		return best
	}
	return phases[0]
}

func getMatureTimeSec(phases []*plantpb.PlantPhaseInfo) int64 {
	for _, p := range phases {
		if plantpb.PlantPhase(p.Phase) == plantpb.PlantPhase_MATURE {
			return toTimeSec(p.BeginTime)
		}
	}
	return 0
}

func getPlantStartTimeSec(phases []*plantpb.PlantPhaseInfo) int64 {
	if len(phases) > 0 {
		return toTimeSec(phases[0].BeginTime)
	}
	return 0
}

func toTimeSec(val int64) int64 {
	if val <= 0 {
		return 0
	}
	if val > 1e15 {
		return val / 1_000_000
	}
	if val > 1e12 {
		return val / 1000
	}
	return val
}

// buildLandMap creates a map from land ID to LandInfo for quick lookups.
func buildLandMap(lands []*plantpb.LandInfo) map[int64]*plantpb.LandInfo {
	m := make(map[int64]*plantpb.LandInfo, len(lands))
	for _, land := range lands {
		m[land.Id] = land
	}
	return m
}

// isOccupiedSlaveLand returns true if a land is a slave tile occupied by a
// multi-tile crop planted on another (master) land.
func isOccupiedSlaveLand(land *plantpb.LandInfo, landMap map[int64]*plantpb.LandInfo) bool {
	masterID := land.MasterLandId
	if masterID == 0 || masterID == land.Id {
		return false
	}
	master, ok := landMap[masterID]
	if !ok {
		return false
	}
	// Verify the master land actually has plant data
	if master.Plant == nil || len(master.Plant.Phases) == 0 {
		return false
	}
	// Validate that this land ID is listed in the master's slave land IDs.
	// This matches the reference implementation's getLinkedMasterLand check.
	if len(master.SlaveLandIds) > 0 {
		found := false
		for _, sid := range master.SlaveLandIds {
			if sid == land.Id {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (f *FarmWorker) harvest(landIDs []int64) error {
	gid, _, _, _, _ := f.net.state.Get()
	req := &plantpb.HarvestRequest{LandIds: landIDs, HostGid: gid, IsAll: true}
	body, _ := proto.Marshal(req)
	_, err := f.net.SendRequest("gamepb.plantpb.PlantService", "Harvest", body)
	return err
}

func (f *FarmWorker) waterLand(landIDs []int64) error {
	gid, _, _, _, _ := f.net.state.Get()
	req := &plantpb.WaterLandRequest{LandIds: landIDs, HostGid: gid}
	body, _ := proto.Marshal(req)
	_, err := f.net.SendRequest("gamepb.plantpb.PlantService", "WaterLand", body)
	return err
}

func (f *FarmWorker) weedOut(landIDs []int64) error {
	gid, _, _, _, _ := f.net.state.Get()
	req := &plantpb.WeedOutRequest{LandIds: landIDs, HostGid: gid}
	body, _ := proto.Marshal(req)
	_, err := f.net.SendRequest("gamepb.plantpb.PlantService", "WeedOut", body)
	return err
}

func (f *FarmWorker) insecticide(landIDs []int64) error {
	gid, _, _, _, _ := f.net.state.Get()
	req := &plantpb.InsecticideRequest{LandIds: landIDs, HostGid: gid}
	body, _ := proto.Marshal(req)
	_, err := f.net.SendRequest("gamepb.plantpb.PlantService", "Insecticide", body)
	return err
}

// removePlantAndCollectFreed removes dead/withered plants and returns all freed
// land IDs (including slave lands for multi-tile crops) by parsing the server reply.
func (f *FarmWorker) removePlantAndCollectFreed(landIDs []int64) ([]int64, error) {
	req := &plantpb.RemovePlantRequest{LandIds: landIDs}
	body, _ := proto.Marshal(req)
	replyBody, err := f.net.SendRequest("gamepb.plantpb.PlantService", "RemovePlant", body)
	if err != nil {
		return landIDs, err
	}

	// Parse reply to find all freed land IDs (including slaves of multi-tile crops)
	reply := &plantpb.RemovePlantReply{}
	proto.Unmarshal(replyBody, reply)

	seen := make(map[int64]bool, len(landIDs))
	for _, id := range landIDs {
		seen[id] = true
	}
	for _, land := range reply.Land {
		if !seen[land.Id] {
			seen[land.Id] = true
		}
	}

	freed := make([]int64, 0, len(seen))
	for id := range seen {
		freed = append(freed, id)
	}
	return freed, nil
}

func (f *FarmWorker) fertilize(landIDs []int64) int {
	success := 0
	for _, id := range landIDs {
		req := &plantpb.FertilizeRequest{LandIds: []int64{id}, FertilizerId: normalFertilizerID}
		body, _ := proto.Marshal(req)
		if _, err := f.net.SendRequest("gamepb.plantpb.PlantService", "Fertilize", body); err != nil {
			break
		}
		success++
		time.Sleep(50 * time.Millisecond)
	}
	return success
}

func (f *FarmWorker) autoPlant(deadLands, emptyLands []int64, unlockedCount int, allLands []*plantpb.LandInfo) {
	toLant := append([]int64{}, emptyLands...)
	for _, id := range deadLands {
		delete(f.fertilized, id)
	}

	if len(deadLands) > 0 {
		deadLandMap := buildLandMap(allLands)
		var deadDesc []string
		for _, id := range deadLands {
			if land, ok := deadLandMap[id]; ok && land.Plant != nil {
				cropName := f.gc.GetPlantName(int(land.Plant.Id))
				deadDesc = append(deadDesc, fmt.Sprintf("#%d(%s)", id, cropName))
			} else {
				deadDesc = append(deadDesc, fmt.Sprintf("#%d", id))
			}
		}
		f.logger.Infof("铲除", "铲除枯萎作物 %d 块: %s", len(deadLands), strings.Join(deadDesc, " "))
		freedIDs, err := f.removePlantAndCollectFreed(deadLands)
		if err == nil {
			slaveCount := len(freedIDs) - len(deadLands)
			if slaveCount > 0 {
				f.logger.Infof("铲除", "释放附属地 %d 块，共腾出 %d 块", slaveCount, len(freedIDs))
			}
			for _, id := range freedIDs {
				delete(f.fertilized, id)
			}
			toLant = append(toLant, freedIDs...)
		} else {
			toLant = append(toLant, deadLands...)
		}
	}

	if len(toLant) == 0 {
		return
	}

	// Phase 0: Handle size>=2 (big) seeds from bag — prioritize 2×2 planting
	toLant = f.handleBigSeedPlanting(toLant, allLands)
	if len(toLant) == 0 {
		return
	}

	// Phase 1: plant from bag seeds if PreferBagSeeds is enabled
	if f.cfg.PreferBagSeeds {
		plantedFromBag := f.plantFromBag(toLant)
		if plantedFromBag >= len(toLant) {
			return
		}
		toLant = toLant[plantedFromBag:]
	}

	// Phase 2: buy seeds from shop and plant remaining lands
	f.buyAndPlant(toLant, unlockedCount)
}

// plantFromBag checks the bag for seeds and plants them on the given lands.
// Returns the number of lands successfully planted.
func (f *FarmWorker) plantFromBag(lands []int64) int {
	req := &itempb.BagRequest{}
	body, _ := proto.Marshal(req)
	replyBody, err := f.net.SendRequest("gamepb.itempb.ItemService", "Bag", body)
	if err != nil {
		return 0
	}
	reply := &itempb.BagReply{}
	proto.Unmarshal(replyBody, reply)

	if reply.ItemBag == nil || len(reply.ItemBag.Items) == 0 {
		return 0
	}

	// Collect available seeds from bag
	type bagSeed struct {
		itemID int64
		count  int64
	}
	var seeds []bagSeed
	for _, item := range reply.ItemBag.Items {
		if item.Count > 0 && f.gc.IsSeedID(int(item.Id)) {
			if f.gc.GetPlantSizeBySeedID(int(item.Id)) >= 2 {
				continue
			}
			seeds = append(seeds, bagSeed{itemID: item.Id, count: item.Count})
		}
	}

	if len(seeds) == 0 {
		return 0
	}

	planted := 0
	pendingLands := make(map[int64]bool, len(lands))
	for _, id := range lands {
		pendingLands[id] = true
	}
	for _, seed := range seeds {
		if len(pendingLands) == 0 {
			break
		}
		seedName := f.gc.GetPlantNameBySeedID(int(seed.itemID))
		plantSize := f.gc.GetPlantSizeBySeedID(int(seed.itemID))
		landFootprint := plantSize * plantSize
		count := int(seed.count)
		// For multi-tile crops, limit count by available land / footprint
		availableForSeed := len(pendingLands) / landFootprint
		if availableForSeed <= 0 {
			continue
		}
		if count > availableForSeed {
			count = availableForSeed
		}
		seedPlanted := 0
		var plantedOnLands []string
		for _, landID := range lands {
			if !pendingLands[landID] {
				continue
			}
			if seedPlanted >= count {
				break
			}
			plantReq := &plantpb.PlantRequest{
				Items: []*plantpb.PlantItem{{SeedId: seed.itemID, LandIds: []int64{landID}}},
			}
			plantBody, _ := proto.Marshal(plantReq)
			replyBody, err := f.net.SendRequest("gamepb.plantpb.PlantService", "Plant", plantBody)
			if err != nil {
				break
			}
			seedPlanted++
			planted++
			plantedOnLands = append(plantedOnLands, fmt.Sprintf("#%d", landID))
			delete(pendingLands, landID)
			delete(f.fertilized, landID)
			if landFootprint > 1 {
				plantReply := &plantpb.PlantReply{}
				proto.Unmarshal(replyBody, plantReply)
				for _, changedLand := range plantReply.Land {
					cid := changedLand.Id
					if cid != landID {
						delete(pendingLands, cid)
						delete(f.fertilized, cid)
					}
				}
			}
			time.Sleep(50 * time.Millisecond)
		}
		if seedPlanted > 0 {
			f.logger.Infof("种植", "背包种子 %s x%d → 地%s", seedName, seedPlanted, strings.Join(plantedOnLands, " "))
		}
	}

	if planted > 0 {
		f.logger.Infof("种植", "从背包种植 %d 块", planted)
		f.sc.RecordSimple(model.OpPlant, int64(planted))
	}
	return planted
}

func (f *FarmWorker) buyAndPlant(toLant []int64, unlockedCount int) {
	// Find best seed from shop (respects PlantCropID config)
	bestSeed, err := f.findBestSeed(unlockedCount)
	if err != nil || bestSeed == nil {
		return
	}

	seedName := f.gc.GetPlantNameBySeedID(int(bestSeed.ItemId))
	f.logger.Infof("商店", "最佳种子: %s 价格=%d金币", seedName, bestSeed.Price)

	// Calculate land footprint for multi-tile crops
	plantSize := f.gc.GetPlantSizeBySeedID(int(bestSeed.ItemId))
	landFootprint := plantSize * plantSize

	// Buy seeds
	_, _, _, gold, _ := f.net.state.Get()
	needCount := int64(len(toLant))
	if landFootprint > 1 {
		needCount = int64(len(toLant) / landFootprint)
		if needCount <= 0 {
			f.logger.Warnf("种植", "%s 需要至少 %d 块空地才能种植，当前仅 %d 块", seedName, landFootprint, len(toLant))
			return
		}
	}
	totalCost := bestSeed.Price * needCount
	if totalCost > gold {
		canBuy := gold / bestSeed.Price
		if canBuy <= 0 {
			f.logger.Warnf("商店", "金币不足")
			return
		}
		needCount = canBuy
	}

	buyReq := &shoppb.BuyGoodsRequest{GoodsId: bestSeed.Id, Num: needCount, Price: bestSeed.Price}
	buyBody, _ := proto.Marshal(buyReq)
	buyReplyBody, err := f.net.SendRequest("gamepb.shoppb.ShopService", "BuyGoods", buyBody)
	if err != nil {
		f.logger.Warnf("购买", "%v", err)
		return
	}
	buyReply := &shoppb.BuyGoodsReply{}
	proto.Unmarshal(buyReplyBody, buyReply)

	actualSeedID := bestSeed.ItemId
	if len(buyReply.GetItems) > 0 && buyReply.GetItems[0].Id > 0 {
		actualSeedID = buyReply.GetItems[0].Id
	}
	f.logger.Infof("购买", "已购买 %s种子 x%d", f.gc.GetPlantNameBySeedID(int(actualSeedID)), needCount)
	seedCost := bestSeed.Price * needCount
	f.sc.Record(model.OpBuySeed, needCount, -seedCost, 0)

	planted := 0
	var plantedOnLands []string
	pendingLands := make(map[int64]bool, len(toLant))
	for _, id := range toLant {
		pendingLands[id] = true
	}
	actualSeedName := f.gc.GetPlantNameBySeedID(int(actualSeedID))
	for _, landID := range toLant {
		if !pendingLands[landID] {
			continue
		}
		if planted >= int(needCount) {
			break
		}
		req := &plantpb.PlantRequest{
			Items: []*plantpb.PlantItem{{SeedId: actualSeedID, LandIds: []int64{landID}}},
		}
		body, _ := proto.Marshal(req)
		replyBody, err := f.net.SendRequest("gamepb.plantpb.PlantService", "Plant", body)
		if err != nil {
			break
		}
		planted++
		plantedOnLands = append(plantedOnLands, fmt.Sprintf("#%d", landID))
		delete(pendingLands, landID)
		delete(f.fertilized, landID)
		if landFootprint > 1 {
			plantReply := &plantpb.PlantReply{}
			proto.Unmarshal(replyBody, plantReply)
			for _, changedLand := range plantReply.Land {
				cid := changedLand.Id
				if cid != landID {
					delete(pendingLands, cid)
					delete(f.fertilized, cid)
				}
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	if planted > 0 {
		f.logger.Infof("种植", "商店种子 %s x%d → 地%s", actualSeedName, planted, strings.Join(plantedOnLands, " "))
		f.sc.RecordSimple(model.OpPlant, int64(planted))
	}
}

func (f *FarmWorker) findBestSeed(landsCount int) (*shoppb.GoodsInfo, error) {
	req := &shoppb.ShopInfoRequest{ShopId: 2} // Seed shop
	body, _ := proto.Marshal(req)
	replyBody, err := f.net.SendRequest("gamepb.shoppb.ShopService", "ShopInfo", body)
	if err != nil {
		return nil, err
	}
	reply := &shoppb.ShopInfoReply{}
	proto.Unmarshal(replyBody, reply)
	if len(reply.GoodsList) == 0 {
		return nil, fmt.Errorf("种子商店无商品")
	}

	_, level, _, _, _ := f.net.state.Get()

	var available []shopSeedCandidate

	for _, goods := range reply.GoodsList {
		if !goods.Unlocked {
			continue
		}
		meetsConditions := true
		var reqLevel int64
		for _, cond := range goods.Conds {
			if cond.Type == 1 { // MIN_LEVEL
				reqLevel = cond.Param
				if level < reqLevel {
					meetsConditions = false
					break
				}
			}
		}
		if !meetsConditions {
			continue
		}
		if goods.LimitCount > 0 && goods.BoughtNum >= goods.LimitCount {
			continue
		}
		available = append(available, shopSeedCandidate{goods: goods, requiredLevel: reqLevel})
	}

	if len(available) == 0 {
		return nil, fmt.Errorf("没有可购买的种子")
	}

	// If a specific crop is configured, try to find its seed
	if f.cfg.PlantCropID > 0 {
		targetSeedID := f.gc.GetSeedIDForCrop(f.cfg.PlantCropID)
		if targetSeedID > 0 {
			for _, c := range available {
				if int(c.goods.ItemId) == targetSeedID {
					return c.goods, nil
				}
			}
			f.logger.Warnf("商店", "指定作物(ID:%d)的种子不可购买，使用自动选择", f.cfg.PlantCropID)
		}
	}
	// Strategy-based selection: composable rules pipeline
	strategy := ParsePlantingStrategy(f.cfg.PlantingStrategy)
	if strategy != nil {
		result := f.selectSeedByStrategy(strategy, available, landsCount)
		if result != nil {
			return result, nil
		}
		f.logger.Warnf("策略", "策略筛选无匹配作物，回退默认选择")
	}

	if f.cfg.ForceLowest {
		best := available[0]
		for _, c := range available[1:] {
			if c.requiredLevel < best.requiredLevel || (c.requiredLevel == best.requiredLevel && c.goods.Price < best.goods.Price) {
				best = c
			}
		}
		return best.goods, nil
	}

	// Try efficiency-based selection first
	if f.gc != nil {
		rec := f.gc.GetPlantingRecommendation(int(level), landsCount, 50)
		for _, r := range rec {
			for _, c := range available {
				if c.goods.ItemId == int64(r.SeedID) {
					return c.goods, nil
				}
			}
		}
	}

	// Fallback: level-based selection
	if level <= 28 {
		best := available[0]
		for _, c := range available[1:] {
			if c.requiredLevel < best.requiredLevel {
				best = c
			}
		}
		return best.goods, nil
	}

	best := available[0]
	for _, c := range available[1:] {
		if c.requiredLevel > best.requiredLevel {
			best = c
		}
	}
	return best.goods, nil
}

func (f *FarmWorker) findFastestLevelUpSeed(emptyLandIDs []int64, available []shopSeedCandidate) *shoppb.GoodsInfo {
	if len(emptyLandIDs) == 0 || f.gc == nil || f.lands == nil {
		return nil
	}

	_, level, exp, gold, _ := f.net.state.Get()

	nextLevelExp, hasNext := f.gc.GetNextLevelExp(int(level))
	if !hasNext {
		return nil
	}
	expToNextLevel := nextLevelExp - exp
	if expToNextLevel <= 0 {
		return nil
	}

	harvestInfos := f.lands.GetHarvestInfo()
	landBuffs := f.lands.GetLandBuffsByID(emptyLandIDs)

	nowSec := time.Now().Unix()

	type harvestEvent struct {
		timeSec int64
		exp     int64
	}

	var existingEvents []harvestEvent
	var totalExpPerMin float64

	for _, h := range harvestInfos {
		adjustedExp := float64(h.CropExp) * (10000 + float64(h.ExpBonusPct)) / 10000.0
		if adjustedExp <= 0 {
			continue
		}

		seasons := 1
		var season2GrowSec int64
		if h.CropID > 0 {
			seasons = f.gc.GetPlantSeasons(int(h.CropID))
			if seasons >= 2 {
				if pd := f.gc.GetPlantPhaseData(int(h.CropID)); pd != nil && pd.Season2GrowTime > 0 {
					s2Base := int64(pd.Season2GrowTime)
					if h.TimeReducePct > 0 {
						s2Base = s2Base * (10000 - h.TimeReducePct) / 10000
					}
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

		if h.IsMature {
			existingEvents = append(existingEvents, harvestEvent{timeSec: nowSec, exp: int64(adjustedExp)})
			if currentSeason <= 1 && seasons >= 2 && season2GrowSec > 0 {
				existingEvents = append(existingEvents, harvestEvent{timeSec: nowSec + season2GrowSec, exp: int64(adjustedExp)})
			}
		} else if h.IsGrowing && h.MatureTimeSec > nowSec {
			existingEvents = append(existingEvents, harvestEvent{timeSec: h.MatureTimeSec, exp: int64(adjustedExp)})
			if currentSeason <= 1 && seasons >= 2 && season2GrowSec > 0 {
				existingEvents = append(existingEvents, harvestEvent{timeSec: h.MatureTimeSec + season2GrowSec, exp: int64(adjustedExp)})
			}
		}
	}

	yieldRows := f.gc.GetSeedYieldRows()
	yieldMap := make(map[int]*SeedYieldRow, len(yieldRows))
	for i := range yieldRows {
		yieldMap[yieldRows[i].SeedID] = &yieldRows[i]
	}

	bestSeedGoods := (*shoppb.GoodsInfo)(nil)
	bestLevelUpSec := int64(math.MaxInt64)

	for _, c := range available {
		seedID := int(c.goods.ItemId)
		yr, ok := yieldMap[seedID]
		if !ok || yr.GrowTimeSec <= 0 {
			continue
		}

		plantSize := f.gc.GetPlantSizeBySeedID(seedID)
		landFootprint := plantSize * plantSize
		if landFootprint > len(emptyLandIDs) {
			continue
		}

		effectiveLandCount := len(emptyLandIDs)
		if landFootprint > 1 {
			effectiveLandCount = len(emptyLandIDs) / landFootprint
		}
		if effectiveLandCount <= 0 {
			continue
		}

		if c.goods.Price <= 0 {
			continue
		}

		if c.goods.Price*int64(effectiveLandCount) > gold {
			canBuy := gold / c.goods.Price
			if canBuy <= 0 {
				continue
			}
			effectiveLandCount = int(canBuy)
		}

		var newEvents []harvestEvent
		var newExpPerMin float64

		for i, landID := range emptyLandIDs {
			if i >= effectiveLandCount*landFootprint {
				break
			}
			if landFootprint > 1 && i%landFootprint != 0 {
				continue
			}

			buff := landBuffs[landID]

			s1Base := int64(yr.GrowTimeSec)
			if buff.TimeReducePct > 0 {
				s1Base = s1Base * (10000 - buff.TimeReducePct) / 10000
			}
			s1Fert := int64(yr.NormalFertReduceSec)
			if buff.TimeReducePct > 0 {
				s1Fert = s1Fert * (10000 - buff.TimeReducePct) / 10000
			}
			s1Effective := s1Base - s1Fert
			if s1Effective < 1 {
				s1Effective = 1
			}

			adjustedExp := int64(yr.ExpHarvest) * (10000 + buff.ExpBonusPct) / 10000
			if adjustedExp <= 0 {
				adjustedExp = int64(yr.ExpHarvest)
			}

			harvestTime1 := nowSec + s1Effective
			newEvents = append(newEvents, harvestEvent{timeSec: harvestTime1, exp: adjustedExp})

			var s2Effective int64
			if yr.Seasons >= 2 && yr.Season2GrowTimeSec > 0 {
				s2Base := int64(yr.Season2GrowTimeSec)
				if buff.TimeReducePct > 0 {
					s2Base = s2Base * (10000 - buff.TimeReducePct) / 10000
				}
				s2Fert := int64(yr.Season2FertReduceSec)
				if buff.TimeReducePct > 0 {
					s2Fert = s2Fert * (10000 - buff.TimeReducePct) / 10000
				}
				s2Effective = s2Base - s2Fert
				if s2Effective < 1 {
					s2Effective = 1
				}
				newEvents = append(newEvents, harvestEvent{timeSec: harvestTime1 + s2Effective, exp: adjustedExp})
			}

			totalCycleSec := float64(s1Effective)
			totalCycleExp := float64(adjustedExp)
			if s2Effective > 0 {
				totalCycleSec += float64(s2Effective)
				totalCycleExp += float64(adjustedExp)
			}
			if totalCycleSec > 0 {
				newExpPerMin += totalCycleExp / (totalCycleSec / 60.0)
			}
		}

		allEvents := make([]harvestEvent, 0, len(existingEvents)+len(newEvents))
		allEvents = append(allEvents, existingEvents...)
		allEvents = append(allEvents, newEvents...)
		sort.Slice(allEvents, func(i, j int) bool {
			return allEvents[i].timeSec < allEvents[j].timeSec
		})

		combinedExpPerMin := totalExpPerMin + newExpPerMin
		remaining := expToNextLevel
		lastEventTime := nowSec
		levelUpSec := int64(0)
		found := false

		for _, e := range allEvents {
			remaining -= e.exp
			if remaining <= 0 {
				secsUntil := e.timeSec - nowSec
				if secsUntil < 0 {
					secsUntil = 0
				}
				levelUpSec = nowSec + secsUntil
				found = true
				break
			}
			lastEventTime = e.timeSec
		}

		if !found {
			if combinedExpPerMin > 0 {
				additionalSecs := float64(remaining) / combinedExpPerMin * 60
				totalSecs := float64(lastEventTime-nowSec) + additionalSecs
				if totalSecs < 0 {
					totalSecs = 0
				}
				levelUpSec = nowSec + int64(totalSecs)
			} else {
				levelUpSec = int64(math.MaxInt64)
			}
		}

		if levelUpSec < bestLevelUpSec {
			bestLevelUpSec = levelUpSec
			bestSeedGoods = c.goods
		}
	}

	if bestSeedGoods != nil {
		seedName := f.gc.GetPlantNameBySeedID(int(bestSeedGoods.ItemId))
		hoursToLevelUp := float64(bestLevelUpSec-nowSec) / 3600.0
		if hoursToLevelUp < 0 {
			hoursToLevelUp = 0
		}
		f.logger.Infof("策略", "最快升级模式 → %s (预计%.1f小时后升级)", seedName, hoursToLevelUp)
	}

	return bestSeedGoods
}

// autoUnlockAndUpgrade checks all lands and attempts to unlock/upgrade eligible ones.
func (f *FarmWorker) autoUnlockAndUpgrade(lands []*plantpb.LandInfo) (unlocked, upgraded int) {
	_, level, _, gold, _ := f.net.state.Get()

	for _, land := range lands {
		if !land.Unlocked && land.CouldUnlock {
			cond := land.UnlockCondition
			if cond != nil && level >= cond.NeedLevel && gold >= cond.NeedGold {
				if _, err := f.net.UnlockLand(land.Id); err != nil {
					f.logger.Warnf("\u89e3\u9501", "\u571f\u5730#%d \u5931\u8d25: %v", land.Id, err)
				} else {
					f.logger.Infof("解锁", "土地#%d 成功 (花费%d金币)", land.Id, cond.NeedGold)
					f.sc.Record(model.OpUnlockLand, 1, -cond.NeedGold, 0)
					unlocked++
					gold -= cond.NeedGold
				}
				time.Sleep(200 * time.Millisecond)
			}
		}

		if land.Unlocked && land.CouldUpgrade {
			cond := land.UpgradeCondition
			if cond != nil && level >= cond.NeedLevel && gold >= cond.NeedGold {
				if _, err := f.net.UpgradeLand(land.Id); err != nil {
					f.logger.Warnf("\u5347\u7ea7", "\u571f\u5730#%d Lv%d\u2192Lv%d \u5931\u8d25: %v", land.Id, land.Level, land.Level+1, err)
				} else {
					f.logger.Infof("升级", "土地#%d Lv%d→Lv%d (花费%d金币)", land.Id, land.Level, land.Level+1, cond.NeedGold)
					f.sc.Record(model.OpUpgradeLand, 1, -cond.NeedGold, 0)
					upgraded++
					gold -= cond.NeedGold
				}
				time.Sleep(200 * time.Millisecond)
			}
		}
	}
	return
}

// selectSeedByStrategy builds SeedCandidates from shop data + yield cache,
// applies the composable strategy rules, and returns the best matching shop goods.
func (f *FarmWorker) selectSeedByStrategy(strategy *PlantingStrategyConfig, available []shopSeedCandidate, landsCount int) *shoppb.GoodsInfo {
	if f.gc == nil {
		return nil
	}
	if strategy != nil && strategy.Mode == StrategyModeFastestLevelUp {
		_, _, statuses := f.lands.Get()
		cropByLandID := make(map[int64]int64, len(statuses))
		for _, ls := range statuses {
			cropByLandID[ls.ID] = ls.CropID
		}
		emptyLandIDs := make([]int64, 0, len(statuses))
		for _, ls := range statuses {
			if !ls.Unlocked || ls.CropID > 0 {
				continue
			}
			if ls.MasterLandID > 0 && ls.MasterLandID != ls.ID {
				if cropByLandID[ls.MasterLandID] > 0 {
					continue
				}
			}
			emptyLandIDs = append(emptyLandIDs, ls.ID)
		}
		return f.findFastestLevelUpSeed(emptyLandIDs, available)
	}

	// Build yield lookup from game config
	yieldRows := f.gc.GetSeedYieldRows()
	yieldMap := make(map[int]*SeedYieldRow, len(yieldRows))
	for i := range yieldRows {
		yieldMap[yieldRows[i].SeedID] = &yieldRows[i]
	}

	// Build SeedCandidates from available shop goods enriched with yield data
	var candidates []SeedCandidate
	for _, c := range available {
		seedID := int(c.goods.ItemId)
		sc := SeedCandidate{
			SeedID:        seedID,
			GoodsID:       c.goods.Id,
			Name:          f.gc.GetPlantNameBySeedID(seedID),
			RequiredLevel: int(c.requiredLevel),
			Price:         int(c.goods.Price),
		}

		// Enrich with yield data if available
		if yr, ok := yieldMap[seedID]; ok {
			sc.ExpPerHarvest = yr.ExpHarvest
			sc.Seasons = yr.Seasons
			sc.GrowTimeSec = yr.GrowTimeSec
			sc.ExpEfficiency = yr.FarmExpPerHourNormal
			sc.GrowTimeNormalFert = yr.GrowTimeNormalFert
			if yr.Price > 0 {
				sc.GoldEfficiency = float64(yr.ExpHarvest*yr.Seasons) / float64(yr.Price)
			}
		}

		candidates = append(candidates, sc)
	}

	if len(candidates) == 0 {
		return nil
	}

	// Apply strategy pipeline
	result := ApplyStrategy(strategy, candidates)
	if len(result) == 0 {
		return nil
	}

	// Find the shop goods matching the top candidate
	topSeedID := result[0].SeedID
	for _, c := range available {
		if int(c.goods.ItemId) == topSeedID {
			desc := FormatStrategyDescription(strategy)
			f.logger.Infof("策略", "%s → %s", desc, result[0].Name)
			return c.goods
		}
	}
	return nil
}
