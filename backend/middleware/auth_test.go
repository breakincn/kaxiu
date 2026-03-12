package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"kabao/config"
	"kabao/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAuthMiddlewareTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:middleware_auth_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Merchant{}, &models.ServiceRole{}, &models.Technician{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	if err := db.Create(&models.User{Username: "u1", Password: "x"}).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	merchant := models.Merchant{Name: "m1", Phone: "18800000001", Password: "x"}
	if err := db.Create(&merchant).Error; err != nil {
		t.Fatalf("create merchant failed: %v", err)
	}
	roleMerchantID := merchant.ID
	role := models.ServiceRole{
		MerchantID: &roleMerchantID,
		Key:        config.BuildMerchantServiceRoleKey(merchant.ID, "js"),
		Name:       "技师",
		IsActive:   true,
	}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("create role failed: %v", err)
	}
	tech := models.Technician{
		MerchantID:    merchant.ID,
		ServiceRoleID: role.ID,
		Name:          "技师A",
		Code:          "0001",
		Account:       "js0001",
		Password:      "x",
		IsActive:      true,
	}
	if err := db.Create(&tech).Error; err != nil {
		t.Fatalf("create technician failed: %v", err)
	}
	return db
}

func signedJWT(t *testing.T, secret string, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	raw, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token failed: %v", err)
	}
	return raw
}

func authTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AuthMiddleware())
	r.GET("/probe", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"auth_type":       c.GetString("auth_type"),
			"user_id":         c.GetUint("user_id"),
			"merchant_id":     c.GetUint("merchant_id"),
			"staff_id":        c.GetUint("staff_id"),
			"service_role_id": c.GetUint("service_role_id"),
		})
	})
	return r
}

func TestAuthMiddlewareRejectsLegacyUserToken(t *testing.T) {
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupAuthMiddlewareTestDB(t)

	r := authTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("Authorization", "Bearer user_1_123456")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Auth-Legacy-Token"); got != "" {
		t.Fatalf("want no legacy header, got %q", got)
	}
}

func TestAuthMiddlewareAcceptsUserJWTAndRejectsWrongTypeOrExpired(t *testing.T) {
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupAuthMiddlewareTestDB(t)
	t.Setenv("KABAO_USER_JWT_SECRET", "user-secret")
	t.Setenv("KABAO_JWT_SECRET", "merchant-secret")

	r := authTestRouter()

	okToken := signedJWT(t, config.UserJWTSecret(), jwt.MapClaims{
		"type":    "user",
		"user_id": 1,
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("Authorization", "Bearer "+okToken)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	wrongTypeToken := signedJWT(t, config.UserJWTSecret(), jwt.MapClaims{
		"type":    "merchant",
		"user_id": 1,
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	reqWrongType := httptest.NewRequest(http.MethodGet, "/probe", nil)
	reqWrongType.Header.Set("Authorization", "Bearer "+wrongTypeToken)
	recWrongType := httptest.NewRecorder()
	r.ServeHTTP(recWrongType, reqWrongType)
	if recWrongType.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 for wrong type, got %d body=%s", recWrongType.Code, recWrongType.Body.String())
	}

	expiredToken := signedJWT(t, config.UserJWTSecret(), jwt.MapClaims{
		"type":    "user",
		"user_id": 1,
		"exp":     time.Now().Add(-time.Hour).Unix(),
	})
	reqExpired := httptest.NewRequest(http.MethodGet, "/probe", nil)
	reqExpired.Header.Set("Authorization", "Bearer "+expiredToken)
	recExpired := httptest.NewRecorder()
	r.ServeHTTP(recExpired, reqExpired)
	if recExpired.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 for expired token, got %d body=%s", recExpired.Code, recExpired.Body.String())
	}
}

func TestAuthMiddlewareAcceptsMerchantJWTAndRejectsWrongTypeOrExpired(t *testing.T) {
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupAuthMiddlewareTestDB(t)
	t.Setenv("KABAO_USER_JWT_SECRET", "user-secret")
	t.Setenv("KABAO_JWT_SECRET", "merchant-secret")

	r := authTestRouter()

	okToken := signedJWT(t, config.JWTSecret(), jwt.MapClaims{
		"type":        "merchant",
		"merchant_id": 1,
		"exp":         time.Now().Add(time.Hour).Unix(),
	})
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("Authorization", "Bearer "+okToken)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	wrongTypeToken := signedJWT(t, config.JWTSecret(), jwt.MapClaims{
		"type":        "user",
		"merchant_id": 1,
		"exp":         time.Now().Add(time.Hour).Unix(),
	})
	reqWrongType := httptest.NewRequest(http.MethodGet, "/probe", nil)
	reqWrongType.Header.Set("Authorization", "Bearer "+wrongTypeToken)
	recWrongType := httptest.NewRecorder()
	r.ServeHTTP(recWrongType, reqWrongType)
	if recWrongType.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 for wrong type, got %d body=%s", recWrongType.Code, recWrongType.Body.String())
	}

	expiredToken := signedJWT(t, config.JWTSecret(), jwt.MapClaims{
		"type":        "merchant",
		"merchant_id": 1,
		"exp":         time.Now().Add(-time.Hour).Unix(),
	})
	reqExpired := httptest.NewRequest(http.MethodGet, "/probe", nil)
	reqExpired.Header.Set("Authorization", "Bearer "+expiredToken)
	recExpired := httptest.NewRecorder()
	r.ServeHTTP(recExpired, reqExpired)
	if recExpired.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 for expired token, got %d body=%s", recExpired.Code, recExpired.Body.String())
	}
}

func TestAuthMiddlewareAcceptsStaffJWTAndRejectsWrongTypeOrExpired(t *testing.T) {
	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupAuthMiddlewareTestDB(t)
	t.Setenv("KABAO_USER_JWT_SECRET", "user-secret")
	t.Setenv("KABAO_JWT_SECRET", "merchant-secret")

	r := authTestRouter()

	okToken := signedJWT(t, config.JWTSecret(), jwt.MapClaims{
		"type":            "staff",
		"merchant_id":     1,
		"staff_id":        1,
		"service_role_id": 1,
		"exp":             time.Now().Add(time.Hour).Unix(),
	})
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("Authorization", "Bearer "+okToken)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	wrongTypeToken := signedJWT(t, config.JWTSecret(), jwt.MapClaims{
		"type":            "staffx",
		"merchant_id":     1,
		"staff_id":        1,
		"service_role_id": 1,
		"exp":             time.Now().Add(time.Hour).Unix(),
	})
	reqWrongType := httptest.NewRequest(http.MethodGet, "/probe", nil)
	reqWrongType.Header.Set("Authorization", "Bearer "+wrongTypeToken)
	recWrongType := httptest.NewRecorder()
	r.ServeHTTP(recWrongType, reqWrongType)
	if recWrongType.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 for wrong type, got %d body=%s", recWrongType.Code, recWrongType.Body.String())
	}

	expiredToken := signedJWT(t, config.JWTSecret(), jwt.MapClaims{
		"type":            "staff",
		"merchant_id":     1,
		"staff_id":        1,
		"service_role_id": 1,
		"exp":             time.Now().Add(-time.Hour).Unix(),
	})
	reqExpired := httptest.NewRequest(http.MethodGet, "/probe", nil)
	reqExpired.Header.Set("Authorization", "Bearer "+expiredToken)
	recExpired := httptest.NewRecorder()
	r.ServeHTTP(recExpired, reqExpired)
	if recExpired.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 for expired token, got %d body=%s", recExpired.Code, recExpired.Body.String())
	}
}
