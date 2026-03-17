package handlers

import (
	"errors"
	"fmt"
	"kabao/config"
	"kabao/models"
	"kabao/queue"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func parseHHMMToMinuteOfDay(s string) (int, bool) {
	ss := strings.TrimSpace(s)
	if ss == "" {
		return 0, false
	}
	parts := strings.Split(ss, ":")
	if len(parts) != 2 {
		return 0, false
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return 0, false
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}

type businessInterval struct {
	Start time.Time
	End   time.Time
}

func getMerchantBusinessIntervalsForDate(merchant models.Merchant, date time.Time) ([]businessInterval, bool) {
	intervals := make([]businessInterval, 0, 3)

	add := func(startStr, endStr string) {
		sm, ok1 := parseHHMMToMinuteOfDay(startStr)
		em, ok2 := parseHHMMToMinuteOfDay(endStr)
		if !ok1 || !ok2 {
			return
		}
		// 支持跨天：如 10:00 - 00:00 视为 10:00 - 24:00
		if em == 0 {
			em = 24 * 60
		}
		if em <= sm {
			return
		}
		start := time.Date(date.Year(), date.Month(), date.Day(), sm/60, sm%60, 0, 0, date.Location())
		end := time.Date(date.Year(), date.Month(), date.Day(), em/60, em%60, 0, 0, date.Location())
		if !end.After(start) {
			return
		}
		intervals = append(intervals, businessInterval{Start: start, End: end})
	}

	allDayStart := strings.TrimSpace(merchant.AllDayStart)
	allDayEnd := strings.TrimSpace(merchant.AllDayEnd)
	if allDayStart != "" && allDayEnd != "" {
		add(allDayStart, allDayEnd)
		return intervals, len(intervals) > 0
	}

	add(merchant.MorningStart, merchant.MorningEnd)
	add(merchant.AfternoonStart, merchant.AfternoonEnd)
	add(merchant.EveningStart, merchant.EveningEnd)

	return intervals, len(intervals) > 0
}

func isWithinBusinessIntervals(t time.Time, intervals []businessInterval) bool {
	for _, it := range intervals {
		if (t.Equal(it.Start) || t.After(it.Start)) && t.Before(it.End) {
			return true
		}
	}
	return false
}

func getAppointmentServiceMinutes(merchantID uint, appt models.Appointment) int {
	if appt.ProjectID != nil && *appt.ProjectID > 0 {
		var p models.MerchantProject
		if err := config.DB.Where("id = ? AND merchant_id = ? AND is_active = ?", *appt.ProjectID, merchantID, true).First(&p).Error; err == nil {
			if p.Duration > 0 {
				return p.Duration
			}
		}
	}
	return 30
}

func appointmentCompensationPreload(db *gorm.DB) *gorm.DB {
	return db.Order("id DESC")
}

func appointmentRescheduleRequestPreload(db *gorm.DB) *gorm.DB {
	return db.Order("id DESC")
}

func hydrateAppointmentRelations(tx *gorm.DB, appt *models.Appointment) error {
	if tx == nil || appt == nil || appt.ID == 0 {
		return nil
	}
	// 预约主记录统一通过 raw loader 读取，避免 sqlite 在测试环境里把 datetime 字符串直接扫进 *time.Time。
	// 这里单独补齐关联关系，保证接口返回结构与原来保持一致。
	if err := tx.Where("id = ?", appt.UserID).First(&appt.User).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err := tx.Where("id = ?", appt.CardID).First(&appt.Card).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err := tx.Where("id = ?", appt.MerchantID).First(&appt.Merchant).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if appt.ProjectID != nil && *appt.ProjectID > 0 {
		var project models.MerchantProject
		if err := tx.Where("id = ?", *appt.ProjectID).First(&project).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		} else {
			appt.Project = &project
		}
	}
	if appt.TechnicianID != nil && *appt.TechnicianID > 0 {
		var technician models.Technician
		if err := tx.Preload("ServiceRole").Where("id = ?", *appt.TechnicianID).First(&technician).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		} else {
			appt.Technician = &technician
		}
	}
	var compensations []models.AppointmentCompensation
	if err := tx.Scopes(appointmentCompensationPreload).Where("appointment_id = ?", appt.ID).Find(&compensations).Error; err != nil {
		return err
	}
	appt.Compensations = compensations
	var requests []models.AppointmentRescheduleRequest
	if err := tx.Scopes(appointmentRescheduleRequestPreload).Where("appointment_id = ?", appt.ID).Find(&requests).Error; err != nil {
		return err
	}
	appt.RescheduleRequests = requests
	return nil
}

func latestPendingRescheduleRequest(appt *models.Appointment) *models.AppointmentRescheduleRequest {
	if appt == nil {
		return nil
	}
	for i := range appt.RescheduleRequests {
		status := strings.TrimSpace(appt.RescheduleRequests[i].Status)
		if status == "pending_user" || status == "pending_merchant" {
			return &appt.RescheduleRequests[i]
		}
	}
	return nil
}

func loadAppointmentRescheduleRequestByID(tx *gorm.DB, requestID uint) (*models.AppointmentRescheduleRequest, error) {
	if tx == nil || requestID == 0 {
		return nil, nil
	}
	rows, err := tx.Table("appointment_reschedule_requests").
		Select("id, appointment_id, merchant_id, user_id, current_project_id, current_technician_id, current_time, new_project_id, new_technician_id, new_appointment_time, status, reason, proposed_by_type, proposed_by_id, confirmed_by_type, confirmed_by_id, confirmed_at, result_appointment_id, created_at").
		Where("id = ?", requestID).
		Limit(1).
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, nil
	}

	var (
		req                    models.AppointmentRescheduleRequest
		currentProjectIDRaw    interface{}
		currentTechnicianIDRaw interface{}
		currentTimeRaw         interface{}
		newProjectIDRaw        interface{}
		newTechnicianIDRaw     interface{}
		newTimeRaw             interface{}
		proposedByIDRaw        interface{}
		confirmedByIDRaw       interface{}
		confirmedAtRaw         interface{}
		resultAppointmentIDRaw interface{}
		createdAtRaw           interface{}
	)
	if err := rows.Scan(
		&req.ID,
		&req.AppointmentID,
		&req.MerchantID,
		&req.UserID,
		&currentProjectIDRaw,
		&currentTechnicianIDRaw,
		&currentTimeRaw,
		&newProjectIDRaw,
		&newTechnicianIDRaw,
		&newTimeRaw,
		&req.Status,
		&req.Reason,
		&req.ProposedByType,
		&proposedByIDRaw,
		&req.ConfirmedByType,
		&confirmedByIDRaw,
		&confirmedAtRaw,
		&resultAppointmentIDRaw,
		&createdAtRaw,
	); err != nil {
		return nil, err
	}
	if v, ok := gormValueToUint(currentProjectIDRaw); ok {
		req.CurrentProjectID = &v
	}
	if v, ok := gormValueToUint(currentTechnicianIDRaw); ok {
		req.CurrentTechnicianID = &v
	}
	if v, ok := parseDBTimeValue(currentTimeRaw); ok {
		req.CurrentTime = &v
	}
	if v, ok := gormValueToUint(newProjectIDRaw); ok {
		req.NewProjectID = &v
	}
	if v, ok := gormValueToUint(newTechnicianIDRaw); ok {
		req.NewTechnicianID = &v
	}
	if v, ok := parseDBTimeValue(newTimeRaw); ok {
		req.NewAppointmentTime = &v
	}
	if v, ok := gormValueToUint(proposedByIDRaw); ok {
		req.ProposedByID = &v
	}
	if v, ok := gormValueToUint(confirmedByIDRaw); ok {
		req.ConfirmedByID = &v
	}
	if v, ok := parseDBTimeValue(confirmedAtRaw); ok {
		req.ConfirmedAt = &v
	}
	if v, ok := gormValueToUint(resultAppointmentIDRaw); ok {
		req.ResultAppointmentID = &v
	}
	if v, ok := parseDBTimeValue(createdAtRaw); ok {
		req.CreatedAt = &v
	}
	return &req, nil
}

