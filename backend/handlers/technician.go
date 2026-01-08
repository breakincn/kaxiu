package handlers

import (
	"kabao/config"
	"kabao/models"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func GetCurrentTechnician(c *gin.Context) {
	authType, _ := c.Get("auth_type")
	if authType != "technician" {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅技师账号可操作"})
		return
	}

	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	technicianIDAny, ok := c.Get("technician_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	technicianID, ok := technicianIDAny.(uint)
	if !ok || technicianID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var tech models.Technician
	if err := config.DB.Where("id = ? AND merchant_id = ?", technicianID, merchantID).First(&tech).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "技师不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": tech})
}

func BindTechnicianPhone(c *gin.Context) {
	authType, _ := c.Get("auth_type")
	if authType != "technician" {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅技师账号可操作"})
		return
	}

	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	technicianIDAny, ok := c.Get("technician_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	technicianID, ok := technicianIDAny.(uint)
	if !ok || technicianID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var input struct {
		Phone string `json:"phone" binding:"required"`
		Code  string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.Phone = strings.TrimSpace(input.Phone)
	input.Code = strings.TrimSpace(input.Code)
	if input.Phone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供手机号"})
		return
	}
	if input.Code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入验证码"})
		return
	}

	var tech models.Technician
	if err := config.DB.Where("id = ? AND merchant_id = ?", technicianID, merchantID).First(&tech).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "技师不存在"})
		return
	}

	// 检查手机号是否已被其他技师绑定（同一商户下唯一即可）
	var existing models.Technician
	if err := config.DB.Where("merchant_id = ? AND phone = ? AND id != ?", merchantID, input.Phone, technicianID).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该手机号已被其他账号绑定"})
		return
	}

	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := consumeSMSCode(tx, input.Phone, "technician_bind_phone", input.Code); err != nil {
			return err
		}
		return tx.Model(&models.Technician{}).Where("id = ? AND merchant_id = ?", technicianID, merchantID).Updates(map[string]interface{}{
			"phone":      input.Phone,
			"updated_at": func() *time.Time { t := time.Now(); return &t }(),
		}).Error
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var updated models.Technician
	if err := config.DB.Where("id = ? AND merchant_id = ?", technicianID, merchantID).First(&updated).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "绑定失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": updated})
}

func GetMerchantTechnicians(c *gin.Context) {
	authType, _ := c.Get("auth_type")
	if authType == "technician" {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅商户可操作"})
		return
	}

	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	// 获取角色参数
	roleKey := c.Query("role")
	if roleKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少角色参数"})
		return
	}

	// 查询角色ID
	var role models.ServiceRole
	if err := config.DB.Where("`key` = ?", roleKey).First(&role).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "角色不存在"})
		return
	}

	var list []models.Technician
	config.DB.Where("merchant_id = ? AND service_role_id = ?", merchantID, role.ID).Order("id desc").Find(&list)
	c.JSON(http.StatusOK, gin.H{"data": list})
}

func UpdateMerchantTechnician(c *gin.Context) {
	authType, _ := c.Get("auth_type")
	if authType == "technician" {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅商户可操作"})
		return
	}

	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	idStr := strings.TrimSpace(c.Param("id"))
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || id64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的技师ID"})
		return
	}

	var input struct {
		Name     *string `json:"name"`
		IsActive *bool   `json:"is_active"`
		Code     *string `json:"code"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var tech models.Technician
	if err := config.DB.Where("id = ? AND merchant_id = ?", uint(id64), merchantID).First(&tech).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "技师不存在"})
		return
	}

	if input.Code != nil {
		// 当前版本不支持修改 code/account，避免破坏账号体系
		newCode := strings.TrimSpace(*input.Code)
		if newCode != "" && newCode != tech.Code {
			c.JSON(http.StatusBadRequest, gin.H{"error": "暂不支持修改技师编号"})
			return
		}
	}

	updates := map[string]interface{}{}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请输入技师姓名"})
			return
		}
		updates["name"] = name
	}
	if input.IsActive != nil {
		updates["is_active"] = *input.IsActive
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有可更新的字段"})
		return
	}

	if err := config.DB.Model(&models.Technician{}).Where("id = ? AND merchant_id = ?", tech.ID, merchantID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	var updated models.Technician
	config.DB.First(&updated, tech.ID)
	c.JSON(http.StatusOK, gin.H{"data": updated})
}

func DeleteMerchantTechnician(c *gin.Context) {
	authType, _ := c.Get("auth_type")
	if authType == "technician" {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅商户可操作"})
		return
	}

	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	idStr := strings.TrimSpace(c.Param("id"))
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || id64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的技师ID"})
		return
	}

	if err := config.DB.Where("id = ? AND merchant_id = ?", uint(id64), merchantID).Delete(&models.Technician{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func CreateMerchantTechnician(c *gin.Context) {
	authType, _ := c.Get("auth_type")
	if authType == "technician" {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅商户可操作"})
		return
	}

	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	var input struct {
		Name string `json:"name" binding:"required"`
		Code string `json:"code" binding:"required"`
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 查询角色
	var role models.ServiceRole
	if err := config.DB.Where("`key` = ?", input.Role).First(&role).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "角色不存在"})
		return
	}

	name := strings.TrimSpace(input.Name)
	code := strings.TrimSpace(input.Code)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入技师姓名"})
		return
	}
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入技师编号"})
		return
	}

	account := "js" + code
	defaultPassword := code + "12345"

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	tech := models.Technician{
		MerchantID:    merchantID,
		ServiceRoleID: role.ID,
		Name:          name,
		Code:          code,
		Account:       account,
		Password:      string(hashedPassword),
		IsActive:      true,
	}
	if err := config.DB.Create(&tech).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "创建失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"id":               tech.ID,
			"merchant_id":      tech.MerchantID,
			"name":             tech.Name,
			"code":             tech.Code,
			"account":          tech.Account,
			"default_password": defaultPassword,
		},
	})
}

func GetTechniciansByMerchantID(c *gin.Context) {
	merchantID := c.Param("id")
	if strings.TrimSpace(merchantID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "商户ID不能为空"})
		return
	}

	// 查找预约权限
	var appointmentViewPerm models.Permission
	if err := config.DB.Where("`key` = ?", "merchant.appointment.view").First(&appointmentViewPerm).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"data": []models.Technician{}})
		return
	}

	// 获取所有技师
	var list []models.Technician
	config.DB.Where("merchant_id = ? AND is_active = ?", merchantID, true).Order("id desc").Find(&list)

	// 过滤出具有预约权限的技师
	var result []models.Technician
	merchantIDUint, _ := strconv.ParseUint(merchantID, 10, 32)
	for _, tech := range list {
		// 检查商户级别的权限覆盖
		var override models.MerchantRolePermissionOverride
		err := config.DB.Where("merchant_id = ? AND service_role_id = ? AND permission_id = ?", uint(merchantIDUint), tech.ServiceRoleID, appointmentViewPerm.ID).First(&override).Error
		if err == nil {
			if override.Allowed {
				result = append(result, tech)
			}
			continue
		}

		// 检查全局角色权限
		var rolePerm models.RolePermission
		err = config.DB.Where("service_role_id = ? AND permission_id = ? AND allowed = ?", tech.ServiceRoleID, appointmentViewPerm.ID, true).First(&rolePerm).Error
		if err == nil {
			result = append(result, tech)
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}
