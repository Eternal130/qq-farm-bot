package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"qq-farm-bot/internal/bot"
	"qq-farm-bot/internal/model"
	"qq-farm-bot/internal/store"
)

// APIKeyMiddleware validates the API key from X-API-Key header or api_key query parameter.
// It supports two modes:
//   - Global API key (from config): full access to all accounts
//   - Per-account API key (from database): restricted to that account only
//
// When a per-account key matches, "externalAccountID" is set in the gin context.
func APIKeyMiddleware(globalAPIKey string, s *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-API-Key")
		if key == "" {
			key = c.Query("api_key")
		}
		if key == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or missing API key"})
			return
		}

		// 1. Check global API key — full access
		if globalAPIKey != "" && key == globalAPIKey {
			c.Next()
			return
		}

		// 2. Check per-account API key — restricted to that account
		account, err := s.GetAccountByAPIKey(key)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or missing API key"})
			return
		}

		// Store the restricted account ID in context
		c.Set("externalAccountID", account.ID)
		c.Next()
	}
}

// getRestrictedAccountID returns the account ID if the request is restricted to a single account.
// Returns (accountID, true) for per-account key, (0, false) for global key.
func getRestrictedAccountID(c *gin.Context) (int64, bool) {
	id, exists := c.Get("externalAccountID")
	if !exists {
		return 0, false
	}
	return id.(int64), true
}

// checkAccountAccess verifies the requested account ID is allowed under the current API key.
// Returns true if access is allowed, false if denied (and sends 403 response).
func checkAccountAccess(c *gin.Context, requestedID int64) bool {
	restrictedID, restricted := getRestrictedAccountID(c)
	if restricted && restrictedID != requestedID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied: API key can only access its own account"})
		return false
	}
	return true
}

