package handlers

import (
	"kabao/config"
	"kabao/models"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	systemConfigKeyProfessionalBasePermissionKeys = "professional_base_permission_keys"
)

func AdminGetProfessionalBasePermissions(c *gin.Context) {
	var perms []models.Permission
	config.DB.Order("sort asc, id asc").Find(&perms)

	// 读取系统配置
	baseKeys := map[string]bool{}
	var sc models.SystemConfig
	if err := config.DB.Where("`key` = ?", systemConfigKeyProfessionalBasePermissionKeys).First(&sc).Error; err == nil {
		parts := strings.Split(sc.Value, ",")
		for _, p := range parts {
			k := strings.TrimSpace(p)
			if k != "" {
				baseKeys[k] = true
			}
		}
	}

	type item struct {
		Permission models.Permission `json:"permission"`
		Allowed    bool              `json:"allowed"`
	}
	resp := make([]item, 0, len(perms))
	for _, p := range perms {
		allowed := false
		if baseKeys[strings.TrimSpace(p.Key)] {
			allowed = true
		}
		resp = append(resp, item{Permission: p, Allowed: allowed})
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"items": resp}})
}

func AdminSetProfessionalBasePermissions(c *gin.Context) {
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

	// 仅保存 allowed=true 的 key，使用逗号分隔
	keys := make([]string, 0, len(input.Items))
	seen := map[string]bool{}
	for _, it := range input.Items {
		k := strings.TrimSpace(it.PermissionKey)
		if k == "" || !it.Allowed {
			continue
		}
		if seen[k] {
			continue
		}
		seen[k] = true
		keys = append(keys, k)
	}
	sort.Strings(keys)
	v := strings.Join(keys, ",")

	var sc models.SystemConfig
	err := config.DB.Where("`key` = ?", systemConfigKeyProfessionalBasePermissionKeys).First(&sc).Error
	if err == nil {
		if err2 := config.DB.Model(&models.SystemConfig{}).Where("id = ?", sc.ID).Updates(map[string]interface{}{"value": v}).Error; err2 != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}
	if err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}
	if err2 := config.DB.Create(&models.SystemConfig{Key: systemConfigKeyProfessionalBasePermissionKeys, Value: v}).Error; err2 != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

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
	if !config.IsFixedServiceRoleKey(role.Key) || role.MerchantID != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "平台后台仅可配置固定角色默认权限"})
		return
	}

	var perms []models.Permission
	config.DB.Order("sort asc, id asc").Find(&perms)

	// 专业客服：基础默认权限（全局）
	baseKeys := map[string]bool{}
	if strings.TrimSpace(role.RoleType) == "professional" {
		var sc models.SystemConfig
		if err := config.DB.Where("`key` = ?", systemConfigKeyProfessionalBasePermissionKeys).First(&sc).Error; err == nil {
			parts := strings.Split(sc.Value, ",")
			for _, p := range parts {
				k := strings.TrimSpace(p)
				if k != "" {
					baseKeys[k] = true
				}
			}
		}
	}

	type item struct {
		Permission models.Permission `json:"permission"`
		Allowed    bool              `json:"allowed"`
		IsBase     bool              `json:"is_base"`
	}
	resp := make([]item, 0, len(perms))

	for _, p := range perms {
		allowed := false
		isBase := false
		var rp models.RolePermission
		err := config.DB.Where("service_role_id = ? AND permission_id = ?", role.ID, p.ID).First(&rp).Error
		if err == nil {
			allowed = rp.Allowed
		}

		// 专业客服：基础权限强制允许（叠加）
		if strings.TrimSpace(role.RoleType) == "professional" {
			pk := strings.TrimSpace(p.Key)
			if pk != "" && baseKeys[pk] {
				allowed = true
				isBase = true
			}
		}

		resp = append(resp, item{Permission: p, Allowed: allowed, IsBase: isBase})
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
	if !config.IsFixedServiceRoleKey(role.Key) || role.MerchantID != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "平台后台仅可配置固定角色默认权限"})
		return
	}

	// 专业客服：基础默认权限（全局）
	baseKeys := map[string]bool{}
	if strings.TrimSpace(role.RoleType) == "professional" {
		var sc models.SystemConfig
		if err := config.DB.Where("`key` = ?", systemConfigKeyProfessionalBasePermissionKeys).First(&sc).Error; err == nil {
			parts := strings.Split(sc.Value, ",")
			for _, p := range parts {
				k := strings.TrimSpace(p)
				if k != "" {
					baseKeys[k] = true
				}
			}
		}
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

		// 专业客服：基础权限不可减少
		allowed := it.Allowed
		if strings.TrimSpace(role.RoleType) == "professional" {
			if baseKeys[key] {
				allowed = true
			}
		}
		var perm models.Permission
		if err := config.DB.Where("`key` = ?", key).First(&perm).Error; err != nil {
			continue
		}

		var rp models.RolePermission
		err := config.DB.Where("service_role_id = ? AND permission_id = ?", role.ID, perm.ID).First(&rp).Error
		if err == nil {
			config.DB.Model(&models.RolePermission{}).Where("id = ?", rp.ID).Updates(map[string]interface{}{"allowed": allowed})
			continue
		}
		if err != gorm.ErrRecordNotFound {
			continue
		}
		config.DB.Create(&models.RolePermission{ServiceRoleID: role.ID, PermissionID: perm.ID, Allowed: allowed})
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
