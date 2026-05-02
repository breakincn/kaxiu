package handlers

import (
	"errors"
	"fmt"
	"kabao/config"
	"kabao/models"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MerchantRegister 商户注册
func MerchantRegister(c *gin.Context) {
	var input struct {
		Phone      string `json:"phone" binding:"required"`
		Password   string `json:"password" binding:"required,min=6"`
		Name       string `json:"name" binding:"required"`
		Type       string `json:"type"`
		Code       string `json:"code"`
		InviteCode string `json:"invite_code" binding:"required"`
		ReferralCode string `json:"referral_code"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查手机号是否已注册
	var existingMerchant models.Merchant
	if err := config.DB.Where("phone = ?", input.Phone).First(&existingMerchant).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该手机号已注册"})
		return
	}

	var merchant models.Merchant
	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		if !config.MerchantRegisterSMSVerificationDisabled() {
			if strings.TrimSpace(input.Code) == "" {
				return errors.New("请输入验证码")
			}
			if err := consumeSMSCode(tx, input.Phone, "merchant_register", input.Code); err != nil {
				return err
			}
		}

		var invite models.InviteCode
		if err := tx.Where("code = ? AND used = ?", input.InviteCode, false).First(&invite).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("邀请码无效或已使用")
			}
			return err
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		merchant = models.Merchant{
			Phone:     input.Phone,
			Password:  string(hashedPassword),
			Name:      input.Name,
			Type:      input.Type,
			CreatedAt: func() *time.Time { t := time.Now(); return &t }(),
		}
		if err := tx.Create(&merchant).Error; err != nil {
			return err
		}

		usedAt := time.Now()
		updates := map[string]interface{}{
			"used":                true,
			"used_at":             &usedAt,
			"used_by_merchant_id": merchant.ID,
		}
		res := tx.Model(&models.InviteCode{}).
			Where("code = ? AND used = ?", input.InviteCode, false).
			Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return errors.New("邀请码无效或已使用")
		}
		if err := bindMerchantReferralByCode(tx, input.ReferralCode, merchant.ID, usedAt); err != nil {
			return err
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("商户注册成功: ID=%d, 手机号=%s, 名称=%s", merchant.ID, merchant.Phone, merchant.Name)

	c.JSON(http.StatusOK, gin.H{
		"message": "注册成功",
		"data": gin.H{
			"id":    merchant.ID,
			"phone": merchant.Phone,
			"name":  merchant.Name,
		},
	})
}

// MerchantLogin 商户登录
func MerchantLogin(c *gin.Context) {
	var input struct {
		Phone    string `json:"phone" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	account := strings.TrimSpace(input.Phone)
	password := strings.TrimSpace(input.Password)
	rlKey, ok := enforceLoginRateLimit(c, "merchant", account)
	if !ok {
		return
	}
	if strings.HasPrefix(strings.ToLower(account), "js") {
		recordLoginFailure(rlKey)
		c.JSON(http.StatusBadRequest, gin.H{"error": "技师账号请使用店铺登录地址 /s/:slug/login"})
		return
	}

	var merchant models.Merchant
	if err := config.DB.Where("phone = ?", account).First(&merchant).Error; err != nil {
		recordLoginFailure(rlKey)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "手机号或密码错误"})
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(merchant.Password), []byte(password)); err != nil {
		recordLoginFailure(rlKey)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "手机号或密码错误"})
		return
	}

	// 生成 JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"merchant_id": merchant.ID,
		"phone":       merchant.Phone,
		"type":        "merchant",
		"exp":         time.Now().Add(time.Hour * 24 * 7).Unix(),
	})

	secret := config.JWTSecret()
	if secret == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "登录服务未配置"})
		return
	}
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		log.Printf("生成token失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "登录失败"})
		return
	}
	recordLoginSuccess(rlKey)

	log.Printf("商户登录成功: ID=%d, 手机号=%s", merchant.ID, merchant.Phone)

	c.JSON(http.StatusOK, gin.H{
		"token": tokenString,
		"merchant": gin.H{
			"id":    merchant.ID,
			"phone": merchant.Phone,
			"name":  merchant.Name,
			"type":  merchant.Type,
		},
	})
}

func GetMerchants(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}
	if !requireMerchantPermissionInHandler(c, "merchant.info.manage") {
		return
	}

	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": []models.Merchant{merchant}})
}

