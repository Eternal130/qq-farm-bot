package bot

import (
	"encoding/json"
	"fmt"
	"sort"
)

// StrategyRuleType defines the attribute a rule operates on.
type StrategyRuleType string

const (
	RuleGrowthTime     StrategyRuleType = "growth_time"     // total grow time (seconds)
	RuleExpEfficiency  StrategyRuleType = "exp_efficiency"  // farm exp per hour (with fertilizer)
	RuleGoldEfficiency StrategyRuleType = "gold_efficiency" // exp per gold spent on seed
	RuleExpPerHarvest  StrategyRuleType = "exp_per_harvest" // base exp per single harvest
	RulePrice          StrategyRuleType = "price"           // seed price (gold)
	RuleSeasons        StrategyRuleType = "seasons"         // number of seasons (1 or 2)
	RuleLevel          StrategyRuleType = "level"           // required player level
)

// StrategyOperator defines filter comparison operations.
type StrategyOperator string

const (
	OpEq  StrategyOperator = "eq"  // equal
	OpLte StrategyOperator = "lte" // less than or equal
	OpGte StrategyOperator = "gte" // greater than or equal
	OpLt  StrategyOperator = "lt"  // less than
	OpGt  StrategyOperator = "gt"  // greater than
)

// SortOrder defines sort direction.
type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

// StrategyRule is a single composable rule in the planting strategy pipeline.
// A rule can be a filter (has Operator+Value), a sorter (has Order), or both.
type StrategyRule struct {
	Type     StrategyRuleType `json:"type"`
	Operator StrategyOperator `json:"operator,omitempty"` // filter mode
	Value    float64          `json:"value,omitempty"`    // filter comparison value
	Order    SortOrder        `json:"order,omitempty"`    // sort mode
}

// PlantingStrategyConfig is the top-level JSON structure stored in account.planting_strategy.
type PlantingStrategyConfig struct {
	Rules []StrategyRule `json:"rules"`
}

// SeedCandidate holds all attributes of a seed available for strategy evaluation.
// Built from SeedYieldRow + shop availability data.
type SeedCandidate struct {
	SeedID             int
	GoodsID            int64 // shop goods ID (for purchasing)
	Name               string
	RequiredLevel      int
	Price              int
	ExpPerHarvest      int // base exp per single season harvest
	Seasons            int
	GrowTimeSec        int     // season 1 total grow time
	ExpEfficiency      float64 // farm exp per hour (with fertilizer, across all seasons)
	GoldEfficiency     float64 // exp per gold spent on seed
	GrowTimeNormalFert int     // effective grow time with fertilizer
}

// ParsePlantingStrategy parses the JSON strategy config string.
// Returns nil if the string is empty or invalid (caller should fallback to default logic).
func ParsePlantingStrategy(raw string) *PlantingStrategyConfig {
	if raw == "" {
		return nil
	}
	var cfg PlantingStrategyConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil
	}
	if len(cfg.Rules) == 0 {
		return nil
	}
	return &cfg
}

// ApplyStrategy applies the composable rules pipeline to filter and sort seed candidates.
// Returns the ordered list of candidates after all rules are applied.
// If no candidates remain after filtering, returns nil.
func ApplyStrategy(strategy *PlantingStrategyConfig, candidates []SeedCandidate) []SeedCandidate {
	if strategy == nil || len(strategy.Rules) == 0 || len(candidates) == 0 {
		return candidates
	}

	// Work on a copy to avoid mutating the original
	result := make([]SeedCandidate, len(candidates))
	copy(result, candidates)

	// Collect sort rules in order for multi-level sorting
	var sortRules []StrategyRule

	for _, rule := range strategy.Rules {
		// Apply filter if operator is set
		if rule.Operator != "" {
			result = filterCandidates(result, rule)
			if len(result) == 0 {
				return nil
			}
		}

		// Collect sort rule if order is set
		if rule.Order != "" {
			sortRules = append(sortRules, rule)
		}
	}

	// Apply multi-level sort: first rule = primary sort, subsequent = tiebreakers
	if len(sortRules) > 0 {
		sortCandidates(result, sortRules)
	}

	return result
}

// getFieldValue extracts the numeric value of a candidate for a given rule type.
func getFieldValue(c *SeedCandidate, ruleType StrategyRuleType) float64 {
	switch ruleType {
	case RuleGrowthTime:
		return float64(c.GrowTimeSec)
	case RuleExpEfficiency:
		return c.ExpEfficiency
	case RuleGoldEfficiency:
		return c.GoldEfficiency
	case RuleExpPerHarvest:
		return float64(c.ExpPerHarvest)
	case RulePrice:
		return float64(c.Price)
	case RuleSeasons:
		return float64(c.Seasons)
	case RuleLevel:
		return float64(c.RequiredLevel)
	default:
		return 0
	}
}

