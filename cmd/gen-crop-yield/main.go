// cmd/gen-crop-yield/main.go generates web/src/data/cropYield.ts from game config data.
// Usage: go run ./cmd/gen-crop-yield > web/src/data/cropYield.ts
package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
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

type SeedShopExport struct {
	Rows []SeedShopEntry `json:"rows"`
}

type SeedShopEntry struct {
	SeedID      int    `json:"seedId"`
	PlantID     int    `json:"plantId"`
	Name        string `json:"name"`
	GrowTimeSec int    `json:"growTimeSec"`
	Exp         int    `json:"exp"`
	Price       int    `json:"price"`
	FruitID     int    `json:"fruitId"`
	FruitCount  int    `json:"fruitCount"`
}

type ItemInfo struct {
	ID    int `json:"id"`
	Price int `json:"price"`
}

type phaseData struct {
	durations       []int
	maxPhaseDur     int
	maxPhaseIdx     int
	totalGrow       int
	allEqual        bool
	season2Phases   []int
	season2Grow     int
	season2MaxPhase int
	season2MaxIdx   int
	season2AllEqual bool
}

const (
	lands                   = 18
	normalFertPlantsPer2Sec = 12
	normalFertPlantSpeed    = normalFertPlantsPer2Sec / 2 // 6 lands/sec
)

