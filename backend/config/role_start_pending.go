package config

import (
	"kabao/models"
	"strings"

	"gorm.io/gorm"
)

const DefaultRoleStartPendingTimeoutSeconds = 300

func NormalizeRoleStartPendingTimeoutSeconds(v int) int {
	if v < 1 {
		return DefaultRoleStartPendingTimeoutSeconds
	}
	if v > 3600 {
		return 3600
	}
	return v
}

func EffectiveRoleStartPendingTimeoutSeconds(role *models.ServiceRole, override *models.MerchantRoleStartPendingConfig) int {
	if override != nil && override.StartPendingTimeoutSeconds > 0 {
		return NormalizeRoleStartPendingTimeoutSeconds(override.StartPendingTimeoutSeconds)
	}
	if role != nil && role.StartPendingTimeoutSeconds > 0 {
		return NormalizeRoleStartPendingTimeoutSeconds(role.StartPendingTimeoutSeconds)
	}
	return DefaultRoleStartPendingTimeoutSeconds
}

func GetMerchantRoleStartPendingTimeoutSeconds(tx *gorm.DB, merchantID uint, serviceRoleID uint) int {
	if tx == nil || merchantID == 0 || serviceRoleID == 0 {
		return DefaultRoleStartPendingTimeoutSeconds
	}
	var role models.ServiceRole
	if err := tx.Select("id", "start_pending_timeout_seconds").First(&role, serviceRoleID).Error; err != nil {
		return DefaultRoleStartPendingTimeoutSeconds
	}
	var override models.MerchantRoleStartPendingConfig
	if err := tx.Where("merchant_id = ? AND service_role_id = ?", merchantID, serviceRoleID).First(&override).Error; err == nil {
		return EffectiveRoleStartPendingTimeoutSeconds(&role, &override)
	}
	return EffectiveRoleStartPendingTimeoutSeconds(&role, nil)
}

func GetProjectStartPendingTimeoutSeconds(tx *gorm.DB, projectID uint) int {
	if tx == nil || projectID == 0 {
		return 0
	}
	var project struct {
		StartPendingTimeoutSeconds int `gorm:"column:start_pending_timeout_seconds"`
	}
	if err := tx.Table("merchant_projects").
		Select("start_pending_timeout_seconds").
		Where("id = ?", projectID).
		First(&project).Error; err != nil {
		return 0
	}
	return NormalizeRoleStartPendingTimeoutSeconds(project.StartPendingTimeoutSeconds)
}

func ResolveServiceSessionStartPendingTimeoutSeconds(tx *gorm.DB, merchantID uint, technicianID uint, projectID *uint) int {
	if projectID != nil && *projectID > 0 {
		if seconds := GetProjectStartPendingTimeoutSeconds(tx, *projectID); seconds > 0 {
			return seconds
		}
	}
	return GetMerchantTechnicianStartPendingTimeoutSeconds(tx, merchantID, technicianID)
}

func ResolveServiceSessionStartPendingTimeoutSecondsByRole(tx *gorm.DB, merchantID uint, serviceRoleID uint, projectID *uint) int {
	if projectID != nil && *projectID > 0 {
		if seconds := GetProjectStartPendingTimeoutSeconds(tx, *projectID); seconds > 0 {
			return seconds
		}
	}
	return GetMerchantRoleStartPendingTimeoutSeconds(tx, merchantID, serviceRoleID)
}

func GetMerchantTechnicianStartPendingTimeoutSeconds(tx *gorm.DB, merchantID uint, technicianID uint) int {
	if tx == nil || merchantID == 0 || technicianID == 0 {
		return DefaultRoleStartPendingTimeoutSeconds
	}
	var tech struct {
		ServiceRoleID uint `gorm:"column:service_role_id"`
	}
	if err := tx.Table("technicians").
		Select("service_role_id").
		Where("id = ? AND merchant_id = ?", technicianID, merchantID).
		First(&tech).Error; err != nil {
		return DefaultRoleStartPendingTimeoutSeconds
	}
	return GetMerchantRoleStartPendingTimeoutSeconds(tx, merchantID, tech.ServiceRoleID)
}

func GetMerchantRoleStartPendingConfig(tx *gorm.DB, merchantID uint, serviceRoleID uint) (*models.MerchantRoleStartPendingConfig, error) {
	if tx == nil || merchantID == 0 || serviceRoleID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	var cfg models.MerchantRoleStartPendingConfig
	if err := tx.Where("merchant_id = ? AND service_role_id = ?", merchantID, serviceRoleID).First(&cfg).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

func RoleStartPendingLabelForMerchant(m *models.Merchant) string {
	startTerm := "起单"
	if m != nil {
		if m.SupportQueue && !m.SupportCustomerServiceMode && (m.QueueMode == "auto" || m.QueueMode == "manual") {
			startTerm = "叫号"
		} else if s := strings.TrimSpace(m.StartTerm); s != "" {
			startTerm = s
		}
	}
	return "待" + startTerm + "超时秒数"
}
