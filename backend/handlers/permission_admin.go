package handlers

import (
	"kabao/config"
	"kabao/models"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func AdminListPermissions(c *gin.Context) {
	var list []models.Permission
	config.DB.Order("`group` asc, sort asc, id asc").Find(&list)
	c.JSON(http.StatusOK, gin.H{"data": list})
}

func AdminCreatePermission(c *gin.Context) {
	var input struct {
		Key         string `json:"key"`
		Name        string `json:"name"`
		Group       string `json:"group"`
		Description string `json:"description"`
		Sort        *int   `json:"sort"`
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

	var existing models.Permission
	if err := config.DB.Where("`key` = ?", key).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key已存在"})
		return
	}

	p := models.Permission{
		Key:         key,
		Name:        name,
		Group:       strings.TrimSpace(input.Group),
		Description: strings.TrimSpace(input.Description),
		Sort:        0,
	}
	if input.Sort != nil {
		p.Sort = *input.Sort
	}

	if err := config.DB.Create(&p).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "创建失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": p})
}

func AdminUpdatePermission(c *gin.Context) {
	idStr := strings.TrimSpace(c.Param("id"))
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || id64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效ID"})
		return
	}

	var input struct {
		Name        *string `json:"name"`
		Group       *string `json:"group"`
		Description *string `json:"description"`
		Sort        *int    `json:"sort"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
	if input.Group != nil {
		updates["group"] = strings.TrimSpace(*input.Group)
	}
	if input.Description != nil {
		updates["description"] = strings.TrimSpace(*input.Description)
	}
	if input.Sort != nil {
		updates["sort"] = *input.Sort
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有可更新字段"})
		return
	}

	if err := config.DB.Model(&models.Permission{}).Where("id = ?", uint(id64)).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	var updated models.Permission
	config.DB.First(&updated, uint(id64))
	c.JSON(http.StatusOK, gin.H{"data": updated})
}

func AdminDeletePermission(c *gin.Context) {
	idStr := strings.TrimSpace(c.Param("id"))
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || id64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效ID"})
		return
	}

	if err := config.DB.Delete(&models.Permission{}, uint(id64)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