func parseGrowPhases(gp string) []int {
	var durations []int
	for _, phase := range strings.Split(gp, ";") {
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

// parseAllPhaseDurations returns ALL phase durations including zero (mature).
func parseAllPhaseDurations(gp string) []int {
	var durations []int
	for _, phase := range strings.Split(gp, ";") {
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

func buildPhaseData(durations []int, seasons int, growPhases string) *phaseData {
	pd := &phaseData{durations: durations}
	for i, d := range durations {
		pd.totalGrow += d
		if d > pd.maxPhaseDur {
			pd.maxPhaseDur = d
			pd.maxPhaseIdx = i
		}
	}
	pd.allEqual = true
	for _, d := range durations {
		if d != durations[0] {
			pd.allEqual = false
			break
		}
	}
	if seasons >= 2 {
		allPhases := parseAllPhaseDurations(growPhases)
		if len(allPhases) >= 3 {
			last3 := allPhases[len(allPhases)-3:]
			var s2Phases []int
			for _, d := range last3 {
				if d > 0 {
					s2Phases = append(s2Phases, d)
				}
			}
			if len(s2Phases) > 0 {
				pd.season2Phases = s2Phases
				for i, d := range s2Phases {
					pd.season2Grow += d
					if d > pd.season2MaxPhase {
						pd.season2MaxPhase = d
						pd.season2MaxIdx = i
					}
				}
				pd.season2AllEqual = true
				for _, d := range s2Phases {
					if d != s2Phases[0] {
						pd.season2AllEqual = false
						break
					}
				}
			}
		}
	}
	return pd
}

func formatTime(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%d秒", seconds)
	}
	if seconds < 3600 {
		mins := seconds / 60
		secs := seconds % 60
		if secs > 0 {
			return fmt.Sprintf("%d分%d秒", mins, secs)
		}
		return fmt.Sprintf("%d分", mins)
	}
	hours := seconds / 3600
	remainder := seconds % 3600
	mins := remainder / 60
	secs := remainder % 60
	if secs > 0 {
		return fmt.Sprintf("%d时%d分%d秒", hours, mins, secs)
	}
	if mins > 0 {
		return fmt.Sprintf("%d时%d分", hours, mins)
	}
	return fmt.Sprintf("%d时0分", hours)
}

type cropRow struct {
	rank             int
	cropID           int
	seedID           int
	name             string
	seasons          int
	growTime         string // display string
	growTimeFert     string // display string with fert
	harvestExp       int    // total exp per full cycle (all seasons)
	fruitCount       int
	fruitPrice       int
	expPerMinNoFert  float64
	expPerMinFert    float64
	goldPerMinNoFert float64
	goldPerMinFert   float64
}

func main() {
	configDir := filepath.Join("gameConfig")

	// Load Plant.json
	var plants []PlantConfig
	data, err := os.ReadFile(filepath.Join(configDir, "Plant.json"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading Plant.json: %v\n", err)
		os.Exit(1)
	}
	if err := json.Unmarshal(data, &plants); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing Plant.json: %v\n", err)
		os.Exit(1)
	}

	// Build plantMap
	plantMap := make(map[int]*PlantConfig)
	for i := range plants {
		plantMap[plants[i].ID] = &plants[i]
	}

	// Load seed-shop-merged-export.json
	var shopExport SeedShopExport
	data, err = os.ReadFile(filepath.Join(configDir, "seed-shop-merged-export.json"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading seed-shop-merged-export.json: %v\n", err)
		os.Exit(1)
	}
	if err := json.Unmarshal(data, &shopExport); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing seed-shop-merged-export.json: %v\n", err)
		os.Exit(1)
	}

	// Load ItemInfo.json for fruit sell prices
	fruitPriceMap := make(map[int]int)
	{
		var items []ItemInfo
		data, err := os.ReadFile(filepath.Join(configDir, "ItemInfo.json"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading ItemInfo.json: %v\n", err)
			os.Exit(1)
		}
		if err := json.Unmarshal(data, &items); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing ItemInfo.json: %v\n", err)
			os.Exit(1)
		}
		for _, item := range items {
			fruitPriceMap[item.ID] = item.Price
		}
	}

	plantSecondsNormalFert := float64(lands) / float64(normalFertPlantSpeed)

	var rows []cropRow

	for _, s := range shopExport.Rows {
		if s.SeedID <= 0 || s.GrowTimeSec <= 0 {
			continue
		}

		plant := plantMap[s.PlantID]
		if plant == nil {
			// Try via seedToPlant
			for _, p := range plants {
				if p.SeedID == s.SeedID {
					plant = &p
					break
				}
			}
		}

		seasons := 1
		var pd *phaseData

		if plant != nil {
			if plant.Seasons >= 2 {
				seasons = plant.Seasons
			}
			durations := parseGrowPhases(plant.GrowPhases)
			if len(durations) > 0 {
				pd = buildPhaseData(durations, seasons, plant.GrowPhases)
			}
		}

		var s1FertReduce, s2FertReduce, s2GrowTime int
		if pd != nil {
			s1FertReduce = pd.maxPhaseDur
			if seasons >= 2 {
				s2GrowTime = pd.season2Grow
				s2FertReduce = pd.season2MaxPhase
			}
		}

		// Season 1 with fert
		s1GrowFert := s.GrowTimeSec - s1FertReduce
		if s1GrowFert < 1 {
			s1GrowFert = 1
		}

		// Total effective grow time (all seasons with fert)
		totalGrowFert := s1GrowFert
		totalExp := s.Exp
		if seasons >= 2 && s2GrowTime > 0 {
			s2GrowFert := s2GrowTime - s2FertReduce
			if s2GrowFert < 1 {
				s2GrowFert = 1
			}
			totalGrowFert += s2GrowFert
			totalExp += s.Exp
		}

		// Total without fert
		totalGrowNoFert := s.GrowTimeSec
		totalExpNoFert := s.Exp
		if seasons >= 2 && s2GrowTime > 0 {
			totalGrowNoFert += s2GrowTime
			totalExpNoFert += s.Exp
		}

		// Fruit value per cycle
		fruitCount := s.FruitCount
		fruitPrice := fruitPriceMap[s.FruitID]
		totalFruitValue := float64(fruitCount) * float64(fruitPrice) * float64(seasons)

		// Rates: per land per minute, then multiply by lands for farm-wide
		cycleSecNoFert := float64(totalGrowNoFert) + plantSecondsNormalFert
		cycleSecFert := float64(totalGrowFert) + plantSecondsNormalFert

		expPerMinNoFert := float64(totalExpNoFert) / (cycleSecNoFert / 60.0)
		expPerMinFert := float64(totalExp) / (cycleSecFert / 60.0)

		goldPerMinNoFert := totalFruitValue / (cycleSecNoFert / 60.0)
		goldPerMinFert := totalFruitValue / (cycleSecFert / 60.0)

		row := cropRow{
			cropID:           s.PlantID,
			seedID:           s.SeedID,
			name:             s.Name,
			seasons:          seasons,
			growTime:         formatTime(totalGrowNoFert),
			growTimeFert:     formatTime(totalGrowFert),
			harvestExp:       totalExp,
			fruitCount:       fruitCount,
			fruitPrice:       fruitPrice,
			expPerMinNoFert:  math.Round(expPerMinNoFert*100) / 100,
			expPerMinFert:    math.Round(expPerMinFert*100) / 100,
			goldPerMinNoFert: math.Round(goldPerMinNoFert*100) / 100,
			goldPerMinFert:   math.Round(goldPerMinFert*100) / 100,
		}

		rows = append(rows, row)
	}

	// Sort by expPerMinFert descending
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].expPerMinFert > rows[j].expPerMinFert
	})

	// Assign ranks
	for i := range rows {
		rows[i].rank = i + 1
	}

	// Output TypeScript
	fmt.Println("export interface CropYield {")
	fmt.Println("  rank: number")
	fmt.Println("  cropId: number")
	fmt.Println("  seedId: number")
	fmt.Println("  name: string")
	fmt.Println("  seasons: number")
	fmt.Println("  growTime: string")
	fmt.Println("  growTimeFert: string")
	fmt.Println("  harvestExp: number")
	fmt.Println("  fruitCount: number")
	fmt.Println("  fruitPrice: number")
	fmt.Println("  expPerMinNoFert: number")
	fmt.Println("  expPerMinFert: number")
	fmt.Println("  goldPerMinNoFert: number")
	fmt.Println("  goldPerMinFert: number")
	fmt.Println("}")
	fmt.Println("")
	fmt.Println("// Auto-generated from gameConfig data (18 lands, normal fertilizer, optimal phase)")
	fmt.Println("// Multi-season crops show combined exp/time across all seasons.")
	fmt.Println("export const cropYieldData: CropYield[] = [")

	for _, r := range rows {
		fmt.Printf("  { rank: %d, cropId: %d, seedId: %d, name: '%s', seasons: %d, growTime: '%s', growTimeFert: '%s', harvestExp: %d, fruitCount: %d, fruitPrice: %d, expPerMinNoFert: %.2f, expPerMinFert: %.2f, goldPerMinNoFert: %.2f, goldPerMinFert: %.2f },\n",
			r.rank, r.cropID, r.seedID, r.name, r.seasons, r.growTime, r.growTimeFert, r.harvestExp, r.fruitCount, r.fruitPrice, r.expPerMinNoFert, r.expPerMinFert, r.goldPerMinNoFert, r.goldPerMinFert)
	}

	fmt.Println("]")
}
