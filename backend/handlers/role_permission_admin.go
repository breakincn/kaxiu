package handlers

import (
	"kabao/config"
	"kabao/models"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func AdminGetRolePermissions(c *gin.Context) {
	roleIDStr := strings.TrimSpace(c.Param("roleId"))
	roleID64, err := strconv.ParseUint(roleIDStr, 10, 32)
	if err != nil || roleID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效角色ID"})
		return
	}

	var role models.ServiceRole
	if err := config.DB.First(&role, uint(roleID64)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
		return
	}

	var perms []models.Permission
	config.DB.Order("sort asc, id asc").Find(&perms)

	type item struct {
		Permission models.Permission `json:"permission"`
		Allowed    bool              `json:"allowed"`
	}
	resp := make([]item, 0, len(perms))

	for _, p := range perms {
		allowed := false
		var rp models.RolePermission
		err := config.DB.Where("service_role_id = ? AND permission_id = ?", role.ID, p.ID).First(&rp).Error
		if err == nil {
			allowed = rp.Allowed
		}
		resp = append(resp, item{Permission: p, Allowed: allowed})
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"role": role, "items": resp}})
}

func AdminSetRolePermissions(c *gin.Context) {
	roleIDStr := strings.TrimSpace(c.Param("roleId"))
	roleID64, err := strconv.ParseUint(roleIDStr, 10, 32)
	if err != nil || roleID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效角色ID"})
		return
	}

	var role models.ServiceRole
	if err := config.DB.First(&role, uint(roleID64)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
		return
	}

	var input struct {
		Items []struct {
			PermissionKey string `json:"permission_key"`
			Allowed       bool   `json:"allowed"`
		} `json:"items"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for _, it := range input.Items {
		key := strings.TrimSpace(it.PermissionKey)
		if key == "" {
			continue
		}
		var perm models.Permission
		if err := config.DB.Where("`key` = ?", key).First(&perm).Error; err != nil {
			continue
		}

		var rp models.RolePermission
		err := config.DB.Where("service_role_id = ? AND permission_id = ?", role.ID, perm.ID).First(&rp).Error
		if err == nil {
			config.DB.Model(&models.RolePermission{}).Where("id = ?", rp.ID).Updates(map[string]interface{}{"allowed": it.Allowed})
			continue
		}
		if err != gorm.ErrRecordNotFound {
			continue
		}
		config.DB.Create(&models.RolePermission{ServiceRoleID: role.ID, PermissionID: perm.ID, Allowed: it.Allowed})
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
