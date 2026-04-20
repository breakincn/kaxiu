package handlers

import (
	"errors"
	"kabao/config"
	"kabao/middleware"
	"kabao/models"
	"kabao/queue"
	"kabao/sessionflow"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type queuePendingItem struct {
	UsageID                      uint       `json:"usage_id"`
	QueueNo                      int        `json:"queue_no"`
	QueueCalledAt                *time.Time `json:"queue_called_at"`
	SessionID                    uint       `json:"session_id"`
	SessionStatus                string     `json:"session_status"`
	StartConfirmedAt             *time.Time `json:"start_confirmed_at"`
	StartPendingTimeoutSeconds   int        `json:"start_pending_timeout_seconds"`
	StartPendingRemainingSeconds int        `json:"start_pending_remaining_seconds"`
	StartedAt                    *time.Time `json:"started_at"`
	ScheduledFinishAt            *time.Time `json:"scheduled_finish_at"`
	DurationMinutes              int        `json:"duration_minutes"`
	TechnicianID                 *uint      `json:"technician_id"`
	TechnicianName               string     `json:"technician_name"`
	TechnicianAvailable          bool       `json:"technician_available"`
	TechnicianUnavailableReason  string     `json:"technician_unavailable_reason"`
	ProjectName                  string     `json:"project_name"`
	UserNickname                 string     `json:"user_nickname"`
	CreatedAt                    *time.Time `json:"created_at"`
	UpdatedAt                    *time.Time `json:"updated_at"`
}

type queueTimeoutWaitingItem struct {
	UsageID        uint       `json:"usage_id"`
	SessionID      uint       `json:"session_id"`
	SessionStatus  string     `json:"session_status"`
	TimeoutAt      *time.Time `json:"timeout_at"`
	TechnicianID   *uint      `json:"technician_id"`
	TechnicianName string     `json:"technician_name"`
	WindowNo       string     `json:"window_no"`
	ProjectName    string     `json:"project_name"`
	UserNickname   string     `json:"user_nickname"`
}

type queueCallInfoSession struct {
	SessionID                    uint       `json:"session_id"`
	Status                       string     `json:"status"`
	StartConfirmedAt             *time.Time `json:"start_confirmed_at"`
	StartPendingTimeoutSeconds   int        `json:"start_pending_timeout_seconds"`
	StartPendingRemainingSeconds int        `json:"start_pending_remaining_seconds"`
	InitialUsageID               uint       `json:"initial_usage_id"`
	ProjectName                  string     `json:"project_name"`
	StartedAt                    *time.Time `json:"started_at"`
	ScheduledFinishAt            *time.Time `json:"scheduled_finish_at"`
	DurationMinutes              int        `json:"duration_minutes"`
	CreatedAt                    *time.Time `json:"created_at"`
	UpdatedAt                    *time.Time `json:"updated_at"`
}

type queueCallInfo struct {
	WindowNo    string                `json:"window_no"`
	QueuePrefix string                `json:"queue_prefix"`
	QueueNo     int                   `json:"queue_no"`
	TrackingID  uint                  `json:"tracking_id"`
	Session     *queueCallInfoSession `json:"session"`
}

func isManualQueueMode(mode string) bool {
	return strings.TrimSpace(mode) == "manual"
}

func shouldExcludeFromPendingByStatus(normalizedStatus string) bool {
	s := strings.TrimSpace(normalizedStatus)
	return s == "finished" || s == "canceled" || s == "timeout_waiting" || s == "timeout_failed"
}

func normalizeStartDelaySeconds(v int) int {
	if v <= 0 {
		return 60
	}
	return v
}

func currentQueueScopedTechnicianID(c *gin.Context) *uint {
	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)
	if authType != "staff" {
		return nil
	}

	techIDAny, ok := c.Get("technician_id")
	if !ok {
		return nil
	}

	techID, _ := techIDAny.(uint)
	if techID == 0 {
		return nil
	}

	return &techID
}

type queuePendingTechMeta struct {
	Name              string
	QueuePaused       bool
	AttendanceStatus  string
	HasOpenAttendance bool
}

func resolveQueuePendingTechUnavailableReason(meta queuePendingTechMeta) string {
	if strings.TrimSpace(meta.Name) == "" && !meta.HasOpenAttendance {
		return "当前客服不可服务"
	}
	if meta.QueuePaused {
		return "当前客服已暂停叫号"
	}
	switch strings.TrimSpace(meta.AttendanceStatus) {
	case "":
		if !meta.HasOpenAttendance {
			return "当前客服已下班"
		}
	case "paused":
		return "当前客服已暂停服务"
	case "rest":
		return "当前客服已下班"
	}
	if !meta.HasOpenAttendance {
		return "当前客服已下班"
	}
	return ""
}

func listQueuePendingTechMeta(merchantID uint, techIDs []uint) map[uint]queuePendingTechMeta {
	out := make(map[uint]queuePendingTechMeta, len(techIDs))
	if merchantID == 0 || len(techIDs) == 0 {
		return out
	}

	uniqIDs := make([]uint, 0, len(techIDs))
	seen := make(map[uint]struct{}, len(techIDs))
	for _, id := range techIDs {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniqIDs = append(uniqIDs, id)
	}
	if len(uniqIDs) == 0 {
		return out
	}

	var techs []models.Technician
	_ = config.DB.
		Select("id", "name", "queue_paused").
		Where("merchant_id = ? AND id IN ?", merchantID, uniqIDs).
		Find(&techs).Error
	for _, tech := range techs {
		out[tech.ID] = queuePendingTechMeta{
			Name:        strings.TrimSpace(tech.Name),
			QueuePaused: tech.QueuePaused,
		}
	}

	start := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location())
	type attendanceLite struct {
		TechnicianID uint   `gorm:"column:technician_id"`
		Status       string `gorm:"column:status"`
	}
	var atts []attendanceLite
	_ = config.DB.
		Table("technician_attendances").
		Select("technician_id, status").
		Where("merchant_id = ? AND technician_id IN ? AND checked_in_at >= ? AND checked_out_at IS NULL", merchantID, uniqIDs, start).
		Order("id desc").
		Find(&atts).Error

	attSeen := make(map[uint]struct{}, len(atts))
	for _, att := range atts {
		if att.TechnicianID == 0 {
			continue
		}
		if _, ok := attSeen[att.TechnicianID]; ok {
			continue
		}
		attSeen[att.TechnicianID] = struct{}{}
		meta := out[att.TechnicianID]
		meta.AttendanceStatus = strings.TrimSpace(att.Status)
		meta.HasOpenAttendance = true
		out[att.TechnicianID] = meta
	}

	return out
}

func currentStaffRoleType(c *gin.Context) string {
	roleAny, ok := c.Get("service_role")
	if !ok {
		return ""
	}
	switch v := roleAny.(type) {
	case models.ServiceRole:
		return strings.TrimSpace(v.RoleType)
	case *models.ServiceRole:
		if v != nil {
			return strings.TrimSpace(v.RoleType)
		}
	}
	return ""
}

func supportsCurrentPendingReassign(merchant *models.Merchant) bool {
	if merchant == nil {
		return false
	}
	if merchant.SupportCustomerServiceMode {
		return true
	}
	return merchant.SupportQueue && merchant.SupportMultiCustomerService
}

