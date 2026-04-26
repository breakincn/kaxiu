package config

import (
	"fmt"
	"log"
	"os"
	"strings"
)

func IsTruthyEnv(key string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func EnvString(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

func JWTSecret() string {
	return strings.TrimSpace(os.Getenv("KABAO_JWT_SECRET"))
}

func UserJWTSecret() string {
	return strings.TrimSpace(os.Getenv("KABAO_USER_JWT_SECRET"))
}

func UserCodeSecret() string {
	return strings.TrimSpace(os.Getenv("KABAO_USER_CODE_SECRET"))
}

func SMSDebugEnabled() bool {
	return IsTruthyEnv("KABAO_SMS_DEBUG")
}

func MerchantRegisterSMSVerificationDisabled() bool {
	return !IsTruthyEnv("KABAO_MERCHANT_REGISTER_REQUIRE_SMS")
}

func UserRegisterSMSVerificationDisabled() bool {
	return !IsTruthyEnv("KABAO_USER_REGISTER_REQUIRE_SMS")
}

func ValidateCriticalSecrets() error {
	required := []string{
		"KABAO_JWT_SECRET",
		"KABAO_USER_JWT_SECRET",
		"KABAO_USER_CODE_SECRET",
	}

	if !enforceSecretValidation() {
		ensureDevSecrets(required)
		return nil
	}

	missing := make([]string, 0, len(required))
	for _, key := range required {
		if strings.TrimSpace(os.Getenv(key)) == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required secrets: %s", strings.Join(missing, ", "))
	}
	return nil
}

func enforceSecretValidation() bool {
	if IsTruthyEnv("KABAO_ENFORCE_SECRET") {
		return true
	}
	env := strings.ToLower(strings.TrimSpace(os.Getenv("KABAO_ENV")))
	return env == "prod" || env == "production"
}

func ensureDevSecrets(keys []string) {
	for _, key := range keys {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			continue
		}
		_ = os.Setenv(key, "dev-insecure-"+strings.ToLower(key))
		log.Printf("WARN: %s 未配置，已注入开发环境临时密钥（请勿用于生产）", key)
	}
}
