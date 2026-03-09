package config

import (
	"fmt"
	"kabao/models"
	"sort"
	"strings"

	"gorm.io/gorm"
)

var fixedServiceRoleKeys = map[string]struct{}{
	"store_manager": {},
	"front_desk":    {},
}

var reservedServiceRoleNames = map[string]struct{}{
	"店长": {},
	"前台": {},
}

func FixedServiceRoleKeys() []string {
	return []string{"store_manager", "front_desk"}
}

func IsFixedServiceRoleKey(key string) bool {
	_, ok := fixedServiceRoleKeys[strings.TrimSpace(key)]
	return ok
}

func IsReservedServiceRoleName(name string) bool {
	_, ok := reservedServiceRoleNames[strings.TrimSpace(name)]
	return ok
}

func BuildMerchantServiceRoleKey(merchantID uint, prefix string) string {
	return fmt.Sprintf("m%d_%s", merchantID, strings.ToLower(strings.TrimSpace(prefix)))
}

func IsMerchantUsableRole(role *models.ServiceRole, merchantID uint) bool {
	if role == nil || merchantID == 0 || !role.IsActive {
		return false
	}
	if IsFixedServiceRoleKey(role.Key) {
		return role.MerchantID == nil
	}
	return role.MerchantID != nil && *role.MerchantID == merchantID
}

func FindMerchantUsableRole(tx *gorm.DB, merchantID uint, roleKey string) (*models.ServiceRole, error) {
	if tx == nil || merchantID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	var role models.ServiceRole
	if err := tx.Where("`key` = ?", strings.TrimSpace(roleKey)).First(&role).Error; err != nil {
		return nil, err
	}
	if !IsMerchantUsableRole(&role, merchantID) {
		return nil, gorm.ErrRecordNotFound
	}
	return &role, nil
}

func ProfessionalBasePermissionKeys(tx *gorm.DB) []string {
	if tx == nil {
		return nil
	}
	var keys []string
	var sc models.SystemConfig
	if err := tx.Where("`key` = ?", "professional_base_permission_keys").First(&sc).Error; err == nil {
		seen := map[string]struct{}{}
		for _, raw := range strings.Split(sc.Value, ",") {
			key := strings.TrimSpace(raw)
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			keys = append(keys, key)
		}
		sort.Strings(keys)
	}
	return keys
}
