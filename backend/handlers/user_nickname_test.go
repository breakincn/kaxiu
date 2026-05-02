package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kabao/config"
	"kabao/models"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupUserNicknameTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_loc=auto"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	return db
}

func TestUpdateUserNicknameRejectsInvalidValues(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldDB := config.DB
	defer func() { config.DB = oldDB }()

	db := setupUserNicknameTestDB(t)
	config.DB = db

	user := models.User{Username: "nickname-user", Password: "pwd", Nickname: "原昵称"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	testCases := []struct {
		name    string
		payload map[string]any
		wantErr string
	}{
		{
			name:    "contains spaces",
			payload: map[string]any{"nickname": "张 三"},
			wantErr: "昵称不能包含空格",
		},
		{
			name:    "too long",
			payload: map[string]any{"nickname": "一二三四五六七八"},
			wantErr: "昵称最多 7 个字",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.payload)
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPut, "/user/nickname", bytes.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Set("user_id", user.ID)

			UpdateUserNickname(c)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("want 400, got %d body=%s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), tc.wantErr) {
				t.Fatalf("want error %q, got body=%s", tc.wantErr, rec.Body.String())
			}
		})
	}
}

