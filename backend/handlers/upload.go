package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func UploadPaymentQRCode(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少文件"})
		return
	}

	if file.Size <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件为空"})
		return
	}

	const maxSize = 5 * 1024 * 1024
	if file.Size > maxSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件过大，最大支持5MB"})
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持jpg/jpeg/png/webp"})
		return
	}

	relDir := filepath.Join("payment_qrcodes", "merchant_"+strconv.FormatUint(uint64(merchantID), 10))
	absDir := filepath.Join("uploads", relDir)
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建目录失败"})
		return
	}

	filename := time.Now().Format("20060102150405") + "_" + uuid.New().String() + ext
	absPath := filepath.Join(absDir, filename)
	if err := c.SaveUploadedFile(file, absPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}

	urlPath := "/api/uploads/" + filepath.ToSlash(filepath.Join(relDir, filename))
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"url": urlPath}})
}
