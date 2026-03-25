package handlers

import (
	"kabao/config"
	"kabao/scheduler"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func parseSchedulerStaleMinutes(raw string) int {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0
	}
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

func getSchedulerHealthResponse(c *gin.Context) {
	report, err := scheduler.GetSchedulerHealthReport(config.DB, time.Now(), parseSchedulerStaleMinutes(c.Query("stale_minutes")))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取调度器健康状态失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": report})
}

func GetMerchantSchedulerHealth(c *gin.Context) {
	getSchedulerHealthResponse(c)
}

func GetAdminSchedulerHealth(c *gin.Context) {
	getSchedulerHealthResponse(c)
}
