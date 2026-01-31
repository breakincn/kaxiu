package handlers

import (
	"kabao/config"
	"kabao/models"
	"kabao/queue"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var tableActiveSessionStatuses = models.ExpandStatusesWithKnownPrefixes([]string{"room_locked", "staff_selecting", "start_pending", "delay_pending", "serving", "auto_finishing"})

func lazyReleaseStartPendingTimeout(merchantID uint, now time.Time) {
	// 手动叫号模式：待上号不做超时释放，避免状态被自动回退
	{
		var m models.Merchant
		if err := config.DB.Select("id,support_customer_service_mode,support_queue,queue_mode").First(&m, merchantID).Error; err == nil {
			if !m.SupportCustomerServiceMode && m.SupportQueue && m.QueueMode == "manual" {
				return
			}
		}
	}

	// 仅处理“待起单”且已超时、还绑着技师的会话
	var ids []uint
	if err := config.DB.
		Model(&models.ServiceSession{}).
		Select("id").
		Where("merchant_id = ? AND status IN ? AND start_confirmed_at IS NULL AND technician_id IS NOT NULL AND updated_at IS NOT NULL", merchantID, models.ExpandStatusWithKnownPrefixes("start_pending")).
		Limit(200).
		Pluck("id", &ids).Error; err != nil {
		return
	}
	if len(ids) == 0 {
		return
	}

	_ = config.DB.Transaction(func(tx *gorm.DB) error {
		for _, id := range ids {
			var s models.ServiceSession
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND merchant_id = ?", id, merchantID).First(&s).Error; err != nil {
				continue
			}
			if models.NormalizeSessionStatus(s.Status) != "start_pending" || s.StartConfirmedAt != nil || s.UpdatedAt == nil {
				continue
			}
			timeout := config.StartPendingTimeout()
			if s.StartPendingTimeoutSeconds > 0 {
				timeout = time.Duration(s.StartPendingTimeoutSeconds) * time.Second
			}
			startDeadline := s.UpdatedAt.Add(timeout)
			if now.Before(startDeadline) {
				continue
			}

			oldTechID := s.TechnicianID
			updates := map[string]interface{}{
				"status":                        models.ApplyStatusPrefix(s.Status, "staff_selecting"),
				"technician_id":                 nil,
				"staff_select_entered_at":       nil,
				"start_pending_timeout_seconds": 0,
				"start_timeout_count":           gorm.Expr("start_timeout_count + ?", 1),
				"start_timeout_last_at":         now,
			}
			if err := tx.Model(&models.ServiceSession{}).
				Where("id = ? AND status IN ? AND start_confirmed_at IS NULL", s.ID, models.ExpandStatusWithKnownPrefixes("start_pending")).
				Updates(updates).Error; err != nil {
				continue
			}
			if s.InitialUsageID > 0 && queue.Default != nil {
				date := now.Format("2006-01-02")
				queue.Default.Uncall(s.MerchantID, date, queue.QueueTypeOnsite, s.InitialUsageID)
			}

			if oldTechID != nil && *oldTechID > 0 {
				_ = tx.Model(&models.TechnicianAttendance{}).
					Where("merchant_id = ? AND technician_id = ? AND status = ?", s.MerchantID, *oldTechID, "busy").
					Updates(map[string]interface{}{"status": "idle"}).Error
			}
		}
		return nil
	})
}