func selectQueueReassignTarget(tx *gorm.DB, merchant *models.Merchant, excludeTechID uint, now time.Time) (*models.Technician, *models.TechnicianAttendance, error) {
	if tx == nil || merchant == nil || merchant.ID == 0 {
		return nil, nil, nil
	}
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	type idleTechRow struct {
		ID   uint   `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}
	var candidates []idleTechRow
	if err := tx.
		Table("technician_attendances ta").
		Select("t.id, t.name").
		Joins("JOIN technicians t ON t.id = ta.technician_id").
		Where("ta.merchant_id = ? AND ta.checked_in_at >= ? AND ta.checked_out_at IS NULL AND ta.status = ?", merchant.ID, start, "idle").
		Where("t.merchant_id = ? AND t.is_active = ? AND t.queue_paused = ?", merchant.ID, true, false).
		Where("t.id <> ?", excludeTechID).
		Order("ta.updated_at asc, ta.id asc").
		Find(&candidates).Error; err != nil {
		return nil, nil, err
	}

	activeStatuses := models.ExpandStatusesWithKnownPrefixes([]string{"start_pending", "delay_pending", "serving", "auto_finishing"})
	for _, cand := range candidates {
		if cand.ID == 0 {
			continue
		}

		var cnt int64
		activeSessionQuery := tx.Model(&models.ServiceSession{}).
			Where("merchant_id = ? AND status IN ?", merchant.ID, activeStatuses)
		activeSessionQuery = applyServiceSessionTechnicianFilter(activeSessionQuery, "service_sessions", cand.ID, false)
		if err := activeSessionQuery.Count(&cnt).Error; err != nil {
			return nil, nil, err
		}
		if cnt > 0 {
			continue
		}

		var att struct {
			ID           uint    `gorm:"column:id"`
			TechnicianID uint    `gorm:"column:technician_id"`
			Status       string  `gorm:"column:status"`
			NextStatus   *string `gorm:"column:next_status"`
		}
		attRes := tx.
			Model(&models.TechnicianAttendance{}).
			Select("id", "technician_id", "status", "next_status").
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("merchant_id = ? AND technician_id = ? AND checked_in_at >= ? AND checked_out_at IS NULL", merchant.ID, cand.ID, start).
			Order("id desc").
			Limit(1).
			Find(&att)
		if attRes.Error != nil || attRes.RowsAffected == 0 {
			continue
		}
		if strings.TrimSpace(att.Status) != "idle" {
			continue
		}

		var tech models.Technician
		if err := tx.Where("id = ? AND merchant_id = ?", cand.ID, merchant.ID).First(&tech).Error; err != nil {
			return nil, nil, err
		}
		if tech.QueuePaused || !tech.IsActive {
			continue
		}
		return &tech, &models.TechnicianAttendance{
			ID:           att.ID,
			TechnicianID: att.TechnicianID,
			Status:       att.Status,
			NextStatus:   att.NextStatus,
		}, nil
	}

	return nil, nil, nil
}

func releaseTechnicianAfterPendingReassign(tx *gorm.DB, merchantID uint, techID uint, now time.Time) error {
	if tx == nil || merchantID == 0 || techID == 0 {
		return nil
	}

	var tech models.Technician
	if err := tx.Where("id = ? AND merchant_id = ?", techID, merchantID).First(&tech).Error; err != nil {
		return err
	}

	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var att struct {
		ID         uint    `gorm:"column:id"`
		Status     string  `gorm:"column:status"`
		NextStatus *string `gorm:"column:next_status"`
	}
	res := tx.
		Model(&models.TechnicianAttendance{}).
		Select("id", "status", "next_status").
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("merchant_id = ? AND technician_id = ? AND checked_in_at >= ? AND checked_out_at IS NULL", merchantID, techID, start).
		Order("id desc").
		Limit(1).
		Find(&att)
	if res.Error != nil || res.RowsAffected == 0 {
		return res.Error
	}
	if strings.TrimSpace(att.Status) != "busy" {
		return nil
	}

	updates := map[string]interface{}{
		"next_status": nil,
	}
	if tech.QueuePaused || (att.NextStatus != nil && strings.TrimSpace(*att.NextStatus) == "paused") {
		updates["status"] = "paused"
	} else {
		updates["status"] = "idle"
	}
	return tx.Model(&models.TechnicianAttendance{}).Where("id = ?", att.ID).Updates(updates).Error
}

func isMerchantInBusinessHours(m *models.Merchant, now time.Time) bool {
	if m == nil {
		return true
	}

	// IsOpen 优先
	if !m.IsOpen {
		return false
	}

	parseHM := func(s string) (hour int, min int, ok bool) {
		s = strings.TrimSpace(s)
		if s == "" {
			return 0, 0, false
		}
		t, err := time.Parse("15:04", s)
		if err != nil {
			return 0, 0, false
		}
		return t.Hour(), t.Minute(), true
	}
	inRange := func(start, end string) bool {
		sh, sm, ok1 := parseHM(start)
		eh, em, ok2 := parseHM(end)
		if !ok1 || !ok2 {
			return false
		}
		curMin := now.Hour()*60 + now.Minute()
		startMin := sh*60 + sm
		endMin := eh*60 + em
		// 允许跨天营业（如 22:00-02:00）
		if startMin <= endMin {
			return curMin >= startMin && curMin <= endMin
		}
		return curMin >= startMin || curMin <= endMin
	}

	allDayStart := strings.TrimSpace(m.AllDayStart)
	allDayEnd := strings.TrimSpace(m.AllDayEnd)
	if allDayStart != "" && allDayEnd != "" {
		return inRange(allDayStart, allDayEnd)
	}

	if inRange(m.MorningStart, m.MorningEnd) {
		return true
	}
	if inRange(m.AfternoonStart, m.AfternoonEnd) {
		return true
	}
	if inRange(m.EveningStart, m.EveningEnd) {
		return true
	}

	// 未配置时间段：默认认为营业中（以 IsOpen 为准）
	return true
}

func finalizeSessionManual(tx *gorm.DB, merchant *models.Merchant, s *models.ServiceSession, now time.Time) error {
	if tx == nil || merchant == nil || s == nil {
		return nil
	}
	return sessionflow.FinishServiceSession(tx, s, merchant, now, sessionflow.FinishOptions{
		AllowedBaseStatuses: []string{"serving", "auto_finishing"},
		MarkQueueDone:       true,
	})
}

func assignNextSessionToTechnicianManual(tx *gorm.DB, merchant *models.Merchant, technicianID uint, now time.Time) (nextUsageID uint, assigned bool, err error) {
	if tx == nil || merchant == nil || merchant.ID == 0 || technicianID == 0 {
		return 0, false, nil
	}
	if queue.Default == nil {
		return 0, false, nil
	}
	date := now.Format("2006-01-02")

	{
		var cnt int64
		activeSessionQuery := tx.Model(&models.ServiceSession{}).
			Where("merchant_id = ? AND status IN ?", merchant.ID,
				models.ExpandStatusesWithKnownPrefixes([]string{"start_pending", "delay_pending", "serving", "auto_finishing"}))
		activeSessionQuery = applyServiceSessionTechnicianFilter(activeSessionQuery, "service_sessions", technicianID, false)
		if err := activeSessionQuery.Count(&cnt).Error; err != nil || cnt > 0 {
			return 0, false, nil
		}
	}

	// 策略1：优先从队列取下一个未叫号
	nextUsageID = queue.Default.CallNextUncalled(merchant.ID, date, queue.QueueTypeOnsite, now)
	if nextUsageID > 0 {
		// 尝试找到对应的可分配 session
		q := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("merchant_id = ? AND initial_usage_id = ? AND start_confirmed_at IS NULL", merchant.ID, nextUsageID)
		q = applyServiceSessionUnassignedFilter(q, "service_sessions")
		q = q.Where("status IN ?", models.ExpandStatusWithKnownPrefixes("staff_selecting"))

		var nextSession models.ServiceSession
		if err := q.Order("id desc").First(&nextSession).Error; err == nil {
			// 模式守卫：拒绝跨模式会话被叫号流程推进
			if err := models.ValidateSessionModeForEntry(&nextSession, merchant); err != nil {
				queue.Default.Uncall(merchant.ID, date, queue.QueueTypeOnsite, nextUsageID)
				return 0, false, nil
			}
			// 找到了，分配给当前技师
			updates := map[string]interface{}{
				"last_technician_id":            technicianID,
				"service_technician_ids":        models.MerchantProjectDefaultServiceTechnicianIDs{technicianID},
				"start_confirmed_technician_ids": models.MerchantProjectDefaultServiceTechnicianIDs{},
				"status":                        models.ApplyStatusPrefix(nextSession.Status, "start_pending"),
				"staff_select_entered_at":       nil,
				"staff_select_cooldown_until":   nil,
				"start_pending_timeout_seconds": config.MerchantQueueWaitingStartSeconds(merchant),
			}
			if err := tx.Model(&models.ServiceSession{}).
				Where("id = ? AND merchant_id = ? AND start_confirmed_at IS NULL", nextSession.ID, merchant.ID).
				Scopes(func(db *gorm.DB) *gorm.DB { return applyServiceSessionUnassignedFilter(db, "service_sessions") }).
				Updates(updates).Error; err != nil {
				queue.Default.Uncall(merchant.ID, date, queue.QueueTypeOnsite, nextUsageID)
				return 0, false, err
			}
			{
				start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
				var att models.TechnicianAttendance
				attRes := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
					Where("merchant_id = ? AND technician_id = ? AND checked_in_at >= ? AND checked_out_at IS NULL", merchant.ID, technicianID, start).
					Order("id desc").
					Limit(1).
					Find(&att)
				if attRes.Error == nil && attRes.RowsAffected > 0 {
					_ = tx.Model(&models.TechnicianAttendance{}).
						Where("id = ? AND merchant_id = ? AND technician_id = ? AND status = ?", att.ID, merchant.ID, technicianID, "idle").
						Updates(map[string]interface{}{"status": "busy"}).Error
				}
			}
			return nextUsageID, true, nil
		} else if err != gorm.ErrRecordNotFound {
			// 数据库错误
			queue.Default.Uncall(merchant.ID, date, queue.QueueTypeOnsite, nextUsageID)
			return 0, false, err
		}
		// 队列返回的号找不到对应的可分配 session（可能已被分配），回退队列状态
		queue.Default.Uncall(merchant.ID, date, queue.QueueTypeOnsite, nextUsageID)
	}

	// 策略2：队列无可叫号或队列返回的号已被分配，直接从数据库查找任意一个待分配的 session
	var anySession models.ServiceSession
	qAny := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("merchant_id = ? AND start_confirmed_at IS NULL", merchant.ID)
	qAny = applyServiceSessionUnassignedFilter(qAny, "service_sessions")
	qAny = qAny.
		Where("status IN ?", models.ExpandStatusWithKnownPrefixes("staff_selecting"))
	if err := qAny.Order("id asc").First(&anySession).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, false, nil
		}
		return 0, false, err
	}
	// 模式守卫：拒绝跨模式会话被叫号流程推进
	if err := models.ValidateSessionModeForEntry(&anySession, merchant); err != nil {
		return 0, false, nil
	}

	// 找到了一个待分配的 session，分配给当前技师
	updates := map[string]interface{}{
		"technician_id":                 technicianID,
		"last_technician_id":            technicianID,
		"service_technician_ids":        models.MerchantProjectDefaultServiceTechnicianIDs{technicianID},
		"start_confirmed_technician_ids": models.MerchantProjectDefaultServiceTechnicianIDs{},
		"status":                        models.ApplyStatusPrefix(anySession.Status, "start_pending"),
		"staff_select_entered_at":       nil,
		"staff_select_cooldown_until":   nil,
		"start_pending_timeout_seconds": config.MerchantQueueWaitingStartSeconds(merchant),
	}
	if err := tx.Model(&models.ServiceSession{}).
		Where("id = ? AND merchant_id = ? AND start_confirmed_at IS NULL", anySession.ID, merchant.ID).
		Scopes(func(db *gorm.DB) *gorm.DB { return applyServiceSessionUnassignedFilter(db, "service_sessions") }).
		Updates(updates).Error; err != nil {
		return 0, false, err
	}
	{
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		var att models.TechnicianAttendance
		attRes := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("merchant_id = ? AND technician_id = ? AND checked_in_at >= ? AND checked_out_at IS NULL", merchant.ID, technicianID, start).
			Order("id desc").
			Limit(1).
			Find(&att)
		if attRes.Error == nil && attRes.RowsAffected > 0 {
			_ = tx.Model(&models.TechnicianAttendance{}).
				Where("id = ? AND merchant_id = ? AND technician_id = ? AND status = ?", att.ID, merchant.ID, technicianID, "idle").
				Updates(map[string]interface{}{"status": "busy"}).Error
		}
	}

	// 如果这个 session 有对应的 usage_id，尝试在队列中标记为已叫（如果还没叫的话）
	if anySession.InitialUsageID > 0 {
		_ = queue.Default.CallNextUncalled(merchant.ID, date, queue.QueueTypeOnsite, now)
	}

	return anySession.InitialUsageID, true, nil
}

func assignNextForAllIdleTechsManual(tx *gorm.DB, merchant *models.Merchant, now time.Time) {
	if tx == nil || merchant == nil || merchant.ID == 0 {
		return
	}
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var ids []uint
	tx.Model(&models.TechnicianAttendance{}).
		Distinct().
		Where("merchant_id = ? AND checked_in_at >= ? AND checked_out_at IS NULL AND status = ?", merchant.ID, start, "idle").
		Pluck("technician_id", &ids)
	for _, tid := range ids {
		if tid == 0 {
			continue
		}
		assignNextSessionToTechnicianManual(tx, merchant, tid, now)
	}
}

// GetQueueCallingStatus 获取叫号状态
// GET /queue/calling-status
func GetQueueCallingStatus(c *gin.Context) {
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

	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}

	now := time.Now()
	isInBusinessHours := isMerchantInBusinessHours(&merchant, now)

	// 获取账号类型
	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)

	var technicianQueuePaused *bool
	if authType == "staff" {
		techIDAny, _ := c.Get("technician_id")
		techID, _ := techIDAny.(uint)
		if techID > 0 {
			var tech models.Technician
			if err := config.DB.First(&tech, techID).Error; err == nil {
				technicianQueuePaused = &tech.QueuePaused
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"queue_paused":            merchant.QueuePaused,
			"queue_ended_at":          merchant.QueueEndedAt,
			"queue_mode":              merchant.QueueMode,
			"support_queue":           merchant.SupportQueue,
			"is_in_business_hours":    isInBusinessHours,
			"technician_queue_paused": technicianQueuePaused,
		},
	})
}

// GetQueueCallInfo 获取“叫号信息”所需的最小数据（技师端服务页用）
// GET /queue/call-info
func GetQueueCallInfo(c *gin.Context) {
	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)

	merchantIDAny, ok := c.Get("merchant_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	merchantID, _ := merchantIDAny.(uint)
	if merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}

	out := queueCallInfo{QueuePrefix: strings.TrimSpace(merchant.QueuePrefix)}

	// 仅技师账号返回窗口号与当前会话信息
	if authType != "staff" {
		c.JSON(http.StatusOK, gin.H{"data": out})
		return
	}
	techIDAny, ok := c.Get("technician_id")
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅工作人员可查看"})
		return
	}
	technicianID, _ := techIDAny.(uint)
	if technicianID == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅工作人员可查看"})
		return
	}

	var tech models.Technician
	if err := config.DB.Where("id = ? AND merchant_id = ?", technicianID, merchantID).First(&tech).Error; err == nil {
		out.WindowNo = strings.TrimSpace(tech.WindowNo)
	}

	// 当前技师活跃会话（待上号/服务中）
	type sessLite struct {
		ID                         uint       `gorm:"column:id"`
		InitialUsageID             uint       `gorm:"column:initial_usage_id"`
		Status                     string     `gorm:"column:status"`
		StartConfirmedAt           *time.Time `gorm:"column:start_confirmed_at"`
		StartPendingTimeoutSeconds int        `gorm:"column:start_pending_timeout_seconds"`
		ProjectName                string     `gorm:"column:project_name"`
		StartedAt                  *time.Time `gorm:"column:started_at"`
		ScheduledFinishAt          *time.Time `gorm:"column:scheduled_finish_at"`
		DurationMinutes            int        `gorm:"column:duration_minutes"`
		CreatedAt                  *time.Time `gorm:"column:created_at"`
		UpdatedAt                  *time.Time `gorm:"column:updated_at"`
	}
	var s sessLite
	active := []string{"start_pending", "delay_pending", "serving", "auto_finishing"}
	if err := config.DB.
		Table("service_sessions ss").
		Select("ss.id, ss.initial_usage_id, ss.status, ss.start_confirmed_at, ss.start_pending_timeout_seconds, COALESCE(p.name,'') AS project_name, ss.started_at, ss.scheduled_finish_at, ss.duration_minutes, ss.created_at, ss.updated_at").
		Joins("LEFT JOIN merchant_projects p ON p.id = ss.project_id").
		Scopes(func(db *gorm.DB) *gorm.DB { return applyServiceSessionTechnicianFilter(db, "ss", technicianID, false) }).
		Where("ss.status IN ?", models.ExpandStatusesWithKnownPrefixes(active)).
		Order("ss.id desc").
		Limit(1).
		Scan(&s).Error; err == nil {
		if s.ID > 0 {
			sessModel := models.ServiceSession{
				Status:                     s.Status,
				StartConfirmedAt:           s.StartConfirmedAt,
				StartPendingTimeoutSeconds: s.StartPendingTimeoutSeconds,
				CreatedAt:                  s.CreatedAt,
				UpdatedAt:                  s.UpdatedAt,
			}
			out.Session = &queueCallInfoSession{
				SessionID:                    s.ID,
				Status:                       s.Status,
				StartConfirmedAt:             s.StartConfirmedAt,
				StartPendingTimeoutSeconds:   s.StartPendingTimeoutSeconds,
				StartPendingRemainingSeconds: computeStartPendingRemainingSecondsForSession(&sessModel, time.Now()),
				InitialUsageID:               s.InitialUsageID,
				ProjectName:                  s.ProjectName,
				StartedAt:                    s.StartedAt,
				ScheduledFinishAt:            s.ScheduledFinishAt,
				DurationMinutes:              s.DurationMinutes,
				CreatedAt:                    s.CreatedAt,
				UpdatedAt:                    s.UpdatedAt,
			}
			// 单号口径A：优先 usage_id
			if s.InitialUsageID > 0 {
				out.TrackingID = s.InitialUsageID
			} else {
				out.TrackingID = s.ID
			}
		}
	}

	// 计算当前会话对应的叫号号数（若支持队列且存在 usage_id）
	if merchant.SupportQueue && queue.Default != nil && out.Session != nil && out.Session.InitialUsageID > 0 {
		now := time.Now()
		date := now.Format("2006-01-02")
		if no, ok := queue.Default.GetNo(merchantID, date, queue.QueueTypeOnsite, out.Session.InitialUsageID); ok {
			out.QueueNo = no
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": out})
}

// GetQueuePendingList 获取全店待叫号列表（当天现场叫号队列）
// GET /queue/pending-list
func GetQueuePendingList(c *gin.Context) {
	merchantIDAny, ok := c.Get("merchant_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	merchantID, _ := merchantIDAny.(uint)
	if merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	if queue.Default == nil {
		c.JSON(http.StatusOK, gin.H{"data": []queuePendingItem{}})
		return
	}
	scopedTechID := currentQueueScopedTechnicianID(c)

	now := time.Now()
	date := strings.TrimSpace(c.Query("date"))
	if date == "" {
		date = now.Format("2006-01-02")
	}

	limit := 80
	if s := strings.TrimSpace(c.Query("limit")); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			if n > 0 {
				limit = n
			}
		}
	}
	if limit > 200 {
		limit = 200
	}

	snap := queue.Default.Snapshot(merchantID, date, queue.QueueTypeOnsite)
	if len(snap.Tickets) == 0 {
		c.JSON(http.StatusOK, gin.H{"data": []queuePendingItem{}})
		return
	}

	// Snapshot.Tickets 理论上已按号码顺序；这里兜底排序一次
	tickets := make([]queue.Ticket, 0, len(snap.Tickets))
	for _, t := range snap.Tickets {
		if t.ID == 0 || t.No <= 0 {
			continue
		}
		tickets = append(tickets, t)
	}
	sort.Slice(tickets, func(i, j int) bool { return tickets[i].No < tickets[j].No })
	if len(tickets) == 0 {
		c.JSON(http.StatusOK, gin.H{"data": []queuePendingItem{}})
		return
	}
	if len(tickets) > limit {
		tickets = tickets[:limit]
	}

	usageIDs := make([]uint, 0, len(tickets))
	for _, t := range tickets {
		usageIDs = append(usageIDs, t.ID)
	}

	// 仅展示进行中的 usage
	var activeUsageIDs []uint
	if err := config.DB.Model(&models.Usage{}).
		Where("merchant_id = ? AND id IN ? AND status = ?", merchantID, usageIDs, "in_progress").
		Pluck("id", &activeUsageIDs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	activeSet := make(map[uint]struct{}, len(activeUsageIDs))
	for _, id := range activeUsageIDs {
		activeSet[id] = struct{}{}
	}

	// 取每个 usage 最新的一条 session
	type sessLite struct {
		ID                         uint                                              `gorm:"column:id"`
		InitialUsageID             uint                                              `gorm:"column:initial_usage_id"`
		Status                     string                                            `gorm:"column:status"`
		StartConfirmedAt           *time.Time                                        `gorm:"column:start_confirmed_at"`
		StartPendingTimeoutSeconds int                                               `gorm:"column:start_pending_timeout_seconds"`
		StartedAt                  *time.Time                                        `gorm:"column:started_at"`
		ScheduledFinishAt          *time.Time                                        `gorm:"column:scheduled_finish_at"`
		DurationMinutes            int                                               `gorm:"column:duration_minutes"`
		LastTechnicianID           *uint                                             `gorm:"column:last_technician_id"`
		ServiceTechnicianIDs       models.MerchantProjectDefaultServiceTechnicianIDs `gorm:"column:service_technician_ids"`
		ProjectName                string                                            `gorm:"column:project_name"`
		UserNickname               string                                            `gorm:"column:user_nickname"`
		CreatedAt                  *time.Time                                        `gorm:"column:created_at"`
		UpdatedAt                  *time.Time                                        `gorm:"column:updated_at"`
	}

	sub := config.DB.
		Table("service_sessions").
		Select("MAX(id) AS id").
		Where("merchant_id = ? AND initial_usage_id IN ?", merchantID, usageIDs).
		Group("initial_usage_id")

	var sessions []sessLite
	if err := config.DB.
		Table("service_sessions ss").
		Select("ss.id, ss.initial_usage_id, ss.status, ss.start_confirmed_at, ss.start_pending_timeout_seconds, ss.started_at, ss.scheduled_finish_at, ss.duration_minutes, ss.last_technician_id, ss.service_technician_ids, COALESCE(p.name,'') AS project_name, COALESCE(u.nickname,'') AS user_nickname, ss.created_at, ss.updated_at").
		Joins("LEFT JOIN merchant_projects p ON p.id = ss.project_id").
		Joins("LEFT JOIN users u ON u.id = ss.user_id").
		Where("ss.id IN (?)", sub).
		Find(&sessions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	byUsageIDSession := make(map[uint]sessLite, len(sessions))
	techIDs := make([]uint, 0, len(sessions))
	for _, s := range sessions {
		if s.InitialUsageID == 0 {
			continue
		}
		sessionModel := models.ServiceSession{
			LastTechnicianID:     s.LastTechnicianID,
			ServiceTechnicianIDs: s.ServiceTechnicianIDs,
		}
		assignedTechIDs := serviceSessionAssignedTechnicianIDs(&sessionModel)
		if scopedTechID != nil {
			if !uintIDInSlice(assignedTechIDs, *scopedTechID) {
				continue
			}
		}
		byUsageIDSession[s.InitialUsageID] = s
		techIDs = append(techIDs, assignedTechIDs...)
		if len(techIDs) > 0 {
			techIDs = uniqueOrderedUintIDs(techIDs)
		}
	}
	techMeta := listQueuePendingTechMeta(merchantID, techIDs)

	out := make([]queuePendingItem, 0, len(tickets))
	for _, t := range tickets {
		if _, ok := activeSet[t.ID]; !ok {
			continue
		}
		s, ok := byUsageIDSession[t.ID]
		if !ok {
			continue
		}
		ns := models.NormalizeSessionStatus(s.Status)
		if shouldExcludeFromPendingByStatus(ns) {
			continue
		}
		sessModel := models.ServiceSession{
			Status:                     s.Status,
			StartConfirmedAt:           s.StartConfirmedAt,
			StartPendingTimeoutSeconds: s.StartPendingTimeoutSeconds,
			CreatedAt:                  s.CreatedAt,
			UpdatedAt:                  s.UpdatedAt,
		}
		technicianName := ""
		technicianAvailable := true
		technicianUnavailableReason := ""
		var primaryTechID uint
		if len(s.ServiceTechnicianIDs) > 0 {
			primaryTechID = s.ServiceTechnicianIDs[0]
		} else if s.LastTechnicianID != nil && *s.LastTechnicianID > 0 {
			primaryTechID = *s.LastTechnicianID
		}
		if primaryTechID > 0 {
			meta := techMeta[primaryTechID]
			technicianName = meta.Name
			technicianUnavailableReason = resolveQueuePendingTechUnavailableReason(meta)
			technicianAvailable = technicianUnavailableReason == ""
		}
		var primaryTechIDPtr *uint
		if primaryTechID > 0 {
			primaryTechIDCopy := primaryTechID
			primaryTechIDPtr = &primaryTechIDCopy
		}
		out = append(out, queuePendingItem{
			UsageID:                      t.ID,
			QueueNo:                      t.No,
			QueueCalledAt:                t.CalledAt,
			SessionID:                    s.ID,
			SessionStatus:                s.Status,
			StartConfirmedAt:             s.StartConfirmedAt,
			StartPendingTimeoutSeconds:   s.StartPendingTimeoutSeconds,
			StartPendingRemainingSeconds: computeStartPendingRemainingSecondsForSession(&sessModel, now),
			StartedAt:                    s.StartedAt,
			ScheduledFinishAt:            s.ScheduledFinishAt,
			DurationMinutes:              s.DurationMinutes,
			TechnicianID:                 primaryTechIDPtr,
			TechnicianName:               technicianName,
			TechnicianAvailable:          technicianAvailable,
			TechnicianUnavailableReason:  technicianUnavailableReason,
			ProjectName:                  s.ProjectName,
			UserNickname:                 s.UserNickname,
			CreatedAt:                    s.CreatedAt,
			UpdatedAt:                    s.UpdatedAt,
		})
	}

	// 若过滤后为空，也按空数组返回
	c.JSON(http.StatusOK, gin.H{"data": out})
}

// GetQueueTimeoutWaitingList 获取全店“超时过号等待”列表（用于手动叫号插队窗口展示）
// GET /queue/timeout-waiting-list
func GetQueueTimeoutWaitingList(c *gin.Context) {
	merchantIDAny, ok := c.Get("merchant_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	merchantID, _ := merchantIDAny.(uint)
	if merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	scopedTechID := currentQueueScopedTechnicianID(c)

	limit := 80
	if s := strings.TrimSpace(c.Query("limit")); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			if n > 0 {
				limit = n
			}
		}
	}
	if limit > 200 {
		limit = 200
	}

	// 仅返回 usage 仍处于 in_progress 的 timeout_waiting 会话
	// 注意：timeout_waiting 可能带前缀（cs_/qs_/...），统一使用 ExpandStatusWithKnownPrefixes
	type row struct {
		ID             uint       `gorm:"column:id"`
		InitialUsageID uint       `gorm:"column:initial_usage_id"`
		Status         string     `gorm:"column:status"`
		TimeoutAt      *time.Time `gorm:"column:timeout_at"`
		LastTechnician *uint      `gorm:"column:last_technician_id"`
		TechnicianName string     `gorm:"column:technician_name"`
		WindowNo       string     `gorm:"column:window_no"`
		ProjectName    string     `gorm:"column:project_name"`
		UserNickname   string     `gorm:"column:user_nickname"`
	}

	var rows []row
	q := config.DB.
		Table("service_sessions ss").
		Select(strings.Join([]string{
			"ss.id",
			"ss.initial_usage_id",
			"ss.status",
			"ss.start_timeout_last_at AS timeout_at",
			"ss.last_technician_id",
			"COALESCE(t.name,'') AS technician_name",
			"COALESCE(t.window_no,'') AS window_no",
			"COALESCE(p.name,'') AS project_name",
			"COALESCE(u.nickname,'') AS user_nickname",
		}, ", ")).
		Joins("JOIN usages ug ON ug.id = ss.initial_usage_id AND ug.status = 'in_progress'").
		Joins("LEFT JOIN technicians t ON t.id = ss.last_technician_id").
		Joins("LEFT JOIN merchant_projects p ON p.id = ss.project_id").
		Joins("LEFT JOIN users u ON u.id = ss.user_id").
		Where("ss.merchant_id = ? AND ss.initial_usage_id > 0", merchantID).
		Where("ss.status IN ?", models.ExpandStatusWithKnownPrefixes("timeout_waiting")).
		Order("ss.id desc").
		Limit(limit)
	if scopedTechID != nil {
		q = q.Where("ss.last_technician_id = ?", *scopedTechID)
	}
	if err := q.Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	out := make([]queueTimeoutWaitingItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, queueTimeoutWaitingItem{
			UsageID:        r.InitialUsageID,
			SessionID:      r.ID,
			SessionStatus:  r.Status,
			TimeoutAt:      r.TimeoutAt,
			TechnicianID:   r.LastTechnician,
			TechnicianName: strings.TrimSpace(r.TechnicianName),
			WindowNo:       strings.TrimSpace(r.WindowNo),
			ProjectName:    r.ProjectName,
			UserNickname:   r.UserNickname,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": out})
}

// EnqueueOnsiteUsages 补偿：将指定 usage_id 补入现场叫号队列（当日）
// POST /queue/enqueue-onsite
// 仅用于历史数据未入队的补偿。
func EnqueueOnsiteUsages(c *gin.Context) {
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
		UsageIDs  []uint `json:"usage_ids"`
		CallNext  bool   `json:"call_next"`
		StartNo   *int   `json:"start_no"`
		QueueType string `json:"queue_type"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(input.UsageIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "usage_ids 不能为空"})
		return
	}

	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}
	if !merchant.SupportQueue {
		c.JSON(http.StatusBadRequest, gin.H{"error": "商户未开启叫号功能"})
		return
	}
	if merchant.QueueMode != "auto" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅自动叫号模式支持补入队列"})
		return
	}
	if queue.Default == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "叫号服务未初始化"})
		return
	}

	qt := queue.QueueTypeOnsite
	if queueType := strings.TrimSpace(input.QueueType); queueType != "" && queueType != string(queue.QueueTypeOnsite) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持现场队列"})
		return
	}

	now := time.Now()
	date := now.Format("2006-01-02")
	startNo := merchant.QueueStartNo
	if input.StartNo != nil && *input.StartNo > 0 {
		startNo = *input.StartNo
	}
	autoCallFirst := false

	createdMap := make(map[uint]bool, len(input.UsageIDs))
	for _, uid := range input.UsageIDs {
		if uid == 0 {
			continue
		}
		// 校验 usage 属于本商户且仍在进行中，并存在对应 service_session
		var u models.Usage
		if err := config.DB.Where("id = ? AND merchant_id = ?", uid, merchantID).First(&u).Error; err != nil {
			createdMap[uid] = false
			continue
		}
		if u.Status != "in_progress" {
			createdMap[uid] = false
			continue
		}
		var ss models.ServiceSession
		if err := config.DB.Where("merchant_id = ? AND initial_usage_id = ?", merchantID, uid).Order("id desc").First(&ss).Error; err != nil {
			createdMap[uid] = false
			continue
		}

		tk, created := queue.Default.Enqueue(merchantID, date, qt, uid, startNo, autoCallFirst, now)
		createdMap[uid] = created
		if os.Getenv("KABAO_QUEUE_DEBUG") == "1" {
			log.Printf("[queue-debug] compensate enqueue: merchant=%d date=%s usage_id=%d created=%v queue_no=%d called_at=%v\n", merchantID, date, uid, created, tk.No, tk.CalledAt)
		}
		_ = tk
	}

	var nextUsageID uint
	if input.CallNext {
		nextUsageID = queue.Default.CallNextUncalled(merchantID, date, qt, now)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"date":          date,
			"queue_type":    string(qt),
			"created_map":   createdMap,
			"next_usage_id": nextUsageID,
		},
	})
}

