package handlers

import (
	"fmt"
	"kabao/config"
	"kabao/models"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func duplicateRolePrefixError(prefix, roleName string) string {
	return fmt.Sprintf("%q前缀已被岗位%q使用", prefix, roleName)
}

func GetPlatformServiceRoles(c *gin.Context) {
	var list []models.ServiceRole
	config.DB.Where("is_active = ? AND merchant_id IS NULL AND `key` IN ?", true, config.FixedServiceRoleKeys()).Order("sort asc, id asc").Find(&list)
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
	config.DB.
		Where("merchant_id = ? AND role_type = ? AND is_active = ?", merchantID, "professional", true).
		Order("sort asc, id asc").
		Find(&list)
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
	if config.IsReservedServiceRoleName(name) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该岗位名称为系统保留名称"})
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

	var existing models.ServiceRole
	if err := config.DB.Where("merchant_id = ? AND account_prefix = ?", merchantID, prefix).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": duplicateRolePrefixError(prefix, existing.Name)})
		return
	}

	var existingByName models.ServiceRole
	if err := config.DB.Where("merchant_id = ? AND name = ?", merchantID, name).First(&existingByName).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该岗位称谓已经存在,请不要重复添加"})
		return
	}

	key := config.BuildMerchantServiceRoleKey(merchantID, prefix)
	if config.IsFixedServiceRoleKey(key) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该岗位标识为系统保留标识"})
		return
	}
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

	defaultPermKeys := config.ProfessionalBasePermissionKeys(config.DB)
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

	var list []models.ServiceRole
	config.DB.
		Where("is_active = ? AND ((merchant_id IS NULL AND `key` IN ?) OR (merchant_id = ? AND role_type = ?))", true, config.FixedServiceRoleKeys(), merchantID, "operational").
		Order("sort asc, id asc").
		Find(&list)
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
	if config.IsReservedServiceRoleName(name) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该岗位名称为系统保留名称"})
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

	var existing models.ServiceRole
	if err := config.DB.Where("merchant_id = ? AND account_prefix = ?", merchantID, prefix).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": duplicateRolePrefixError(prefix, existing.Name)})
		return
	}

	var existingByName models.ServiceRole
	if err := config.DB.Where("merchant_id = ? AND name = ?", merchantID, name).First(&existingByName).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该岗位称谓已经存在,请不要重复添加"})
		return
	}

	key := config.BuildMerchantServiceRoleKey(merchantID, prefix)
	if config.IsFixedServiceRoleKey(key) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该岗位标识为系统保留标识"})
		return
	}
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
	config.DB.Where("merchant_id IS NULL AND `key` IN ?", config.FixedServiceRoleKeys()).Order("sort asc, id asc").Find(&list)
	c.JSON(http.StatusOK, gin.H{"data": list})
}

func AdminCreateServiceRole(c *gin.Context) {
	c.JSON(http.StatusForbidden, gin.H{"error": "平台后台不再支持新增客服角色"})
}

func AdminUpdateServiceRole(c *gin.Context) {
	c.JSON(http.StatusForbidden, gin.H{"error": "平台后台不再支持编辑客服角色"})
}

func AdminDeleteServiceRole(c *gin.Context) {
	c.JSON(http.StatusForbidden, gin.H{"error": "平台后台不再支持删除客服角色"})
}
