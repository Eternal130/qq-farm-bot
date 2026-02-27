package bot

import (
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"

	"qq-farm-bot/internal/model"

	"qq-farm-bot/proto/plantpb"
	"qq-farm-bot/proto/shoppb"
)

const normalFertilizerID = 1011

// FarmWorker handles all farm automation logic.
type FarmWorker struct {
	net    *Network
	logger *Logger
	cfg    *BotConfig
	gc     *GameConfig
	lands  *LandCache
}

func NewFarmWorker(net *Network, logger *Logger, cfg *BotConfig, lands *LandCache) *FarmWorker {
	return &FarmWorker{net: net, logger: logger, cfg: cfg, gc: GetGameConfig(), lands: lands}
}

// RunLoop runs the farm check loop until context is cancelled.
func (f *FarmWorker) RunLoop() {
	// Initial delay
	select {
	case <-time.After(2 * time.Second):
	case <-f.net.ctx.Done():
		return
	}

	for {
		f.checkFarm()
		select {
		case <-time.After(time.Duration(f.cfg.FarmInterval) * time.Second):
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

	// Auto unlock & upgrade lands before analyzing
	unlockedNew, upgradedNew := f.autoUnlockAndUpgrade(lands)
	if unlockedNew > 0 || upgradedNew > 0 {
		// Re-fetch lands after unlock/upgrade to get updated state
		landsReply, err = f.net.AllLands()
		if err != nil {
			f.logger.Warnf("巡田", "重新获取土地失败: %v", err)
			return
		}
		lands = landsReply.Lands
	}

	status := f.analyzeLands(lands)
	unlockedCount := 0
	for _, land := range lands {
		if land.Unlocked {
			unlockedCount++
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
		parts = append(parts, fmt.Sprintf("空:%d", len(status.empty)))
	}
	parts = append(parts, fmt.Sprintf("长:%d", len(status.growing)))

	hasWork := len(status.harvestable) > 0 || len(status.needWeed) > 0 || len(status.needBug) > 0 ||
		len(status.needWater) > 0 || len(status.dead) > 0 || len(status.empty) > 0

	var actions []string

	// Record unlock/upgrade actions
	if unlockedNew > 0 {
		actions = append(actions, fmt.Sprintf("解锁%d", unlockedNew))
	}
	if upgradedNew > 0 {
		actions = append(actions, fmt.Sprintf("升级%d", upgradedNew))
	}
	if unlockedNew > 0 || upgradedNew > 0 {
		hasWork = true
	}

	// Batch operations: weed, bug, water
	if len(status.needWeed) > 0 {
		if err := f.weedOut(status.needWeed); err == nil {
			actions = append(actions, fmt.Sprintf("除草%d", len(status.needWeed)))
		}
	}
	if len(status.needBug) > 0 {
		if err := f.insecticide(status.needBug); err == nil {
			actions = append(actions, fmt.Sprintf("除虫%d", len(status.needBug)))
		}
	}
	if len(status.needWater) > 0 {
		if err := f.waterLand(status.needWater); err == nil {
			actions = append(actions, fmt.Sprintf("浇水%d", len(status.needWater)))
		}
	}

	// Harvest
	var harvestedLands []int64
	if len(status.harvestable) > 0 {
		if err := f.harvest(status.harvestable); err == nil {
			actions = append(actions, fmt.Sprintf("收获%d", len(status.harvestable)))
			harvestedLands = status.harvestable
		}
	}

	// Remove dead + plant + fertilize
	allDead := append(status.dead, harvestedLands...)
	allEmpty := status.empty
	if len(allDead) > 0 || len(allEmpty) > 0 {
		f.autoPlant(allDead, allEmpty, unlockedCount)
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

func (f *FarmWorker) updateLandCache(lands []*plantpb.LandInfo) {
	if f.lands == nil {
		return
	}
	nowSec := time.Now().Unix()
	totalLands := len(lands)
	unlockedCount := 0
	var statuses []model.LandStatus
	var harvestInfos []LandHarvestInfo
	for _, land := range lands {
		ls := model.LandStatus{
			ID:       land.Id,
			Level:    land.Level,
			MaxLevel: land.MaxLevel,
			Unlocked: land.Unlocked,
		}
		if land.Unlocked {
			unlockedCount++
		}
		if land.Plant != nil && len(land.Plant.Phases) > 0 {
			ls.CropID = land.Plant.Id
			ls.CropName = f.gc.GetPlantName(int(land.Plant.Id))
			currentPhase := getCurrentPhase(land.Plant.Phases, nowSec)
			if currentPhase != nil {
				if name, ok := phaseNames[currentPhase.Phase]; ok {
					ls.Phase = name
				} else {
					ls.Phase = fmt.Sprintf("阶段%d", currentPhase.Phase)
				}
			}

			// Build harvest info for level-up estimation
			matureTime := getMatureTimeSec(land.Plant.Phases)
			plantTime := getPlantStartTimeSec(land.Plant.Phases)
			if matureTime > 0 && plantTime > 0 && matureTime > plantTime {
				hi := LandHarvestInfo{
					LandID:        land.Id,
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

	for _, land := range lands {
		id := land.Id
		if !land.Unlocked {
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

		switch plantpb.PlantPhase(phase.Phase) {
		case plantpb.PlantPhase_DEAD:
			s.dead = append(s.dead, id)
		case plantpb.PlantPhase_MATURE:
			s.harvestable = append(s.harvestable, id)
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
		}
	}
	return s
}

func getCurrentPhase(phases []*plantpb.PlantPhaseInfo, nowSec int64) *plantpb.PlantPhaseInfo {
	if len(phases) == 0 {
		return nil
	}
	for i := len(phases) - 1; i >= 0; i-- {
		bt := toTimeSec(phases[i].BeginTime)
		if bt > 0 && bt <= nowSec {
			return phases[i]
		}
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
	if val > 1e12 {
		return val / 1000
	}
	return val
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

func (f *FarmWorker) removePlant(landIDs []int64) error {
	req := &plantpb.RemovePlantRequest{LandIds: landIDs}
	body, _ := proto.Marshal(req)
	_, err := f.net.SendRequest("gamepb.plantpb.PlantService", "RemovePlant", body)
	return err
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

func (f *FarmWorker) autoPlant(deadLands, emptyLands []int64, unlockedCount int) {
	toLant := append([]int64{}, emptyLands...)

	// Remove dead plants
	if len(deadLands) > 0 {
		if err := f.removePlant(deadLands); err == nil {
			f.logger.Infof("铲除", "已铲除 %d 块", len(deadLands))
		}
		toLant = append(toLant, deadLands...)
	}

	if len(toLant) == 0 {
		return
	}

	// Find best seed from shop
	bestSeed, err := f.findBestSeed(unlockedCount)
	if err != nil || bestSeed == nil {
		return
	}

	seedName := f.gc.GetPlantNameBySeedID(int(bestSeed.ItemId))
	f.logger.Infof("商店", "最佳种子: %s 价格=%d金币", seedName, bestSeed.Price)

	// Buy seeds
	_, _, _, gold, _ := f.net.state.Get()
	needCount := int64(len(toLant))
	totalCost := bestSeed.Price * needCount
	if totalCost > gold {
		canBuy := gold / bestSeed.Price
		if canBuy <= 0 {
			f.logger.Warnf("商店", "金币不足")
			return
		}
		toLant = toLant[:canBuy]
	}

	buyReq := &shoppb.BuyGoodsRequest{GoodsId: bestSeed.Id, Num: int64(len(toLant)), Price: bestSeed.Price}
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
	f.logger.Infof("购买", "已购买 %s种子 x%d", f.gc.GetPlantNameBySeedID(int(actualSeedID)), len(toLant))

	// Plant seeds one by one
	planted := 0
	for _, landID := range toLant {
		req := &plantpb.PlantRequest{
			Items: []*plantpb.PlantItem{{SeedId: actualSeedID, LandIds: []int64{landID}}},
		}
		body, _ := proto.Marshal(req)
		if _, err := f.net.SendRequest("gamepb.plantpb.PlantService", "Plant", body); err == nil {
			planted++
		}
		time.Sleep(50 * time.Millisecond)
	}
	f.logger.Infof("种植", "已种植 %d 块", planted)

	// Fertilize
	if planted > 0 {
		fertilized := f.fertilize(toLant[:planted])
		if fertilized > 0 {
			f.logger.Infof("施肥", "已为 %d/%d 块地施肥", fertilized, planted)
		}
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

	type candidate struct {
		goods         *shoppb.GoodsInfo
		requiredLevel int64
	}
	var available []candidate

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
		available = append(available, candidate{goods: goods, requiredLevel: reqLevel})
	}

	if len(available) == 0 {
		return nil, fmt.Errorf("没有可购买的种子")
	}

	if f.cfg.ForceLowest {
		// Sort by level asc, then price asc
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
			// Find matching goods in available shop items
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

// autoUnlockAndUpgrade checks all lands and attempts to unlock/upgrade eligible ones.
// Returns counts of successfully unlocked and upgraded lands.
func (f *FarmWorker) autoUnlockAndUpgrade(lands []*plantpb.LandInfo) (unlocked, upgraded int) {
	_, level, _, gold, _ := f.net.state.Get()

	for _, land := range lands {
		// Try unlock
		if !land.Unlocked && land.CouldUnlock {
			cond := land.UnlockCondition
			if cond != nil && level >= cond.NeedLevel && gold >= cond.NeedGold {
				if _, err := f.net.UnlockLand(land.Id); err != nil {
					f.logger.Warnf("\u89e3\u9501", "\u571f\u5730#%d \u5931\u8d25: %v", land.Id, err)
				} else {
					f.logger.Infof("\u89e3\u9501", "\u571f\u5730#%d \u6210\u529f (\u82b1\u8d39%d\u91d1\u5e01)", land.Id, cond.NeedGold)
					unlocked++
					gold -= cond.NeedGold // track remaining gold
				}
				time.Sleep(200 * time.Millisecond)
			}
		}

		// Try upgrade
		if land.Unlocked && land.CouldUpgrade {
			cond := land.UpgradeCondition
			if cond != nil && level >= cond.NeedLevel && gold >= cond.NeedGold {
				if _, err := f.net.UpgradeLand(land.Id); err != nil {
					f.logger.Warnf("\u5347\u7ea7", "\u571f\u5730#%d Lv%d\u2192Lv%d \u5931\u8d25: %v", land.Id, land.Level, land.Level+1, err)
				} else {
					f.logger.Infof("\u5347\u7ea7", "\u571f\u5730#%d Lv%d\u2192Lv%d (\u82b1\u8d39%d\u91d1\u5e01)", land.Id, land.Level, land.Level+1, cond.NeedGold)
					upgraded++
					gold -= cond.NeedGold
				}
				time.Sleep(200 * time.Millisecond)
			}
		}
	}
	return
}
