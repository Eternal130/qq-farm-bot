package model

import "time"

// Account represents a game account managed by the system.
type Account struct {
	ID        int64  `json:"id"`
	UserID    int64  `json:"user_id"`    // owner user id
	Name      string `json:"name"`       // display name
	Platform  string `json:"platform"`   // "qq" or "wx"
	Code      string `json:"code"`       // login code
	AutoStart bool   `json:"auto_start"` // auto start bot on server launch

	// Bot config
	FarmInterval   int  `json:"farm_interval"`   // farm check seconds
	FriendInterval int  `json:"friend_interval"` // friend check seconds
	EnableSteal    bool `json:"enable_steal"`
	ForceLowest    bool `json:"force_lowest"` // force lowest level crop

	// Farm automation toggles (all default true for backward compatibility)
	EnableHarvest     bool `json:"enable_harvest"`
	EnablePlant       bool `json:"enable_plant"`
	EnableSell        bool `json:"enable_sell"`
	EnableWeed        bool `json:"enable_weed"`
	EnableBug         bool `json:"enable_bug"`
	EnableWater       bool `json:"enable_water"`
	EnableRemoveDead  bool `json:"enable_remove_dead"`
	EnableUpgradeLand bool `json:"enable_upgrade_land"`
	EnableHelpFriend  bool `json:"enable_help_friend"`
	EnableClaimTask   bool `json:"enable_claim_task"`

	// Crop selection & filtering
	PlantCropID  int    `json:"plant_crop_id"`  // specific crop to plant (0 = auto select)
	SellCropIDs  string `json:"sell_crop_ids"`  // comma-separated crop IDs to sell (empty = all)
	StealCropIDs string `json:"steal_crop_ids"` // comma-separated crop IDs to steal (empty = all)

	// Fertilizer config
	AutoUseFertilizer       bool `json:"auto_use_fertilizer"`
	AutoBuyFertilizer       bool `json:"auto_buy_fertilizer"`
	FertilizerTargetCount   int  `json:"fertilizer_target_count"`
	FertilizerBuyDailyLimit int  `json:"fertilizer_buy_daily_limit"`

	// Anti-detection
	EnableAntiDetection bool `json:"enable_anti_detection"`
	// Planting preference
	PreferBagSeeds bool `json:"prefer_bag_seeds"` // prioritize planting seeds from bag

	// External API
	APIKey string `json:"api_key"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BotStatus represents the runtime status of a bot instance.
type BotStatus struct {
	AccountID int64      `json:"account_id"`
	Running   bool       `json:"running"`
	GID       int64      `json:"gid,omitempty"`
	Name      string     `json:"name,omitempty"`
	Level     int64      `json:"level,omitempty"`
	Exp       int64      `json:"exp,omitempty"`
	Gold      int64      `json:"gold,omitempty"`
	Platform  string     `json:"platform,omitempty"`
	StartedAt *time.Time `json:"started_at,omitempty"`
	Error     string     `json:"error,omitempty"`

	// Exp tracking for level up estimation
	ExpRatePerHour   float64 `json:"exp_rate_per_hour,omitempty"`
	NextLevelExp     int64   `json:"next_level_exp,omitempty"`
	ExpToNextLevel   int64   `json:"exp_to_next_level,omitempty"`
	HoursToNextLevel float64 `json:"hours_to_next_level,omitempty"`

	// Farm stats
	TotalHarvest  int64        `json:"total_harvest"`
	TotalSteal    int64        `json:"total_steal"`
	TotalHelp     int64        `json:"total_help"`
	FriendsCount  int          `json:"friends_count"`
	TotalLands    int          `json:"total_lands"`
	UnlockedLands int          `json:"unlocked_lands"`
	Lands         []LandStatus `json:"lands,omitempty"`
}

// LandStatus represents the status of a single farm land.
type LandStatus struct {
	ID       int64  `json:"id"`
	Level    int64  `json:"level"`
	MaxLevel int64  `json:"max_level"`
	Unlocked bool   `json:"unlocked"`
	CropName string `json:"crop_name,omitempty"`
	CropID   int64  `json:"crop_id,omitempty"`
	Phase    string `json:"phase,omitempty"`
}

// LogEntry represents a bot log message.
type LogEntry struct {
	ID        int64     `json:"id"`
	AccountID int64     `json:"account_id"`
	Tag       string    `json:"tag"`
	Message   string    `json:"message"`
	Level     string    `json:"level"` // "info", "warn", "error"
	CreatedAt time.Time `json:"created_at"`
}

// OpRecord represents a single operation statistics record.
type OpRecord struct {
	ID        int64     `json:"id"`
	AccountID int64     `json:"account_id"`
	OpType    string    `json:"op_type"`    // harvest, plant, sell, steal, weed, bug, water, fertilize, task_claim, fert_buy, fert_open, fert_use, unlock_land, upgrade_land
	Count     int64     `json:"count"`      // number of items/lands in this operation
	GoldDelta int64     `json:"gold_delta"` // gold change: positive=earned, negative=spent
	ExpDelta  int64     `json:"exp_delta"`  // exp earned
	Detail    string    `json:"detail"`     // optional: crop name (sell), friend name (steal), etc.
	CreatedAt time.Time `json:"created_at"`
}

// OpType constants for statistics tracking.
const (
	OpHarvest     = "harvest"
	OpPlant       = "plant"
	OpSell        = "sell"
	OpSteal       = "steal"
	OpWeed        = "weed"
	OpBug         = "bug"
	OpWater       = "water"
	OpFertilize   = "fertilize"
	OpTaskClaim   = "task_claim"
	OpFertBuy     = "fert_buy"
	OpFertOpen    = "fert_open"
	OpFertUse     = "fert_use"
	OpUnlockLand  = "unlock_land"
	OpUpgradeLand = "upgrade_land"
	OpHelpWeed    = "help_weed"
	OpHelpBug     = "help_bug"
	OpHelpWater   = "help_water"
	OpBuySeed     = "buy_seed"
)

// AggregatedStats represents aggregated operation statistics for a time bucket.
type AggregatedStats struct {
	Period    string         `json:"period"`    // time bucket label, e.g. "2026-03-09 10:00"
	OpCounts  map[string]int64 `json:"op_counts"` // op_type -> total count
	GoldIn    int64          `json:"gold_in"`    // total gold earned
	GoldOut   int64          `json:"gold_out"`   // total gold spent (absolute)
	ExpGained int64          `json:"exp_gained"` // total exp earned
}