// filterCandidates applies a single filter rule, keeping only candidates that match.
func filterCandidates(candidates []SeedCandidate, rule StrategyRule) []SeedCandidate {
	var result []SeedCandidate
	for i := range candidates {
		val := getFieldValue(&candidates[i], rule.Type)
		if matchesFilter(val, rule.Operator, rule.Value) {
			result = append(result, candidates[i])
		}
	}
	return result
}

// matchesFilter checks if a value satisfies the filter condition.
func matchesFilter(val float64, op StrategyOperator, target float64) bool {
	switch op {
	case OpEq:
		// For growth_time, use a tolerance window of ±5% to account for minor variations
		diff := val - target
		if diff < 0 {
			diff = -diff
		}
		tolerance := target * 0.05
		if tolerance < 1 {
			tolerance = 1
		}
		return diff <= tolerance
	case OpLte:
		return val <= target
	case OpGte:
		return val >= target
	case OpLt:
		return val < target
	case OpGt:
		return val > target
	default:
		return true
	}
}

// sortCandidates sorts candidates by multiple rules (first rule = primary, etc.).
func sortCandidates(candidates []SeedCandidate, rules []StrategyRule) {
	sort.SliceStable(candidates, func(i, j int) bool {
		for _, rule := range rules {
			vi := getFieldValue(&candidates[i], rule.Type)
			vj := getFieldValue(&candidates[j], rule.Type)
			if vi == vj {
				continue // tie — use next sort rule
			}
			if rule.Order == SortDesc {
				return vi > vj
			}
			return vi < vj
		}
		return false // all equal
	})
}

// FormatStrategyDescription returns a human-readable Chinese description of a strategy.
func FormatStrategyDescription(strategy *PlantingStrategyConfig) string {
	if strategy == nil || len(strategy.Rules) == 0 {
		return "默认策略"
	}

	var parts []string
	for _, rule := range strategy.Rules {
		parts = append(parts, formatRuleDescription(rule))
	}

	desc := ""
	for i, p := range parts {
		if i > 0 {
			desc += " → "
		}
		desc += p
	}
	return desc
}

// formatRuleDescription formats a single rule as a Chinese description.
func formatRuleDescription(rule StrategyRule) string {
	typeName := ruleTypeName(rule.Type)

	if rule.Operator != "" && rule.Order != "" {
		// Combined filter+sort
		return fmt.Sprintf("筛选%s%s%s并%s排序",
			typeName, operatorName(rule.Operator), formatValue(rule.Type, rule.Value), orderName(rule.Order))
	}
	if rule.Operator != "" {
		return fmt.Sprintf("筛选%s%s%s",
			typeName, operatorName(rule.Operator), formatValue(rule.Type, rule.Value))
	}
	if rule.Order != "" {
		return fmt.Sprintf("按%s%s排序", typeName, orderName(rule.Order))
	}
	return typeName
}

func ruleTypeName(t StrategyRuleType) string {
	switch t {
	case RuleGrowthTime:
		return "生长时长"
	case RuleExpEfficiency:
		return "经验效率"
	case RuleGoldEfficiency:
		return "金币性价比"
	case RuleExpPerHarvest:
		return "单次经验"
	case RulePrice:
		return "种子价格"
	case RuleSeasons:
		return "季节数"
	case RuleLevel:
		return "等级需求"
	default:
		return string(t)
	}
}

func operatorName(op StrategyOperator) string {
	switch op {
	case OpEq:
		return "等于"
	case OpLte:
		return "≤"
	case OpGte:
		return "≥"
	case OpLt:
		return "<"
	case OpGt:
		return ">"
	default:
		return string(op)
	}
}

func orderName(o SortOrder) string {
	switch o {
	case SortAsc:
		return "升序"
	case SortDesc:
		return "降序"
	default:
		return string(o)
	}
}

func formatValue(t StrategyRuleType, v float64) string {
	switch t {
	case RuleGrowthTime:
		sec := int(v)
		if sec >= 3600 {
			h := sec / 3600
			m := (sec % 3600) / 60
			if m > 0 {
				return fmt.Sprintf("%d小时%d分", h, m)
			}
			return fmt.Sprintf("%d小时", h)
		}
		if sec >= 60 {
			return fmt.Sprintf("%d分钟", sec/60)
		}
		return fmt.Sprintf("%d秒", sec)
	case RuleSeasons:
		return fmt.Sprintf("%d季", int(v))
	default:
		if v == float64(int(v)) {
			return fmt.Sprintf("%d", int(v))
		}
		return fmt.Sprintf("%.1f", v)
	}
}
