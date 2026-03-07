package handlers

import (
	"errors"
	"fmt"
	"kabao/config"
	"kabao/middleware"
	"kabao/models"
	"kabao/queue"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const preparedVerifyTokenTTL = 45 * time.Second

type preparedVerifyClaims struct {
	MerchantID   uint   `json:"merchant_id"`
	VerifyCodeID uint   `json:"verify_code_id"`
	CardID       uint   `json:"card_id"`
	ProjectID    uint   `json:"project_id,omitempty"`
	AuthType     string `json:"auth_type"`
	TechnicianID uint   `json:"technician_id,omitempty"`
	jwt.RegisteredClaims
}

type verifyCommitResult struct {
	Merchant            models.Merchant
	Card                models.Card
	UsedAt              time.Time
	RemainTimes         int
	SessionID           uint
	NextStep            string
	UsageID             uint
	ShouldEnqueueOnsite bool
	Action              string
}

func verifyTokenSecret() string {
	secret := strings.TrimSpace(config.JWTSecret())
	if secret != "" {
		return secret
	}
	return strings.TrimSpace(config.UserCodeSecret())
}

func currentVerifyActor(c *gin.Context) (string, uint) {
	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)
	authType = strings.TrimSpace(authType)
	if authType == "" {
		authType = "merchant"
	}
	var technicianID uint
	if authType == "staff" {
		if v, ok := c.Get("technician_id"); ok {
			if tid, ok := v.(uint); ok {
				technicianID = tid
			}
		}
	}
	return authType, technicianID
}

func signPreparedVerifyToken(claims preparedVerifyClaims) (string, error) {
	secret := verifyTokenSecret()
	if secret == "" {
		return "", fmt.Errorf("签发核销令牌失败")
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func parsePreparedVerifyToken(raw string) (*preparedVerifyClaims, error) {
	secret := verifyTokenSecret()
	if secret == "" {
		return nil, fmt.Errorf("核销令牌密钥未配置")
	}
	claims := &preparedVerifyClaims{}
	parsed, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrTokenSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if parsed == nil || !parsed.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}

func loadVerifyPrepareContext(tx *gorm.DB, merchantID uint, code string, now time.Time) (models.Merchant, models.VerifyCode, models.Card, *models.MerchantProject, error) {
	var merchant models.Merchant
	if err := tx.First(&merchant, merchantID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return merchant, models.VerifyCode{}, models.Card{}, nil, apiErr{status: http.StatusNotFound, msg: "商户不存在"}
		}
		return merchant, models.VerifyCode{}, models.Card{}, nil, err
	}

	var verifyCode models.VerifyCode
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("code = ?", code).First(&verifyCode).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return merchant, verifyCode, models.Card{}, nil, apiErr{status: http.StatusNotFound, msg: "核销码不存在"}
		}
		return merchant, verifyCode, models.Card{}, nil, err
	}
	if verifyCode.Used {
		return merchant, verifyCode, models.Card{}, nil, apiErr{status: http.StatusBadRequest, msg: "核销码已使用"}
	}
	if now.Unix() > verifyCode.ExpireAt {
		return merchant, verifyCode, models.Card{}, nil, apiErr{status: http.StatusBadRequest, msg: "核销码已过期"}
	}

	var card models.Card
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&card, verifyCode.CardID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return merchant, verifyCode, card, nil, apiErr{status: http.StatusNotFound, msg: "卡片不存在"}
		}
		return merchant, verifyCode, card, nil, err
	}
	if card.MerchantID != merchantID {
		return merchant, verifyCode, card, nil, apiErr{status: http.StatusForbidden, msg: "无权核销此卡"}
	}
	if err := verifyHandCardUnreturnedAgeGuard(tx, merchantID, merchant.SupportHandCard, now); err != nil {
		return merchant, verifyCode, card, nil, err
	}
	if card.Locked {
		msg := "卡片已锁定"
		if strings.TrimSpace(card.LockedReason) != "" {
			msg = card.LockedReason
		}
		return merchant, verifyCode, card, nil, apiErr{status: http.StatusBadRequest, msg: msg}
	}
	if card.EndDate != nil && now.After(*card.EndDate) {
		return merchant, verifyCode, card, nil, apiErr{status: http.StatusBadRequest, msg: "卡片已过期"}
	}
	if card.RemainTimes <= 0 {
		return merchant, verifyCode, card, nil, apiErr{status: http.StatusBadRequest, msg: "剩余次数不足"}
	}

	var project *models.MerchantProject
	if verifyCode.ProjectID != nil {
		var p models.MerchantProject
		if err := tx.Where("id = ? AND merchant_id = ?", *verifyCode.ProjectID, merchantID).First(&p).Error; err == nil {
			project = &p
		}
	}

	return merchant, verifyCode, card, project, nil
}

