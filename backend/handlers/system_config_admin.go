package handlers

import (
	"kabao/config"
	"kabao/models"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	systemConfigKeyUserAppDomain     = "user_app_domain"
	systemConfigKeyMerchantAppDomain = "merchant_app_domain"
	systemConfigKeyAPIDomain         = "api_domain"
)

func AdminGetSystemConfig(c *gin.Context) {
	defaults := gin.H{
		"user_app_domain":     "kabao.app",
		"merchant_app_domain": "kabao.shop",
		"api_domain":          "api.kabao.app",
	}

	get := func(key string) string {
		var sc models.SystemConfig
		err := config.DB.Where("`key` = ?", key).First(&sc).Error
		if err == nil {
			return strings.TrimSpace(sc.Value)
		}
		return ""
	}

	if v := get(systemConfigKeyUserAppDomain); v != "" {
		defaults["user_app_domain"] = v
	}
	if v := get(systemConfigKeyMerchantAppDomain); v != "" {
		defaults["merchant_app_domain"] = v
	}
	if v := get(systemConfigKeyAPIDomain); v != "" {
		defaults["api_domain"] = v
	}

	c.JSON(http.StatusOK, gin.H{"data": defaults})
}

func AdminUpdateSystemConfig(c *gin.Context) {
	var input struct {
		UserAppDomain     *string `json:"user_app_domain"`
		MerchantAppDomain *string `json:"merchant_app_domain"`
		APIDomain         *string `json:"api_domain"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	save := func(key string, val *string) error {
		if val == nil {
			return nil
		}
		v := strings.TrimSpace(*val)
		if v == "" {
			return nil
		}
		var sc models.SystemConfig
		err := config.DB.Where("`key` = ?", key).First(&sc).Error
		if err == nil {
			return config.DB.Model(&models.SystemConfig{}).Where("id = ?", sc.ID).Updates(map[string]interface{}{"value": v}).Error
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}
		return config.DB.Create(&models.SystemConfig{Key: key, Value: v}).Error
	}

	if err := save(systemConfigKeyUserAppDomain, input.UserAppDomain); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}
	if err := save(systemConfigKeyMerchantAppDomain, input.MerchantAppDomain); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}
	if err := save(systemConfigKeyAPIDomain, input.APIDomain); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}

	AdminGetSystemConfig(c)
}