func hasPendingRescheduleRequest(tx *gorm.DB, appointmentID uint) (bool, error) {
	if tx == nil || appointmentID == 0 {
		return false, nil
	}
	var count int64
	if err := tx.Table("appointment_reschedule_requests").
		Where("appointment_id = ? AND status IN ?", appointmentID, []string{"pending_user", "pending_merchant"}).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func getAppointmentActor(c *gin.Context) (string, *uint) {
	authType := strings.TrimSpace(c.GetString("auth_type"))
	switch authType {
	case "merchant":
		if merchantID, ok := c.Get("merchant_id"); ok {
			if id, ok := merchantID.(uint); ok && id > 0 {
				return "merchant", &id
			}
		}
	case "staff":
		if technicianID, ok := c.Get("technician_id"); ok {
			if id, ok := technicianID.(uint); ok && id > 0 {
				return "staff", &id
			}
		}
	case "user":
		if userID, ok := c.Get("user_id"); ok {
			if id, ok := userID.(uint); ok && id > 0 {
				return "user", &id
			}
		}
	}
	return authType, nil
}

func appointmentAllowsReschedule(status string) bool {
	switch normalizeAppointmentStatus(status) {
	case "pending", "confirmed", "arrived", "failed":
		return true
	default:
		return false
	}
}

func appendAppointmentResolutionNote(current, next string) string {
	left := strings.TrimSpace(current)
	right := strings.TrimSpace(next)
	if left == "" {
		return right
	}
	if right == "" {
		return left
	}
	return left + "\n" + right
}

func cancelUnstartedAppointmentArrival(tx *gorm.DB, appt *models.Appointment, now time.Time) error {
	if appt == nil || appt.UsageID == nil || appt.ServiceSessionID == nil {
		return apiErr{status: http.StatusBadRequest, msg: "当前预约未生成可撤回的到店记录"}
	}

	var usage models.Usage
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", *appt.UsageID).First(&usage).Error; err != nil {
		return err
	}
	if usage.Status != "in_progress" {
		return apiErr{status: http.StatusBadRequest, msg: "当前预约已进入不可改签阶段"}
	}

	var session models.ServiceSession
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", *appt.ServiceSessionID).First(&session).Error; err != nil {
		return err
	}
	baseStatus := models.NormalizeSessionStatus(session.Status)
	switch baseStatus {
	case "room_selecting", "room_locked", "staff_selecting", "appointment_waiting", "start_pending", "delay_pending":
	default:
		return apiErr{status: http.StatusBadRequest, msg: "当前预约已开始服务，不能改签"}
	}

	var card models.Card
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&card, appt.CardID).Error; err != nil {
		return err
	}
	used := usage.UsedTimes
	if used <= 0 {
		used = 1
	}
	if err := tx.Model(&models.Card{}).Where("id = ?", card.ID).Updates(map[string]interface{}{
		"remain_times": gorm.Expr("remain_times + ?", used),
		"used_times":   gorm.Expr("CASE WHEN used_times >= ? THEN used_times - ? ELSE 0 END", used, used),
	}).Error; err != nil {
		return err
	}
	if err := tx.Model(&models.Usage{}).Where("id = ? AND status = ?", usage.ID, "in_progress").Updates(map[string]interface{}{
		"status":        "failed",
		"finished_at":   &now,
		"technician_id": nil,
	}).Error; err != nil {
		return err
	}

	updates := map[string]interface{}{
		"status":                  models.ApplyStatusPrefix(session.Status, "canceled"),
		"technician_id":           nil,
		"room_id":                 nil,
		"room_locked_at":          nil,
		"room_select_deadline_at": nil,
		"start_confirmed_at":      nil,
		"predicted_ready_at":      nil,
	}
	if err := tx.Model(&models.ServiceSession{}).
		Where("id = ? AND status NOT IN ?", session.ID, models.ExpandStatusesWithKnownPrefixes([]string{"finished", "canceled"})).
		Updates(updates).Error; err != nil {
		return err
	}
	return nil
}

func compensationDisplayText(t string, value int, remark string) string {
	switch t {
	case "extra_times":
		return fmt.Sprintf("补偿次数 +%d", value)
	case "extend_minutes":
		return fmt.Sprintf("补偿时长 +%d分钟", value)
	case "discount_note":
		if strings.TrimSpace(remark) != "" {
			return "补偿优惠：" + strings.TrimSpace(remark)
		}
		return "补偿优惠"
	default:
		return strings.TrimSpace(remark)
	}
}

func cancelAppointmentWithTime(appointment *models.Appointment, canceledAt time.Time) error {
	return config.DB.Model(appointment).Updates(map[string]interface{}{
		"status":      "canceled",
		"canceled_at": canceledAt,
	}).Error
}

func autoCancelAppointmentIfOverdue(appointment *models.Appointment, now time.Time) (bool, error) {
	if appointment == nil || appointment.AppointmentTime == nil {
		return false, nil
	}
	if appointment.Status != "pending" && appointment.Status != "confirmed" {
		return false, nil
	}
	deadline := appointment.AppointmentTime.Add(35 * time.Minute)
	if now.Before(deadline) {
		return false, nil
	}
	if err := cancelAppointmentWithTime(appointment, now); err != nil {
		return false, err
	}
	return true, nil
}

func getMerchantAppointmentPermissionState(c *gin.Context) (canView bool, canManage bool, err error) {
	canView, err = hasMerchantPermissionInHandler(c, "merchant.appointment.view")
	if err != nil {
		return false, false, err
	}
	canManage, err = hasMerchantPermissionInHandler(c, "merchant.appointment.manage")
	if err != nil {
		return false, false, err
	}
	return canView, canManage, nil
}

func getCurrentTechnicianID(c *gin.Context) uint {
	technicianIDAny, ok := c.Get("technician_id")
	if !ok {
		return 0
	}
	technicianID, ok := technicianIDAny.(uint)
	if !ok {
		return 0
	}
	return technicianID
}

func requireMerchantAppointmentAccess(c *gin.Context) (canManage bool, technicianID uint, ok bool) {
	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)
	if authType != "merchant" && authType != "staff" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return false, 0, false
	}

	canView, canManage, err := getMerchantAppointmentPermissionState(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "权限校验失败"})
		return false, 0, false
	}
	if !canView && !canManage {
		c.JSON(http.StatusForbidden, gin.H{"error": "无预约权限"})
		return false, 0, false
	}

	if authType == "staff" {
		technicianID = getCurrentTechnicianID(c)
		if technicianID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
			return false, 0, false
		}
	}

	return canManage, technicianID, true
}