func performVerifyCommit(tx *gorm.DB, c *gin.Context, merchant models.Merchant, verifyCode models.VerifyCode, card models.Card, handCardNo string) (verifyCommitResult, error) {
	result := verifyCommitResult{
		Merchant: merchant,
		Card:     card,
		Action:   "verify",
	}
	now := time.Now()
	authType, _ := currentVerifyActor(c)
	effectiveSupportOrderComplete := merchant.SupportCustomerServiceMode && merchant.SupportOrderComplete
	effectiveSupportRoom := merchant.SupportCustomerServiceMode && merchant.SupportRoom
	isQueueMode := !merchant.SupportCustomerServiceMode && merchant.SupportQueue && (merchant.QueueMode == "auto" || merchant.QueueMode == "manual")
	usageStatus := "success"
	autoFinish := false

	if merchant.SupportCustomerServiceMode {
		usageStatus = "in_progress"
		if authType == "staff" {
			okVF, err := middleware.HasPermission(c, "merchant.card.verify_finish")
			if err != nil {
				return result, err
			}
			if okVF {
				autoFinish = true
				usageStatus = "success"
			}
		}
	} else if isQueueMode || effectiveSupportOrderComplete {
		usageStatus = "in_progress"
	}

	res := tx.Model(&models.VerifyCode{}).
		Where("id = ? AND used = ?", verifyCode.ID, false).
		Updates(map[string]interface{}{"used": true, "used_at": now})
	if res.Error != nil {
		return result, res.Error
	}
	if res.RowsAffected == 0 {
		return result, apiErr{status: http.StatusBadRequest, msg: "核销码已使用"}
	}

	res = tx.Model(&models.Card{}).
		Where("id = ? AND remain_times > 0", card.ID).
		Updates(map[string]interface{}{
			"remain_times": gorm.Expr("remain_times - ?", 1),
			"used_times":   gorm.Expr("used_times + ?", 1),
			"last_used_at": now,
		})
	if res.Error != nil {
		return result, res.Error
	}
	if res.RowsAffected == 0 {
		return result, apiErr{status: http.StatusBadRequest, msg: "剩余次数不足"}
	}

	card.RemainTimes--
	card.UsedTimes++
	card.LastUsedAt = &now
	result.Card = card
	result.RemainTimes = card.RemainTimes
	result.UsedAt = now

	usage := models.Usage{
		CardID:             card.ID,
		MerchantID:         card.MerchantID,
		ProjectID:          verifyCode.ProjectID,
		UsedTimes:          1,
		UsedAt:             &now,
		VerifyCode:         verifyCode.Code,
		VerifyCodeExpireAt: verifyCode.ExpireAt,
		Status:             usageStatus,
	}
	if handCardNo != "" {
		usage.HandCardNo = &handCardNo
		usage.HandCardAssignedAt = &now
	}
	if err := tx.Create(&usage).Error; err != nil {
		return result, err
	}
	result.UsageID = usage.ID

	if !isQueueMode && !merchant.SupportCustomerServiceMode && !effectiveSupportOrderComplete {
		finishedAt := now
		if err := tx.Model(&models.Usage{}).Where("id = ?", usage.ID).Updates(map[string]interface{}{
			"status":      "success",
			"finished_at": &finishedAt,
		}).Error; err != nil {
			return result, err
		}
		return result, nil
	}

	if (!merchant.SupportCustomerServiceMode && effectiveSupportOrderComplete) || isQueueMode {
		durationMinutes := 15
		if verifyCode.ProjectID != nil {
			var project models.MerchantProject
			if err := tx.Where("id = ? AND merchant_id = ?", verifyCode.ProjectID, merchant.ID).First(&project).Error; err == nil && project.Duration > 0 {
				durationMinutes = project.Duration
			}
		}

		delaySeconds := merchant.StartDelaySeconds
		if delaySeconds <= 0 {
			delaySeconds = 60
		}
		startAt := now.Add(time.Duration(delaySeconds) * time.Second)

		status := "delay_pending"
		var roomSelectDeadlineAt *time.Time
		var startConfirmedAt *time.Time
		var scheduledStartAt *time.Time
		if isQueueMode {
			status = "staff_selecting"
			result.NextStep = ""
		} else if effectiveSupportRoom {
			status = "room_selecting"
			dl := now.Add(90 * time.Second)
			roomSelectDeadlineAt = &dl
			result.NextStep = "room_select"
		} else {
			startConfirmedAt = &now
			scheduledStartAt = &startAt
			result.NextStep = ""
		}

		sessionMode := models.ResolveSessionMode(&merchant)
		if isQueueMode {
			status = models.WithModePrefix(sessionMode, status)
		} else if merchant.SupportCustomerServiceMode {
			status = models.WithCSPrefix(status)
		}

		session := models.ServiceSession{
			MerchantID:             merchant.ID,
			UserID:                 card.UserID,
			CardID:                 card.ID,
			ProjectID:              verifyCode.ProjectID,
			InitialUsageID:         usage.ID,
			VerifyCode:             verifyCode.Code,
			SessionMode:            sessionMode,
			Status:                 status,
			RoomSelectDeadlineAt:   roomSelectDeadlineAt,
			StartConfirmedAt:       startConfirmedAt,
			StartDelaySeconds:      delaySeconds,
			ScheduledStartAt:       scheduledStartAt,
			DurationMinutes:        durationMinutes,
			AutoFinishDelaySeconds: 300,
			AutoIdleAfterSeconds:   180,
		}
		if err := tx.Create(&session).Error; err != nil {
			return result, err
		}
		result.SessionID = session.ID
		result.ShouldEnqueueOnsite = isQueueMode
		return result, nil
	}

	if autoFinish {
		finishedAt := now
		if err := tx.Model(&models.Usage{}).Where("id = ?", usage.ID).Updates(map[string]interface{}{
			"status":      "success",
			"finished_at": &finishedAt,
		}).Error; err != nil {
			return result, err
		}
		return result, nil
	}

	status := "staff_selecting"
	var roomSelectDeadlineAt *time.Time
	if effectiveSupportRoom {
		status = "room_selecting"
		dl := now.Add(90 * time.Second)
		roomSelectDeadlineAt = &dl
		result.NextStep = "room_select"
	} else {
		result.NextStep = "staff_select"
	}

	sessionMode := models.ResolveSessionMode(&merchant)
	if isQueueMode {
		status = models.WithModePrefix(sessionMode, status)
	} else if merchant.SupportCustomerServiceMode {
		status = models.WithCSPrefix(status)
	}

	durationMinutes := 50
	if verifyCode.ProjectID != nil {
		var project models.MerchantProject
		if err := tx.Where("id = ? AND merchant_id = ?", verifyCode.ProjectID, merchant.ID).First(&project).Error; err == nil && project.Duration > 0 {
			durationMinutes = project.Duration
		}
	}

	session := models.ServiceSession{
		MerchantID:             merchant.ID,
		UserID:                 card.UserID,
		CardID:                 card.ID,
		ProjectID:              verifyCode.ProjectID,
		InitialUsageID:         usage.ID,
		VerifyCode:             verifyCode.Code,
		SessionMode:            sessionMode,
		Status:                 status,
		RoomSelectDeadlineAt:   roomSelectDeadlineAt,
		StartDelaySeconds:      60,
		DurationMinutes:        durationMinutes,
		AutoFinishDelaySeconds: 60,
		AutoIdleAfterSeconds:   180,
	}
	if err := tx.Create(&session).Error; err != nil {
		return result, err
	}
	result.SessionID = session.ID
	result.ShouldEnqueueOnsite = true
	return result, nil
}

