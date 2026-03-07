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
)

func TestPrepareVerifyDoesNotCreateUsageOrConsumeVerifyCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("KABAO_JWT_SECRET", "test-jwt-secret")

	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupHandCardVerifyAgeGuardTestDB(t)

	merchant, card, verifyCode := seedAgeGuardFixture(t, config.DB, true)

	body, _ := json.Marshal(map[string]any{"code": verifyCode.Code})
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/merchant/verify/prepare", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("merchant_id", merchant.ID)
	c.Set("auth_type", "merchant")

	PrepareVerify(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Data struct {
			VerifyToken  string `json:"verify_token"`
			NeedHandCard bool   `json:"need_hand_card"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if payload.Data.VerifyToken == "" {
		t.Fatalf("want verify token in prepare response")
	}
	if !payload.Data.NeedHandCard {
		t.Fatalf("want need_hand_card=true")
	}

	assertNoVerifySideEffects(t, config.DB, verifyCode.ID, card.ID, 0)
}

func TestCommitPreparedVerifyCreatesUsageWithHandCardAtomically(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("KABAO_JWT_SECRET", "test-jwt-secret")

	oldDB := config.DB
	defer func() { config.DB = oldDB }()
	config.DB = setupHandCardVerifyAgeGuardTestDB(t)

	merchant, card, verifyCode := seedAgeGuardFixture(t, config.DB, true)

	prepareBody, _ := json.Marshal(map[string]any{"code": verifyCode.Code})
	prepareRec := httptest.NewRecorder()
	prepareCtx, _ := gin.CreateTestContext(prepareRec)
	prepareCtx.Request = httptest.NewRequest(http.MethodPost, "/merchant/verify/prepare", bytes.NewReader(prepareBody))
	prepareCtx.Request.Header.Set("Content-Type", "application/json")
	prepareCtx.Set("merchant_id", merchant.ID)
	prepareCtx.Set("auth_type", "merchant")

	PrepareVerify(prepareCtx)
	if prepareRec.Code != http.StatusOK {
		t.Fatalf("prepare want 200, got %d body=%s", prepareRec.Code, prepareRec.Body.String())
	}

	var preparePayload struct {
		Data struct {
			VerifyToken string `json:"verify_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(prepareRec.Body.Bytes(), &preparePayload); err != nil {
		t.Fatalf("unmarshal prepare response failed: %v", err)
	}
	if preparePayload.Data.VerifyToken == "" {
		t.Fatalf("prepare verify token should not be empty")
	}

	commitBody, _ := json.Marshal(map[string]any{
		"verify_token": preparePayload.Data.VerifyToken,
		"hand_card_no": "H012",
	})
	commitRec := httptest.NewRecorder()
	commitCtx, _ := gin.CreateTestContext(commitRec)
	commitCtx.Request = httptest.NewRequest(http.MethodPost, "/merchant/verify/commit", bytes.NewReader(commitBody))
	commitCtx.Request.Header.Set("Content-Type", "application/json")
	commitCtx.Set("merchant_id", merchant.ID)
	commitCtx.Set("auth_type", "merchant")

	CommitPreparedVerify(commitCtx)
	if commitRec.Code != http.StatusOK {
		t.Fatalf("commit want 200, got %d body=%s", commitRec.Code, commitRec.Body.String())
	}

	var usage struct {
		HandCardNo         *string `gorm:"column:hand_card_no"`
		HandCardAssignedAt string  `gorm:"column:hand_card_assigned_at"`
		UsedAt             string  `gorm:"column:used_at"`
	}
	if err := config.DB.Model(&models.Usage{}).
		Select("hand_card_no, hand_card_assigned_at, used_at").
		Where("card_id = ?", card.ID).
		Order("id asc").
		Take(&usage).Error; err != nil {
		t.Fatalf("load usage failed: %v", err)
	}
	if usage.HandCardNo == nil || *usage.HandCardNo != "H012" {
		t.Fatalf("want hand card H012, got %+v", usage.HandCardNo)
	}
	if strings.TrimSpace(usage.HandCardAssignedAt) == "" {
		t.Fatalf("want hand_card_assigned_at set")
	}
	if strings.TrimSpace(usage.UsedAt) == "" {
		t.Fatalf("want used_at set")
	}

	var refreshedVerifyCode struct {
		Used bool `gorm:"column:used"`
	}
	if err := config.DB.Model(&models.VerifyCode{}).
		Select("used").
		Where("id = ?", verifyCode.ID).
		Take(&refreshedVerifyCode).Error; err != nil {
		t.Fatalf("load verify code failed: %v", err)
	}
	if !refreshedVerifyCode.Used {
		t.Fatalf("verify code should be consumed after commit")
	}

	var refreshedCard struct {
		RemainTimes int `gorm:"column:remain_times"`
		UsedTimes   int `gorm:"column:used_times"`
	}
	if err := config.DB.Model(&models.Card{}).
		Select("remain_times, used_times").
		Where("id = ?", card.ID).
		Take(&refreshedCard).Error; err != nil {
		t.Fatalf("load card failed: %v", err)
	}
	if refreshedCard.RemainTimes != 9 || refreshedCard.UsedTimes != 1 {
		t.Fatalf("unexpected card counts remain=%d used=%d", refreshedCard.RemainTimes, refreshedCard.UsedTimes)
	}
}
