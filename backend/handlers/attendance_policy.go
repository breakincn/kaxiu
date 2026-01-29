package handlers

import (
	"kabao/config"
	"kabao/models"
	"time"

	"gorm.io/gorm"
)

func isRoleAttendanceRequired(tx *gorm.DB, merchantID uint, serviceRoleID uint) (bool, error) {
	if merchantID == 0 || serviceRoleID == 0 {
		return true, nil
	}
	if tx == nil {
		tx = config.DB
	}

	// 商户覆盖优先
	var cfg models.MerchantRoleAttendanceConfig
	err := tx.Where("merchant_id = ? AND service_role_id = ?", merchantID, serviceRoleID).First(&cfg).Error
	if err == nil {
		return cfg.RequireAttendance, nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return true, err
	}

	// 否则取岗位默认
	var role models.ServiceRole
	if err := tx.Select("id,require_attendance").Where("id = ?", serviceRoleID).First(&role).Error; err != nil {
		return true, err
	}
	return role.RequireAttendance, nil
}

// ensureAttendanceForNoCheckinRole: 当岗位不需要签到时，确保当天有一条有效考勤记录（默认空闲）
func ensureAttendanceForNoCheckinRole(tx *gorm.DB, merchantID uint, technicianID uint, now time.Time) error {
	if merchantID == 0 || technicianID == 0 {
		return nil
	}
	if tx == nil {
		tx = config.DB
	}

	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var att models.TechnicianAttendance
	err := tx.Where("merchant_id = ? AND technician_id = ? AND checked_in_at >= ? AND checked_out_at IS NULL", merchantID, technicianID, start).
		Order("id desc").
		First(&att).Error
	if err == nil {
		return nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	att = models.TechnicianAttendance{MerchantID: merchantID, TechnicianID: technicianID, CheckedInAt: &now, CheckedOutAt: nil, Status: "idle", NextStatus: nil}
	return tx.Create(&att).Error
}