func TableRooms(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	lazyReleaseStartPendingTimeout(merchantID, time.Now())

	var rooms []models.Room
	config.DB.Where("merchant_id = ?", merchantID).Order("id asc").Find(&rooms)

	var sessions []models.ServiceSession
	config.DB.
		Preload("Room").
		Preload("Technician").
		Preload("Technician.ServiceRole").
		Where("merchant_id = ? AND room_id IS NOT NULL AND status IN ?", merchantID, tableActiveSessionStatuses).
		Order("id desc").
		Find(&sessions)

	byRoom := map[uint]models.ServiceSession{}
	for _, s := range sessions {
		if s.RoomID == nil {
			continue
		}
		// 取最新一条
		if _, exists := byRoom[*s.RoomID]; !exists {
			byRoom[*s.RoomID] = s
		}
	}

	now := time.Now()
	type roomItem struct {
		Room                    models.Room            `json:"room"`
		Occupied                bool                   `json:"occupied"`
		Session                 *models.ServiceSession `json:"session"`
		Technician              *models.Technician     `json:"technician"`
		TechnicianRole          *models.ServiceRole    `json:"technician_role"`
		StartedAt               *time.Time             `json:"started_at"`
		FinishAt                *time.Time             `json:"finish_at"`
		RoomLockedAt            *time.Time             `json:"room_locked_at"`
		UpdatedAt               *time.Time             `json:"updated_at"`
		RoomSelectDeadlineAt    *time.Time             `json:"room_select_deadline_at"`
		ElapsedSeconds          int64                  `json:"elapsed_seconds"`
		RemainSeconds           int64                  `json:"remain_seconds"`
		Status                  string                 `json:"status"`
		PhaseText               string                 `json:"phase_text"`
		PhaseClass              string                 `json:"phase_class"`
		StartRemainSeconds      int64                  `json:"start_remain_seconds"`
		RoomSelectRemainSeconds int64                  `json:"room_select_remain_seconds"`
		Now                     time.Time              `json:"now"`
	}

	// 查询商户信息以获取自定义术语
	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取商户信息失败"})
		return
	}

	// 获取自定义起单术语，默认为"起单"
	startTerm := "起单"
	if merchant.StartTerm != "" {
		startTerm = merchant.StartTerm
	}

	out := make([]roomItem, 0, len(rooms))
	for _, r := range rooms {
		it := roomItem{Room: r, Occupied: false, Session: nil, Technician: nil, TechnicianRole: nil, StartedAt: nil, FinishAt: nil, RoomLockedAt: nil, UpdatedAt: nil, RoomSelectDeadlineAt: nil, ElapsedSeconds: 0, RemainSeconds: 0, Status: "idle", PhaseText: "", PhaseClass: "", StartRemainSeconds: 0, RoomSelectRemainSeconds: 0, Now: now}
		s, ok := byRoom[r.ID]
		if ok {
			it.Occupied = true
			it.Status = s.Status
			it.Session = &s
			if s.Technician != nil {
				it.Technician = s.Technician
				it.TechnicianRole = &s.Technician.ServiceRole
			}
			it.StartedAt = s.StartedAt
			it.RoomLockedAt = s.RoomLockedAt
			it.UpdatedAt = s.UpdatedAt
			it.RoomSelectDeadlineAt = s.RoomSelectDeadlineAt
			finishAt := s.ScheduledFinishAt
			if finishAt == nil && s.StartedAt != nil && s.DurationMinutes > 0 {
				t := s.StartedAt.Add(time.Duration(s.DurationMinutes) * time.Minute)
				finishAt = &t
			}
			it.FinishAt = finishAt
			if s.StartedAt != nil {
				it.ElapsedSeconds = int64(now.Sub(*s.StartedAt).Seconds())
				if it.ElapsedSeconds < 0 {
					it.ElapsedSeconds = 0
				}
			}
			if finishAt != nil {
				it.RemainSeconds = int64(finishAt.Sub(now).Seconds())
				if it.RemainSeconds < 0 {
					it.RemainSeconds = 0
				}
			}

			if s.Status == "start_pending" && s.StartConfirmedAt == nil && s.UpdatedAt != nil {
				startDeadline := s.UpdatedAt.Add(config.StartPendingTimeout())
				startRemain := int64(startDeadline.Sub(now).Seconds())
				if startRemain < 0 {
					startRemain = 0
				}
				it.StartRemainSeconds = startRemain

				if now.Before(startDeadline) {
					it.PhaseText = "待" + startTerm
					it.PhaseClass = "start_pending"
				} else {
					it.PhaseText = startTerm + "超时 重新选择客服"
					it.PhaseClass = "start_timeout"
				}
			}
		}
		out = append(out, it)
	}

	c.JSON(http.StatusOK, gin.H{"data": out})
}

