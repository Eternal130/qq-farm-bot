package bot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type PlantConfig struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	SeedID     int    `json:"seed_id"`
	Exp        int    `json:"exp"`
	GrowPhases string `json:"grow_phases"`
	Seasons    int    `json:"seasons"`
	Fruit      struct {
		ID    int `json:"id"`
		Count int `json:"count"`
	} `json:"fruit"`
}

type RoleLevelConfig struct {
	Level int   `json:"level"`
	Exp   int64 `json:"exp"`
}

// SeedShopExport represents the seed shop export JSON structure
type SeedShopExport struct {
	ExportedAt string          `json:"exportedAt"`
	Source     string          `json:"source"`
	Count      int             `json:"count"`
	Rows       []SeedShopEntry `json:"rows"`
}

// SeedShopEntry represents a single seed entry from shop export
type SeedShopEntry struct {
	SeedID        int    `json:"seedId"`
	GoodsID       int    `json:"goodsId"`
	PlantID       int    `json:"plantId"`
	Name          string `json:"name"`
	RequiredLevel int    `json:"requiredLevel"`
	Price         int    `json:"price"`
	Exp           int    `json:"exp"`
	GrowTimeSec   int    `json:"growTimeSec"`
	FruitID       int    `json:"fruitId"`
	FruitCount    int    `json:"fruitCount"`
}

// PlantPhaseData holds parsed phase info for fertilizer optimization.
type PlantPhaseData struct {
	PhaseDurations       []int // all non-zero growth phase durations
	MaxPhaseDuration     int   // longest phase in season 1
	MaxPhaseIndex        int   // 0-based index of longest phase
	TotalGrowTime        int   // sum of all phase durations
	AllPhasesEqual       bool  // true if all phases have the same duration
	Season2Phases        []int // last 3 non-zero phases (for multi-season crops)
	Season2GrowTime      int   // sum of season 2 phases
	Season2MaxPhase      int   // longest phase in season 2
	Season2MaxPhaseIndex int   // index of longest phase within Season2Phases
	Season2AllEqual      bool  // true if all season 2 phases are equal
}

// SeedYieldRow contains calculated yield info for a seed
type SeedYieldRow struct {
	SeedID               int
	Name                 string
	RequiredLevel        int
	Price                int
	ExpHarvest           int // base exp per season
	Seasons              int
	GrowTimeSec          int // season 1 total grow time
	Season2GrowTimeSec   int // season 2 total grow time (0 if single season)
	NormalFertReduceSec  int // time saved by fertilizer in season 1 (max phase)
	Season2FertReduceSec int // time saved by fertilizer in season 2
	GrowTimeNormalFert   int // effective grow time with fert (both seasons combined)
	FarmExpPerHourNormal float64
}

type GameConfig struct {
	mu             sync.RWMutex
	plants         []PlantConfig
	plantMap       map[int]*PlantConfig // id -> plant
	seedToPlant    map[int]*PlantConfig // seed_id -> plant
	fruitToPlant   map[int]*PlantConfig // fruit_id -> plant
	levelExp       []RoleLevelConfig
	levelExpMap    map[int]int64 // level -> cumulative exp
	seedShopData   *SeedShopExport
	seedYieldCache []SeedYieldRow
	plantPhaseData map[int]*PlantPhaseData // seed_id -> phase data
}

var globalGameConfig *GameConfig
var gameConfigOnce sync.Once

func LoadGameConfig(configDir string) *GameConfig {
	gameConfigOnce.Do(func() {
		globalGameConfig = &GameConfig{
			plantMap:       make(map[int]*PlantConfig),
			seedToPlant:    make(map[int]*PlantConfig),
			fruitToPlant:   make(map[int]*PlantConfig),
			levelExpMap:    make(map[int]int64),
			plantPhaseData: make(map[int]*PlantPhaseData),
		}
		globalGameConfig.load(configDir)
	})
	return globalGameConfig
}

func GetGameConfig() *GameConfig {
	return globalGameConfig
}