// UpdateQueueCallingStatus 更新商户叫号全局状态（开始/暂停）
// PUT /queue/calling-status
func UpdateQueueCallingStatus(c *gin.Context) {
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
		QueuePaused *bool `json:"queue_paused"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if input.QueuePaused == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "queue_paused 不能为空"})
		return
	}

	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}

	now := time.Now()
	// 恢复叫号：清空结束叫号时间，避免结束叫号后的自动收尾误触发
	if !*input.QueuePaused {
		updates := map[string]interface{}{
			"queue_paused":   false,
			"queue_ended_at": nil,
		}
		if err := config.DB.Model(&merchant).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"queue_paused":   false,
				"queue_ended_at": nil,
			},
		})
		return
	}
	// 打烊时：暂停叫号升级为结束叫号，并记录 queue_ended_at
	if *input.QueuePaused && !isMerchantInBusinessHours(&merchant, now) {
		updates := map[string]interface{}{
			"queue_paused":   true,
			"queue_ended_at": now,
		}
		if err := config.DB.Model(&merchant).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"queue_paused":   true,
				"queue_ended_at": now,
			},
		})
		return
	}

	if err := config.DB.Model(&merchant).Update("queue_paused", *input.QueuePaused).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"queue_paused": *input.QueuePaused,
		},
	})
}

// UpdateTechnicianQueuePaused 更新技师自己的叫号状态（开始/暂停）
// PUT /technician/queue-paused
func UpdateTechnicianQueuePaused(c *gin.Context) {
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

	// 必须是技师账号
	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)
	if authType != "staff" {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅工作人员账号可操作"})
		return
	}

	techIDAny, _ := c.Get("technician_id")
	techID, _ := techIDAny.(uint)
	if techID == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "获取工作人员信息失败"})
		return
	}

	var input struct {
		QueuePaused *bool `json:"queue_paused"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if input.QueuePaused == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "queue_paused 不能为空"})
		return
	}

	var tech models.Technician
	if err := config.DB.Where("id = ? AND merchant_id = ?", techID, merchantID).First(&tech).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "工作人员不存在"})
		return
	}

	// 若已打烊：技师点击“暂停叫号”时升级为“结束叫号”，写入 merchants.queue_ended_at
	if *input.QueuePaused {
		var merchant models.Merchant
		if err := config.DB.First(&merchant, merchantID).Error; err == nil {
			now := time.Now()
			if !isMerchantInBusinessHours(&merchant, now) {
				_ = config.DB.Model(&merchant).Updates(map[string]interface{}{
					"queue_paused":   true,
					"queue_ended_at": now,
				}).Error
			}
		}
	}

	if err := config.DB.Model(&tech).Update("queue_paused", *input.QueuePaused).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"queue_paused": *input.QueuePaused,
		},
	})
}

