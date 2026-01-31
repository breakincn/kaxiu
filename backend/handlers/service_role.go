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

	// 同称谓去重：优先商户自定义岗位，若商户已定义同名岗位，则不返回平台同名岗位
	// 硬性排除运营岗位key：店长/前台及所有role_type=operational的岗位
	var merchantRoles []models.ServiceRole
	config.DB.Where("merchant_id = ? AND role_type = ? AND is_active = ? AND `key` NOT IN ('store_manager','front_desk')", merchantID, "professional", true).Order("sort asc, id asc").Find(&merchantRoles)

	var platformRoles []models.ServiceRole
	config.DB.Where("merchant_id IS NULL AND role_type = ? AND is_active = ? AND `key` NOT IN ('store_manager','front_desk')", "professional", true).Order("sort asc, id asc").Find(&platformRoles)

	seenName := map[string]bool{}
	list := make([]models.ServiceRole, 0, len(merchantRoles)+len(platformRoles))
	for _, r := range merchantRoles {
		n := strings.TrimSpace(r.Name)
		if n == "" {
			continue
		}
		seenName[n] = true
		list = append(list, r)
	}
	for _, r := range platformRoles {
		n := strings.TrimSpace(r.Name)
		if n == "" {
			continue
		}
		if seenName[n] {
			continue
		}
		seenName[n] = true
		list = append(list, r)
	}

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

	// 称谓不可重复：对齐“新增岗位（称谓）”下拉框口径（平台 + 商户，若同名则平台会被剔除，但依然视为已存在）
	var existingByName models.ServiceRole
	if err := config.DB.Where("role_type = ? AND is_active = ? AND name = ? AND (merchant_id IS NULL OR merchant_id = ?)", "professional", true, name, merchantID).First(&existingByName).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该岗位称谓已经存在,请不要重复添加"})
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

func GetMerchantOperationalRoles(c *gin.Context) {
	authType, _ := c.Get("auth_type")
	if authType == "staff" {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅商户可操作"})
		return
	}
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	// 运营岗位：平台默认 + 商户自定义
	// 策略：先查所有运营岗位（role_type=operational 或 key=店长/前台），再排除专业岗位key
	var allOperational []models.ServiceRole
	config.DB.
		Where("is_active = ? AND (role_type = ? OR `key` IN ('store_manager','front_desk')) AND (merchant_id IS NULL OR merchant_id = ?)", true, "operational", merchantID).
		Order("sort asc, id asc").
		Find(&allOperational)

	// 获取所有专业岗位key，用于排除
	var professionalKeys []string
	config.DB.Model(&models.ServiceRole{}).
		Where("role_type = ? AND is_active = ?", "professional", true).
		Pluck("`key`", &professionalKeys)
	professionalKeySet := make(map[string]bool)
	for _, k := range professionalKeys {
		if strings.TrimSpace(k) != "" {
			professionalKeySet[k] = true
		}
	}

	// 过滤掉专业岗位key
	var list []models.ServiceRole
	for _, r := range allOperational {
		if professionalKeySet[r.Key] {
			continue
		}
		list = append(list, r)
	}
	
	c.JSON(http.StatusOK, gin.H{"data": list})
}

func CreateMerchantOperationalRole(c *gin.Context) {
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

	// 前缀不可重复：对齐账号生成逻辑（同商户同前缀会导致不同岗位账号混在一起）
	var existing models.ServiceRole
	if err := config.DB.Where("role_type = ? AND account_prefix = ? AND is_active = ? AND (merchant_id IS NULL OR merchant_id = ?)", "operational", prefix, true, merchantID).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该前缀已存在"})
		return
	}

	// 称谓不可重复：平台 + 本商户口径
	var existingByName models.ServiceRole
	if err := config.DB.Where("role_type = ? AND is_active = ? AND name = ? AND (merchant_id IS NULL OR merchant_id = ?)", "operational", true, name, merchantID).First(&existingByName).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该岗位称谓已经存在,请不要重复添加"})
		return
	}

	key := fmt.Sprintf("m%d_%s_%d", merchantID, prefix, time.Now().Unix())
	role := models.ServiceRole{
		MerchantID:            func() *uint { v := merchantID; return &v }(),
		RoleType:              "operational",
		Key:                   key,
		Name:                  name,
		AccountPrefix:         prefix,
		Description:           "商户自定义运营客服岗位",
		IsActive:              true,
		AllowPermissionAdjust: true,
		Sort:                  100,
	}

	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&role).Error; err != nil {
			return err
		}

		// 默认权限：复制前台(front_desk)默认权限
		var frontDesk models.ServiceRole
		if err := tx.Where("`key` = ?", "front_desk").First(&frontDesk).Error; err != nil {
			return nil
		}
		var base []models.RolePermission
		tx.Where("service_role_id = ? AND allowed = ?", frontDesk.ID, true).Find(&base)
		for _, rp := range base {
			if rp.PermissionID == 0 {
				continue
			}
			var existingRP models.RolePermission
			err := tx.Where("service_role_id = ? AND permission_id = ?", role.ID, rp.PermissionID).First(&existingRP).Error
			if err == nil {
				if !existingRP.Allowed {
					tx.Model(&models.RolePermission{}).Where("id = ?", existingRP.ID).Updates(map[string]interface{}{"allowed": true})
				}
				continue
			}
			if err != gorm.ErrRecordNotFound {
				continue
			}
			tx.Create(&models.RolePermission{ServiceRoleID: role.ID, PermissionID: rp.PermissionID, Allowed: true})
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
