package bot

import (
	"sort"
	"time"

	"google.golang.org/protobuf/proto"

	"qq-farm-bot/internal/model"

	"qq-farm-bot/proto/itempb"
	"qq-farm-bot/proto/plantpb"
)

type bigSeed struct {
	itemID int64
	count  int64
}

type landBlock struct {
	ids        [4]int64
	totalLevel int64
}

// handleBigSeedPlanting checks the backpack for size>=2 seeds and tries to plant
// them on the highest-level 2×2 land blocks. If not enough empty blocks are
// available, it reserves the highest-level empty lands for future 2×2 planting.
// Returns the remaining empty lands available for normal (size=1) planting.
func (f *FarmWorker) handleBigSeedPlanting(emptyLands []int64, allLands []*plantpb.LandInfo) []int64 {
	bagSeeds := f.getBigSeedsFromBag()
	if len(bagSeeds) == 0 {
		f.reservedForBigSeed = make(map[int64]bool)
		return emptyLands
	}

	totalSeedCount := 0
	for _, s := range bagSeeds {
		totalSeedCount += int(s.count)
	}

	emptySet := make(map[int64]bool, len(emptyLands))
	for _, id := range emptyLands {
		emptySet[id] = true
	}
	levelMap := make(map[int64]int64)
	for _, land := range allLands {
		levelMap[land.Id] = land.Level
	}

	allBlocks := all2x2BlockPositions(allLands)

	var emptyBlocks []landBlock
	for _, block := range allBlocks {
		allEmpty := true
		var total int64
		for _, id := range block {
			if !emptySet[id] {
				allEmpty = false
				break
			}
			total += levelMap[id]
		}
		if allEmpty {
			emptyBlocks = append(emptyBlocks, landBlock{ids: block, totalLevel: total})
		}
	}
	sort.Slice(emptyBlocks, func(i, j int) bool {
		return emptyBlocks[i].totalLevel > emptyBlocks[j].totalLevel
	})

	consumed := make(map[int64]bool)
	totalPlanted := 0

	for si := range bagSeeds {
		seed := &bagSeeds[si]
		for seed.count > 0 {
			picked := -1
			for bi, block := range emptyBlocks {
				conflict := false
				for _, id := range block.ids {
					if consumed[id] {
						conflict = true
						break
					}
				}
				if !conflict {
					picked = bi
					break
				}
			}
			if picked < 0 {
				break
			}

			block := emptyBlocks[picked]
			blockLandIDs := block.ids[:]
			seedName := f.gc.GetPlantNameBySeedID(int(seed.itemID))

			if f.plantBigSeedOnLands(seed.itemID, blockLandIDs, consumed) {
				f.logger.Infof("大种子", "种植 %s 于土地#%d (等级合计%d)", seedName, blockLandIDs[0], block.totalLevel)
				seed.count--
				totalPlanted++
			}
			emptyBlocks = append(emptyBlocks[:picked], emptyBlocks[picked+1:]...)
		}
	}

	seedsLeft := 0
	for _, s := range bagSeeds {
		seedsLeft += int(s.count)
	}
	if seedsLeft > 0 {
		sorted := sortLandIDsByLevel(emptyLands, levelMap)
		for si := range bagSeeds {
			seed := &bagSeeds[si]
			for seed.count > 0 {
				planted := false
				for _, landID := range sorted {
					if consumed[landID] {
						continue
					}
					seedName := f.gc.GetPlantNameBySeedID(int(seed.itemID))
					if f.plantBigSeedOnLands(seed.itemID, []int64{landID}, consumed) {
						f.logger.Infof("大种子", "种植 %s 于土地#%d", seedName, landID)
						seed.count--
						totalPlanted++
						planted = true
						break
					}
				}
				if !planted {
					break
				}
			}
		}
	}

	if totalPlanted > 0 {
		f.sc.RecordSimple(model.OpPlant, int64(totalPlanted))
	}

	var remaining []int64
	for _, id := range emptyLands {
		if !consumed[id] {
			remaining = append(remaining, id)
		}
	}

	seedsLeft = 0
	for _, s := range bagSeeds {
		seedsLeft += int(s.count)
	}

	if seedsLeft > 0 && len(remaining) > 0 {
		potential := findPotential2x2Lands(allBlocks, emptySet, consumed)

		type ll struct {
			id    int64
			level int64
		}
		var candidates []ll
		for id := range potential {
			candidates = append(candidates, ll{id, levelMap[id]})
		}
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].level > candidates[j].level
		})

		reserveCount := seedsLeft
		if reserveCount > len(candidates) {
			reserveCount = len(candidates)
		}

		f.reservedForBigSeed = make(map[int64]bool)
		for i := 0; i < reserveCount; i++ {
			f.reservedForBigSeed[candidates[i].id] = true
		}

		var unreserved []int64
		for _, id := range remaining {
			if !f.reservedForBigSeed[id] {
				unreserved = append(unreserved, id)
			}
		}

		f.logger.Infof("大种子", "预留 %d 块空地等待凑齐 2×2 种植 (%d颗大种子待种)", reserveCount, seedsLeft)
		return unreserved
	}

	f.reservedForBigSeed = make(map[int64]bool)
	return remaining
}

