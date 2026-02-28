package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"qq-farm-bot/internal/bot"
	"qq-farm-bot/internal/config"
	"qq-farm-bot/internal/model"
	"qq-farm-bot/internal/store"
)

func RegisterAccountRoutes(r *gin.RouterGroup, s *store.Store, mgr *bot.Manager, cfg *config.Config) {
	r.GET("/accounts", func(c *gin.Context) {
		userID := c.GetInt64("userID")
		isAdmin := c.GetBool("isAdmin")

		var accounts []model.Account
		var err error

		if isAdmin {
			accounts, err = s.ListAccounts()
		} else {
			accounts, err = s.ListAccountsByUserID(userID)
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		type accountResponse struct {
			model.Account
			Status string `json:"status"`
			Level  int64  `json:"level,omitempty"`
			Gold   int64  `json:"gold,omitempty"`
			Exp    int64  `json:"exp,omitempty"`
		}
		var result []accountResponse
		for _, a := range accounts {
			ar := accountResponse{Account: a}
			bs := mgr.GetStatus(a.ID)
			if bs.Running {
				ar.Status = "running"
				ar.Level = bs.Level
				ar.Gold = bs.Gold
				ar.Exp = bs.Exp
			} else if bs.Error != "" {
				ar.Status = "error"
			} else {
				ar.Status = "stopped"
			}
			if len(ar.Code) > 8 {
				ar.Code = ar.Code[:8] + "..."
			}
			result = append(result, ar)
		}
		c.JSON(http.StatusOK, result)
	})

	r.POST("/accounts", func(c *gin.Context) {
		userID := c.GetInt64("userID")

		var req struct {
			Name           string `json:"name"`
			Platform       string `json:"platform"`
			Code           string `json:"code"`
			AutoStart      bool   `json:"auto_start"`
			FarmInterval   int    `json:"farm_interval"`
			FriendInterval int    `json:"friend_interval"`
			EnableSteal    *bool  `json:"enable_steal"`
			ForceLowest    bool   `json:"force_lowest"`
			// Farm automation toggles
			EnableHarvest     *bool `json:"enable_harvest"`
			EnablePlant       *bool `json:"enable_plant"`
			EnableSell        *bool `json:"enable_sell"`
			EnableWeed        *bool `json:"enable_weed"`
			EnableBug         *bool `json:"enable_bug"`
			EnableWater       *bool `json:"enable_water"`
			EnableRemoveDead  *bool `json:"enable_remove_dead"`
			EnableUpgradeLand *bool `json:"enable_upgrade_land"`
			EnableHelpFriend  *bool `json:"enable_help_friend"`
			EnableClaimTask   *bool `json:"enable_claim_task"`
			// Crop selection
			PlantCropID  int    `json:"plant_crop_id"`
			SellCropIDs  string `json:"sell_crop_ids"`
			StealCropIDs string `json:"steal_crop_ids"`
			// Fertilizer
			AutoUseFertilizer       bool `json:"auto_use_fertilizer"`
			AutoBuyFertilizer       bool `json:"auto_buy_fertilizer"`
			FertilizerTargetCount   int  `json:"fertilizer_target_count"`
			FertilizerBuyDailyLimit int  `json:"fertilizer_buy_daily_limit"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.Platform == "" {
			req.Platform = "qq"
		}
		if req.FarmInterval == 0 {
			req.FarmInterval = 10
		}
		if req.FriendInterval == 0 {
			req.FriendInterval = 10
		}

		account := &model.Account{
			UserID:         userID,
			Name:           req.Name,
			Platform:       req.Platform,
			Code:           req.Code,
			AutoStart:      req.AutoStart,
			FarmInterval:   req.FarmInterval,
			FriendInterval: req.FriendInterval,
			EnableSteal:    ptrBoolDefault(req.EnableSteal, true),
			ForceLowest:    req.ForceLowest,
			// Default all automation toggles to true
			EnableHarvest:           ptrBoolDefault(req.EnableHarvest, true),
			EnablePlant:             ptrBoolDefault(req.EnablePlant, true),
			EnableSell:              ptrBoolDefault(req.EnableSell, true),
			EnableWeed:              ptrBoolDefault(req.EnableWeed, true),
			EnableBug:               ptrBoolDefault(req.EnableBug, true),
			EnableWater:             ptrBoolDefault(req.EnableWater, true),
			EnableRemoveDead:        ptrBoolDefault(req.EnableRemoveDead, true),
			EnableUpgradeLand:       ptrBoolDefault(req.EnableUpgradeLand, true),
			EnableHelpFriend:        ptrBoolDefault(req.EnableHelpFriend, true),
			EnableClaimTask:         ptrBoolDefault(req.EnableClaimTask, true),
			PlantCropID:             req.PlantCropID,
			SellCropIDs:             req.SellCropIDs,
			StealCropIDs:            req.StealCropIDs,
			AutoUseFertilizer:       req.AutoUseFertilizer,
			AutoBuyFertilizer:       req.AutoBuyFertilizer,
			FertilizerTargetCount:   req.FertilizerTargetCount,
			FertilizerBuyDailyLimit: req.FertilizerBuyDailyLimit,
		}
		if err := s.CreateAccount(account); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, account)
	})

	r.PUT("/accounts/:id", func(c *gin.Context) {
		userID := c.GetInt64("userID")
		isAdmin := c.GetBool("isAdmin")

		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		account, err := s.GetAccount(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
			return
		}

		if !isAdmin && account.UserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}

		var req struct {
			Name           *string `json:"name"`
			Platform       *string `json:"platform"`
			Code           *string `json:"code"`
			AutoStart      *bool   `json:"auto_start"`
			FarmInterval   *int    `json:"farm_interval"`
			FriendInterval *int    `json:"friend_interval"`
			EnableSteal    *bool   `json:"enable_steal"`
			ForceLowest    *bool   `json:"force_lowest"`
			// Farm automation toggles
			EnableHarvest     *bool `json:"enable_harvest"`
			EnablePlant       *bool `json:"enable_plant"`
			EnableSell        *bool `json:"enable_sell"`
			EnableWeed        *bool `json:"enable_weed"`
			EnableBug         *bool `json:"enable_bug"`
			EnableWater       *bool `json:"enable_water"`
			EnableRemoveDead  *bool `json:"enable_remove_dead"`
			EnableUpgradeLand *bool `json:"enable_upgrade_land"`
			EnableHelpFriend  *bool `json:"enable_help_friend"`
			EnableClaimTask   *bool `json:"enable_claim_task"`
			// Crop selection
			PlantCropID  *int    `json:"plant_crop_id"`
			SellCropIDs  *string `json:"sell_crop_ids"`
			StealCropIDs *string `json:"steal_crop_ids"`
			// Fertilizer
			AutoUseFertilizer       *bool `json:"auto_use_fertilizer"`
			AutoBuyFertilizer       *bool `json:"auto_buy_fertilizer"`
			FertilizerTargetCount   *int  `json:"fertilizer_target_count"`
			FertilizerBuyDailyLimit *int  `json:"fertilizer_buy_daily_limit"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if req.Name != nil {
			account.Name = *req.Name
		}
		if req.Platform != nil {
			account.Platform = *req.Platform
		}
		if req.Code != nil {
			account.Code = *req.Code
		}
		if req.AutoStart != nil {
			account.AutoStart = *req.AutoStart
		}
		if req.FarmInterval != nil {
			account.FarmInterval = *req.FarmInterval
		}
		if req.FriendInterval != nil {
			account.FriendInterval = *req.FriendInterval
		}
		if req.EnableSteal != nil {
			account.EnableSteal = *req.EnableSteal
		}
		if req.ForceLowest != nil {
			account.ForceLowest = *req.ForceLowest
		}
		if req.EnableHarvest != nil {
			account.EnableHarvest = *req.EnableHarvest
		}
		if req.EnablePlant != nil {
			account.EnablePlant = *req.EnablePlant
		}
		if req.EnableSell != nil {
			account.EnableSell = *req.EnableSell
		}
		if req.EnableWeed != nil {
			account.EnableWeed = *req.EnableWeed
		}
		if req.EnableBug != nil {
			account.EnableBug = *req.EnableBug
		}
		if req.EnableWater != nil {
			account.EnableWater = *req.EnableWater
		}
		if req.EnableRemoveDead != nil {
			account.EnableRemoveDead = *req.EnableRemoveDead
		}
		if req.EnableUpgradeLand != nil {
			account.EnableUpgradeLand = *req.EnableUpgradeLand
		}
		if req.EnableHelpFriend != nil {
			account.EnableHelpFriend = *req.EnableHelpFriend
		}
		if req.EnableClaimTask != nil {
			account.EnableClaimTask = *req.EnableClaimTask
		}
		if req.PlantCropID != nil {
			account.PlantCropID = *req.PlantCropID
		}
		if req.SellCropIDs != nil {
			account.SellCropIDs = *req.SellCropIDs
		}
		if req.StealCropIDs != nil {
			account.StealCropIDs = *req.StealCropIDs
		}
		if req.AutoUseFertilizer != nil {
			account.AutoUseFertilizer = *req.AutoUseFertilizer
		}
		if req.AutoBuyFertilizer != nil {
			account.AutoBuyFertilizer = *req.AutoBuyFertilizer
		}
		if req.FertilizerTargetCount != nil {
			account.FertilizerTargetCount = *req.FertilizerTargetCount
		}
		if req.FertilizerBuyDailyLimit != nil {
			account.FertilizerBuyDailyLimit = *req.FertilizerBuyDailyLimit
		}

		if err := s.UpdateAccount(account); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, account)
	})

	r.DELETE("/accounts/:id", func(c *gin.Context) {
		userID := c.GetInt64("userID")
		isAdmin := c.GetBool("isAdmin")

		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

		if !isAdmin {
			account, err := s.GetAccount(id)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
				return
			}
			if account.UserID != userID {
				c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
				return
			}
		}

		mgr.StopBot(id)
		if err := s.DeleteAccount(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "deleted"})
	})

	// Crops list endpoint for frontend dropdown
	r.GET("/crops", func(c *gin.Context) {
		gc := bot.GetGameConfig()
		if gc == nil {
			c.JSON(http.StatusOK, []interface{}{})
			return
		}
		c.JSON(http.StatusOK, gc.GetCropList())
	})
}

func ptrBoolDefault(p *bool, defaultVal bool) bool {
	if p == nil {
		return defaultVal
	}
	return *p
}
