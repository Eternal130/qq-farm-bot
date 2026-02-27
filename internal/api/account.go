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
			// Admin sees all accounts
			accounts, err = s.ListAccounts()
		} else {
			// Regular user sees only their own accounts
			accounts, err = s.ListAccountsByUserID(userID)
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Flatten status into response — frontend expects a flat "status" string field
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
			// Mask code for security
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
			EnableSteal    bool   `json:"enable_steal"`
			ForceLowest    bool   `json:"force_lowest"`
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
			EnableSteal:    req.EnableSteal,
			ForceLowest:    req.ForceLowest,
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

		// Check ownership (admin can edit any)
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

		// Check ownership (admin can delete any)
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

		// Stop bot if running
		mgr.StopBot(id)
		if err := s.DeleteAccount(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "deleted"})
	})
}