// GetTechnicianQueuePaused 获取技师自己的叫号状态
// GET /technician/queue-paused
func GetTechnicianQueuePaused(c *gin.Context) {
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

	// 必须是技师账号
	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)
	if authType != "staff" {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅工作人员账号可操作"})
		return
	}

	techIDAny, _ := c.Get("technician_id")
	techID, _ := techIDAny.(uint)
	if techID == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "获取工作人员信息失败"})
		return
	}

	var tech models.Technician
	if err := config.DB.Where("id = ? AND merchant_id = ?", techID, merchantID).First(&tech).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "工作人员不存在"})
		return
	}

	// 同时获取商户的全局叫号状态
	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"queue_paused":          tech.QueuePaused,
			"merchant_queue_paused": merchant.QueuePaused,
		},
	})
}

// 未使用的变量占位
var _ = middleware.RequirePermission

func promoteManualSingleCalledSession(tx *gorm.DB, merchant *models.Merchant, usageID uint, now time.Time) {
	if tx == nil || merchant == nil || merchant.ID == 0 || usageID == 0 {
		return
	}
	if merchant.QueueMode != "manual" || merchant.SupportMultiCustomerService {
		return
	}

	date := now.Format("2006-01-02")
	var s models.ServiceSession
	q := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("merchant_id = ? AND initial_usage_id = ?", merchant.ID, usageID).
		Where("status IN ?", models.ExpandStatusWithKnownPrefixes("staff_selecting")).
		Order("id desc")
	if err := q.First(&s).Error; err != nil {
		if queue.Default != nil {
			queue.Default.Uncall(merchant.ID, date, queue.QueueTypeOnsite, usageID)
		}
		return
	}
	// 模式守卫：拒绝跨模式会话被叫号流程推进
	if err := models.ValidateSessionModeForEntry(&s, merchant); err != nil {
		if queue.Default != nil {
			queue.Default.Uncall(merchant.ID, date, queue.QueueTypeOnsite, usageID)
		}
		return
	}

	delaySeconds := normalizeStartDelaySeconds(merchant.StartDelaySeconds)
	startAt := now.Add(time.Duration(delaySeconds) * time.Second)
	updates := map[string]interface{}{
		"start_confirmed_at":  now,
		"scheduled_start_at":  startAt,
		"status":              models.ApplyStatusPrefix(s.Status, "delay_pending"),
		"start_delay_seconds": delaySeconds,
	}
	if err := tx.Model(&models.ServiceSession{}).
		Where("id = ? AND start_confirmed_at IS NULL AND status IN ?", s.ID, models.ExpandStatusWithKnownPrefixes("staff_selecting")).
		Updates(updates).Error; err != nil {
		if queue.Default != nil {
			queue.Default.Uncall(merchant.ID, date, queue.QueueTypeOnsite, usageID)
		}
		return
	}
}