func enqueueVerifyUsageIfNeeded(merchant models.Merchant, card models.Card, usageID uint, shouldEnqueueOnsite bool) {
	if !merchant.SupportQueue || !shouldEnqueueOnsite || usageID == 0 {
		return
	}

	now := time.Now()
	isAppointment := false
	var appt models.Appointment
	if err := config.DB.
		Where("card_id = ? AND merchant_id = ? AND status = 'confirmed' AND appointment_time IS NOT NULL", card.ID, merchant.ID).
		Order("appointment_time asc").
		First(&appt).Error; err == nil && appt.AppointmentTime != nil {
		at := *appt.AppointmentTime
		if at.Format("2006-01-02") == now.Format("2006-01-02") {
			diff := now.Sub(at)
			if diff < 0 {
				diff = -diff
			}
			if diff <= 30*time.Minute {
				isAppointment = true
			}
		}
	}
	if isAppointment {
		return
	}

	date := now.Format("2006-01-02")
	autoCallFirst := false
	if merchant.QueueMode == "auto" && !merchant.SupportMultiCustomerService {
		autoCallFirst = true
	}
	tk, created := queue.Default.Enqueue(merchant.ID, date, queue.QueueTypeOnsite, usageID, merchant.QueueStartNo, autoCallFirst, now)
	if os.Getenv("KABAO_QUEUE_DEBUG") == "1" {
		log.Printf("[queue-debug] verify enqueue onsite: merchant=%d date=%s usage_id=%d created=%v queue_no=%d called_at=%v\n", merchant.ID, date, usageID, created, tk.No, tk.CalledAt)
	}
	if merchant.QueueMode == "auto" && merchant.SupportMultiCustomerService {
		tryAutoCallNextForIdleTechnicians(config.DB, merchant.ID, now)
	}
}

