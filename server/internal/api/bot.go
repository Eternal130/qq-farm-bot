package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"qq-farm-bot/internal/bot"
	"qq-farm-bot/internal/store"
)

func RegisterBotRoutes(r *gin.RouterGroup, s *store.Store, mgr *bot.Manager) {
	r.POST("/accounts/:id/start", func(c *gin.Context) {
		userID := c.GetInt64("userID")
		isAdmin := c.GetBool("isAdmin")

		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		account, err := s.GetAccount(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
			return
		}

		// Check ownership (admin can start any)
		if !isAdmin && account.UserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
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
		c.JSON(http.StatusOK, gin.H{"message": "started"})
	})

	r.POST("/accounts/:id/stop", func(c *gin.Context) {
		userID := c.GetInt64("userID")
		isAdmin := c.GetBool("isAdmin")

		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

		// Check ownership (admin can stop any)
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

		if err := mgr.StopBot(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "stopped"})
	})

	r.GET("/accounts/:id/status", func(c *gin.Context) {
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

		status := mgr.GetStatus(id)
		c.JSON(http.StatusOK, status)
	})

	// QR code login
	r.POST("/accounts/:id/qrcode", func(c *gin.Context) {
		userID := c.GetInt64("userID")
		isAdmin := c.GetBool("isAdmin")

		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

		// Check ownership (admin can access any)
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

		result, err := bot.RequestQRCode()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})

	r.GET("/accounts/:id/qrcode/poll", func(c *gin.Context) {
		userID := c.GetInt64("userID")
		isAdmin := c.GetBool("isAdmin")

		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

		account, err := s.GetAccount(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
			return
		}

		// Check ownership (admin can access any)
		if !isAdmin && account.UserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}

		loginCode := c.Query("login_code")
		if loginCode == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing login_code"})
			return
		}
		status, err := bot.PollQRStatus(loginCode)
		if err != nil {
			c.JSON(http.StatusOK, &bot.QRLoginStatus{Status: "error", Message: err.Error()})
			return
		}
		// If login succeeded, save the code to the account
		if status.Status == "ok" && status.Code != "" {
			account.Code = status.Code
			s.UpdateAccount(account)
		}
		c.JSON(http.StatusOK, status)
	})
}
