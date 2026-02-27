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

// SeedYieldRow contains calculated yield info for a seed
type SeedYieldRow struct {
	SeedID               int
	Name                 string
	RequiredLevel        int
	Price                int
	ExpHarvest           int
	GrowTimeSec          int
	NormalFertReduceSec  int
	GrowTimeNormalFert   int
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
	firstPhaseTime map[int]int // seed_id -> first phase time (for fertilizer reduction)
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
			firstPhaseTime: make(map[int]int),
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

	// Build first phase time map for fertilizer reduction calculation
	gc.buildFirstPhaseTimeMap()

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

// buildFirstPhaseTimeMap extracts the first phase duration for each plant
// This is used to calculate fertilizer reduction (normal fert removes one phase)
func (gc *GameConfig) buildFirstPhaseTimeMap() {
	for _, p := range gc.plants {
		if p.GrowPhases == "" {
			continue
		}
		// Parse grow_phases like "种子:1;发芽:1;成熟:0;"
		for _, phase := range strings.Split(p.GrowPhases, ";") {
			phase = strings.TrimSpace(phase)
			if phase == "" {
				continue
			}
			parts := strings.Split(phase, ":")
			if len(parts) == 2 {
				if v, err := strconv.Atoi(parts[1]); err == nil && v > 0 {
					gc.firstPhaseTime[p.SeedID] = v
					break // Only need first non-zero phase
				}
			}
		}
	}
}

// calculateSeedYield calculates experience yield for all seeds
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

		reduceSec := gc.firstPhaseTime[s.SeedID]
		growTimeNormalFert := s.GrowTimeSec - reduceSec
		if growTimeNormalFert < 1 {
			growTimeNormalFert = 1
		}

		cycleSecNormalFert := float64(growTimeNormalFert) + plantSecondsNormalFert
		farmExpPerHourNormal := float64(lands*s.Exp) / cycleSecNormalFert * 3600

		rows = append(rows, SeedYieldRow{
			SeedID:               s.SeedID,
			Name:                 s.Name,
			RequiredLevel:        s.RequiredLevel,
			Price:                s.Price,
			ExpHarvest:           s.Exp,
			GrowTimeSec:          s.GrowTimeSec,
			NormalFertReduceSec:  reduceSec,
			GrowTimeNormalFert:   growTimeNormalFert,
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