// TriggerNextCalling 触发下一个叫号（手动叫号模式下使用）
// POST /queue/call-next
// 运营客服点击"开始叫号"时：恢复商户叫号 + 触发下一个
// 专业客服点击"开始叫号"时：恢复商户叫号 + 触发下一个
// 扫码结单时：判断是否有权限，若有则触发下一个
func TriggerNextCalling(c *gin.Context) {
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

	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}

	// 检查是否开启了叫号
	if !merchant.SupportQueue {
		c.JSON(http.StatusBadRequest, gin.H{"error": "商户未开启叫号功能"})
		return
	}
	if !isManualQueueMode(merchant.QueueMode) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前非人工叫号模式"})
		return
	}

	// 检查商户是否暂停叫号，若暂停则先恢复
	// 恢复叫号时也需要清空 queue_ended_at：否则调度器会把 queue_ended_at 最近 15 分钟内的商户批量收尾，导致“待叫号”刷新后变“完成”
	if merchant.QueuePaused || merchant.QueueEndedAt != nil {
		updates := map[string]interface{}{
			"queue_paused":   false,
			"queue_ended_at": nil,
		}
		if err := config.DB.Model(&merchant).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "恢复叫号失败"})
			return
		}
	}

	// 触发下一个叫号
	if queue.Default == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "叫号服务未初始化"})
		return
	}

	now := time.Now()
	if merchant.SupportMultiCustomerService {
		_ = config.DB.Transaction(func(tx *gorm.DB) error {
			assignNextForAllIdleTechsManual(tx, &merchant, now)
			return nil
		})
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"next_usage_id": uint(0),
				"queue_paused":  false,
			},
		})
		return
	}

	date := now.Format("2006-01-02")
	nextUsageID := queue.Default.CallNextUncalled(merchant.ID, date, queue.QueueTypeOnsite, now)
	if nextUsageID > 0 {
		_ = config.DB.Transaction(func(tx *gorm.DB) error {
			promoteManualSingleCalledSession(tx, &merchant, nextUsageID, now)
			return nil
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"next_usage_id": nextUsageID,
			"queue_paused":  false,
		},
	})
}