func (gc *GameConfig) load(configDir string) {
	// Load Plant.json
	plantPath := filepath.Join(configDir, "Plant.json")
	if data, err := os.ReadFile(plantPath); err == nil {
		var plants []PlantConfig
		if err := json.Unmarshal(data, &plants); err == nil {
			gc.plants = plants
			for i := range gc.plants {
				p := &gc.plants[i]
				gc.plantMap[p.ID] = p
				if p.SeedID > 0 {
					gc.seedToPlant[p.SeedID] = p
				}
				if p.Fruit.ID > 0 {
					gc.fruitToPlant[p.Fruit.ID] = p
				}
			}
			fmt.Printf("[配置] 已加载植物配置 (%d 种)\n", len(plants))
		}
	}

	// Load RoleLevel.json
	levelPath := filepath.Join(configDir, "RoleLevel.json")
	if data, err := os.ReadFile(levelPath); err == nil {
		if err := json.Unmarshal(data, &gc.levelExp); err == nil {
			for _, l := range gc.levelExp {
				gc.levelExpMap[l.Level] = l.Exp
			}
			fmt.Printf("[配置] 已加载等级经验表 (%d 级)\n", len(gc.levelExp))
		}
	}

	// Load seed-shop-merged-export.json for yield calculation
	seedShopPath := filepath.Join(configDir, "seed-shop-merged-export.json")
	if data, err := os.ReadFile(seedShopPath); err == nil {
		var export SeedShopExport
		if err := json.Unmarshal(data, &export); err == nil {
			gc.seedShopData = &export
			fmt.Printf("[配置] 已加载种子商店数据 (%d 种)\n", len(export.Rows))
		}
	}

	// Build phase data for fertilizer optimization
	gc.buildPlantPhaseData()

	// Calculate yield for all seeds
	gc.calculateSeedYield(18) // default 18 lands
}

func (gc *GameConfig) GetPlantName(plantID int) string {
	if gc == nil {
		return fmt.Sprintf("植物%d", plantID)
	}
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	if p, ok := gc.plantMap[plantID]; ok {
		return p.Name
	}
	return fmt.Sprintf("植物%d", plantID)
}

func (gc *GameConfig) GetPlantNameBySeedID(seedID int) string {
	if gc == nil {
		return fmt.Sprintf("种子%d", seedID)
	}
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	if p, ok := gc.seedToPlant[seedID]; ok {
		return p.Name
	}
	return fmt.Sprintf("种子%d", seedID)
}

func (gc *GameConfig) GetPlantExp(plantID int) int {
	if gc == nil {
		return 0
	}
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	if p, ok := gc.plantMap[plantID]; ok {
		return p.Exp
	}
	return 0
}

func (gc *GameConfig) GetFruitName(fruitID int) string {
	if gc == nil {
		return fmt.Sprintf("果实%d", fruitID)
	}
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	if p, ok := gc.fruitToPlant[fruitID]; ok {
		return p.Name
	}
	return fmt.Sprintf("果实%d", fruitID)
}

func (gc *GameConfig) IsFruitID(id int) bool {
	if gc == nil {
		return false
	}
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	_, ok := gc.fruitToPlant[id]
	return ok
}

func (gc *GameConfig) GetPlantGrowTime(plantID int) int {
	if gc == nil {
		return 0
	}
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	p, ok := gc.plantMap[plantID]
	if !ok || p.GrowPhases == "" {
		return 0
	}
	total := 0
	for _, phase := range strings.Split(p.GrowPhases, ";") {
		phase = strings.TrimSpace(phase)
		if phase == "" {
			continue
		}
		parts := strings.Split(phase, ":")
		if len(parts) == 2 {
			if v, err := strconv.Atoi(parts[1]); err == nil {
				total += v
			}
		}
	}
	return total
}

func (gc *GameConfig) FormatGrowTime(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%d秒", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%d分钟", seconds/60)
	}
	hours := seconds / 3600
	mins := (seconds % 3600) / 60
	if mins > 0 {
		return fmt.Sprintf("%d小时%d分", hours, mins)
	}
	return fmt.Sprintf("%d小时", hours)
}

// Constants for yield calculation
const (
	normalFertPlantsPer2Sec = 12
	normalFertPlantSpeed    = normalFertPlantsPer2Sec / 2 // 6 块/秒
)