func PrepareVerify(c *gin.Context) {
	merchantIDAny, ok := c.Get("merchant_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	merchantID, ok := merchantIDAny.(uint)
	if !ok || merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var input struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	code := strings.TrimSpace(input.Code)
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code 不能为空"})
		return
	}
	if strings.HasPrefix(code, "SS:") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请扫描核销码"})
		return
	}

	authType, technicianID := currentVerifyActor(c)
	now := time.Now()
	var merchant models.Merchant
	var verifyCode models.VerifyCode
	var card models.Card
	var project *models.MerchantProject

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		merchant, verifyCode, card, project, err = loadVerifyPrepareContext(tx, merchantID, code, now)
		return err
	})
	if err != nil {
		var ae apiErr
		if errors.As(err, &ae) {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	claims := preparedVerifyClaims{
		MerchantID:   merchantID,
		VerifyCodeID: verifyCode.ID,
		CardID:       card.ID,
		AuthType:     authType,
		TechnicianID: technicianID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Subject:   fmt.Sprintf("merchant-verify:%d", verifyCode.ID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(preparedVerifyTokenTTL)),
		},
	}
	if verifyCode.ProjectID != nil {
		claims.ProjectID = *verifyCode.ProjectID
	}
	token, signErr := signPreparedVerifyToken(claims)
	if signErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": signErr.Error()})
		return
	}

	out := gin.H{
		"verify_token":    token,
		"need_hand_card":  merchant.SupportHandCard,
		"verify_code":     verifyCode.Code,
		"expires_in_secs": int(preparedVerifyTokenTTL / time.Second),
		"card": gin.H{
			"id":           card.ID,
			"card_no":      card.CardNo,
			"card_type":    card.CardType,
			"remain_times": card.RemainTimes,
		},
	}
	if project != nil {
		out["project"] = gin.H{
			"id":               project.ID,
			"name":             project.Name,
			"duration_minutes": project.Duration,
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "校验成功", "data": out})
}