// TriggerContinueCalling 手动叫号：技师点击“继续叫号”= 完成当前服务 + 推进下一号
// POST /queue/continue-call
func TriggerContinueCalling(c *gin.Context) {
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

	// 必须是技师账号
	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)
	if authType != "staff" {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅工作人员账号可操作"})
		return
	}
	techIDAny, _ := c.Get("technician_id")
	techID, _ := techIDAny.(uint)
	if techID == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "获取工作人员信息失败"})
		return
	}

	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}
	if !merchant.SupportQueue {
		c.JSON(http.StatusBadRequest, gin.H{"error": "商户未开启叫号功能"})
		return
	}
	if strings.TrimSpace(merchant.QueueMode) != "manual" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前非人工叫号模式"})
		return
	}
	if merchant.QueuePaused {
		c.JSON(http.StatusBadRequest, gin.H{"error": "叫号已暂停"})
		return
	}
	if queue.Default == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "叫号服务未初始化"})
		return
	}

	// 检查技师是否暂停了自己的叫号
	var tech models.Technician
	if err := config.DB.Where("id = ? AND merchant_id = ?", techID, merchantID).First(&tech).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "工作人员不存在"})
		return
	}
	if tech.QueuePaused {
		c.JSON(http.StatusBadRequest, gin.H{"error": "您的叫号已暂停"})
		return
	}

	now := time.Now()
	date := now.Format("2006-01-02")

	out := gin.H{
		"finished_session_id": uint(0),
		"next_usage_id":       uint(0),
		"assigned":            false,
		"reason":              "",
	}

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		// 锁定该技师当前进行中的会话
		var s models.ServiceSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("merchant_id = ? AND technician_id = ? AND status IN ? AND start_confirmed_at IS NOT NULL", merchantID, techID, models.ExpandStatusesWithKnownPrefixes([]string{"serving", "auto_finishing"})).
			Order("id desc").
			First(&s).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				// 若当前技师已有待上号(start_pending)的会话：
				// - 倒计时未结束：提示等待（前端置灰继续叫号按钮）
				// - 倒计时结束：允许跳过该号并继续叫下一个（不退核销，允许后续扫码回补）
				var pending models.ServiceSession
				err2 := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
					Where("merchant_id = ? AND technician_id = ? AND status IN ? AND start_confirmed_at IS NULL", merchantID, techID, models.ExpandStatusWithKnownPrefixes("start_pending")).
					Order("id desc").
					First(&pending).Error
				if err2 == nil {
					baseAt := pending.UpdatedAt
					if baseAt == nil {
						baseAt = pending.CreatedAt
					}
					if baseAt == nil {
						out["reason"] = "当前已有待上号用户，请先扫码上号"
						out["pending_session_id"] = pending.ID
						return nil
					}
					timeout := config.StartPendingTimeout()
					if pending.StartPendingTimeoutSeconds > 0 {
						timeout = time.Duration(pending.StartPendingTimeoutSeconds) * time.Second
					}
					deadline := baseAt.Add(timeout)
					remain := int(deadline.Sub(now).Seconds())
					if remain > 0 {
						out["reason"] = "当前已有待上号用户，请等待倒计时结束后再继续叫号"
						out["pending_session_id"] = pending.ID
						out["need_wait"] = true
						out["pending_remaining_seconds"] = remain
						out["pending_timeout_seconds"] = int(timeout.Seconds())
						return nil
					}

					// 倒计时已到：跳过该号（不退核销），释放窗口，继续分配下一号
					skipUpdates := map[string]interface{}{
						"status":                        models.ApplyStatusPrefix(pending.Status, "timeout_waiting"),
						"last_technician_id":            techID,
						"technician_id":                 nil,
						"staff_select_entered_at":       nil,
						"staff_select_cooldown_until":   nil,
						"start_pending_timeout_seconds": 0,
						"start_timeout_count":           gorm.Expr("start_timeout_count + ?", 1),
						"start_timeout_last_at":         now,
					}
					if err3 := tx.Model(&models.ServiceSession{}).
						Where("id = ? AND merchant_id = ? AND status IN ? AND start_confirmed_at IS NULL", pending.ID, merchantID, models.ExpandStatusWithKnownPrefixes("start_pending")).
						Updates(skipUpdates).Error; err3 != nil {
						return err3
					}
					if pending.InitialUsageID > 0 && queue.Default != nil {
						queue.Default.Uncall(merchant.ID, date, queue.QueueTypeOnsite, pending.InitialUsageID)
					}
					out["skipped_session_id"] = pending.ID
				} else if err2 != gorm.ErrRecordNotFound {
					return err2
				}

				// 没有进行中服务：允许“继续叫号”作为“推进下一号/分配下一位”的触发
				if !merchant.SupportMultiCustomerService {
					nextUsageID := queue.Default.CallNextUncalled(merchant.ID, date, queue.QueueTypeOnsite, now)
					if nextUsageID > 0 {
						promoteManualSingleCalledSession(tx, &merchant, nextUsageID, now)
					}
					out["next_usage_id"] = nextUsageID
					out["assigned"] = false
					if nextUsageID == 0 {
						out["reason"] = "当前无进行中的服务，且暂无可叫号的用户"
					}
					return nil
				}

				nextUsageID, assigned, err := assignNextSessionToTechnicianManual(tx, &merchant, techID, now)
				if err != nil {
					return err
				}
				out["next_usage_id"] = nextUsageID
				out["assigned"] = assigned
				if nextUsageID == 0 {
					out["reason"] = "当前无进行中的服务，且暂无可分配的用户"
				}
				return nil
			}
			return err
		}
		out["finished_session_id"] = s.ID
		// 模式守卫：拒绝跨模式操作
		if err := models.ValidateSessionModeForEntry(&s, &merchant); err != nil {
			return apiErr{status: http.StatusBadRequest, msg: err.Error()}
		}

		// 检查服务是否达到项目设定的服务时长
		if s.StartedAt != nil && s.DurationMinutes > 0 {
			servedMinutes := int(now.Sub(*s.StartedAt).Minutes())
			if servedMinutes < s.DurationMinutes {
				// 未达到服务时长，返回二次确认信息，不执行结束操作
				out["need_confirm"] = true
				out["served_minutes"] = servedMinutes
				out["required_minutes"] = s.DurationMinutes
				out["remaining_minutes"] = s.DurationMinutes - servedMinutes
				return nil
			}
		}

		// 调试日志：检查为什么没有触发二次确认
		if s.StartedAt == nil {
			log.Printf("[DEBUG] ServiceSession %d: StartedAt is nil, skipping duration check", s.ID)
		} else if s.DurationMinutes <= 0 {
			log.Printf("[DEBUG] ServiceSession %d: DurationMinutes is %d, skipping duration check", s.ID, s.DurationMinutes)
		} else {
			servedMinutes := int(now.Sub(*s.StartedAt).Minutes())
			log.Printf("[DEBUG] ServiceSession %d: served %d minutes, required %d minutes, no confirmation needed", s.ID, servedMinutes, s.DurationMinutes)
		}

		if err := finalizeSessionManual(tx, &merchant, &s, now); err != nil {
			return err
		}

		if !merchant.SupportMultiCustomerService {
			nextUsageID := queue.Default.CallNextUncalled(merchant.ID, date, queue.QueueTypeOnsite, now)
			if nextUsageID > 0 {
				promoteManualSingleCalledSession(tx, &merchant, nextUsageID, now)
			}
			out["next_usage_id"] = nextUsageID
			out["assigned"] = false
			return nil
		}

		nextUsageID, assigned, err := assignNextSessionToTechnicianManual(tx, &merchant, techID, now)
		if err != nil {
			return err
		}
		out["next_usage_id"] = nextUsageID
		out["assigned"] = assigned
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

	c.JSON(http.StatusOK, gin.H{"data": out})
}

