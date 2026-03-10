package handlers

import (
	"errors"
	"kabao/config"
	"kabao/models"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func ListMerchantProjects(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	var list []models.MerchantProject
	config.DB.Where("merchant_id = ?", merchantID).Order("sort_order asc, id desc").Find(&list)
	c.JSON(http.StatusOK, gin.H{"data": list})
}

func CreateMerchantProject(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	var input struct {
		Name              string  `json:"name" binding:"required"`
		Duration          int     `json:"duration" binding:"required,min=1"`
		StartDelaySeconds *int    `json:"start_delay_seconds"`
		Price             float64 `json:"price"`
		Description       string  `json:"description"`
		IsActive          *bool   `json:"is_active"`
		SortOrder         *int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "项目名称不能为空"})
		return
	}
	if input.Duration < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "服务时长必须大于等于1"})
		return
	}
	if input.Price < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "项目价格不能为负数"})
		return
	}
	startDelaySeconds := 60
	if input.StartDelaySeconds != nil {
		if *input.StartDelaySeconds < 0 || *input.StartDelaySeconds > 3600 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "服务开始延迟时间范围应为 0-3600 秒"})
			return
		}
		startDelaySeconds = *input.StartDelaySeconds
	}

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}
	sortOrder := 0
	if input.SortOrder != nil {
		sortOrder = *input.SortOrder
	}

	p := models.MerchantProject{
		MerchantID:        merchantID,
		Name:              name,
		Duration:          input.Duration,
		StartDelaySeconds: startDelaySeconds,
		Price:             input.Price,
		Description:       strings.TrimSpace(input.Description),
		IsActive:          isActive,
		SortOrder:         sortOrder,
	}
	if err := config.DB.Create(&p).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": p})
}

func UpdateMerchantProject(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	id := c.Param("id")
	var p models.MerchantProject
	if err := config.DB.Where("id = ? AND merchant_id = ?", id, merchantID).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "项目不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	var input struct {
		Name              *string  `json:"name"`
		Duration          *int     `json:"duration"`
		StartDelaySeconds *int     `json:"start_delay_seconds"`
		Price             *float64 `json:"price"`
		Description       *string  `json:"description"`
		IsActive          *bool    `json:"is_active"`
		SortOrder         *int     `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if input.Name != nil {
		n := strings.TrimSpace(*input.Name)
		if n == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "项目名称不能为空"})
			return
		}
		updates["name"] = n
	}
	if input.Duration != nil {
		if *input.Duration < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "服务时长必须大于等于1"})
			return
		}
		updates["duration"] = *input.Duration
	}
	if input.StartDelaySeconds != nil {
		if *input.StartDelaySeconds < 0 || *input.StartDelaySeconds > 3600 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "服务开始延迟时间范围应为 0-3600 秒"})
			return
		}
		updates["start_delay_seconds"] = *input.StartDelaySeconds
	}
	if input.Price != nil {
		if *input.Price < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "项目价格不能为负数"})
			return
		}
		updates["price"] = *input.Price
	}
	if input.Description != nil {
		updates["description"] = strings.TrimSpace(*input.Description)
	}
	if input.IsActive != nil {
		updates["is_active"] = *input.IsActive
	}
	if input.SortOrder != nil {
		updates["sort_order"] = *input.SortOrder
	}

	if len(updates) == 0 {
		c.JSON(http.StatusOK, gin.H{"data": p})
		return
	}

	if err := config.DB.Model(&p).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}
	config.DB.Where("id = ? AND merchant_id = ?", p.ID, merchantID).First(&p)
	c.JSON(http.StatusOK, gin.H{"data": p})
}

func DeleteMerchantProject(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	id := c.Param("id")
	res := config.DB.Where("id = ? AND merchant_id = ?", id, merchantID).Delete(&models.MerchantProject{})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "项目不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
