package handlers

import (
	"kabao/config"
	"kabao/models"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func GetPlatformServiceRoles(c *gin.Context) {
	var list []models.ServiceRole
	config.DB.Where("is_active = ?", true).Order("sort asc, id asc").Find(&list)
	c.JSON(http.StatusOK, gin.H{"data": list})
}

func AdminListServiceRoles(c *gin.Context) {
	var list []models.ServiceRole
	config.DB.Order("sort asc, id asc").Find(&list)
	c.JSON(http.StatusOK, gin.H{"data": list})
}

func AdminCreateServiceRole(c *gin.Context) {
	var input struct {
		Key                   string `json:"key"`
		Name                  string `json:"name"`
		Description           string `json:"description"`
		IsActive              *bool  `json:"is_active"`
		AllowPermissionAdjust *bool  `json:"allow_permission_adjust"`
		Sort                  *int   `json:"sort"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	key := strings.TrimSpace(input.Key)
	name := strings.TrimSpace(input.Name)
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key不能为空"})
		return
	}
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name不能为空"})
		return
	}

	var existing models.ServiceRole
	if err := config.DB.Where("`key` = ?", key).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key已存在"})
		return
	}

	role := models.ServiceRole{
		Key:         key,
		Name:        name,
		Description: strings.TrimSpace(input.Description),
		IsActive:    true,
		Sort:        0,
	}
	if input.IsActive != nil {
		role.IsActive = *input.IsActive
	}
	if input.AllowPermissionAdjust != nil {
		role.AllowPermissionAdjust = *input.AllowPermissionAdjust
	}
	if input.Sort != nil {
		role.Sort = *input.Sort
	}

	if err := config.DB.Create(&role).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "创建失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": role})
}

func AdminUpdateServiceRole(c *gin.Context) {
	idStr := strings.TrimSpace(c.Param("id"))
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || id64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效ID"})
		return
	}

	var input struct {
		Name                  *string `json:"name"`
		Description           *string `json:"description"`
		IsActive              *bool   `json:"is_active"`
		AllowPermissionAdjust *bool   `json:"allow_permission_adjust"`
		Sort                  *int    `json:"sort"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var role models.ServiceRole
	if err := config.DB.First(&role, uint(id64)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "不存在"})
		return
	}

	updates := map[string]interface{}{}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name不能为空"})
			return
		}
		updates["name"] = name
	}
	if input.Description != nil {
		updates["description"] = strings.TrimSpace(*input.Description)
	}
	if input.IsActive != nil {
		updates["is_active"] = *input.IsActive
	}
	if input.AllowPermissionAdjust != nil {
		updates["allow_permission_adjust"] = *input.AllowPermissionAdjust
	}
	if input.Sort != nil {
		updates["sort"] = *input.Sort
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有可更新字段"})
		return
	}

	if err := config.DB.Model(&models.ServiceRole{}).Where("id = ?", role.ID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	var updated models.ServiceRole
	config.DB.First(&updated, role.ID)
	c.JSON(http.StatusOK, gin.H{"data": updated})
}

func AdminDeleteServiceRole(c *gin.Context) {
	idStr := strings.TrimSpace(c.Param("id"))
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || id64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效ID"})
		return
	}

	if err := config.DB.Delete(&models.ServiceRole{}, uint(id64)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