// TriggerContinueCallingForce 手动叫号：技师点击“继续叫号”强制结束 = 强制完成当前服务 + 推进下一号
// POST /queue/continue-call-force
func TriggerContinueCallingForce(c *gin.Context) {
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

	// 必须是技师账号
	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)
	if authType != "staff" {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅工作人员账号可操作"})
		return
	}
	techIDAny, _ := c.Get("technician_id")
	techID, _ := techIDAny.(uint)
	if techID == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "获取工作人员信息失败"})
		return
	}

	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}
	if !merchant.SupportQueue {
		c.JSON(http.StatusBadRequest, gin.H{"error": "商户未开启叫号功能"})
		return
	}
	if strings.TrimSpace(merchant.QueueMode) != "manual" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前非人工叫号模式"})
		return
	}
	if merchant.QueuePaused {
		c.JSON(http.StatusBadRequest, gin.H{"error": "叫号已暂停"})
		return
	}
	if queue.Default == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "叫号服务未初始化"})
		return
	}

	// 检查技师是否暂停了自己的叫号
	var tech models.Technician
	if err := config.DB.Where("id = ? AND merchant_id = ?", techID, merchantID).First(&tech).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "工作人员不存在"})
		return
	}
	if tech.QueuePaused {
		c.JSON(http.StatusBadRequest, gin.H{"error": "您的叫号已暂停"})
		return
	}

	now := time.Now()
	date := now.Format("2006-01-02")

	out := gin.H{
		"finished_session_id": uint(0),
		"next_usage_id":       uint(0),
		"assigned":            false,
		"reason":              "",
	}

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		// 锁定该技师当前进行中的会话
		var s models.ServiceSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("merchant_id = ? AND technician_id = ? AND status IN ? AND start_confirmed_at IS NOT NULL", merchantID, techID, models.ExpandStatusesWithKnownPrefixes([]string{"serving", "auto_finishing"})).
			Order("id desc").
			First(&s).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				out["reason"] = "当前无进行中的服务"
				return nil
			}
			return err
		}
		out["finished_session_id"] = s.ID
		// 模式守卫：拒绝跨模式操作
		if err := models.ValidateSessionModeForEntry(&s, &merchant); err != nil {
			return apiErr{status: http.StatusBadRequest, msg: err.Error()}
		}

		// 强制结束，不检查服务时长
		if err := finalizeSessionManual(tx, &merchant, &s, now); err != nil {
			return err
		}

		if !merchant.SupportMultiCustomerService {
			nextUsageID := queue.Default.CallNextUncalled(merchant.ID, date, queue.QueueTypeOnsite, now)
			if nextUsageID > 0 {
				promoteManualSingleCalledSession(tx, &merchant, nextUsageID, now)
			}
			out["next_usage_id"] = nextUsageID
			out["assigned"] = false
			return nil
		}

		nextUsageID, assigned, err := assignNextSessionToTechnicianManual(tx, &merchant, techID, now)
		if err != nil {
			return err
		}
		out["next_usage_id"] = nextUsageID
		out["assigned"] = assigned
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

	c.JSON(http.StatusOK, gin.H{"data": out})
}

