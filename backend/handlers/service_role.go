package handlers

import (
	"fmt"
	"kabao/config"
	"kabao/models"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetPlatformServiceRoles(c *gin.Context) {
	var list []models.ServiceRole
	// 平台公开接口：仅返回平台内置角色（merchant_id IS NULL）
	config.DB.Where("is_active = ? AND merchant_id IS NULL", true).Order("sort asc, id asc").Find(&list)
	c.JSON(http.StatusOK, gin.H{"data": list})
}

func GetMerchantProfessionalRoles(c *gin.Context) {
	authType, _ := c.Get("auth_type")
	if authType == "staff" {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅商户可操作"})
		return
	}
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	var list []models.ServiceRole
	config.DB.Where("(merchant_id IS NULL AND role_type = ? AND is_active = ?) OR (merchant_id = ? AND role_type = ? AND is_active = ?)", "professional", true, merchantID, "professional", true).Order("sort asc, id asc").Find(&list)
	c.JSON(http.StatusOK, gin.H{"data": list})
}

func CreateMerchantProfessionalRole(c *gin.Context) {
	authType, _ := c.Get("auth_type")
	if authType == "staff" {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅商户可操作"})
		return
	}
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	var input struct {
		Name          string `json:"name" binding:"required"`
		AccountPrefix string `json:"account_prefix" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	name := strings.TrimSpace(input.Name)
	prefix := strings.ToLower(strings.TrimSpace(input.AccountPrefix))
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "称谓不能为空"})
		return
	}
	if prefix == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "账号前缀不能为空"})
		return
	}
	if len(prefix) > 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "账号前缀最多5个英文字母"})
		return
	}
	for _, ch := range prefix {
		if ch < 'a' || ch > 'z' {
			c.JSON(http.StatusBadRequest, gin.H{"error": "账号前缀只能包含英文字母"})
			return
		}
	}

	// 同一商户内前缀不可重复（否则账号生成会冲突）
	var existing models.ServiceRole
	if err := config.DB.Where("merchant_id = ? AND role_type = ? AND account_prefix = ?", merchantID, "professional", prefix).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该前缀已存在"})
		return
	}

	// key 全局唯一：生成稳定且可读的 key
	key := fmt.Sprintf("m%d_%s_%d", merchantID, prefix, time.Now().Unix())
	role := models.ServiceRole{
		MerchantID:            func() *uint { v := merchantID; return &v }(),
		RoleType:              "professional",
		Key:                   key,
		Name:                  name,
		AccountPrefix:         prefix,
		Description:           "商户自定义专业客服岗位",
		IsActive:              true,
		AllowPermissionAdjust: true,
		Sort:                  100,
	}

	// 默认权限：核销、结单、售卡（核销即结单默认不勾）
	defaultPermKeys := []string{"merchant.card.verify", "merchant.card.finish", "merchant.card.sell"}
	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&role).Error; err != nil {
			return err
		}
		for _, pk := range defaultPermKeys {
			k := strings.TrimSpace(pk)
			if k == "" {
				continue
			}
			var perm models.Permission
			if err := tx.Where("`key` = ?", k).First(&perm).Error; err != nil {
				continue
			}
			var rp models.RolePermission
			err := tx.Where("service_role_id = ? AND permission_id = ?", role.ID, perm.ID).First(&rp).Error
			if err == nil {
				if !rp.Allowed {
					tx.Model(&models.RolePermission{}).Where("id = ?", rp.ID).Updates(map[string]interface{}{"allowed": true})
				}
				continue
			}
			if err != gorm.ErrRecordNotFound {
				continue
			}
			tx.Create(&models.RolePermission{ServiceRoleID: role.ID, PermissionID: perm.ID, Allowed: true})
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": role})
}

func AdminListServiceRoles(c *gin.Context) {
	var list []models.ServiceRole
	// 只返回平台创建的客服角色（merchant_id IS NULL）
	config.DB.Where("merchant_id IS NULL").Order("sort asc, id asc").Find(&list)
	c.JSON(http.StatusOK, gin.H{"data": list})
}

func AdminCreateServiceRole(c *gin.Context) {
	var input struct {
		Key                   string `json:"key"`
		Name                  string `json:"name"`
		AccountPrefix         string `json:"account_prefix"`
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
		Key:           key,
		Name:          name,
		AccountPrefix: strings.ToLower(strings.TrimSpace(input.AccountPrefix)),
		Description:   strings.TrimSpace(input.Description),
		IsActive:      true,
		Sort:          0,
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
		AccountPrefix         *string `json:"account_prefix"`
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
	if input.AccountPrefix != nil {
		updates["account_prefix"] = strings.ToLower(strings.TrimSpace(*input.AccountPrefix))
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
