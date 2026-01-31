package handlers

import (
	"kabao/config"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetConfig 获取前端需要的配置信息
func GetConfig(c *gin.Context) {
	response := gin.H{
		"startPendingTimeoutSeconds": int64(config.StartPendingTimeout().Seconds()),
		"startScanTimeoutSeconds":   int64(config.StartScanTimeout().Seconds()),
	}
	c.JSON(http.StatusOK, gin.H{"data": response})
}