func GetMerchant(c *gin.Context) {
	_, ok := ensureMerchantScope(c, "id")
	if !ok {
		return
	}
	id := c.Param("id")
	var merchant models.Merchant
	if err := config.DB.First(&merchant, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": merchant})
}

func migrateSessionsAfterDisableCustomerService(tx *gorm.DB, m *models.Merchant, now time.Time) error {
	if tx == nil || m == nil || m.ID == 0 {
		return nil
	}

	// 迁移目标：
	// - 若已锁定房间（room_id 有值）或商户不支持房间：进入 delay_pending（非客服流程的“待起单”）
	// - 若未锁定房间且支持房间：回到 room_selecting 让用户选房（不再选客服）
	statuses := models.ExpandStatusesWithKnownPrefixes([]string{"room_locked", "staff_selecting", "start_pending"})

	var sessions []models.ServiceSession
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("merchant_id = ? AND status IN ? AND start_confirmed_at IS NULL", m.ID, statuses).
		Find(&sessions).Error; err != nil {
		return err
	}
	if len(sessions) == 0 {
		return nil
	}
	log.Printf("[merchant-config] migrate sessions after disabling customer service mode: merchant=%d affected_sessions=%d", m.ID, len(sessions))

	delaySeconds := m.StartDelaySeconds
	if delaySeconds <= 0 {
		delaySeconds = 60
	}
	startAt := now.Add(time.Duration(delaySeconds) * time.Second)

	for i := range sessions {
		s := sessions[i]

		// 释放技师占用（如果有）
		for _, techID := range serviceSessionPrimaryTechnicianIDs(&s) {
			_ = tx.Model(&models.TechnicianAttendance{}).
				Where("merchant_id = ? AND technician_id = ? AND status = ?", s.MerchantID, techID, "busy").
				Updates(map[string]interface{}{"status": "idle"}).Error
		}
		updates := map[string]interface{}{
			"last_technician_id":            nil,
			"service_technician_ids":        models.MerchantProjectDefaultServiceTechnicianIDs{},
			"start_confirmed_technician_ids": models.MerchantProjectDefaultServiceTechnicianIDs{},
			"staff_select_cooldown_until":   nil,
			"staff_select_entered_at":       nil,
			"start_pending_timeout_seconds": 0,
		}

		// 已锁定房间：直接进入非客服流程 delay_pending
		if (s.RoomID != nil && *s.RoomID > 0) || !m.SupportRoom {
			updates["status"] = models.ApplyStatusPrefix(s.Status, "delay_pending")
			updates["start_confirmed_at"] = &now
			updates["scheduled_start_at"] = &startAt
			if m.SupportRoom {
				// 保留 room_id，但房间锁定时间不再需要
				updates["room_locked_at"] = nil
			}
		} else {
			// 未锁定房间：回到选房（不再选客服）
			dl := now.Add(time.Duration(config.GetProjectRoomSelectTimeoutSeconds(tx, m.ID, s.ProjectID)) * time.Second)
			updates["status"] = models.ApplyStatusPrefix(s.Status, "room_selecting")
			updates["room_id"] = nil
			updates["room_locked_at"] = nil
			updates["room_select_deadline_at"] = &dl
		}

		if err := tx.Model(&models.ServiceSession{}).
			Where("id = ? AND status IN ? AND start_confirmed_at IS NULL", s.ID, statuses).
			Updates(updates).Error; err != nil {
			return err
		}
	}
	return nil
}

func UpdateCurrentMerchantServices(c *gin.Context) {
	merchantIDAny, exists := c.Get("merchant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	merchantID, ok := merchantIDAny.(uint)
	if !ok || merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}
	oldSupportCustomerServiceMode := merchant.SupportCustomerServiceMode

	var input struct {
		SupportAppointment                              *bool   `json:"support_appointment"`
		SupportQueue                                    *bool   `json:"support_queue"`
		SupportProject                                  *bool   `json:"support_project"`
		SupportRoom                                     *bool   `json:"support_room"`
		SupportTechnicianCheckin                        *bool   `json:"support_technician_checkin"`
		SupportDirectSale                               *bool   `json:"support_direct_sale"`
		SupportCustomerService                          *bool   `json:"support_customer_service"`
		SupportCustomerServiceMode                      *bool   `json:"support_customer_service_mode"`
		SupportMultiCustomerService                     *bool   `json:"support_multi_customer_service"`
		SupportOrderComplete                            *bool   `json:"support_order_complete"`
		StartDelaySeconds                               *int    `json:"start_delay_seconds"`
		SupportHandCard                                 *bool   `json:"support_hand_card"`
		QueuePrefix                                     *string `json:"queue_prefix"`
		QueueStartNo                                    *int    `json:"queue_start_no"`
		QueueMode                                       *string `json:"queue_mode"`
		QueueWindowTerm                                 *string `json:"queue_window_term"`
		QueueWaitingStartSeconds                        *int    `json:"queue_waiting_start_seconds"`
		QueueTimeoutWaitingSeconds                      *int    `json:"queue_timeout_waiting_seconds"`
		AppointmentReserveBufferMinutes                 *int    `json:"appointment_reserve_buffer_minutes"`
		AppointmentGraceWindowMinutes                   *int    `json:"appointment_grace_window_minutes"`
		AppointmentPredictionBufferMinute               *int    `json:"appointment_prediction_buffer_minutes"`
		AppointmentSlotGranularityMinutes               *int    `json:"appointment_slot_granularity_minutes"`
		AppointmentSchedulingMode                       *string `json:"appointment_scheduling_mode"`
		AppointmentRescheduleDeadlineMinutesBeforeStart *int    `json:"appointment_reschedule_deadline_minutes_before_start"`
		AppointmentRescheduleRecommendationEnabled      *bool   `json:"appointment_reschedule_recommendation_enabled"`
		HandCardPrefix                                  *string `json:"hand_card_prefix"`
		HandCardStartNo                                 *int    `json:"hand_card_start_no"`
		HandCardEndNo                                   *int    `json:"hand_card_end_no"`
		RoomNumberCardPrefix                            *string `json:"room_number_card_prefix"`
		RoomNumberCardStartNo                           *int    `json:"room_number_card_start_no"`
		RoomNumberCardEndNo                             *int    `json:"room_number_card_end_no"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 配置互斥：客服模式 与 叫号模式 不允许同时开启。
	// 说明：运行时 ResolveSessionMode 会以客服模式优先，但允许冲突值落库会增加排障成本。
	{
		targetSupportQueue := merchant.SupportQueue
		if input.SupportQueue != nil {
			targetSupportQueue = *input.SupportQueue
		}
		targetSupportCustomerServiceMode := merchant.SupportCustomerServiceMode
		if input.SupportCustomerServiceMode != nil {
			targetSupportCustomerServiceMode = *input.SupportCustomerServiceMode
		}
		if targetSupportQueue && targetSupportCustomerServiceMode {
			c.JSON(http.StatusBadRequest, gin.H{"error": "客服模式与叫号模式互斥，请先关闭其中一个"})
			return
		}
	}

	// 叫号模式切换保护：当存在未完成的服务会话时，不允许切换 queue_mode /
	// support_multi_customer_service，也不允许直接关闭 support_queue 或切到
	// support_customer_service_mode。
	// 说明：已核销但仍处于待选房/待选客服/过号等待等状态的会话，当前不会被安全迁移到新的
	// 运行模式；放行切换会导致旧会话被新配置以错误语义继续推进。
	{
		targetSupportQueue := merchant.SupportQueue
		if input.SupportQueue != nil {
			targetSupportQueue = *input.SupportQueue
		}
		targetQueueMode := strings.TrimSpace(merchant.QueueMode)
		if input.QueueMode != nil {
			targetQueueMode = strings.TrimSpace(*input.QueueMode)
		}
		targetSupportMulti := merchant.SupportMultiCustomerService
		if input.SupportMultiCustomerService != nil {
			targetSupportMulti = *input.SupportMultiCustomerService
		}
		targetSupportCustomerServiceMode := merchant.SupportCustomerServiceMode
		if input.SupportCustomerServiceMode != nil {
			targetSupportCustomerServiceMode = *input.SupportCustomerServiceMode
		}

		queueModeChanged := input.QueueMode != nil && strings.TrimSpace(merchant.QueueMode) != targetQueueMode
		multiChanged := input.SupportMultiCustomerService != nil && merchant.SupportMultiCustomerService != targetSupportMulti
		queueSupportChanged := input.SupportQueue != nil && merchant.SupportQueue != targetSupportQueue
		customerServiceModeChanged := input.SupportCustomerServiceMode != nil && merchant.SupportCustomerServiceMode != targetSupportCustomerServiceMode
		affectsQueueRuntime := queueModeChanged || multiChanged ||
			(queueSupportChanged && (merchant.SupportQueue || targetSupportQueue)) ||
			(customerServiceModeChanged && (merchant.SupportQueue || targetSupportQueue))
		if affectsQueueRuntime {
			activeStatuses := []string{
				"room_selecting",
				"room_locked",
				"staff_selecting",
				"start_pending",
				"delay_pending",
				"timeout_waiting",
				"serving",
				"auto_finishing",
			}
			var cnt int64
			if err := config.DB.Model(&models.ServiceSession{}).
				Where("merchant_id = ? AND status IN ?", merchantID, models.ExpandStatusesWithKnownPrefixes(activeStatuses)).
				Count(&cnt).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
				return
			}
			if cnt > 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "当前有未完成服务会话，请等待本轮服务全部完成后再切换"})
				return
			}
		}
	}

	if input.SupportAppointment != nil && *input.SupportAppointment {
		allDayStart := strings.TrimSpace(merchant.AllDayStart)
		allDayEnd := strings.TrimSpace(merchant.AllDayEnd)
		morningStart := strings.TrimSpace(merchant.MorningStart)
		morningEnd := strings.TrimSpace(merchant.MorningEnd)
		afternoonStart := strings.TrimSpace(merchant.AfternoonStart)
		afternoonEnd := strings.TrimSpace(merchant.AfternoonEnd)
		eveningStart := strings.TrimSpace(merchant.EveningStart)
		eveningEnd := strings.TrimSpace(merchant.EveningEnd)

		hasBusinessTime := false
		if allDayStart != "" && allDayEnd != "" {
			hasBusinessTime = true
		} else if (morningStart != "" && morningEnd != "") || (afternoonStart != "" && afternoonEnd != "") || (eveningStart != "" && eveningEnd != "") {
			hasBusinessTime = true
		}

		if !hasBusinessTime {
			c.JSON(http.StatusBadRequest, gin.H{"error": "先设置营业时间，才能开启预约服务"})
			return
		}
	}

	updates := make(map[string]interface{})
	// 本次更新后的目标客服开关，用于校验 support_customer_service_mode
	targetSupportCustomerService := merchant.SupportCustomerService
	if input.SupportCustomerService != nil {
		targetSupportCustomerService = *input.SupportCustomerService
	}

	// 客服模式需要先开启客服
	if input.SupportCustomerServiceMode != nil && *input.SupportCustomerServiceMode {
		if !targetSupportCustomerService {
			c.JSON(http.StatusBadRequest, gin.H{"error": "开启客服模式前，请先开启\"开启客服\""})
			return
		}
	}

	// 关闭客服时，同步关闭客服模式
	if input.SupportCustomerService != nil && !*input.SupportCustomerService {
		if merchant.SupportCustomerServiceMode {
			updates["support_customer_service_mode"] = false
		}
	}
	if input.SupportAppointment != nil {
		updates["support_appointment"] = *input.SupportAppointment
	}
	if input.SupportQueue != nil {
		updates["support_queue"] = *input.SupportQueue
	}
	if input.SupportProject != nil {
		updates["support_project"] = *input.SupportProject
	}
	if input.SupportRoom != nil {
		updates["support_room"] = *input.SupportRoom
	}
	if input.SupportTechnicianCheckin != nil {
		updates["support_technician_checkin"] = *input.SupportTechnicianCheckin
	}
	if input.SupportDirectSale != nil {
		updates["support_direct_sale"] = *input.SupportDirectSale
	}
	if input.SupportCustomerService != nil {
		updates["support_customer_service"] = *input.SupportCustomerService
	}
	if input.SupportCustomerServiceMode != nil {
		updates["support_customer_service_mode"] = *input.SupportCustomerServiceMode
	}
	if input.SupportMultiCustomerService != nil {
		updates["support_multi_customer_service"] = *input.SupportMultiCustomerService
	}
	if input.SupportOrderComplete != nil {
		updates["support_order_complete"] = *input.SupportOrderComplete
		// 仅在本次请求真实发生 true->false 时才允许联动降级，
		// 避免手牌等无关开关保存时（前端会回传 support_order_complete 当前值）误改 queue_mode。
		if merchant.SupportOrderComplete && !*input.SupportOrderComplete && input.QueueMode == nil {
			if strings.TrimSpace(merchant.QueueMode) == "auto" {
				updates["queue_mode"] = "manual"
			}
		}
	}
	if input.StartDelaySeconds != nil {
		if *input.StartDelaySeconds < 0 || *input.StartDelaySeconds > 3600 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "start_delay_seconds 范围应为 0-3600"})
			return
		}
		updates["start_delay_seconds"] = *input.StartDelaySeconds
	}
	if input.SupportHandCard != nil {
		updates["support_hand_card"] = *input.SupportHandCard
	}
	if input.QueuePrefix != nil {
		updates["queue_prefix"] = strings.TrimSpace(*input.QueuePrefix)
	}
	if input.QueueStartNo != nil {
		if *input.QueueStartNo < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "叫号起始号码必须大于等于1"})
			return
		}
		updates["queue_start_no"] = *input.QueueStartNo
	}
	if input.QueueMode != nil {
		mode := strings.TrimSpace(*input.QueueMode)
		if mode != "auto" && mode != "manual" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "叫号方式必须是自动(auto)或人工(manual)"})
			return
		}
		updates["queue_mode"] = mode
	}
	if input.QueueWindowTerm != nil {
		term := strings.TrimSpace(*input.QueueWindowTerm)
		if term == "" {
			term = "窗口"
		}
		updates["queue_window_term"] = term
	}
	if input.QueueWaitingStartSeconds != nil {
		if *input.QueueWaitingStartSeconds < 1 || *input.QueueWaitingStartSeconds > 3600 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "等待上号时间范围应为 1-3600 秒"})
			return
		}
		updates["queue_waiting_start_seconds"] = *input.QueueWaitingStartSeconds
	}
	if input.QueueTimeoutWaitingSeconds != nil {
		if *input.QueueTimeoutWaitingSeconds < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "超时过号等待时间必须大于 0 秒"})
			return
		}
		updates["queue_timeout_waiting_seconds"] = *input.QueueTimeoutWaitingSeconds
	}
	if input.AppointmentReserveBufferMinutes != nil {
		if *input.AppointmentReserveBufferMinutes < 0 || *input.AppointmentReserveBufferMinutes > 120 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "预约前保留缓冲范围应为 0-120 分钟"})
			return
		}
		updates["appointment_reserve_buffer_minutes"] = *input.AppointmentReserveBufferMinutes
	}
	if input.AppointmentGraceWindowMinutes != nil {
		if *input.AppointmentGraceWindowMinutes < 0 || *input.AppointmentGraceWindowMinutes > 180 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "预约后宽限范围应为 0-180 分钟"})
			return
		}
		updates["appointment_grace_window_minutes"] = *input.AppointmentGraceWindowMinutes
	}
	if input.AppointmentPredictionBufferMinute != nil {
		if *input.AppointmentPredictionBufferMinute < 0 || *input.AppointmentPredictionBufferMinute > 60 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "预约预测缓冲范围应为 0-60 分钟"})
			return
		}
		updates["appointment_prediction_buffer_minutes"] = *input.AppointmentPredictionBufferMinute
	}
	if input.AppointmentSlotGranularityMinutes != nil {
		if *input.AppointmentSlotGranularityMinutes < 5 || *input.AppointmentSlotGranularityMinutes > 60 || *input.AppointmentSlotGranularityMinutes%5 != 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "预约时段粒度范围应为 5-60 分钟，且需为 5 的倍数"})
			return
		}
		updates["appointment_slot_granularity_minutes"] = *input.AppointmentSlotGranularityMinutes
	}
	if input.AppointmentSchedulingMode != nil {
		mode := strings.TrimSpace(*input.AppointmentSchedulingMode)
		if mode == "" {
			mode = "technician_grouped"
		}
		if mode != "technician_grouped" && mode != "technician_mixed_timeline" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "预约排布模式必须是 technician_grouped 或 technician_mixed_timeline"})
			return
		}
		targetSupportCustomerServiceMode := merchant.SupportCustomerServiceMode
		if input.SupportCustomerServiceMode != nil {
			targetSupportCustomerServiceMode = *input.SupportCustomerServiceMode
		}
		if mode == "technician_mixed_timeline" && !targetSupportCustomerServiceMode {
			c.JSON(http.StatusBadRequest, gin.H{"error": "仅客服模式下可开启 technician_mixed_timeline"})
			return
		}
		updates["appointment_scheduling_mode"] = mode
	}
	if input.AppointmentRescheduleDeadlineMinutesBeforeStart != nil {
		if *input.AppointmentRescheduleDeadlineMinutesBeforeStart < 1 || *input.AppointmentRescheduleDeadlineMinutesBeforeStart > 1440 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "预约改签截止时间范围应为 1-1440 分钟"})
			return
		}
		updates["appointment_reschedule_deadline_minutes_before_start"] = *input.AppointmentRescheduleDeadlineMinutesBeforeStart
	}
	if input.AppointmentRescheduleRecommendationEnabled != nil {
		updates["appointment_reschedule_recommendation_enabled"] = *input.AppointmentRescheduleRecommendationEnabled
	}
	if input.HandCardPrefix != nil {
		updates["hand_card_prefix"] = strings.TrimSpace(*input.HandCardPrefix)
	}
	if input.HandCardStartNo != nil {
		if *input.HandCardStartNo < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "手牌起始号码必须大于等于1"})
			return
		}
		updates["hand_card_start_no"] = *input.HandCardStartNo
	}
	if input.HandCardEndNo != nil {
		if *input.HandCardEndNo < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "手牌结束号码必须大于等于1"})
			return
		}
		updates["hand_card_end_no"] = *input.HandCardEndNo
	}
	if input.RoomNumberCardPrefix != nil {
		updates["room_number_card_prefix"] = strings.TrimSpace(*input.RoomNumberCardPrefix)
	}
	if input.RoomNumberCardStartNo != nil {
		if *input.RoomNumberCardStartNo < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "房间号牌起始号码必须大于等于1"})
			return
		}
		updates["room_number_card_start_no"] = *input.RoomNumberCardStartNo
	}
	if input.RoomNumberCardEndNo != nil {
		if *input.RoomNumberCardEndNo < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "房间号牌结束号码必须大于等于1"})
			return
		}
		updates["room_number_card_end_no"] = *input.RoomNumberCardEndNo
	}

	if len(updates) == 0 {
		config.DB.First(&merchant, merchantID)
		c.JSON(http.StatusOK, gin.H{"data": merchant})
		return
	}

	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&merchant).Updates(updates).Error; err != nil {
			return err
		}
		if err := tx.First(&merchant, merchantID).Error; err != nil {
			return err
		}
		// 仅在 support_customer_service_mode 真正由 true -> false 时迁移会话，
		// 避免手牌等无关开关更新触发跨流程迁移。
		if oldSupportCustomerServiceMode && !merchant.SupportCustomerServiceMode {
			return migrateSessionsAfterDisableCustomerService(tx, &merchant, time.Now())
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	// 如果开启了房间功能，且房间号牌设置有变化，自动创建房间
	if merchant.SupportRoom {
		if err := createOrUpdateRoomsForMerchant(merchant.ID, merchant.RoomNumberCardPrefix, merchant.RoomNumberCardStartNo, merchant.RoomNumberCardEndNo); err != nil {
			// 不影响主流程，只记录错误
			fmt.Printf("自动创建房间失败: %v\n", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": merchant})
}

func CreateMerchant(c *gin.Context) {
	var merchant models.Merchant
	if err := c.ShouldBindJSON(&merchant); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config.DB.Create(&merchant)
	c.JSON(http.StatusOK, gin.H{"data": merchant})
}

func UpdateMerchant(c *gin.Context) {
	if _, hasMerchant := c.Get("merchant_id"); hasMerchant {
		if _, ok := ensureMerchantScope(c, "id"); !ok {
			return
		}
	}
	id := c.Param("id")
	var merchant models.Merchant
	if err := config.DB.First(&merchant, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}

	var input struct {
		Name               string `json:"name"`
		Type               string `json:"type"`
		SupportAppointment *bool  `json:"support_appointment"`
		ShowProvince       *bool  `json:"show_province"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := make(map[string]interface{})
	if input.Name != "" {
		updates["name"] = input.Name
	}
	if input.Type != "" {
		updates["type"] = input.Type
	}
	if input.SupportAppointment != nil {
		updates["support_appointment"] = *input.SupportAppointment
	}
	if input.ShowProvince != nil {
		updates["show_province"] = *input.ShowProvince
	}
	if len(updates) > 0 {
		if err := config.DB.Model(&merchant).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
			return
		}
	}
	if err := config.DB.First(&merchant, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取商户信息失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": merchant})
}

func GetCurrentUserMerchant(c *gin.Context) {
	merchantID, exists := c.Get("merchant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": merchant})
}

// ToggleMerchantBusinessStatus 切换商户营业状态
func ToggleMerchantBusinessStatus(c *gin.Context) {
	merchantIDAny, exists := c.Get("merchant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	merchantID, ok := merchantIDAny.(uint)
	if !ok || merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}

	var input struct {
		IsOpen bool `json:"is_open"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 使用 Select 方法确保布尔值 false 也能被更新
	if err := config.DB.Model(&merchant).Select("is_open").Updates(map[string]interface{}{"is_open": input.IsOpen}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}
	config.DB.First(&merchant, merchantID)
	c.JSON(http.StatusOK, gin.H{"data": merchant})
}

func BindMerchantPhone(c *gin.Context) {
	merchantIDAny, exists := c.Get("merchant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var input struct {
		Phone    string `json:"phone" binding:"required"`
		Code     string `json:"code" binding:"required"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.Phone = strings.TrimSpace(input.Phone)
	input.Code = strings.TrimSpace(input.Code)
	input.Password = strings.TrimSpace(input.Password)
	if input.Phone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供手机号"})
		return
	}
	if input.Code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入验证码"})
		return
	}

	merchantID, _ := merchantIDAny.(uint)
	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}

	// 如果商户已有手机号（换绑场景），需要验证密码
	if merchant.Phone != "" && merchant.Phone != input.Phone {
		if input.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "换绑手机号需要输入商户密码"})
			return
		}

		// 验证密码
		if err := bcrypt.CompareHashAndPassword([]byte(merchant.Password), []byte(input.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
			return
		}
	}

	// 检查新手机号是否已被其他商户使用
	var existingByPhone models.Merchant
	if err := config.DB.Where("phone = ? AND id != ?", input.Phone, merchantID).First(&existingByPhone).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该手机号已被其他商户绑定"})
		return
	}

	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := consumeSMSCode(tx, input.Phone, "merchant_bind_phone", input.Code); err != nil {
			return err
		}
		return tx.Model(&models.Merchant{}).Where("id = ?", merchantID).Update("phone", input.Phone).Error
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var updated models.Merchant
	if err := config.DB.First(&updated, merchantID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "绑定失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": updated})
}

