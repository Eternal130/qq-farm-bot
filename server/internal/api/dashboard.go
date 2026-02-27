package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"qq-farm-bot/internal/bot"
	"qq-farm-bot/internal/model"
	"qq-farm-bot/internal/store"
)

func RegisterDashboardRoutes(r *gin.RouterGroup, s *store.Store, mgr *bot.Manager) {
	r.GET("/dashboard", func(c *gin.Context) {
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

		totalAccounts := len(accounts)
		runningCount := 0
		var totalGold int64

		// Build flat account cards matching frontend DashboardStats.accounts
		type accountCard struct {
			ID              int64              `json:"id"`
			Name            string             `json:"name"`
			Level           int64              `json:"level"`
			Gold            int64              `json:"gold"`
			Exp             int64              `json:"exp"`
			Status          string             `json:"status"`
			Platform        string             `json:"platform"`
			TotalSteal      int64              `json:"total_steal"`
			TotalHelp       int64              `json:"total_help"`
			FriendsCount    int                `json:"friends_count"`
			TotalLands      int                `json:"total_lands"`
			UnlockedLands   int                `json:"unlocked_lands"`
			Lands           []model.LandStatus `json:"lands"`
			// Level up estimation
			ExpRatePerHour   float64 `json:"exp_rate_per_hour"`
			NextLevelExp     int64   `json:"next_level_exp"`
			ExpToNextLevel   int64   `json:"exp_to_next_level"`
			HoursToNextLevel float64 `json:"hours_to_next_level"`
		}
		var cards []accountCard
		for _, a := range accounts {
			card := accountCard{
				ID:       a.ID,
				Name:     a.Name,
				Platform: a.Platform,
				Status:   "stopped",
			}
			bs := mgr.GetStatus(a.ID)
			if bs.Running {
				runningCount++
				totalGold += bs.Gold
				card.Status = "running"
				card.Level = bs.Level
				card.Gold = bs.Gold
				card.Exp = bs.Exp
				card.TotalSteal = bs.TotalSteal
				card.TotalHelp = bs.TotalHelp
				card.FriendsCount = bs.FriendsCount
				card.TotalLands = bs.TotalLands
				card.UnlockedLands = bs.UnlockedLands
				// Level up estimation
				card.ExpRatePerHour = bs.ExpRatePerHour
				card.NextLevelExp = bs.NextLevelExp
				card.ExpToNextLevel = bs.ExpToNextLevel
				card.HoursToNextLevel = bs.HoursToNextLevel
				if bs.Lands != nil {
					card.Lands = bs.Lands
				} else {
					card.Lands = []model.LandStatus{}
				}
			} else if bs.Error != "" {
				card.Status = "error"
			}
			if card.Lands == nil {
				card.Lands = []model.LandStatus{}
			}
			cards = append(cards, card)
		}
		if cards == nil {
			cards = make([]accountCard, 0)
		}

		c.JSON(http.StatusOK, gin.H{
			"total_accounts": totalAccounts,
			"running_bots":   runningCount,
			"total_gold":     totalGold,
			"accounts":       cards,
		})
	})
}
