package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"qq-farm-bot/internal/bot"
	"qq-farm-bot/internal/model"
	"qq-farm-bot/internal/store"
)

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func RegisterLogRoutes(r *gin.RouterGroup, s *store.Store, mgr *bot.Manager) {
	// Get historical logs
	r.GET("/accounts/:id/logs", func(c *gin.Context) {
		userID := c.GetInt64("userID")
		isAdmin := c.GetBool("isAdmin")

		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

		// Check ownership (admin can view any)
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

		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
		beforeID, _ := strconv.ParseInt(c.DefaultQuery("before_id", "0"), 10, 64)

		logs, err := s.GetLogs(id, limit, beforeID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if logs == nil {
			logs = make([]model.LogEntry, 0)
		}
		c.JSON(http.StatusOK, logs)
	})

	// Real-time log WebSocket
	r.GET("/ws/logs", func(c *gin.Context) {
		userID := c.GetInt64("userID")
		isAdmin := c.GetBool("isAdmin")

		idStr := c.Query("account_id")
		if idStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing account_id"})
			return
		}
		accountID, _ := strconv.ParseInt(idStr, 10, 64)

		// Check ownership (admin can view any)
		if !isAdmin {
			account, err := s.GetAccount(accountID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
				return
			}
			if account.UserID != userID {
				c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
				return
			}
		}

		conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		inst := mgr.GetInstance(accountID)
		if inst == nil {
			conn.WriteJSON(map[string]string{"error": "bot not running"})
			return
		}

		logCh := inst.Logger().Subscribe()
		defer inst.Logger().Unsubscribe(logCh)

		// Keep alive
		go func() {
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
			}
		}()

		for entry := range logCh {
			data := map[string]interface{}{
				"id":         entry.ID,
				"account_id": entry.AccountID,
				"tag":        entry.Tag,
				"message":    entry.Message,
				"level":      entry.Level,
				"created_at": entry.CreatedAt.Format(time.RFC3339),
			}
			if err := conn.WriteJSON(data); err != nil {
				return
			}
		}
	})
}