// TriggerNextCallingOnFinish 扫码结单时触发下一个叫号（手动叫号模式下使用）
// POST /queue/finish-call-next
// 仅当：
// 1. 商户开启了叫号
// 2. 商户是手动叫号模式
// 3. 商户未暂停叫号
// 4. 操作人员有叫号权限
// 5. 操作人员未暂停自己的叫号（技师账号）
// 才触发下一个叫号
func TriggerNextCallingOnFinish(c *gin.Context) {
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

	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}

	// 检查是否开启了叫号
	if !merchant.SupportQueue {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"triggered": false, "reason": "商户未开启叫号"}})
		return
	}

	// 检查是否是手动叫号模式
	if merchant.QueueMode != "manual" {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"triggered": false, "reason": "自动叫号模式"}})
		return
	}

	// 检查商户是否暂停叫号
	if merchant.QueuePaused {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"triggered": false, "reason": "商户叫号已暂停"}})
		return
	}
	// 防御性处理：若历史上写入了 queue_ended_at 但又恢复运营（queue_paused=false），这里清理掉，避免调度器 finalizeUsagesAfterQueueEnded 误收尾
	if merchant.QueueEndedAt != nil {
		_ = config.DB.Model(&merchant).Update("queue_ended_at", nil).Error
	}

	// 获取账号类型
	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)

	// 检查叫号权限（商户老板号默认有权限）
	if authType != "merchant" {
		okCalling, err := middleware.HasPermission(c, "merchant.queue.calling")
		if err != nil || !okCalling {
			c.JSON(http.StatusOK, gin.H{"data": gin.H{"triggered": false, "reason": "无叫号权限"}})
			return
		}
	}

	// 检查技师是否暂停了自己的叫号
	if authType == "staff" {
		techIDAny, _ := c.Get("technician_id")
		techID, _ := techIDAny.(uint)
		if techID > 0 {
			var tech models.Technician
			if err := config.DB.Where("id = ? AND merchant_id = ?", techID, merchantID).First(&tech).Error; err == nil {
				if tech.QueuePaused {
					c.JSON(http.StatusOK, gin.H{"data": gin.H{"triggered": false, "reason": "您的叫号已暂停"}})
					return
				}
			}
		}
	}

	// 触发下一个叫号
	if queue.Default == nil {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"triggered": false, "reason": "叫号服务未初始化"}})
		return
	}

	// 模式守卫：finish-call-next 是“结单后推进下一号”的入口。
	// 为避免跨模式串用（如误把非手动叫号会话当作叫号会话推进），这里锁定当前进行中的会话并做一致性校验。
	// 若当前无进行中会话，则仅触发叫号，不做校验（保持原行为）。
	{
		authTypeAny, _ := c.Get("auth_type")
		authType, _ := authTypeAny.(string)
		if authType == "staff" {
			techIDAny, _ := c.Get("technician_id")
			techID, _ := techIDAny.(uint)
			if techID > 0 {
				err := config.DB.Transaction(func(tx *gorm.DB) error {
					var s models.ServiceSession
					err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
						Where("merchant_id = ? AND technician_id = ? AND status IN ? AND start_confirmed_at IS NOT NULL", merchantID, techID, models.ExpandStatusesWithKnownPrefixes([]string{"serving", "auto_finishing"})).
						Order("id desc").
						First(&s).Error
					if err != nil {
						if errors.Is(err, gorm.ErrRecordNotFound) {
							return nil
						}
						return err
					}
					if err := models.ValidateSessionModeForEntry(&s, &merchant); err != nil {
						return apiErr{status: http.StatusBadRequest, msg: err.Error()}
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
			}
		}
	}

	now := time.Now()
	if merchant.SupportMultiCustomerService {
		_ = config.DB.Transaction(func(tx *gorm.DB) error {
			assignNextForAllIdleTechsManual(tx, &merchant, now)
			return nil
		})
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"triggered":     true,
				"next_usage_id": uint(0),
			},
		})
		return
	}

	date := now.Format("2006-01-02")
	nextUsageID := queue.Default.CallNextUncalled(merchant.ID, date, queue.QueueTypeOnsite, now)
	if nextUsageID > 0 {
		_ = config.DB.Transaction(func(tx *gorm.DB) error {
			promoteManualSingleCalledSession(tx, &merchant, nextUsageID, now)
			return nil
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"triggered":     true,
			"next_usage_id": nextUsageID,
		},
	})
}

// TriggerAutoAssign 多客服叫号模式手动触发自动分配（运营兜底接口）
// POST /queue/trigger-auto-assign
// 用于极端情况：有空闲技师但队列卡在待叫号时，运营/商户手动触发重新分配
func TriggerAutoAssign(c *gin.Context) {
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

	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}

	// 仅在多客服叫号模式下可用
	if !merchant.SupportQueue || merchant.QueueMode != "auto" || !merchant.SupportMultiCustomerService {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前模式不支持手动分配"})
		return
	}

	// 权限检查：商户老板号默认有权限；技师账号需要“merchant.queue.calling”权限
	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)
	if authType != "merchant" {
		okAssign, err := middleware.HasPermission(c, "merchant.queue.calling")
		if err != nil || !okAssign {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权限操作"})
			return
		}
	}

	now := time.Now()
	// 调用技师签到模块的批量分配函数
	tryAutoCallNextForIdleTechnicians(config.DB, merchantID, now)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"triggered": true,
			"message":   "已尝试为所有空闲技师分配下一号",
		},
	})
}

// ReassignCurrentPendingSession 自动重分配当前待上号单到其他空闲客服
// POST /queue/sessions/:id/reassign-current-pending
func ReassignCurrentPendingSession(c *gin.Context) {
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

	sessionIDStr := strings.TrimSpace(c.Param("id"))
	sessionID64, err := strconv.ParseUint(sessionIDStr, 10, 32)
	if err != nil || sessionID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的会话ID"})
		return
	}
	sessionID := uint(sessionID64)

	var input struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)
	requesterTechID := uint(0)
	if authType == "staff" {
		techIDAny, _ := c.Get("technician_id")
		requesterTechID, _ = techIDAny.(uint)
	}
	roleType := currentStaffRoleType(c)

	now := time.Now()
	var result gin.H
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		var merchant models.Merchant
		if err := tx.First(&merchant, merchantID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apiErr{status: http.StatusNotFound, msg: "商户不存在"}
			}
			return err
		}
		if !supportsCurrentPendingReassign(&merchant) {
			return apiErr{status: http.StatusBadRequest, msg: "当前模式不支持重分配待上号单"}
		}

		var session models.ServiceSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND merchant_id = ?", sessionID, merchantID).
			First(&session).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apiErr{status: http.StatusNotFound, msg: "待上号单不存在"}
			}
			return err
		}
		if err := models.ValidateSessionModeForEntry(&session, &merchant); err != nil {
			return apiErr{status: http.StatusBadRequest, msg: err.Error()}
		}
		if models.NormalizeSessionStatus(session.Status) != "start_pending" || session.StartConfirmedAt != nil {
			return apiErr{status: http.StatusBadRequest, msg: "当前仅支持重分配待上号单"}
		}
		assignedTechIDs := serviceSessionAssignedTechnicianIDs(&session)
		if len(assignedTechIDs) == 0 {
			return apiErr{status: http.StatusBadRequest, msg: "当前待上号单尚未分配客服"}
		}
		oldTechID := assignedTechIDs[0]

		if authType == "staff" {
			if requesterTechID == 0 {
				return apiErr{status: http.StatusForbidden, msg: "仅工作人员可操作"}
			}
			if strings.TrimSpace(roleType) != "operational" && oldTechID != requesterTechID {
				return apiErr{status: http.StatusForbidden, msg: "仅可转交自己当前待上号单"}
			}
		}

		targetTech, targetAtt, err := selectQueueReassignTarget(tx, &merchant, oldTechID, now)
		if err != nil {
			return err
		}
		if targetTech == nil || targetAtt == nil {
			return apiErr{status: http.StatusBadRequest, msg: "当前没有可接手的空闲客服"}
		}

		timeoutSeconds := config.MerchantQueueWaitingStartSeconds(&merchant)
		updates := map[string]interface{}{
			"last_technician_id":            oldTechID,
			"service_technician_ids":        models.MerchantProjectDefaultServiceTechnicianIDs{targetTech.ID},
			"start_confirmed_technician_ids": models.MerchantProjectDefaultServiceTechnicianIDs{},
			"start_pending_timeout_seconds": timeoutSeconds,
			"updated_at":                    now,
		}
		if err := tx.Model(&models.ServiceSession{}).
			Where("id = ? AND merchant_id = ? AND status IN ? AND start_confirmed_at IS NULL", session.ID, merchantID, models.ExpandStatusWithKnownPrefixes("start_pending")).
			Updates(updates).Error; err != nil {
			return err
		}

		if err := releaseTechnicianAfterPendingReassign(tx, merchantID, oldTechID, now); err != nil {
			return err
		}
		if err := tx.Model(&models.TechnicianAttendance{}).
			Where("id = ? AND merchant_id = ? AND technician_id = ? AND status = ?", targetAtt.ID, merchantID, targetTech.ID, "idle").
			Updates(map[string]interface{}{"status": "busy"}).Error; err != nil {
			return err
		}

		var oldTech models.Technician
		_ = tx.Select("id", "name").Where("id = ? AND merchant_id = ?", oldTechID, merchantID).First(&oldTech).Error
		result = gin.H{
			"session_id":           session.ID,
			"from_technician_id":   oldTechID,
			"from_technician_name": strings.TrimSpace(oldTech.Name),
			"to_technician_id":     targetTech.ID,
			"to_technician_name":   strings.TrimSpace(targetTech.Name),
			"reason":               strings.TrimSpace(input.Reason),
			"reassigned_at":        now.Format("2006-01-02 15:04:05"),
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

	c.JSON(http.StatusOK, gin.H{
		"message": "已完成待上号单重分配",
		"data":    result,
	})
}
