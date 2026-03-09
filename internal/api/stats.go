package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"qq-farm-bot/internal/bot"
	"qq-farm-bot/internal/model"
	"qq-farm-bot/internal/store"
)

func RegisterStatsRoutes(r *gin.RouterGroup, s *store.Store, mgr *bot.Manager) {
	// GET /api/accounts/:id/stats?granularity=hour|day|week|all&from=...&to=...
	r.GET("/accounts/:id/stats", func(c *gin.Context) {
		idStr := c.Param("id")
		accountID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
			return
		}

		granularity := c.DefaultQuery("granularity", "day")
		fromStr := c.Query("from")
		toStr := c.Query("to")

		var from, to time.Time
		if fromStr != "" {
			from, _ = time.Parse(time.RFC3339, fromStr)
			if from.IsZero() {
				from, _ = time.Parse("2006-01-02", fromStr)
			}
		}
		if toStr != "" {
			to, _ = time.Parse(time.RFC3339, toStr)
			if to.IsZero() {
				to, _ = time.Parse("2006-01-02", toStr)
				if !to.IsZero() {
					// End of day
					to = to.Add(24*time.Hour - time.Second)
				}
			}
		}

		// Get aggregated timeline data
		timeline, err := s.GetOpStats(accountID, granularity, from, to)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if timeline == nil {
			timeline = []model.AggregatedStats{}
		}

		// Get overall summary
		opCounts, totalGoldIn, totalGoldOut, totalExp, err := s.GetOpStatsSummary(accountID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if opCounts == nil {
			opCounts = make(map[string]int64)
		}

		// Get bot uptime
		var uptimeSeconds int64
		var startedAt *time.Time
		bs := mgr.GetStatus(accountID)
		if bs != nil && bs.Running && bs.StartedAt != nil {
			startedAt = bs.StartedAt
			uptimeSeconds = int64(time.Since(*bs.StartedAt).Seconds())
		}

		// Calculate averages per hour based on total time covered
		var avgGoldInPerHour, avgGoldOutPerHour, avgExpPerHour float64
		if len(timeline) > 0 && granularity != "all" {
			// Use number of periods as approximation for hours/days
			periods := float64(len(timeline))
			switch granularity {
			case "hour":
				avgGoldInPerHour = float64(totalGoldIn) / periods
				avgGoldOutPerHour = float64(totalGoldOut) / periods
				avgExpPerHour = float64(totalExp) / periods
			case "day":
				avgGoldInPerHour = float64(totalGoldIn) / (periods * 24)
				avgGoldOutPerHour = float64(totalGoldOut) / (periods * 24)
				avgExpPerHour = float64(totalExp) / (periods * 24)
			case "week":
				avgGoldInPerHour = float64(totalGoldIn) / (periods * 168)
				avgGoldOutPerHour = float64(totalGoldOut) / (periods * 168)
				avgExpPerHour = float64(totalExp) / (periods * 168)
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"summary": gin.H{
				"op_counts":             opCounts,
				"total_gold_in":         totalGoldIn,
				"total_gold_out":        totalGoldOut,
				"total_exp":             totalExp,
				"avg_gold_in_per_hour":  avgGoldInPerHour,
				"avg_gold_out_per_hour": avgGoldOutPerHour,
				"avg_exp_per_hour":      avgExpPerHour,
			},
			"timeline":       timeline,
			"uptime_seconds": uptimeSeconds,
			"started_at":     startedAt,
		})
	})
}