// parseGrowPhases extracts all non-zero phase durations from a grow_phases string.
// Format: "name:seconds;name:seconds;...;mature:0;"
func parseGrowPhases(growPhases string) []int {
	var durations []int
	for _, phase := range strings.Split(growPhases, ";") {
		phase = strings.TrimSpace(phase)
		if phase == "" {
			continue
		}
		parts := strings.Split(phase, ":")
		if len(parts) == 2 {
			if v, err := strconv.Atoi(parts[1]); err == nil && v > 0 {
				durations = append(durations, v)
			}
		}
	}
	return durations
}

// parseAllPhaseDurations extracts ALL phase durations including zero (mature).
// Used for season 2 calculation: the game takes the last 3 phases from the full list.
func parseAllPhaseDurations(growPhases string) []int {
	var durations []int
	for _, phase := range strings.Split(growPhases, ";") {
		phase = strings.TrimSpace(phase)
		if phase == "" {
			continue
		}
		parts := strings.Split(phase, ":")
		if len(parts) == 2 {
			if v, err := strconv.Atoi(parts[1]); err == nil {
				durations = append(durations, v)
			}
		}
	}
	return durations
}

// buildPlantPhaseData parses phase durations for each plant and computes
// max-phase info for optimal fertilization.
func (gc *GameConfig) buildPlantPhaseData() {
	for _, p := range gc.plants {
		if p.GrowPhases == "" || p.SeedID <= 0 {
			continue
		}

		durations := parseGrowPhases(p.GrowPhases)
		if len(durations) == 0 {
			continue
		}

		pd := &PlantPhaseData{
			PhaseDurations: durations,
		}

		// Find max phase and total grow time for season 1
		for i, d := range durations {
			pd.TotalGrowTime += d
			if d > pd.MaxPhaseDuration {
				pd.MaxPhaseDuration = d
				pd.MaxPhaseIndex = i
			}
		}

		// Check if all phases are equal (no benefit from delayed fertilization)
		pd.AllPhasesEqual = true
		for _, d := range durations {
			if d != durations[0] {
				pd.AllPhasesEqual = false
				break
			}
		}

		// For multi-season crops: season 2 uses the last 3 phases from the FULL
		// phase list (including 成熟:0), then filters to non-zero growth durations.
		seasons := p.Seasons
		if seasons < 1 {
			seasons = 1
		}
		if seasons >= 2 {
			allPhases := parseAllPhaseDurations(p.GrowPhases)
			if len(allPhases) >= 3 {
				last3 := allPhases[len(allPhases)-3:]
				var s2Phases []int
				for _, d := range last3 {
					if d > 0 {
						s2Phases = append(s2Phases, d)
					}
				}
				if len(s2Phases) > 0 {
					pd.Season2Phases = s2Phases
					for i, d := range s2Phases {
						pd.Season2GrowTime += d
						if d > pd.Season2MaxPhase {
							pd.Season2MaxPhase = d
							pd.Season2MaxPhaseIndex = i
						}
					}
					pd.Season2AllEqual = true
					for _, d := range s2Phases {
						if d != s2Phases[0] {
							pd.Season2AllEqual = false
							break
						}
					}
				}
			}
		}

		gc.plantPhaseData[p.SeedID] = pd
	}
}