// UpdateMerchantInfo 更新商户信息（营业时间和地址）
func UpdateMerchantInfo(c *gin.Context) {
	merchantIDAny, exists := c.Get("merchant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	merchantID, ok := merchantIDAny.(uint)
	if !ok || merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}

	var input struct {
		MorningStart   *string `json:"morning_start"`
		MorningEnd     *string `json:"morning_end"`
		AfternoonStart *string `json:"afternoon_start"`
		AfternoonEnd   *string `json:"afternoon_end"`
		EveningStart   *string `json:"evening_start"`
		EveningEnd     *string `json:"evening_end"`
		AllDayStart    *string `json:"all_day_start"`
		AllDayEnd      *string `json:"all_day_end"`
		Province       *string `json:"province"`
		City           *string `json:"city"`
		District       *string `json:"district"`
		Address        *string `json:"address"`
		ShowProvince   *bool   `json:"show_province"`
		StartTerm      *string `json:"start_term"`
		FinishTerm     *string `json:"finish_term"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := make(map[string]interface{})
	if input.MorningStart != nil {
		updates["morning_start"] = strings.TrimSpace(*input.MorningStart)
	}
	if input.MorningEnd != nil {
		updates["morning_end"] = strings.TrimSpace(*input.MorningEnd)
	}
	if input.AfternoonStart != nil {
		updates["afternoon_start"] = strings.TrimSpace(*input.AfternoonStart)
	}
	if input.AfternoonEnd != nil {
		updates["afternoon_end"] = strings.TrimSpace(*input.AfternoonEnd)
	}
	if input.EveningStart != nil {
		updates["evening_start"] = strings.TrimSpace(*input.EveningStart)
	}
	if input.EveningEnd != nil {
		updates["evening_end"] = strings.TrimSpace(*input.EveningEnd)
	}
	if input.AllDayStart != nil {
		updates["all_day_start"] = strings.TrimSpace(*input.AllDayStart)
	}
	if input.AllDayEnd != nil {
		updates["all_day_end"] = strings.TrimSpace(*input.AllDayEnd)
	}
	if input.Province != nil {
		updates["province"] = strings.TrimSpace(*input.Province)
	}
	if input.City != nil {
		updates["city"] = strings.TrimSpace(*input.City)
	}
	if input.District != nil {
		updates["district"] = strings.TrimSpace(*input.District)
	}
	if input.Address != nil {
		updates["address"] = strings.TrimSpace(*input.Address)
	}
	if input.ShowProvince != nil {
		updates["show_province"] = *input.ShowProvince
	}
	if input.StartTerm != nil {
		updates["start_term"] = strings.TrimSpace(*input.StartTerm)
	}
	if input.FinishTerm != nil {
		updates["finish_term"] = strings.TrimSpace(*input.FinishTerm)
	}

	if len(updates) == 0 {
		config.DB.First(&merchant, merchantID)
		c.JSON(http.StatusOK, gin.H{"data": merchant})
		return
	}

	if err := config.DB.Model(&merchant).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}
	config.DB.First(&merchant, merchantID)
	c.JSON(http.StatusOK, gin.H{"data": merchant})
}

func UpdateTechnicianAlias(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}
	if !requireMerchantPermissionInHandler(c, "merchant.info.manage") {
		return
	}

	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}

	var input struct {
		TechnicianAlias string `json:"technician_alias" binding:"required,max=20"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 清理输入的称谓
	alias := strings.TrimSpace(input.TechnicianAlias)
	if alias == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "技师称谓不能为空"})
		return
	}

	// 更新技师称谓
	if err := config.DB.Model(&merchant).Update("technician_alias", alias).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	// 返回更新后的商户信息
	config.DB.First(&merchant, merchantID)
	c.JSON(http.StatusOK, gin.H{"data": merchant})
}

