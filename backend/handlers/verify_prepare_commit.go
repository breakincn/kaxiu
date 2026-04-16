package handlers

import (
	"errors"
	"fmt"
	"kabao/config"
	"kabao/middleware"
	"kabao/models"
	"net/http"
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
	MerchantID    uint   `json:"merchant_id"`
	VerifyCodeID  uint   `json:"verify_code_id"`
	CardID        uint   `json:"card_id"`
	ProjectID     uint   `json:"project_id,omitempty"`
	AuthType      string `json:"auth_type"`
	TechnicianID  uint   `json:"technician_id,omitempty"`
	VerifyMode    string `json:"verify_mode,omitempty"`
	AppointmentID uint   `json:"appointment_id,omitempty"`
	jwt.RegisteredClaims
}

type verifyCommitResult struct {
	Merchant             models.Merchant
	Card                 models.Card
	UsedAt               time.Time
	RemainTimes          int
	SessionID            uint
	NextStep             string
	UsageID              uint
	ShouldEnqueueOnsite  bool
	Action               string
	AppointmentStatus    string
	PredictedWaitMinutes int
	SessionWaitState     string
	BoundTechnicianID    uint
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
	if err := loadVerifyMerchant(tx, merchantID, &merchant); err != nil {
		return merchant, models.VerifyCode{}, models.Card{}, nil, err
	}
	var verifyCode models.VerifyCode
	if err := loadVerifyCodeForUpdate(tx, code, &verifyCode); err != nil {
		return merchant, verifyCode, models.Card{}, nil, err
	}
	if verifyCode.Used {
		return merchant, verifyCode, models.Card{}, nil, apiErr{status: http.StatusBadRequest, msg: "核销码已使用"}
	}
	if now.Unix() > verifyCode.ExpireAt {
		return merchant, verifyCode, models.Card{}, nil, apiErr{status: http.StatusBadRequest, msg: "核销码已过期"}
	}

	var card models.Card
	if err := loadVerifyCardContext(tx, merchant, verifyCode, now, &card); err != nil {
		return merchant, verifyCode, card, nil, err
	}

	var project *models.MerchantProject
	if p, err := config.ResolveMerchantProject(tx, merchantID, verifyCode.ProjectID); err == nil {
		project = p
	}

	return merchant, verifyCode, card, project, nil
}

func loadVerifyMerchant(tx *gorm.DB, merchantID uint, merchant *models.Merchant) error {
	if err := tx.First(merchant, merchantID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apiErr{status: http.StatusNotFound, msg: "商户不存在"}
		}
		return err
	}
	return nil
}

func loadVerifyCodeForUpdate(tx *gorm.DB, code string, verifyCode *models.VerifyCode) error {
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("code = ?", code).First(verifyCode).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apiErr{status: http.StatusNotFound, msg: "核销码不存在"}
		}
		return err
	}
	return nil
}