// calculateSeedYield calculates experience yield for all seeds, accounting for
// multi-season crops and optimal fertilizer usage (skip longest phase).
func (gc *GameConfig) calculateSeedYield(lands int) {
	if gc.seedShopData == nil || len(gc.seedShopData.Rows) == 0 {
		return
	}

	plantSecondsNormalFert := float64(lands) / normalFertPlantSpeed
	var rows []SeedYieldRow

	for _, s := range gc.seedShopData.Rows {
		if s.SeedID <= 0 || s.GrowTimeSec <= 0 {
			continue
		}

		pd := gc.plantPhaseData[s.SeedID]
		plant := gc.seedToPlant[s.SeedID]

		seasons := 1
		if plant != nil && plant.Seasons >= 2 {
			seasons = plant.Seasons
		}

		var s1FertReduce, s2FertReduce, s2GrowTime int

		if pd != nil {
			// Season 1: skip longest phase
			s1FertReduce = pd.MaxPhaseDuration

			// Season 2: skip longest of last 3 phases
			if seasons >= 2 {
				s2GrowTime = pd.Season2GrowTime
				s2FertReduce = pd.Season2MaxPhase
			}
		}

		// Season 1 grow time with fertilizer
		s1GrowFert := s.GrowTimeSec - s1FertReduce
		if s1GrowFert < 1 {
			s1GrowFert = 1
		}

		// Total effective grow time (both seasons with fert)
		totalGrowFert := s1GrowFert
		totalExp := s.Exp
		if seasons >= 2 && s2GrowTime > 0 {
			s2GrowFert := s2GrowTime - s2FertReduce
			if s2GrowFert < 1 {
				s2GrowFert = 1
			}
			totalGrowFert += s2GrowFert
			totalExp += s.Exp // second season gives same exp
		}

		cycleSecNormalFert := float64(totalGrowFert) + plantSecondsNormalFert
		farmExpPerHourNormal := float64(lands*totalExp) / cycleSecNormalFert * 3600

		rows = append(rows, SeedYieldRow{
			SeedID:               s.SeedID,
			Name:                 s.Name,
			RequiredLevel:        s.RequiredLevel,
			Price:                s.Price,
			ExpHarvest:           s.Exp,
			Seasons:              seasons,
			GrowTimeSec:          s.GrowTimeSec,
			Season2GrowTimeSec:   s2GrowTime,
			NormalFertReduceSec:  s1FertReduce,
			Season2FertReduceSec: s2FertReduce,
			GrowTimeNormalFert:   totalGrowFert,
			FarmExpPerHourNormal: farmExpPerHourNormal,
		})
	}

	// Sort by FarmExpPerHourNormal descending
	for i := 0; i < len(rows); i++ {
		for j := i + 1; j < len(rows); j++ {
			if rows[j].FarmExpPerHourNormal > rows[i].FarmExpPerHourNormal {
				rows[i], rows[j] = rows[j], rows[i]
			}
		}
	}

	gc.seedYieldCache = rows
}

// GetPlantingRecommendation returns seed recommendations based on experience efficiency
func (gc *GameConfig) GetPlantingRecommendation(level, lands int, topN int) []SeedYieldRow {
	if gc == nil || len(gc.seedYieldCache) == 0 {
		return nil
	}

	// Recalculate if lands count differs significantly
	if lands > 0 && lands != 18 {
		gc.calculateSeedYield(lands)
	}

	var result []SeedYieldRow
	for _, r := range gc.seedYieldCache {
		if r.RequiredLevel <= level {
			result = append(result, r)
			if len(result) >= topN {
				break
			}
		}
	}
	return result
}

// GetPlantPhaseData returns phase timing data for a plant (looked up by plant ID).
func (gc *GameConfig) GetPlantPhaseData(plantID int) *PlantPhaseData {
	if gc == nil {
		return nil
	}
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	p, ok := gc.plantMap[plantID]
	if !ok {
		return nil
	}
	return gc.plantPhaseData[p.SeedID]
}

 // GetPlantPhaseDataBySeedID returns phase timing data for a plant (looked up by seed ID).
func (gc *GameConfig) GetPlantPhaseDataBySeedID(seedID int) *PlantPhaseData {
	if gc == nil {
		return nil
	}
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	return gc.plantPhaseData[seedID]
}

// GetPlantSeasons returns the number of seasons for a plant (1 = normal, 2 = multi-season).
func (gc *GameConfig) GetPlantSeasons(plantID int) int {
	if gc == nil {
		return 1
	}
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	if p, ok := gc.plantMap[plantID]; ok && p.Seasons >= 2 {
		return p.Seasons
	}
	return 1
}

// GetNextLevelExp returns the cumulative exp required for the next level.
// Returns (nextLevelExp, hasNextLevel). If already max level, returns (0, false).
func (gc *GameConfig) GetNextLevelExp(currentLevel int) (int64, bool) {
	if gc == nil {
		return 0, false
	}
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	nextLevel := currentLevel + 1
	if exp, ok := gc.levelExpMap[nextLevel]; ok {
		return exp, true
	}
	return 0, false
}