func TableStaff(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	roleType := strings.TrimSpace(c.Query("type"))
	if roleType == "" {
		roleType = "professional"
	}
	// 注意：DB 中运营客服 role_type 实际为 operational，但前端参数使用 operation
	if roleType == "operation" || roleType == "operational" {
		roleType = "operational"
	}
	if roleType != "professional" && roleType != "operational" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效type，支持: professional / operation"})
		return
	}

	lazyReleaseStartPendingTimeout(merchantID, time.Now())

	// 专业客服（排除店长/前台）；运营客服不排除
	var techs []models.Technician
	qTech := config.DB.
		Preload("ServiceRole").
		Joins("JOIN service_roles sr ON sr.id = technicians.service_role_id").
		Where("technicians.merchant_id = ? AND sr.role_type = ?", merchantID, roleType)
	if roleType == "professional" {
		qTech = qTech.Where("sr.`key` NOT IN ('store_manager','front_desk')")
	}
	if roleType == "operational" {
		// 运营客服：店长（account_prefix=sm）优先
		qTech.Order("CASE WHEN sr.account_prefix = 'sm' THEN 1 ELSE 2 END, technicians.id, technicians.updated_at desc")
	} else {
		qTech.Order("technicians.id, technicians.updated_at desc")
	}
	qTech.Find(&techs)

	// 签到/状态 - 和选择工作人员条件一致：当天签到且未下班
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var atts []models.TechnicianAttendance
	qAtt := config.DB.
		Joins("JOIN technicians t ON t.id = technician_attendances.technician_id").
		Joins("JOIN service_roles sr ON sr.id = t.service_role_id").
		Where("technician_attendances.merchant_id = ? AND technician_attendances.checked_in_at >= ? AND technician_attendances.checked_out_at IS NULL", merchantID, start).
		Where("t.is_active = ?", true).
		Where("sr.role_type = ?", roleType)
	if roleType == "professional" {
		qAtt = qAtt.Where("sr.`key` NOT IN ('store_manager','front_desk')")
	}
	qAtt.Find(&atts)
	attByTech := map[uint]models.TechnicianAttendance{}
	for _, a := range atts {
		attByTech[a.TechnicianID] = a
	}

	// 当前会话（服务中/选房/选人/预结单/延迟等）
	var sessions []models.ServiceSession
	config.DB.
		Preload("Room").
		Where("merchant_id = ? AND technician_id IS NOT NULL AND status IN ?", merchantID, tableActiveSessionStatuses).
		Order("id desc").
		Find(&sessions)

	sessionByTech := map[uint]models.ServiceSession{}
	for _, s := range sessions {
		if s.TechnicianID == nil {
			continue
		}
		if _, exists := sessionByTech[*s.TechnicianID]; !exists {
			sessionByTech[*s.TechnicianID] = s
		}
	}

	type staffItem struct {
		Technician          models.Technician            `json:"technician"`
		Attendance          *models.TechnicianAttendance `json:"attendance"`
		CurrentSession      *models.ServiceSession       `json:"current_session"`
		Room                *models.Room                 `json:"room"`
		ServiceStatus       string                       `json:"service_status"`
		PhaseText           string                       `json:"phase_text"`
		PhaseClass          string                       `json:"phase_class"`
		StartRemainSeconds  int64                        `json:"start_remain_seconds"`
		CheckedIn           bool                         `json:"checked_in"`
		CheckedInAt         *time.Time                   `json:"checked_in_at"`
		ServiceStartAt      *time.Time                   `json:"service_start_at"`
		ServiceFinishAt     *time.Time                   `json:"service_finish_at"`
		ElapsedSeconds      int64                        `json:"elapsed_seconds"`
		RemainSeconds       int64                        `json:"remain_seconds"`
		NextAvailableAt     *time.Time                   `json:"next_available_at"`
		NextAvailableInSecs int64                        `json:"next_available_in_seconds"`
		Now                 time.Time                    `json:"now"`
	}

	out := make([]staffItem, 0, len(techs))
	for _, t := range techs {
		it := staffItem{Technician: t, Attendance: nil, CurrentSession: nil, Room: nil, ServiceStatus: "not_checked_in", PhaseText: "", PhaseClass: "", StartRemainSeconds: 0, CheckedIn: false, CheckedInAt: nil, ServiceStartAt: nil, ServiceFinishAt: nil, ElapsedSeconds: 0, RemainSeconds: 0, NextAvailableAt: nil, NextAvailableInSecs: 0, Now: now}

		if a, ok := attByTech[t.ID]; ok {
			it.Attendance = &a
			it.CheckedIn = a.CheckedInAt != nil && a.CheckedOutAt == nil
			it.CheckedInAt = a.CheckedInAt
			if a.Status != "" {
				it.ServiceStatus = a.Status
			}
		}

		if s, ok := sessionByTech[t.ID]; ok {
			it.CurrentSession = &s
			it.Room = s.Room
			it.ServiceStatus = s.Status
			it.ServiceStartAt = s.StartedAt

			finishAt := s.ScheduledFinishAt
			if finishAt == nil && s.StartedAt != nil && s.DurationMinutes > 0 {
				tt := s.StartedAt.Add(time.Duration(s.DurationMinutes) * time.Minute)
				finishAt = &tt
			}
			it.ServiceFinishAt = finishAt

			if s.StartedAt != nil {
				it.ElapsedSeconds = int64(now.Sub(*s.StartedAt).Seconds())
				if it.ElapsedSeconds < 0 {
					it.ElapsedSeconds = 0
				}
			}
			if finishAt != nil {
				it.RemainSeconds = int64(finishAt.Sub(now).Seconds())
				if it.RemainSeconds < 0 {
					it.RemainSeconds = 0
				}

				nextAt := finishAt.Add(time.Duration(s.AutoIdleAfterSeconds) * time.Second)
				it.NextAvailableAt = &nextAt
				it.NextAvailableInSecs = int64(nextAt.Sub(now).Seconds())
				if it.NextAvailableInSecs < 0 {
					it.NextAvailableInSecs = 0
				}
			}

			if s.Status == "start_pending" && s.StartConfirmedAt == nil && s.UpdatedAt != nil {
				startDeadline := s.UpdatedAt.Add(config.StartPendingTimeout())
				startRemain := int64(startDeadline.Sub(now).Seconds())
				if startRemain < 0 {
					startRemain = 0
				}
				it.StartRemainSeconds = startRemain

				if now.Before(startDeadline) {
					it.PhaseText = "待起单"
					it.PhaseClass = "start_pending"
				} else {
					it.PhaseText = "上钟超时 重新选择客服"
					it.PhaseClass = "start_timeout"
				}
			}
		}

		out = append(out, it)
	}

	c.JSON(http.StatusOK, gin.H{"data": out})
}