func checkMerchantAppointmentOwnership(c *gin.Context, appointment models.Appointment) (canManage bool, technicianID uint, ok bool) {
	merchantIDAny, exists := c.Get("merchant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return false, 0, false
	}
	merchantID, okCast := merchantIDAny.(uint)
	if !okCast || merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return false, 0, false
	}
	if appointment.MerchantID != merchantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限：不属于您的商户"})
		return false, 0, false
	}

	canManage, technicianID, ok = requireMerchantAppointmentAccess(c)
	if !ok {
		return false, 0, false
	}

	if technicianID > 0 && !canManage {
		if appointment.TechnicianID == nil || *appointment.TechnicianID != technicianID {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权限：只能操作分配给自己的预约"})
			return false, 0, false
		}
	}

	return canManage, technicianID, true
}

func GetMerchantAppointments(c *gin.Context) {
	merchantID, ok := ensureMerchantScope(c, "id")
	if !ok {
		return
	}

	canManage, technicianID, ok := requireMerchantAppointmentAccess(c)
	if !ok {
		return
	}
	status := c.Query("status")

	var appointments []models.Appointment
	query := config.DB.Preload("User").Preload("Card").Preload("Merchant").Preload("Project").Preload("Technician").Preload("Technician.ServiceRole").Preload("Compensations", appointmentCompensationPreload).Preload("RescheduleRequests", appointmentRescheduleRequestPreload).Where("merchant_id = ?", merchantID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if technicianID > 0 && !canManage {
		query = query.Where("(technician_id = ? OR technician_id IS NULL)", technicianID)
	}

	query.Order("appointment_time ASC").Find(&appointments)
	for i := range appointments {
		appointments[i].Status = normalizeAppointmentStatus(appointments[i].Status)
	}
	c.JSON(http.StatusOK, gin.H{"data": appointments})
}

func GetUserAppointments(c *gin.Context) {
	authUserID, ok := mustUserID(c)
	if !ok {
		return
	}
	userIDStr := strings.TrimSpace(c.Param("id"))
	uid, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil || uint(uid) != authUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问其他用户数据"})
		return
	}
	var appointments []models.Appointment
	config.DB.Preload("Merchant").Preload("Technician").Preload("Technician.ServiceRole").Preload("Compensations", appointmentCompensationPreload).Preload("RescheduleRequests", appointmentRescheduleRequestPreload).Where("user_id = ?", authUserID).Order("appointment_time DESC").Find(&appointments)
	for i := range appointments {
		appointments[i].Status = normalizeAppointmentStatus(appointments[i].Status)
	}
	c.JSON(http.StatusOK, gin.H{"data": appointments})
}

func GetCardAppointment(c *gin.Context) {
	authUserID, ok := mustUserID(c)
	if !ok {
		return
	}
	cardID := c.Param("id")

	var card models.Card
	if err := config.DB.First(&card, cardID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "卡片不存在"})
		return
	}
	if card.UserID != authUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问此卡"})
		return
	}

	// 添加调试日志
	log.Printf("查询预约: 卡片ID=%s, 用户ID=%d, 商户ID=%d", cardID, card.UserID, card.MerchantID)

	var appointment models.Appointment
	err := config.DB.Preload("Merchant").Preload("Project").Preload("Technician").Preload("Technician.ServiceRole").Preload("Compensations", appointmentCompensationPreload).Preload("RescheduleRequests", appointmentRescheduleRequestPreload).
		Where("card_id = ? AND merchant_id = ? AND user_id = ? AND status IN ('pending', 'confirmed', 'arrived', 'failed')", card.ID, card.MerchantID, card.UserID).
		Order("appointment_time ASC").
		First(&appointment).Error

	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("查询预约失败: %v", err)
		} else {
			log.Printf("未找到预约: user_id=%d, merchant_id=%d", card.UserID, card.MerchantID)
		}
		c.JSON(http.StatusOK, gin.H{"data": gin.H{
			"appointment":       nil,
			"queue_before":      0,
			"estimated_minutes": 0,
		}})
		return
	}

	log.Printf("找到预约: ID=%d, 状态=%s, 时间=%v", appointment.ID, appointment.Status, appointment.AppointmentTime)
	appointment.Status = normalizeAppointmentStatus(appointment.Status)

	now := time.Now()
	autoCanceled, autoCancelErr := autoCancelAppointmentIfOverdue(&appointment, now)
	if autoCancelErr != nil {
		log.Printf("自动取消预约失败: %v", autoCancelErr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "自动取消预约失败"})
		return
	}
	if autoCanceled {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{
			"appointment":       nil,
			"queue_before":      0,
			"estimated_minutes": 0,
		}})
		return
	}

	// 计算预约队列排队信息（内存队列）
	queueBefore := int64(0)
	if appointment.AppointmentTime != nil {
		date := appointment.AppointmentTime.Format("2006-01-02")
		snap := queue.Default.Snapshot(card.MerchantID, date, queue.QueueTypeAppointment)
		if snap.ByID != nil {
			if tk, ok := snap.ByID[appointment.ID]; ok {
				queueBefore = int64(tk.No - 1)
				if queueBefore < 0 {
					queueBefore = 0
				}
			}
		}
	}

	var merchant models.Merchant
	config.DB.First(&merchant, card.MerchantID)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"appointment":            appointment,
			"queue_before":           queueBefore,
			"estimated_minutes":      int(queueBefore) * 30,
			"service_session_id":     appointment.ServiceSessionID,
			"predicted_wait_minutes": appointment.PredictedWaitMinutes,
			"can_arrive_now":         canArriveForAppointment(appointment, &merchant, now),
		},
	})
}

