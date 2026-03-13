package bot

import (
	"sync"

	"qq-farm-bot/internal/model"
)

// LandHarvestInfo holds harvest timing data for level-up estimation.
type LandHarvestInfo struct {
	LandID        int64
	CropID        int64 // plant ID for GameConfig lookups
	Season        int64 // current season (1 or 2)
	MatureTimeSec int64 // unix timestamp when crop matures
	CropExp       int   // base exp from GameConfig
	CycleTimeSec  int64 // actual growth duration on this land (seconds)
	IsMature      bool  // already mature, waiting for harvest
	IsGrowing     bool  // currently growing (not yet mature)
	ExpBonusPct   int64 // land buff: plant_exp_bonus percentage
	TimeReducePct int64 // land buff: planting_time_reduction percentage
	YieldBonusPct int64 // land buff: plant_yield_bonus percentage
}

// LandBuffInfo holds per-land buff data for the fastest-levelup simulation.
type LandBuffInfo struct {
	ExpBonusPct   int64
	TimeReducePct int64
}

// LandCache stores the latest farm land status for dashboard display.
type LandCache struct {
	mu            sync.RWMutex
	totalLands    int
	unlockedLands int
	lands         []model.LandStatus
	harvestInfos  []LandHarvestInfo
}

func NewLandCache() *LandCache {
	return &LandCache{}
}

func (lc *LandCache) Update(totalLands, unlockedLands int, lands []model.LandStatus, harvestInfos []LandHarvestInfo) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.totalLands = totalLands
	lc.unlockedLands = unlockedLands
	lc.lands = lands
	lc.harvestInfos = harvestInfos
}

func (lc *LandCache) Get() (totalLands, unlockedLands int, lands []model.LandStatus) {
	lc.mu.RLock()
	defer lc.mu.RUnlock()
	return lc.totalLands, lc.unlockedLands, lc.lands
}

func (lc *LandCache) GetHarvestInfo() []LandHarvestInfo {
	lc.mu.RLock()
	defer lc.mu.RUnlock()
	result := make([]LandHarvestInfo, len(lc.harvestInfos))
	copy(result, lc.harvestInfos)
	return result
}

// GetLandBuffsByID returns buff data for the specified land IDs.
// Missing IDs return zero-value LandBuffInfo (no bonus).
func (lc *LandCache) GetLandBuffsByID(landIDs []int64) map[int64]LandBuffInfo {
	lc.mu.RLock()
	defer lc.mu.RUnlock()
	result := make(map[int64]LandBuffInfo, len(landIDs))
	// build index from lc.lands
	index := make(map[int64]model.LandStatus, len(lc.lands))
	for _, land := range lc.lands {
		index[land.ID] = land
	}
	for _, id := range landIDs {
		if land, ok := index[id]; ok {
			result[id] = LandBuffInfo{
				ExpBonusPct:   land.ExpBonusPct,
				TimeReducePct: land.TimeReducePct,
			}
		} else {
			result[id] = LandBuffInfo{}
		}
	}
	return result
}