func loadVerifyCardContext(tx *gorm.DB, merchant models.Merchant, verifyCode models.VerifyCode, now time.Time, card *models.Card) error {
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(card, verifyCode.CardID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apiErr{status: http.StatusNotFound, msg: "卡片不存在"}
		}
		return err
	}
	if card.MerchantID != merchant.ID {
		return apiErr{status: http.StatusForbidden, msg: "无权核销此卡"}
	}
	if err := verifyHandCardUnreturnedAgeGuard(tx, merchant.ID, merchant.SupportHandCard, now); err != nil {
		return err
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
	if card.RemainTimes <= 0 && !strings.HasPrefix(strings.TrimSpace(verifyCode.Code), "APPT-") {
		return apiErr{status: http.StatusBadRequest, msg: "剩余次数不足"}
	}
	return nil
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
	applyUsageCardSnapshotFromCard(&usage, card)
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

	session, nextStep, shouldEnqueueOnsite, err := createServiceSessionForUsage(tx, merchant, card, verifyCode, usage, now)
	if err != nil {
		return result, err
	}
	result.SessionID = session.ID
	result.NextStep = nextStep
	result.ShouldEnqueueOnsite = shouldEnqueueOnsite
	if session.TechnicianID != nil {
		result.BoundTechnicianID = *session.TechnicianID
	}
	if session.SourceType == serviceSessionSourceAppointment && session.SourceID != nil {
		arrivedAt := now
		appointmentStatus := "arrived"
		if err := tx.Model(&models.Appointment{}).
			Where("id = ? AND status IN ?", *session.SourceID, []string{"confirmed", "arrived"}).
			Updates(map[string]interface{}{
				"status":                  appointmentStatus,
				"arrived_at":              &arrivedAt,
				"actual_arrived_at":       &arrivedAt,
				"usage_id":                usage.ID,
				"service_session_id":      session.ID,
				"predicted_delay_minutes": session.PredictedAppointmentDelayMinutes,
			}).Error; err != nil {
			return result, err
		}
		if session.PredictedAppointmentDelayMinutes > 0 {
			loaded, err := loadAppointmentByID(tx, *session.SourceID)
			if err != nil {
				return result, err
			}
			if loaded == nil {
				return result, gorm.ErrRecordNotFound
			}
			if err := markAppointmentDelayPending(tx, loaded); err != nil {
				return result, err
			}
		}
		result.AppointmentStatus = appointmentStatus
		result.PredictedWaitMinutes = session.PredictedAppointmentDelayMinutes
		result.SessionWaitState = models.NormalizeSessionStatus(session.Status)
	}
	return result, nil
}