func CreateAppointment(c *gin.Context) {
	var input struct {
		CardID          uint   `json:"card_id" binding:"required"`
		MerchantID      uint   `json:"merchant_id" binding:"required"`
		UserID          *uint  `json:"user_id"`
		ProjectID       *uint  `json:"project_id"`
		TechnicianID    *uint  `json:"technician_id"`
		AppointmentTime string `json:"appointment_time" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	authUserID, ok := mustUserID(c)
	if !ok {
		return
	}
	if input.UserID != nil && *input.UserID != authUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "user_id 与登录态不一致"})
		return
	}

	// 校验卡片归属（避免同商户多卡串数据）
	var card models.Card
	if err := config.DB.First(&card, input.CardID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "卡片不存在"})
		return
	}
	if card.Locked {
		msg := "卡片已锁定"
		if strings.TrimSpace(card.LockedReason) != "" {
			msg = card.LockedReason
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	if card.UserID != authUserID || card.MerchantID != input.MerchantID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的卡片"})
		return
	}

	// 检查商户是否支持预约
	var merchant models.Merchant
	if err := config.DB.First(&merchant, input.MerchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}

	if !merchant.SupportAppointment {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该商户不支持预约"})
		return
	}

	loc, locErr := time.LoadLocation("Asia/Shanghai")
	if locErr != nil {
		loc = time.Local
	}

	if input.TechnicianID != nil {
		var tech models.Technician
		if err := config.DB.Where("id = ? AND merchant_id = ?", *input.TechnicianID, input.MerchantID).First(&tech).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的技师"})
			return
		}
	}

	var project models.MerchantProject
	if input.ProjectID != nil {
		if *input.ProjectID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目"})
			return
		}
		if err := config.DB.Where("id = ? AND merchant_id = ? AND is_active = ?", *input.ProjectID, input.MerchantID, true).First(&project).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目"})
			return
		}
		if project.Duration <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "项目时长无效"})
			return
		}
	}

	// 检查该卡片在该商户是否已有活跃的预约
	var existingAppointment models.Appointment
	err := config.DB.Where("card_id = ? AND merchant_id = ? AND user_id = ? AND status IN ('pending', 'confirmed')",
		input.CardID, input.MerchantID, authUserID).First(&existingAppointment).Error

	if err == nil {
		if autoCanceled, autoCancelErr := autoCancelAppointmentIfOverdue(&existingAppointment, time.Now()); autoCancelErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "自动取消预约失败"})
			return
		} else if autoCanceled {
			c.JSON(http.StatusBadRequest, gin.H{"error": "您已有预约但已超时取消，请稍后再试"})
			return
		}

		log.Printf("用户已有活跃预约: ID=%d, 状态=%s", existingAppointment.ID, existingAppointment.Status)
		c.JSON(http.StatusBadRequest, gin.H{"error": "您已有进行中的预约，请先取消后再预约"})
		return
	}

	log.Printf("创建新预约: 用户ID=%d, 商户ID=%d, 时间=%s", authUserID, input.MerchantID, input.AppointmentTime)

	appointmentTime, err := time.ParseInLocation("2006-01-02 15:04:05", input.AppointmentTime, loc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "预约时间格式错误"})
		return
	}

	now := time.Now().In(loc)
	tomorrowDate := now.Add(24 * time.Hour).Format("2006-01-02")
	if appointmentTime.In(loc).Format("2006-01-02") != tomorrowDate {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持预约明天"})
		return
	}

	targetDate, _ := time.ParseInLocation("2006-01-02", tomorrowDate, loc)
	intervals, ok := getMerchantBusinessIntervalsForDate(merchant, targetDate)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "商户未设置营业时间"})
		return
	}
	if !isWithinBusinessIntervals(appointmentTime, intervals) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "预约时间不在营业时间范围内"})
		return
	}
	serviceMinutes := 30
	if input.ProjectID != nil {
		serviceMinutes = project.Duration
	}
	serviceEnd := appointmentTime.Add(time.Duration(serviceMinutes) * time.Minute)
	if !isWithinBusinessIntervals(serviceEnd.Add(-1*time.Second), intervals) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "服务必须在营业时间内完成"})
		return
	}

	appointment := models.Appointment{
		CardID:          input.CardID,
		MerchantID:      input.MerchantID,
		UserID:          authUserID,
		ProjectID:       input.ProjectID,
		TechnicianID:    input.TechnicianID,
		AppointmentTime: &appointmentTime,
		Status:          "pending",
	}

	// 预约创建必须做事务级校验：用户手选客服时，要在落库前重新检查“未来预约占产能”规则，
	// 避免前端时段缓存与并发提交之间出现超卖或越过最大等待上限。
	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		if input.TechnicianID != nil && *input.TechnicianID > 0 {
			availability, err := evaluateBookingTechnicianAvailability(tx, merchant, *input.TechnicianID, appointmentTime, serviceMinutes, 0)
			if err != nil {
				return err
			}
			if availability.State == appointmentAvailabilityUnavailable {
				return apiErr{status: http.StatusBadRequest, msg: availability.Reason}
			}
			appointment.PredictedWaitMinutes = availability.PredictedWaitMinutes
		}
		if err := tx.Create(&appointment).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		log.Printf("创建预约失败: %v", err)
		var ae apiErr
		if errors.As(err, &ae) {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建预约失败"})
		return
	}

	log.Printf("预约创建成功: ID=%d, 状态=%s", appointment.ID, appointment.Status)

	config.DB.Preload("User").Preload("Card").Preload("Merchant").Preload("Project").Preload("Technician").Preload("Compensations", appointmentCompensationPreload).Preload("RescheduleRequests", appointmentRescheduleRequestPreload).First(&appointment, appointment.ID)
	c.JSON(http.StatusOK, gin.H{"data": appointment})
}

func ConfirmAppointment(c *gin.Context) {
	id := c.Param("id")
	var appointment models.Appointment
	if err := config.DB.First(&appointment, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}

	if _, _, ok := checkMerchantAppointmentOwnership(c, appointment); !ok {
		return
	}

	if autoCanceled, autoCancelErr := autoCancelAppointmentIfOverdue(&appointment, time.Now()); autoCancelErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "自动取消预约失败"})
		return
	} else if autoCanceled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "预约已超时取消，无法确认"})
		return
	}

	if appointment.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "只能确认待处理的预约"})
		return
	}

	if appointment.AppointmentTime == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "预约时间为空"})
		return
	}

	now := time.Now()
	if now.After(*appointment.AppointmentTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "预约已过期，无法确认"})
		return
	}

	confirmedAt := time.Now()
	config.DB.Model(&appointment).Updates(map[string]interface{}{
		"status":       "confirmed",
		"confirmed_at": &confirmedAt,
	})
	config.DB.Preload("User").Preload("Card").Preload("Merchant").Preload("Technician").Preload("Compensations", appointmentCompensationPreload).Preload("RescheduleRequests", appointmentRescheduleRequestPreload).First(&appointment, id)
	appointment.Status = normalizeAppointmentStatus(appointment.Status)
	// 入预约队列（内存队列）：仅 confirmed 才进入预约排队
	if appointment.AppointmentTime != nil {
		date := appointment.AppointmentTime.Format("2006-01-02")
		now2 := time.Now()
		queue.Default.Enqueue(appointment.MerchantID, date, queue.QueueTypeAppointment, appointment.ID, 1, true, now2)
	}
	c.JSON(http.StatusOK, gin.H{"data": appointment})
}

func FinishAppointment(c *gin.Context) {
	c.JSON(http.StatusBadRequest, gin.H{"error": "预约完成已改为随核销/服务会话自动闭环"})
}

func UpdateAppointmentResolution(c *gin.Context) {
	id := c.Param("id")
	var appointment models.Appointment
	if err := config.DB.First(&appointment, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}

	if _, _, ok := checkMerchantAppointmentOwnership(c, appointment); !ok {
		return
	}

	var input struct {
		ResolutionNote string `json:"resolution_note" binding:"required,max=255"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	note := strings.TrimSpace(input.ResolutionNote)
	if note == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "处理备注不能为空"})
		return
	}

	// 改签/补偿首版先把人工处理动作落到结构化备注，避免预约冲突只能靠线下口头同步。
	if err := config.DB.Model(&appointment).Update("resolution_note", note).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存处理备注失败"})
		return
	}
	appointment.ResolutionNote = note
	appointment.Status = normalizeAppointmentStatus(appointment.Status)
	c.JSON(http.StatusOK, gin.H{"data": appointment})
}