func CommitPreparedVerify(c *gin.Context) {
	merchantIDAny, ok := c.Get("merchant_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	merchantID, ok := merchantIDAny.(uint)
	if !ok || merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var input struct {
		VerifyToken  string `json:"verify_token" binding:"required"`
		HandCardNo   string `json:"hand_card_no"`
		SkipHandCard bool   `json:"skip_hand_card"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	claims, err := parsePreparedVerifyToken(strings.TrimSpace(input.VerifyToken))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "核销令牌无效或已过期"})
		return
	}

	authType, technicianID := currentVerifyActor(c)
	if claims.MerchantID != merchantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "核销令牌不属于当前商户"})
		return
	}
	if strings.TrimSpace(claims.AuthType) != authType {
		c.JSON(http.StatusForbidden, gin.H{"error": "核销令牌与当前账号不匹配"})
		return
	}
	if authType == "staff" && claims.TechnicianID != technicianID {
		c.JSON(http.StatusForbidden, gin.H{"error": "核销令牌与当前工作人员不匹配"})
		return
	}

	handCardNo := strings.TrimSpace(input.HandCardNo)
	var result verifyCommitResult

	err = config.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		var merchant models.Merchant
		var verifyCode models.VerifyCode
		var card models.Card
		if err := tx.First(&merchant, merchantID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apiErr{status: http.StatusNotFound, msg: "商户不存在"}
			}
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", claims.VerifyCodeID).First(&verifyCode).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apiErr{status: http.StatusNotFound, msg: "核销码不存在"}
			}
			return err
		}
		if verifyCode.CardID != claims.CardID {
			return apiErr{status: http.StatusBadRequest, msg: "核销令牌与核销码不匹配"}
		}
		if verifyCode.ProjectID != nil && claims.ProjectID > 0 && *verifyCode.ProjectID != claims.ProjectID {
			return apiErr{status: http.StatusBadRequest, msg: "核销令牌与项目不匹配"}
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&card, verifyCode.CardID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apiErr{status: http.StatusNotFound, msg: "卡片不存在"}
			}
			return err
		}
		if card.MerchantID != merchantID {
			return apiErr{status: http.StatusForbidden, msg: "无权核销此卡"}
		}
		if err := verifyHandCardUnreturnedAgeGuard(tx, merchantID, merchant.SupportHandCard, now); err != nil {
			return err
		}
		if verifyCode.Used {
			return apiErr{status: http.StatusBadRequest, msg: "核销码已使用"}
		}
		if now.Unix() > verifyCode.ExpireAt {
			return apiErr{status: http.StatusBadRequest, msg: "核销码已过期"}
		}
		if card.Locked {
			msg := "卡片已锁定"
			if strings.TrimSpace(card.LockedReason) != "" {
				msg = card.LockedReason
			}
			return apiErr{status: http.StatusBadRequest, msg: msg}
		}
		if card.EndDate != nil && now.After(*card.EndDate) {
			return apiErr{status: http.StatusBadRequest, msg: "卡片已过期"}
		}
		if card.RemainTimes <= 0 {
			return apiErr{status: http.StatusBadRequest, msg: "剩余次数不足"}
		}

		if merchant.SupportHandCard {
			if handCardNo == "" && !input.SkipHandCard {
				return apiErr{status: http.StatusBadRequest, msg: "请输入手牌号或选择跳过分配"}
			}
			if handCardNo != "" {
				var existing models.Usage
				if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
					Where("merchant_id = ? AND hand_card_no = ? AND hand_card_returned_at IS NULL", merchantID, handCardNo).
					Order("id desc").
					First(&existing).Error; err == nil {
					return apiErr{status: http.StatusBadRequest, msg: "该手牌已被占用，请更换手牌号"}
				} else if !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
			}
		} else if handCardNo != "" {
			return apiErr{status: http.StatusBadRequest, msg: "商户未开启手牌功能"}
		}

		result, err = performVerifyCommit(tx, c, merchant, verifyCode, card, handCardNo)
		return err
	})
	if err != nil {
		var ae apiErr
		if errors.As(err, &ae) {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := gin.H{
		"action":       result.Action,
		"card_id":      result.Card.ID,
		"usage_id":     result.UsageID,
		"remain_times": result.RemainTimes,
		"used_at":      result.UsedAt.Format("2006-01-02 15:04:05"),
		"session_id":   result.SessionID,
		"next_step":    result.NextStep,
	}
	c.JSON(http.StatusOK, gin.H{"message": "核销成功", "data": resp})
	enqueueVerifyUsageIfNeeded(result.Merchant, result.Card, result.UsageID, result.ShouldEnqueueOnsite)
}
