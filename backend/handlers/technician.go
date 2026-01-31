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
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func nextTechnicianCode4(tx *gorm.DB, merchantID uint, serviceRoleID uint, roleType string) (string, error) {
	// 运营客服用3位编号（001），专业客服用4位编号（0001）
	digits := 4
	if roleType == "operational" {
		digits = 3
	}

	var last string
	err := tx.Raw(
		fmt.Sprintf("SELECT code FROM technicians WHERE merchant_id = ? AND service_role_id = ? AND code REGEXP '^[0-9]{%d}$' ORDER BY code DESC LIMIT 1 FOR UPDATE", digits),
		merchantID,
		serviceRoleID,
	).Scan(&last).Error
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(last) == "" {
		return fmt.Sprintf("%0*d", digits, 1), nil
	}
	seq, err := strconv.Atoi(last)
	if err != nil {
		return fmt.Sprintf("%0*d", digits, 1), nil
	}
	seq++
	if seq < 1 {
		seq = 1
	}
	return fmt.Sprintf("%0*d", digits, seq), nil
}

func nextTechnicianCodeByPrefix(tx *gorm.DB, merchantID uint, prefix string, roleType string) (string, error) {
	// 运营客服用3位编号（001），专业客服用4位编号（0001）
	digits := 4
	if roleType == "operational" {
		digits = 3
	}

	p := strings.ToLower(strings.TrimSpace(prefix))
	if p == "" {
		return "", fmt.Errorf("empty prefix")
	}

	// account 固定位数时按字符串倒序即可得到最大号
	// 例：js0003 > js0002
	var lastAccount string
	pattern := fmt.Sprintf("^%s[0-9]{%d}$", p, digits)
	err := tx.Raw(
		"SELECT account FROM technicians WHERE merchant_id = ? AND account REGEXP ? ORDER BY account DESC LIMIT 1 FOR UPDATE",
		merchantID,
		pattern,
	).Scan(&lastAccount).Error
	if err != nil {
		return "", err
	}
	lastAccount = strings.TrimSpace(lastAccount)
	if lastAccount == "" {
		return fmt.Sprintf("%0*d", digits, 1), nil
	}
	if !strings.HasPrefix(lastAccount, p) {
		return fmt.Sprintf("%0*d", digits, 1), nil
	}
	lastCode := strings.TrimPrefix(lastAccount, p)
	seq, err := strconv.Atoi(lastCode)
	if err != nil {
		return fmt.Sprintf("%0*d", digits, 1), nil
	}
	seq++
	if seq < 1 {
		seq = 1
	}
	return fmt.Sprintf("%0*d", digits, seq), nil
}

func GetCurrentTechnician(c *gin.Context) {
	authType, _ := c.Get("auth_type")
	if authType != "staff" {
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
	if err := config.DB.
		Preload("ServiceRole").
		Where("id = ? AND merchant_id = ?", technicianID, merchantID).
		First(&tech).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "技师不存在"})
		return
	}

	// 应用商户对“岗位是否需要签到”的覆盖配置
	{
		var o models.MerchantRoleAttendanceConfig
		if err := config.DB.
			Where("merchant_id = ? AND service_role_id = ?", merchantID, tech.ServiceRoleID).
			First(&o).Error; err == nil {
			tech.ServiceRole.RequireAttendance = o.RequireAttendance
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": tech})
}

func BindTechnicianPhone(c *gin.Context) {
	authType, _ := c.Get("auth_type")
	if authType != "staff" {
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
	if authType == "staff" {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅商户可操作"})
		return
	}

	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	// 角色参数可选：不传则返回全部工作人员
	roleKey := strings.TrimSpace(c.Query("role"))

	q := config.DB.Model(&models.Technician{}).Where("merchant_id = ?", merchantID)
	if roleKey != "" {
		// 查询角色ID
		var role models.ServiceRole
		if err := config.DB.Where("`key` = ?", roleKey).First(&role).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "角色不存在"})
			return
		}
		q = q.Where("service_role_id = ?", role.ID)
	}

	var list []models.Technician
	q.Preload("ServiceRole").Order("id desc").Find(&list)
	c.JSON(http.StatusOK, gin.H{"data": list})
}

func UpdateMerchantTechnician(c *gin.Context) {
	authType, _ := c.Get("auth_type")
	if authType == "staff" {
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
		WindowNo *string `json:"window_no"`
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
	if input.WindowNo != nil {
		windowNo := strings.TrimSpace(*input.WindowNo)
		updates["window_no"] = windowNo
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
	if authType == "staff" {
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
	if authType == "staff" {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅商户可操作"})
		return
	}

	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	var input struct {
		Name string `json:"name" binding:"required"`
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
	// 角色归属校验：
	// - 运营角色：平台默认（merchant_id IS NULL）或本商户自定义（merchant_id = 当前商户）
	// - 专业角色：商户自定义（merchant_id = 当前商户）
	if strings.TrimSpace(role.RoleType) == "operational" {
		if role.MerchantID != nil && *role.MerchantID != merchantID {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权使用该角色"})
			return
		}
	} else if strings.TrimSpace(role.RoleType) == "professional" {
		// 专业角色：允许使用平台创建的角色（merchant_id IS NULL）和商户自定义的角色（merchant_id = 当前商户）
		if role.MerchantID != nil && *role.MerchantID != merchantID {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权使用该岗位"})
			return
		}
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入姓名"})
		return
	}

	prefix := strings.ToLower(strings.TrimSpace(role.AccountPrefix))
	if prefix == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该角色未配置账号前缀"})
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

	var tech models.Technician
	defaultPassword := ""
	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		code, err := nextTechnicianCodeByPrefix(tx, merchantID, prefix, role.RoleType)
		if err != nil {
			return err
		}
		account := prefix + code
		defaultPassword = account + "123"
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		tech = models.Technician{
			MerchantID:    merchantID,
			ServiceRoleID: role.ID,
			Name:          name,
			Code:          code,
			Account:       account,
			Password:      string(hashedPassword),
			IsActive:      true,
		}
		return tx.Create(&tech).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