type appointmentReschedulePayload struct {
	AppointmentTime string `json:"appointment_time" binding:"required"`
	TechnicianID    *uint  `json:"technician_id"`
	ProjectID       *uint  `json:"project_id"`
	Reason          string `json:"reason" binding:"required,max=255"`
}

func parseAppointmentReschedulePayload(input appointmentReschedulePayload) (time.Time, string, *time.Location, error) {
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		return time.Time{}, "", nil, apiErr{status: http.StatusBadRequest, msg: "改签原因不能为空"}
	}
	loc, locErr := time.LoadLocation("Asia/Shanghai")
	if locErr != nil {
		loc = time.Local
	}
	newAppointmentTime, err := time.ParseInLocation("2006-01-02 15:04:05", input.AppointmentTime, loc)
	if err != nil {
		return time.Time{}, "", nil, apiErr{status: http.StatusBadRequest, msg: "预约时间格式错误"}
	}
	if newAppointmentTime.Before(time.Now().In(loc).Add(-1 * time.Minute)) {
		return time.Time{}, "", nil, apiErr{status: http.StatusBadRequest, msg: "不能改签到过去时间"}
	}
	return newAppointmentTime, reason, loc, nil
}

func executeAppointmentReschedule(tx *gorm.DB, appointmentID uint, proposal models.AppointmentRescheduleRequest, actorType string, actorID *uint, now time.Time) (models.Appointment, models.Appointment, error) {
	var oldAppointment models.Appointment
	var newAppointment models.Appointment
	currentPtr, err := loadAppointmentByID(tx, appointmentID)
	if err != nil {
		return oldAppointment, newAppointment, err
	}
	if currentPtr == nil {
		return oldAppointment, newAppointment, gorm.ErrRecordNotFound
	}
	current := *currentPtr
	if !appointmentAllowsReschedule(current.Status) {
		return oldAppointment, newAppointment, apiErr{status: http.StatusBadRequest, msg: "当前预约状态不可改签"}
	}
	if proposal.NewAppointmentTime == nil {
		return oldAppointment, newAppointment, apiErr{status: http.StatusBadRequest, msg: "改签时间不能为空"}
	}

	var merchant models.Merchant
	if err := tx.First(&merchant, current.MerchantID).Error; err != nil {
		return oldAppointment, newAppointment, err
	}
	intervals, ok := getMerchantBusinessIntervalsForDate(merchant, proposal.NewAppointmentTime.In(time.Local))
	if !ok || !isWithinBusinessIntervals(*proposal.NewAppointmentTime, intervals) {
		return oldAppointment, newAppointment, apiErr{status: http.StatusBadRequest, msg: "改签时间不在营业时间内"}
	}

	projectID := current.ProjectID
	serviceMinutes := getAppointmentServiceMinutes(current.MerchantID, current)
	if proposal.NewProjectID != nil && *proposal.NewProjectID > 0 {
		var project models.MerchantProject
		if err := tx.Where("id = ? AND merchant_id = ? AND is_active = ?", *proposal.NewProjectID, current.MerchantID, true).First(&project).Error; err != nil {
			return oldAppointment, newAppointment, apiErr{status: http.StatusBadRequest, msg: "无效的项目"}
		}
		projectID = proposal.NewProjectID
		serviceMinutes = project.Duration
	}
	if serviceMinutes <= 0 {
		serviceMinutes = 30
	}

	technicianID := current.TechnicianID
	if proposal.NewTechnicianID != nil {
		if *proposal.NewTechnicianID == 0 {
			technicianID = nil
		} else {
			technicianID = proposal.NewTechnicianID
		}
	}

	if technicianID != nil && *technicianID > 0 {
		availability, err := evaluateBookingTechnicianAvailability(tx, merchant, *technicianID, *proposal.NewAppointmentTime, serviceMinutes, current.ID)
		if err != nil {
			return oldAppointment, newAppointment, err
		}
		if availability.State == appointmentAvailabilityUnavailable {
			return oldAppointment, newAppointment, apiErr{status: http.StatusBadRequest, msg: "新时间下该客服不可预约"}
		}
		current.PredictedWaitMinutes = availability.PredictedWaitMinutes
	} else {
		current.PredictedWaitMinutes = 0
	}

	if normalizeAppointmentStatus(current.Status) == "arrived" {
		// 到店后接受改签，需要先回滚未开始服务的占用，再关闭旧预约。
		if err := cancelUnstartedAppointmentArrival(tx, &current, now); err != nil {
			return oldAppointment, newAppointment, err
		}
	}

	newStatus := "pending"
	newConfirmedAt := (*time.Time)(nil)
	if normalizeAppointmentStatus(current.Status) == "confirmed" || normalizeAppointmentStatus(current.Status) == "arrived" {
		newStatus = "confirmed"
		newConfirmedAt = &now
	}
	newAppointment = models.Appointment{
		CardID:                current.CardID,
		MerchantID:            current.MerchantID,
		UserID:                current.UserID,
		ProjectID:             projectID,
		TechnicianID:          technicianID,
		AppointmentTime:       proposal.NewAppointmentTime,
		Status:                newStatus,
		ConfirmedAt:           newConfirmedAt,
		PredictedWaitMinutes:  current.PredictedWaitMinutes,
		ReplacesAppointmentID: &current.ID,
		RescheduleReason:      proposal.Reason,
		ResolutionNote:        appendAppointmentResolutionNote("", "改签说明："+proposal.Reason),
	}
	if err := tx.Create(&newAppointment).Error; err != nil {
		return oldAppointment, newAppointment, err
	}

	closeUpdates := map[string]interface{}{
		"status":                     "canceled",
		"closed_reason":              "rescheduled",
		"closed_by_type":             actorType,
		"closed_by_id":               actorID,
		"reschedule_reason":          proposal.Reason,
		"replaced_by_appointment_id": newAppointment.ID,
		"canceled_at":                &now,
		"resolution_note":            appendAppointmentResolutionNote(current.ResolutionNote, fmt.Sprintf("改签到 %s", proposal.NewAppointmentTime.Format("2006-01-02 15:04"))),
		"arrived_at":                 nil,
		"service_session_id":         nil,
		"usage_id":                   nil,
		"predicted_wait_minutes":     0,
	}
	if err := tx.Model(&models.Appointment{}).Where("id = ?", current.ID).Updates(closeUpdates).Error; err != nil {
		return oldAppointment, newAppointment, err
	}

	oldAppointment = current
	oldAppointment.Status = "canceled"
	oldAppointment.ClosedReason = "rescheduled"
	oldAppointment.ClosedByType = actorType
	oldAppointment.ClosedByID = actorID
	oldAppointment.RescheduleReason = proposal.Reason
	oldAppointment.ReplacedByAppointmentID = &newAppointment.ID
	oldAppointment.CanceledAt = &now
	return oldAppointment, newAppointment, nil
}