// createOrUpdateRoomsForMerchant 根据商户的房间号牌设置自动创建或更新房间
func createOrUpdateRoomsForMerchant(merchantID uint, prefix string, startNo, endNo int) error {
	// 参数验证
	if startNo < 1 || endNo < 1 || startNo > endNo {
		return fmt.Errorf("房间号牌参数无效: startNo=%d, endNo=%d", startNo, endNo)
	}

	// 获取现有房间
	var existingRooms []models.Room
	if err := config.DB.Where("merchant_id = ?", merchantID).Order("id asc").Find(&existingRooms).Error; err != nil {
		return fmt.Errorf("查询现有房间失败: %v", err)
	}

	// 创建房间名称映射
	existingRoomNames := make(map[string]bool)
	for _, room := range existingRooms {
		existingRoomNames[room.Name] = true
	}

	// 生成目标房间列表
	targetRooms := make([]string, 0, endNo-startNo+1)
	for i := startNo; i <= endNo; i++ {
		roomName := fmt.Sprintf("%s%d", prefix, i)
		targetRooms = append(targetRooms, roomName)
	}

	// 删除不再需要的房间
	roomsToDelete := make([]uint, 0)
	for _, room := range existingRooms {
		// 检查房间是否在目标列表中
		found := false
		for _, targetName := range targetRooms {
			if room.Name == targetName {
				found = true
				break
			}
		}
		if !found {
			// 检查房间是否正在使用
			var activeSessionCount int64
			activeStatuses := models.ExpandStatusesWithKnownPrefixes([]string{"room_locked", "staff_selecting", "start_pending", "delay_pending", "serving", "auto_finishing"})
			config.DB.Model(&models.ServiceSession{}).
				Where("merchant_id = ? AND room_id = ? AND status IN ?", merchantID, room.ID, activeStatuses).
				Count(&activeSessionCount)

			if activeSessionCount == 0 {
				roomsToDelete = append(roomsToDelete, room.ID)
			}
		}
	}

	// 批量删除不需要的房间
	if len(roomsToDelete) > 0 {
		if err := config.DB.Where("id IN ?", roomsToDelete).Delete(&models.Room{}).Error; err != nil {
			return fmt.Errorf("删除房间失败: %v", err)
		}
	}

	// 创建新房间
	for _, roomName := range targetRooms {
		if !existingRoomNames[roomName] {
			newRoom := models.Room{
				MerchantID: merchantID,
				Name:       roomName,
				IsActive:   true,
			}
			if err := config.DB.Create(&newRoom).Error; err != nil {
				return fmt.Errorf("创建房间 %s 失败: %v", roomName, err)
			}
		}
	}

	return nil
}
