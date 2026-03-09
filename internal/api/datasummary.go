package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"qq-farm-bot/internal/bot"
	"qq-farm-bot/internal/store"
)

func RegisterDataSummaryRoutes(r *gin.RouterGroup, s *store.Store, mgr *bot.Manager) {
	// GET /api/accounts/:id/data-summary?hours=24&days=7
	r.GET("/accounts/:id/data-summary", func(c *gin.Context) {
		idStr := c.Param("id")
		accountID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
			return
		}

		hoursStr := c.DefaultQuery("hours", "24")
		hours, _ := strconv.Atoi(hoursStr)
		if hours <= 0 || hours > 720 {
			hours = 24
		}

		daysStr := c.DefaultQuery("days", "7")
		days, _ := strconv.Atoi(daysStr)
		if days <= 0 || days > 90 {
			days = 7
		}

		now := time.Now()
		hoursSince := now.Add(-time.Duration(hours) * time.Hour)
		daysSince := now.AddDate(0, 0, -days)

		// Summary totals (use the hourly time range)
		totals, err := s.GetDataSummaryTotals(accountID, hoursSince)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Hourly trend
		hourlyTrend, err := s.GetHourlyTrend(accountID, hoursSince)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if hourlyTrend == nil {
			hourlyTrend = []store.HourlyTrendRow{}
		}

		// Crop breakdown (from hourly range)
		cropBreakdown, err := s.GetCropBreakdown(accountID, hoursSince)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if cropBreakdown == nil {
			cropBreakdown = []store.CropBreakdownRow{}
		}

		// Steal ranking (from hourly range)
		stealRanking, err := s.GetStealRanking(accountID, hoursSince)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if stealRanking == nil {
			stealRanking = []store.StealRankingRow{}
		}

		// Daily summary (from days range)
		dailySummary, err := s.GetDailySummary(accountID, daysSince)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if dailySummary == nil {
			dailySummary = []store.DailySummaryRow{}
		}

		c.JSON(http.StatusOK, gin.H{
			"summary":        totals,
			"hourly_trend":   hourlyTrend,
			"crop_breakdown": cropBreakdown,
			"steal_ranking":  stealRanking,
			"daily_summary":  dailySummary,
		})
	})
}