func CreateAppointmentRescheduleRequest(c *gin.Context) {
	id := c.Param("id")
	appointmentID64, err := strconv.ParseUint(strings.TrimSpace(id), 10, 64)
	if err != nil || appointmentID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的预约ID"})
		return
	}
	appointment, err := loadAppointmentByID(config.DB, uint(appointmentID64))
	if err != nil || appointment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}

	authType := strings.TrimSpace(c.GetString("auth_type"))
	switch authType {
	case "merchant", "staff":
		if _, _, ok := checkMerchantAppointmentOwnership(c, *appointment); !ok {
			return
		}
	case "user":
		authUserID, ok := mustUserID(c)
		if !ok {
			return
		}
		if appointment.UserID != authUserID {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此预约"})
			return
		}
	default:
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	var input appointmentReschedulePayload
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	newAppointmentTime, reason, _, err := parseAppointmentReschedulePayload(input)
	if err != nil {
		var ae apiErr
		if errors.As(err, &ae) {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	actorType, actorID := getAppointmentActor(c)
	requestStatus := "pending_user"
	if authType == "user" {
		requestStatus = "pending_merchant"
	}

	var proposal models.AppointmentRescheduleRequest
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		currentPtr, err := loadAppointmentByID(tx, appointment.ID)
		if err != nil {
			return err
		}
		if currentPtr == nil {
			return gorm.ErrRecordNotFound
		}
		current := *currentPtr
		if !appointmentAllowsReschedule(current.Status) {
			return apiErr{status: http.StatusBadRequest, msg: "当前预约状态不可改签"}
		}
		if exists, err := hasPendingRescheduleRequest(tx, current.ID); err != nil {
			return err
		} else if exists {
			return apiErr{status: http.StatusBadRequest, msg: "当前预约已有待确认的改签提议"}
		}

		proposal = models.AppointmentRescheduleRequest{
			AppointmentID:       current.ID,
			MerchantID:          current.MerchantID,
			UserID:              current.UserID,
			CurrentProjectID:    current.ProjectID,
			CurrentTechnicianID: current.TechnicianID,
			CurrentTime:         current.AppointmentTime,
			NewProjectID:        current.ProjectID,
			NewTechnicianID:     current.TechnicianID,
			NewAppointmentTime:  &newAppointmentTime,
			Status:              requestStatus,
			Reason:              reason,
			ProposedByType:      actorType,
			ProposedByID:        actorID,
		}
		if input.ProjectID != nil {
			proposal.NewProjectID = input.ProjectID
		}
		if input.TechnicianID != nil {
			proposal.NewTechnicianID = input.TechnicianID
		}
		return tx.Create(&proposal).Error
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
	c.JSON(http.StatusOK, gin.H{"data": proposal})
}

func AcceptAppointmentRescheduleRequest(c *gin.Context) {
	requestID64, err := strconv.ParseUint(strings.TrimSpace(c.Param("request_id")), 10, 64)
	if err != nil || requestID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的改签提议ID"})
		return
	}
	req, err := loadAppointmentRescheduleRequestByID(config.DB, uint(requestID64))
	if err != nil || req == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "改签提议不存在"})
		return
	}
	appointment, err := loadAppointmentByID(config.DB, req.AppointmentID)
	if err != nil || appointment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}

	authType := strings.TrimSpace(c.GetString("auth_type"))
	switch authType {
	case "merchant", "staff":
		if _, _, ok := checkMerchantAppointmentOwnership(c, *appointment); !ok {
			return
		}
		if req.Status != "pending_merchant" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "当前改签提议无需商户确认"})
			return
		}
	case "user":
		authUserID, ok := mustUserID(c)
		if !ok {
			return
		}
		if appointment.UserID != authUserID {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此改签提议"})
			return
		}
		if req.Status != "pending_user" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "当前改签提议无需用户确认"})
			return
		}
	default:
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	actorType, actorID := getAppointmentActor(c)
	now := time.Now()
	var oldAppointment, newAppointment models.Appointment
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		currentReq, err := loadAppointmentRescheduleRequestByID(tx, req.ID)
		if err != nil {
			return err
		}
		if currentReq == nil {
			return gorm.ErrRecordNotFound
		}
		if strings.TrimSpace(currentReq.Status) != strings.TrimSpace(req.Status) {
			return apiErr{status: http.StatusBadRequest, msg: "改签提议状态已变化，请刷新后重试"}
		}
		oldAppointment, newAppointment, err = executeAppointmentReschedule(tx, currentReq.AppointmentID, *currentReq, actorType, actorID, now)
		if err != nil {
			return err
		}
		return tx.Model(&models.AppointmentRescheduleRequest{}).Where("id = ?", currentReq.ID).Updates(map[string]interface{}{
			"status":                "accepted",
			"confirmed_by_type":     actorType,
			"confirmed_by_id":       actorID,
			"confirmed_at":          &now,
			"result_appointment_id": newAppointment.ID,
		}).Error
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
	if reloadedAppointment, err := loadAppointmentByID(config.DB, newAppointment.ID); err == nil && reloadedAppointment != nil {
		newAppointment = *reloadedAppointment
		_ = hydrateAppointmentRelations(config.DB, &newAppointment)
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"old_appointment": oldAppointment,
		"new_appointment": newAppointment,
	}})
}

func RejectAppointmentRescheduleRequest(c *gin.Context) {
	requestID64, err := strconv.ParseUint(strings.TrimSpace(c.Param("request_id")), 10, 64)
	if err != nil || requestID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的改签提议ID"})
		return
	}
	req, err := loadAppointmentRescheduleRequestByID(config.DB, uint(requestID64))
	if err != nil || req == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "改签提议不存在"})
		return
	}
	appointment, err := loadAppointmentByID(config.DB, req.AppointmentID)
	if err != nil || appointment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}
	authType := strings.TrimSpace(c.GetString("auth_type"))
	switch authType {
	case "merchant", "staff":
		if _, _, ok := checkMerchantAppointmentOwnership(c, *appointment); !ok {
			return
		}
		if req.Status != "pending_merchant" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "当前改签提议无需商户处理"})
			return
		}
	case "user":
		authUserID, ok := mustUserID(c)
		if !ok {
			return
		}
		if appointment.UserID != authUserID {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此改签提议"})
			return
		}
		if req.Status != "pending_user" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "当前改签提议无需用户处理"})
			return
		}
	default:
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	actorType, actorID := getAppointmentActor(c)
	now := time.Now()
	if err := config.DB.Model(&models.AppointmentRescheduleRequest{}).
		Where("id = ? AND status = ?", req.ID, req.Status).
		Updates(map[string]interface{}{
			"status":            "rejected",
			"confirmed_by_type": actorType,
			"confirmed_by_id":   actorID,
			"confirmed_at":      &now,
		}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "拒绝改签提议失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"id": req.ID, "status": "rejected"}})
}

func RescheduleAppointment(c *gin.Context) {
	CreateAppointmentRescheduleRequest(c)
}

func ListAppointmentCompensations(c *gin.Context) {
	id := c.Param("id")
	appointmentID64, err := strconv.ParseUint(strings.TrimSpace(id), 10, 64)
	if err != nil || appointmentID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的预约ID"})
		return
	}
	appointment, err := loadAppointmentByID(config.DB, uint(appointmentID64))
	if err != nil || appointment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}
	if _, _, ok := checkMerchantAppointmentOwnership(c, *appointment); !ok {
		return
	}
	var compensations []models.AppointmentCompensation
	if err := config.DB.Where("appointment_id = ?", appointment.ID).Order("id DESC").Find(&compensations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取补偿记录失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": compensations})
}