func (f *FarmWorker) plantBigSeedOnLands(seedID int64, landIDs []int64, consumed map[int64]bool) bool {
	plantReq := &plantpb.PlantRequest{
		Items: []*plantpb.PlantItem{{SeedId: seedID, LandIds: landIDs}},
	}
	plantBody, _ := proto.Marshal(plantReq)
	replyBody, err := f.net.SendRequest("gamepb.plantpb.PlantService", "Plant", plantBody)
	if err != nil {
		return false
	}

	for _, id := range landIDs {
		consumed[id] = true
		delete(f.fertilized, id)
	}

	plantReply := &plantpb.PlantReply{}
	proto.Unmarshal(replyBody, plantReply)
	for _, changedLand := range plantReply.Land {
		consumed[changedLand.Id] = true
		delete(f.fertilized, changedLand.Id)
	}

	time.Sleep(50 * time.Millisecond)
	return true
}

// getBigSeedsFromBag fetches the backpack and returns all seeds with plant size >= 2.
func (f *FarmWorker) getBigSeedsFromBag() []bigSeed {
	req := &itempb.BagRequest{}
	body, _ := proto.Marshal(req)
	replyBody, err := f.net.SendRequest("gamepb.itempb.ItemService", "Bag", body)
	if err != nil {
		return nil
	}
	reply := &itempb.BagReply{}
	proto.Unmarshal(replyBody, reply)

	if reply.ItemBag == nil || len(reply.ItemBag.Items) == 0 {
		return nil
	}

	var seeds []bigSeed
	for _, item := range reply.ItemBag.Items {
		if item.Count > 0 && f.gc.IsSeedID(int(item.Id)) {
			if f.gc.GetPlantSizeBySeedID(int(item.Id)) >= 2 {
				seeds = append(seeds, bigSeed{itemID: item.Id, count: item.Count})
			}
		}
	}
	return seeds
}

func detectGridCols(allLands []*plantpb.LandInfo) int {
	if len(allLands) == 0 {
		return 0
	}
	var ids []int64
	for _, land := range allLands {
		ids = append(ids, land.Id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	n := len(ids)
	minID := ids[0]
	maxID := ids[n-1]
	totalSlots := int(maxID - minID + 1)

	if totalSlots == n {
		for _, cols := range []int{6, 5, 4} {
			if totalSlots%cols == 0 {
				return cols
			}
		}
	}
	for _, cols := range []int{6, 5, 4} {
		if n%cols == 0 && totalSlots >= cols {
			return cols
		}
	}
	return 4
}

func all2x2BlockPositions(allLands []*plantpb.LandInfo) [][4]int64 {
	if len(allLands) < 4 {
		return nil
	}

	var ids []int64
	idSet := make(map[int64]bool)
	for _, land := range allLands {
		ids = append(ids, land.Id)
		idSet[land.Id] = true
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	minID := ids[0]
	maxID := ids[len(ids)-1]
	totalSlots := int(maxID - minID + 1)

	cols := detectGridCols(allLands)
	if cols == 0 || totalSlots < cols {
		return nil
	}

	rows := totalSlots / cols

	var blocks [][4]int64
	for r := 0; r < rows-1; r++ {
		for c := 0; c < cols-1; c++ {
			tl := minID + int64(r*cols+c)
			tr := minID + int64(r*cols+c+1)
			bl := minID + int64((r+1)*cols+c)
			br := minID + int64((r+1)*cols+c+1)
			if idSet[tl] && idSet[tr] && idSet[bl] && idSet[br] {
				blocks = append(blocks, [4]int64{tl, tr, bl, br})
			}
		}
	}

	return blocks
}

// findPotential2x2Lands returns all empty, non-consumed lands that appear
// in at least one valid 2×2 block position. These are lands worth reserving
// because they could become part of a 2×2 block when neighbors are harvested.
func findPotential2x2Lands(allBlocks [][4]int64, emptySet, consumed map[int64]bool) map[int64]bool {
	potential := make(map[int64]bool)
	for _, block := range allBlocks {
		for _, id := range block {
			if emptySet[id] && !consumed[id] {
				potential[id] = true
			}
		}
	}
	return potential
}

// sortLandIDsByLevel returns a copy of land IDs sorted by level descending.
func sortLandIDsByLevel(ids []int64, levelMap map[int64]int64) []int64 {
	sorted := make([]int64, len(ids))
	copy(sorted, ids)
	sort.Slice(sorted, func(i, j int) bool {
		return levelMap[sorted[i]] > levelMap[sorted[j]]
	})
	return sorted
}