func RegisterExternalRoutes(r *gin.RouterGroup, s *store.Store, mgr *bot.Manager) {

	// ==================== Account Endpoints ====================

	// GET /api/external/accounts — List accounts (filtered by API key scope)
	r.GET("/accounts", func(c *gin.Context) {
		accounts, err := s.ListAccounts()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		restrictedID, restricted := getRestrictedAccountID(c)

		type accountInfo struct {
			ID       int64  `json:"id"`
			Name     string `json:"name"`
			Platform string `json:"platform"`
			HasCode  bool   `json:"has_code"`
			Status   string `json:"status"`
		}
		var result []accountInfo
		for _, a := range accounts {
			// Per-account key: only return the matching account
			if restricted && a.ID != restrictedID {
				continue
			}
			info := accountInfo{
				ID:       a.ID,
				Name:     a.Name,
				Platform: a.Platform,
				HasCode:  a.Code != "",
			}
			bs := mgr.GetStatus(a.ID)
			if bs.Running {
				info.Status = "running"
			} else if bs.Error != "" {
				info.Status = "error"
			} else {
				info.Status = "stopped"
			}
			result = append(result, info)
		}
		if result == nil {
			result = make([]accountInfo, 0)
		}
		c.JSON(http.StatusOK, result)
	})

	// PUT /api/external/accounts/:id/code — Upload login code by account ID
	r.PUT("/accounts/:id/code", func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account ID"})
			return
		}

		if !checkAccountAccess(c, id) {
			return
		}

		var req struct {
			Code     string `json:"code" binding:"required"`
			Platform string `json:"platform"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
			return
		}

		account, err := s.GetAccount(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
			return
		}

		account.Code = req.Code
		if req.Platform != "" {
			account.Platform = req.Platform
		}
		if err := s.UpdateAccount(account); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message":    "code updated",
			"account_id": account.ID,
		})
	})

	// POST /api/external/code — Upload login code by account name (upsert: update if exists, create if not)
	r.POST("/code", func(c *gin.Context) {
		var req struct {
			Name      string `json:"name" binding:"required"`
			Code      string `json:"code" binding:"required"`
			Platform  string `json:"platform"`
			AutoStart bool   `json:"auto_start"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name and code are required"})
			return
		}
		if req.Platform == "" {
			req.Platform = "qq"
		}

		restrictedID, restricted := getRestrictedAccountID(c)

		// Try to find existing account by name
		account, err := s.GetAccountByName(req.Name)
		if err == nil {
			// Per-account key: verify this is the same account
			if restricted && account.ID != restrictedID {
				c.JSON(http.StatusForbidden, gin.H{"error": "access denied: API key can only access its own account"})
				return
			}
			// Account exists — update code
			account.Code = req.Code
			account.Platform = req.Platform
			if err := s.UpdateAccount(account); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"message":    "code updated",
				"account_id": account.ID,
				"created":    false,
			})
			return
		}

		// Per-account key: cannot create new accounts
		if restricted {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied: per-account API key cannot create new accounts"})
			return
		}

		// Account doesn't exist — create new (global key only)
		account = &model.Account{
			UserID:            1, // default to admin user
			Name:              req.Name,
			Platform:          req.Platform,
			Code:              req.Code,
			AutoStart:         req.AutoStart,
			FarmInterval:      10,
			FriendInterval:    10,
			EnableSteal:       true,
			EnableHarvest:     true,
			EnablePlant:       true,
			EnableSell:        true,
			EnableWeed:        true,
			EnableBug:         true,
			EnableWater:       true,
			EnableRemoveDead:  true,
			EnableUpgradeLand: true,
			EnableHelpFriend:  true,
			EnableClaimTask:   true,
		}
		if err := s.CreateAccount(account); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{
			"message":    "account created",
			"account_id": account.ID,
			"created":    true,
		})
	})

	// ==================== Bot Control Endpoints ====================

	// POST /api/external/bot/start-all — Start bots (filtered by API key scope)
	r.POST("/bot/start-all", func(c *gin.Context) {
		accounts, err := s.ListAccounts()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		restrictedID, restricted := getRestrictedAccountID(c)

		started, failed, skipped := 0, 0, 0
		var errors []string
		for _, a := range accounts {
			// Per-account key: only operate on the matching account
			if restricted && a.ID != restrictedID {
				continue
			}
			if a.Code == "" {
				skipped++
				continue
			}
			acct := a
			if err := mgr.StartBot(&acct); err != nil {
				failed++
				errors = append(errors, fmt.Sprintf("#%d(%s): %s", a.ID, a.Name, err.Error()))
			} else {
				started++
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("started %d bots, %d failed, %d skipped (no code)", started, failed, skipped),
			"started": started,
			"failed":  failed,
			"skipped": skipped,
			"errors":  errors,
		})
	})

	// POST /api/external/bot/stop-all — Stop bots (filtered by API key scope)
	r.POST("/bot/stop-all", func(c *gin.Context) {
		accounts, err := s.ListAccounts()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		restrictedID, restricted := getRestrictedAccountID(c)

		stopped := 0
		for _, a := range accounts {
			if restricted && a.ID != restrictedID {
				continue
			}
			if err := mgr.StopBot(a.ID); err == nil {
				stopped++
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("stopped %d bots", stopped),
			"stopped": stopped,
		})
	})

	// POST /api/external/bot/:id/start — Start a single bot
	r.POST("/bot/:id/start", func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account ID"})
			return
		}
		if !checkAccountAccess(c, id) {
			return
		}
		account, err := s.GetAccount(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
			return
		}
		if account.Code == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "account has no login code"})
			return
		}
		if err := mgr.StartBot(account); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "started", "account_id": id})
	})

	// POST /api/external/bot/:id/stop — Stop a single bot
	r.POST("/bot/:id/stop", func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account ID"})
			return
		}
		if !checkAccountAccess(c, id) {
			return
		}
		if err := mgr.StopBot(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "stopped", "account_id": id})
	})

	// POST /api/external/bot/:id/restart — Restart a single bot (stop then start)
	r.POST("/bot/:id/restart", func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account ID"})
			return
		}
		if !checkAccountAccess(c, id) {
			return
		}
		account, err := s.GetAccount(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
			return
		}
		if account.Code == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "account has no login code"})
			return
		}
		// Stop first (ignore error — bot may not be running)
		mgr.StopBot(id)
		// Brief pause to allow goroutine cleanup
		time.Sleep(500 * time.Millisecond)
		// Start
		if err := mgr.StartBot(account); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "restarted", "account_id": id})
	})

	// ==================== Status Endpoints ====================

	// GET /api/external/bot/:id/status — Get single bot status
	r.GET("/bot/:id/status", func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account ID"})
			return
		}
		if !checkAccountAccess(c, id) {
			return
		}
		if _, err := s.GetAccount(id); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
			return
		}
		status := mgr.GetStatus(id)
		c.JSON(http.StatusOK, status)
	})

	// GET /api/external/status — Get bots status overview (filtered by API key scope)
	r.GET("/status", func(c *gin.Context) {
		accounts, err := s.ListAccounts()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		restrictedID, restricted := getRestrictedAccountID(c)

		type botOverview struct {
			AccountID int64  `json:"account_id"`
			Name      string `json:"name"`
			Platform  string `json:"platform"`
			Status    string `json:"status"`
			Level     int64  `json:"level,omitempty"`
			Gold      int64  `json:"gold,omitempty"`
			Exp       int64  `json:"exp,omitempty"`
			Error     string `json:"error,omitempty"`
		}
		var bots []botOverview
		running := 0
		var totalGold int64

		for _, a := range accounts {
			if restricted && a.ID != restrictedID {
				continue
			}
			bs := mgr.GetStatus(a.ID)
			ov := botOverview{
				AccountID: a.ID,
				Name:      a.Name,
				Platform:  a.Platform,
				Level:     bs.Level,
				Gold:      bs.Gold,
				Exp:       bs.Exp,
			}
			if bs.Running {
				ov.Status = "running"
				running++
				totalGold += bs.Gold
			} else if bs.Error != "" {
				ov.Status = "error"
				ov.Error = bs.Error
			} else {
				ov.Status = "stopped"
			}
			bots = append(bots, ov)
		}
		if bots == nil {
			bots = make([]botOverview, 0)
		}
		c.JSON(http.StatusOK, gin.H{
			"total":      len(bots),
			"running":    running,
			"total_gold": totalGold,
			"bots":       bots,
		})
	})
}