func CreateAppointmentCompensation(c *gin.Context) {
	id := c.Param("id")
	appointmentID64, err := strconv.ParseUint(strings.TrimSpace(id), 10, 64)
	if err != nil || appointmentID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的预约ID"})
		return
	}
	appointment, err := loadAppointmentByID(config.DB, uint(appointmentID64))
	if err != nil || appointment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}
	if _, _, ok := checkMerchantAppointmentOwnership(c, *appointment); !ok {
		return
	}

	var input struct {
		Type   string `json:"type" binding:"required"`
		Value  int    `json:"value"`
		Reason string `json:"reason" binding:"required,max=255"`
		Remark string `json:"remark" binding:"max=255"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.Type = strings.TrimSpace(input.Type)
	input.Reason = strings.TrimSpace(input.Reason)
	input.Remark = strings.TrimSpace(input.Remark)
	if input.Reason == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "补偿原因不能为空"})
		return
	}
	switch input.Type {
	case "extra_times":
		if input.Value <= 0 || input.Value > 20 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "补偿次数范围应为 1-20"})
			return
		}
	case "extend_minutes":
		if input.Value < 5 || input.Value > 180 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "补偿时长范围应为 5-180 分钟"})
			return
		}
	case "discount_note", "other_note":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的补偿类型"})
		return
	}

	actorType, actorID := getAppointmentActor(c)
	now := time.Now()
	var compensation models.AppointmentCompensation
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		currentPtr, err := loadAppointmentByID(tx, appointment.ID)
		if err != nil {
			return err
		}
		if currentPtr == nil {
			return gorm.ErrRecordNotFound
		}
		current := *currentPtr
		compensation = models.AppointmentCompensation{
			AppointmentID:    current.ID,
			MerchantID:       current.MerchantID,
			UserID:           current.UserID,
			CardID:           current.CardID,
			ServiceSessionID: current.ServiceSessionID,
			Type:             input.Type,
			Value:            input.Value,
			Status:           "pending",
			Reason:           input.Reason,
			Remark:           input.Remark,
			CreatedByType:    actorType,
			CreatedByID:      actorID,
		}
		if err := tx.Create(&compensation).Error; err != nil {
			return err
		}

		// 不同补偿类型直接驱动对应资源：
		// 1. extra_times 立刻回充卡次数
		// 2. extend_minutes 直接延长当前服务会话
		// 3. discount_note / other_note 先形成正式补偿记录，供线下履约和后续稽核
		switch input.Type {
		case "extra_times":
			if err := tx.Model(&models.Card{}).Where("id = ?", current.CardID).Updates(map[string]interface{}{
				"total_times":  gorm.Expr("total_times + ?", input.Value),
				"remain_times": gorm.Expr("remain_times + ?", input.Value),
			}).Error; err != nil {
				return err
			}
		case "extend_minutes":
			if current.ServiceSessionID == nil {
				return apiErr{status: http.StatusBadRequest, msg: "当前预约没有关联服务会话，不能补时"}
			}
			var session models.ServiceSession
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", *current.ServiceSessionID).First(&session).Error; err != nil {
				return err
			}
			baseStatus := models.NormalizeSessionStatus(session.Status)
			if baseStatus != "serving" && baseStatus != "start_pending" && baseStatus != "delay_pending" && baseStatus != "appointment_waiting" {
				return apiErr{status: http.StatusBadRequest, msg: "当前阶段不能补时"}
			}
			updates := map[string]interface{}{
				"duration_minutes": gorm.Expr("duration_minutes + ?", input.Value),
			}
			if session.ScheduledFinishAt != nil {
				updates["scheduled_finish_at"] = session.ScheduledFinishAt.Add(time.Duration(input.Value) * time.Minute)
			}
			if err := tx.Model(&models.ServiceSession{}).Where("id = ?", session.ID).Updates(updates).Error; err != nil {
				return err
			}
		}

		noteText := compensationDisplayText(input.Type, input.Value, input.Remark)
		if err := tx.Model(&models.Appointment{}).Where("id = ?", current.ID).Updates(map[string]interface{}{
			"resolution_note": appendAppointmentResolutionNote(current.ResolutionNote, noteText),
		}).Error; err != nil {
			return err
		}

		compensation.Status = "applied"
		compensation.AppliedAt = &now
		return tx.Model(&models.AppointmentCompensation{}).Where("id = ?", compensation.ID).Updates(map[string]interface{}{
			"status":     "applied",
			"applied_at": &now,
		}).Error
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
	c.JSON(http.StatusOK, gin.H{"data": compensation})
}

func CancelAppointment(c *gin.Context) {
	id := c.Param("id")
	var appointment models.Appointment
	if err := config.DB.First(&appointment, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}

	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)
	if authType == "merchant" || authType == "staff" {
		if _, _, ok := checkMerchantAppointmentOwnership(c, appointment); !ok {
			return
		}
	} else {
		userIDAny, ok := c.Get("user_id")
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
			return
		}
		userID, ok := userIDAny.(uint)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
			return
		}
		if appointment.UserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权限：只能取消自己的预约"})
			return
		}
	}

	// 只允许取消待确认或已确认的预约
	if appointment.Status != "pending" && appointment.Status != "confirmed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "只能取消待确认或已确认的预约"})
		return
	}

	now := time.Now()
	if err := cancelAppointmentWithTime(&appointment, now); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "取消预约失败"})
		return
	}
	appointment.Status = "canceled"
	appointment.CanceledAt = &now
	c.JSON(http.StatusOK, gin.H{"data": appointment})
}

func GetQueueStatus(c *gin.Context) {
	merchantID, ok := ensureMerchantScope(c, "id")
	if !ok {
		return
	}

	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}

	// 待处理预约数
	var pendingCount int64
	config.DB.Model(&models.Appointment{}).
		Where("merchant_id = ? AND status = 'pending'", merchantID).
		Count(&pendingCount)

	// 今日核销数
	today := time.Now().Format("2006-01-02")
	var todayVerifyCount int64
	config.DB.Model(&models.Usage{}).
		Where("merchant_id = ? AND DATE(used_at) = ?", merchantID, today).
		Count(&todayVerifyCount)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"pending_appointments": pendingCount,
			"today_verify_count":   todayVerifyCount,
		},
	})
}

// GetAvailableTimeSlots 获取商户的可用预约时间段
func GetAvailableTimeSlots(c *gin.Context) {
	authType := c.GetString("auth_type")
	var merchantID uint
	var ok bool
	if authType == "user" {
		merchantID, ok = merchantIDFromRouteParam(c, "id")
	} else {
		merchantID, ok = ensureMerchantScope(c, "id")
	}
	if !ok {
		return
	}
	date := c.Query("date") // 格式: 2024-01-01
	projectIDStr := strings.TrimSpace(c.Query("project_id"))
	var projectID uint
	if projectIDStr != "" {
		pid, err := strconv.ParseUint(projectIDStr, 10, 64)
		if err != nil || pid == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目"})
			return
		}
		projectID = uint(pid)
	}
	log.Printf("GetAvailableTimeSlots: merchant_id=%d date=%s project_id=%d", merchantID, date, projectID)
	loc, locErr := time.LoadLocation("Asia/Shanghai")
	if locErr != nil {
		loc = time.Local
	}

	now := time.Now().In(loc)
	todayDate := now.Format("2006-01-02")
	tomorrowDate := now.Add(24 * time.Hour).Format("2006-01-02")
	if date == "" {
		if authType == "user" {
			date = tomorrowDate
		} else {
			date = todayDate
		}
	}

	// 验证日期格式
	if _, err := time.ParseInLocation("2006-01-02", date, loc); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "日期格式错误，应为 YYYY-MM-DD"})
		return
	}

	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}

	if !merchant.SupportAppointment {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该商户不支持预约功能"})
		return
	}

	if authType == "user" {
		if date != tomorrowDate {
			c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持预约明天"})
			return
		}
	} else {
		selectedDate, _ := time.ParseInLocation("2006-01-02", date, loc)
		if selectedDate.Before(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "不能查询过去的日期"})
			return
		}
	}
	serviceMinutes := 30
	if projectID > 0 {
		var project models.MerchantProject
		if err := config.DB.Where("id = ? AND merchant_id = ? AND is_active = ?", projectID, merchant.ID, true).First(&project).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目"})
			return
		}
		serviceMinutes = project.Duration
		log.Printf("GetAvailableTimeSlots: project_id=%d duration=%d service_minutes=%d", projectID, project.Duration, serviceMinutes)
		if serviceMinutes <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "项目时长无效"})
			return
		}
	} else {
		log.Printf("GetAvailableTimeSlots: project_id=0 use default service_minutes=%d", serviceMinutes)
	}

	// 懒更新：自动取消该商户已超时(>=35分钟)但未核销的预约，避免继续占用时间段
	overdueDeadline := now.Add(-35 * time.Minute)
	config.DB.Model(&models.Appointment{}).
		Where("merchant_id = ? AND status IN ('pending','confirmed') AND appointment_time IS NOT NULL AND appointment_time <= ?", merchantID, overdueDeadline).
		Updates(map[string]interface{}{
			"status":      "canceled",
			"canceled_at": now,
		})

	// 获取当天的所有预约（pending和confirmed状态）
	var appointments []models.Appointment
	// 使用字符串前缀匹配来查询日期，避免DATE()函数的兼容性问题
	datePrefix := date + " %"
	config.DB.Preload("User").Where("merchant_id = ? AND appointment_time LIKE ? AND status IN ('pending', 'confirmed')",
		merchantID, datePrefix).Order("appointment_time ASC").Find(&appointments)

	// 如果商户开启“客服”，则返回可选的专业客服列表（用于前端联动）
	// 说明：历史数据 role_type 可能为空，因此仅排除运营客服 role_type=operational
	var technicians []models.Technician
	if merchant.SupportCustomerServiceMode {
		config.DB.
			Joins("JOIN service_roles sr ON sr.id = technicians.service_role_id").
			Where("technicians.merchant_id = ? AND technicians.is_active = ? AND (sr.role_type IS NULL OR sr.role_type = '' OR sr.role_type <> ?)", merchantID, true, "operational").
			Order("technicians.id desc").
			Find(&technicians)
	}

	// 解析日期
	targetDate, _ := time.ParseInLocation("2006-01-02", date, loc)
	intervals, ok := getMerchantBusinessIntervalsForDate(merchant, targetDate)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "商户未设置营业时间"})
		return
	}

	// 生成所有可能的时间段（仅生成服务能在营业时间内完成的起始时刻）
	var allSlots []string
	for _, it := range intervals {
		// 最晚可开始时间 = 营业结束 - 服务时长
		latestStart := it.End.Add(-time.Duration(serviceMinutes) * time.Minute)
		for t := it.Start; !t.After(latestStart); t = t.Add(time.Duration(serviceMinutes) * time.Minute) {
			allSlots = append(allSlots, t.Format("2006-01-02 15:04:05"))
		}
	}

	// 标记已被占用的时间段（按各自预约项目时长占用）
	type TechnicianCandidate struct {
		TechnicianID         uint   `json:"technician_id"`
		AvailabilityState    string `json:"availability_state"`
		PredictedWaitMinutes int    `json:"predicted_wait_minutes"`
		AvailabilityReason   string `json:"availability_reason,omitempty"`
	}
	type TimeSlot struct {
		Time                 string                `json:"time"`
		Available            bool                  `json:"available"`
		UserName             string                `json:"user_name,omitempty"`
		TechnicianIDs        []uint                `json:"technician_ids,omitempty"`
		TechnicianCandidates []TechnicianCandidate `json:"technician_candidates,omitempty"`
	}

	var timeSlots []TimeSlot
	allTechIDs := make([]uint, 0)
	if len(technicians) > 0 {
		for _, t := range technicians {
			allTechIDs = append(allTechIDs, t.ID)
		}
	}
	for _, slot := range allSlots {
		slotTime, _ := time.Parse("2006-01-02 15:04:05", slot)
		available := true
		userName := ""
		availableTechIDs := make([]uint, 0)
		candidates := make([]TechnicianCandidate, 0, len(technicians))

		// 检查这个时间段是否与现有预约冲突
		for _, apt := range appointments {
			if apt.AppointmentTime == nil {
				continue
			}
			aptTime := *apt.AppointmentTime
			aptMinutes := getAppointmentServiceMinutes(merchant.ID, apt)

			aptEnd := aptTime.Add(time.Duration(aptMinutes) * time.Minute)
			slotEnd := slotTime.Add(time.Duration(serviceMinutes) * time.Minute)
			// 区间相交则冲突：[slotTime, slotEnd) 与 [aptTime, aptEnd)
			if slotTime.Before(aptEnd) && aptTime.Before(slotEnd) {
				// 若该预约未指定技师，则占用全部技师/时间段
				if apt.TechnicianID == nil {
					available = false
					if apt.User.Nickname != "" {
						userName = apt.User.Nickname
					}
					break
				}
			}
		}

		// 若没有“占用全部”的预约冲突，则为该时间段计算可预约技师列表
		if available && len(allTechIDs) > 0 {
			for _, tech := range technicians {
				availability, err := evaluateBookingTechnicianAvailability(config.DB, merchant, tech.ID, slotTime, serviceMinutes, 0)
				if err != nil {
					log.Printf("evaluate booking availability failed: merchant=%d tech=%d slot=%s err=%v", merchant.ID, tech.ID, slot, err)
					continue
				}
				candidates = append(candidates, TechnicianCandidate{
					TechnicianID:         tech.ID,
					AvailabilityState:    string(availability.State),
					PredictedWaitMinutes: availability.PredictedWaitMinutes,
					AvailabilityReason:   availability.Reason,
				})
				if availability.State != appointmentAvailabilityUnavailable {
					availableTechIDs = append(availableTechIDs, tech.ID)
				}
			}
		}

		// 仅返回可预约的时间段（排除已被预约/占用的时间段）
		if available {
			if len(allTechIDs) == 0 || len(availableTechIDs) > 0 {
				timeSlots = append(timeSlots, TimeSlot{
					Time:                 slot,
					Available:            true,
					UserName:             userName,
					TechnicianIDs:        availableTechIDs,
					TechnicianCandidates: candidates,
				})
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"date":            date,
			"service_minutes": serviceMinutes,
			"time_slots":      timeSlots,
			"technicians":     technicians,
		},
	})
}
