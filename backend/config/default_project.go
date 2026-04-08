package config

import (
	"kabao/models"

	"gorm.io/gorm"
)

func GetDefaultMerchantProject(tx *gorm.DB, merchantID uint) (*models.MerchantProject, error) {
	if tx == nil || merchantID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	var project models.MerchantProject
	err := tx.
		Where("merchant_id = ? AND is_default = ?", merchantID, true).
		Order("sort_order asc, id desc").
		First(&project).Error
	if err == nil {
		return &project, nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	err = tx.
		Where("merchant_id = ?", merchantID).
		Order("sort_order asc, id desc").
		First(&project).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func ResolveMerchantProject(tx *gorm.DB, merchantID uint, projectID *uint) (*models.MerchantProject, error) {
	if tx == nil || merchantID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	if projectID != nil && *projectID > 0 {
		var project models.MerchantProject
		if err := tx.Where("merchant_id = ? AND id = ?", merchantID, *projectID).First(&project).Error; err == nil {
			return &project, nil
		} else if err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}
	return GetDefaultMerchantProject(tx, merchantID)
}

func EnsureMerchantDefaultProject(tx *gorm.DB, merchantID uint) error {
	if tx == nil || merchantID == 0 {
		return nil
	}
	var count int64
	if err := tx.Model(&models.MerchantProject{}).Where("merchant_id = ?", merchantID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return nil
	}
	var defaultCount int64
	if err := tx.Model(&models.MerchantProject{}).Where("merchant_id = ? AND is_default = ?", merchantID, true).Count(&defaultCount).Error; err != nil {
		return err
	}
	if defaultCount == 1 {
		return nil
	}
	if err := tx.Model(&models.MerchantProject{}).Where("merchant_id = ?", merchantID).Update("is_default", false).Error; err != nil {
		return err
	}
	var project models.MerchantProject
	if err := tx.Where("merchant_id = ?", merchantID).Order("sort_order asc, id desc").First(&project).Error; err != nil {
		return err
	}
	return tx.Model(&models.MerchantProject{}).Where("id = ?", project.ID).Update("is_default", true).Error
}

func SetMerchantDefaultProject(tx *gorm.DB, merchantID uint, projectID uint) error {
	if tx == nil || merchantID == 0 || projectID == 0 {
		return nil
	}
	return tx.Transaction(func(inner *gorm.DB) error {
		var project models.MerchantProject
		if err := inner.Where("merchant_id = ? AND id = ?", merchantID, projectID).First(&project).Error; err != nil {
			return err
		}
		if err := inner.Model(&models.MerchantProject{}).Where("merchant_id = ?", merchantID).Update("is_default", false).Error; err != nil {
			return err
		}
		return inner.Model(&models.MerchantProject{}).Where("id = ?", projectID).Update("is_default", true).Error
	})
}

func EnsureAllMerchantsDefaultProjects(tx *gorm.DB) error {
	if tx == nil {
		return nil
	}
	var merchantIDs []uint
	if err := tx.Model(&models.MerchantProject{}).Distinct("merchant_id").Pluck("merchant_id", &merchantIDs).Error; err != nil {
		return err
	}
	for _, merchantID := range merchantIDs {
		if err := EnsureMerchantDefaultProject(tx, merchantID); err != nil {
			return err
		}
	}
	return nil
}