func performAppointmentCheckInWithVerifyCode(tx *gorm.DB, merchant models.Merchant, verifyCode models.VerifyCode, card models.Card, appointmentID uint, now time.Time) (verifyCommitResult, error) {
	result := verifyCommitResult{
		Merchant:    merchant,
		Card:        card,
		Action:      "appointment_checkin",
		RemainTimes: card.RemainTimes,
		UsedAt:      now,
	}
	if tx == nil {
		return result, nil
	}

	currentAppt, err := loadAppointmentByID(tx, appointmentID)
	if err != nil {
		return result, err
	}
	if currentAppt == nil {
		return result, gorm.ErrRecordNotFound
	}
	if currentAppt.CardID != card.ID || currentAppt.MerchantID != merchant.ID || currentAppt.UserID != card.UserID {
		return result, apiErr{status: http.StatusBadRequest, msg: "预约与签到码不匹配"}
	}
	currentStatus := normalizeAppointmentStatus(currentAppt.Status)
	if currentStatus != "confirmed" {
		return result, apiErr{status: http.StatusBadRequest, msg: "当前预约状态不可签到"}
	}
	if !canArriveForAppointment(*currentAppt, &merchant, now.In(appointmentLocation())) {
		return result, apiErr{status: http.StatusBadRequest, msg: "当前不在可签到时间窗口内"}
	}
	if currentAppt.ProjectID != nil && verifyCode.ProjectID != nil && *currentAppt.ProjectID != *verifyCode.ProjectID {
		return result, apiErr{status: http.StatusBadRequest, msg: "预约项目与签到码不匹配"}
	}

	res := tx.Model(&models.VerifyCode{}).
		Where("id = ? AND used = ?", verifyCode.ID, false).
		Updates(map[string]interface{}{"used": true, "used_at": now})
	if res.Error != nil {
		return result, res.Error
	}
	if res.RowsAffected == 0 {
		return result, apiErr{status: http.StatusBadRequest, msg: "签到码已使用"}
	}

	usage := models.Usage{
		CardID:             card.ID,
		MerchantID:         card.MerchantID,
		ProjectID:          currentAppt.ProjectID,
		UsedTimes:          1,
		UsedAt:             &now,
		VerifyCode:         verifyCode.Code,
		VerifyCodeExpireAt: verifyCode.ExpireAt,
		Status:             "in_progress",
	}
	applyUsageCardSnapshotFromCard(&usage, card)
	if err := tx.Create(&usage).Error; err != nil {
		return result, err
	}

	session, nextStep, shouldEnqueueOnsite, err := createServiceSessionForUsage(tx, merchant, card, verifyCode, usage, now)
	if err != nil {
		return result, err
	}

	actualArrivedAt := now
	if err := tx.Model(&models.Appointment{}).
		Where("id = ? AND status IN ?", currentAppt.ID, []string{"confirmed", "arrived"}).
		Updates(map[string]interface{}{
			"status":                  "arrived",
			"arrived_at":              &actualArrivedAt,
			"actual_arrived_at":       &actualArrivedAt,
			"usage_id":                usage.ID,
			"service_session_id":      session.ID,
			"predicted_delay_minutes": session.PredictedAppointmentDelayMinutes,
		}).Error; err != nil {
		return result, err
	}

	if session.PredictedAppointmentDelayMinutes > 0 {
		loaded, err := loadAppointmentByID(tx, currentAppt.ID)
		if err != nil {
			return result, err
		}
		if loaded == nil {
			return result, gorm.ErrRecordNotFound
		}
		if err := markAppointmentDelayPending(tx, loaded); err != nil {
			return result, err
		}
	}

	result.UsageID = usage.ID
	result.SessionID = session.ID
	result.NextStep = nextStep
	result.ShouldEnqueueOnsite = shouldEnqueueOnsite
	result.AppointmentStatus = "arrived"
	result.PredictedWaitMinutes = session.PredictedAppointmentDelayMinutes
	result.SessionWaitState = models.NormalizeSessionStatus(session.Status)
	if session.TechnicianID != nil {
		result.BoundTechnicianID = *session.TechnicianID
	}
	return result, nil
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
	verifyMode := "verify"
	var appointmentID uint

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		merchant, verifyCode, card, project, err = loadVerifyPrepareContext(tx, merchantID, code, now)
		if err != nil {
			return err
		}
		if strings.HasPrefix(strings.TrimSpace(verifyCode.Code), "APPT-") {
			source, err := detectServiceSessionSource(tx, merchant, card, now.In(appointmentLocation()))
			if err != nil {
				return err
			}
			if source.SourceType != serviceSessionSourceAppointment || source.SourceID == nil || *source.SourceID == 0 {
				return apiErr{status: http.StatusBadRequest, msg: "预约签到码当前不可用"}
			}
			verifyMode = "appointment_checkin"
			appointmentID = *source.SourceID
		}
		return nil
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
		MerchantID:    merchantID,
		VerifyCodeID:  verifyCode.ID,
		CardID:        card.ID,
		AuthType:      authType,
		TechnicianID:  technicianID,
		VerifyMode:    verifyMode,
		AppointmentID: appointmentID,
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
	out["verify_mode"] = verifyMode
	if appointmentID > 0 {
		out["appointment_id"] = appointmentID
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
		isAppointmentCheckIn := strings.TrimSpace(claims.VerifyMode) == "appointment_checkin"
		if card.RemainTimes <= 0 && !isAppointmentCheckIn {
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

		if isAppointmentCheckIn {
			if claims.AppointmentID == 0 {
				return apiErr{status: http.StatusBadRequest, msg: "预约签到令牌无效"}
			}
			result, err = performAppointmentCheckInWithVerifyCode(tx, merchant, verifyCode, card, claims.AppointmentID, now.In(appointmentLocation()))
			return err
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
		"action":                  result.Action,
		"card_id":                 result.Card.ID,
		"usage_id":                result.UsageID,
		"remain_times":            result.RemainTimes,
		"used_at":                 result.UsedAt.Format("2006-01-02 15:04:05"),
		"session_id":              result.SessionID,
		"next_step":               result.NextStep,
		"appointment_status":      result.AppointmentStatus,
		"predicted_delay_minutes": result.PredictedWaitMinutes,
		"session_wait_state":      result.SessionWaitState,
		"bound_technician_id":     result.BoundTechnicianID,
	}
	c.JSON(http.StatusOK, gin.H{"message": "核销成功", "data": resp})
	enqueueVerifyUsageIfNeeded(result.Merchant, result.Card, result.UsageID, result.ShouldEnqueueOnsite)
}
