package handlers

import (
	"kabao/config"
	"kabao/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

var tableActiveSessionStatuses = []string{"room_locked", "staff_selecting", "precheck_pending", "delay_pending", "serving", "auto_finishing"}

func TableRooms(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

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
		Room           models.Room            `json:"room"`
		Occupied       bool                   `json:"occupied"`
		Session        *models.ServiceSession `json:"session"`
		Technician     *models.Technician     `json:"technician"`
		TechnicianRole *models.ServiceRole    `json:"technician_role"`
		StartedAt      *time.Time             `json:"started_at"`
		FinishAt       *time.Time             `json:"finish_at"`
		ElapsedSeconds int64                  `json:"elapsed_seconds"`
		RemainSeconds  int64                  `json:"remain_seconds"`
		Status         string                 `json:"status"`
		Now            time.Time              `json:"now"`
	}

	out := make([]roomItem, 0, len(rooms))
	for _, r := range rooms {
		it := roomItem{Room: r, Occupied: false, Session: nil, Technician: nil, TechnicianRole: nil, StartedAt: nil, FinishAt: nil, ElapsedSeconds: 0, RemainSeconds: 0, Status: "idle", Now: now}
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

	// 专业客服（排除店长/前台）
	var techs []models.Technician
	config.DB.
		Preload("ServiceRole").
		Joins("JOIN service_roles sr ON sr.id = technicians.service_role_id").
		Where("technicians.merchant_id = ? AND sr.role_type = ? AND sr.`key` NOT IN ('store_manager','front_desk')", merchantID, "professional").
		Order("technicians.id desc").
		Find(&techs)

	// 签到/状态
	var atts []models.TechnicianAttendance
	config.DB.Where("merchant_id = ?", merchantID).Find(&atts)
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

	now := time.Now()
	type staffItem struct {
		Technician          models.Technician            `json:"technician"`
		Attendance          *models.TechnicianAttendance `json:"attendance"`
		CurrentSession      *models.ServiceSession       `json:"current_session"`
		Room                *models.Room                 `json:"room"`
		ServiceStatus       string                       `json:"service_status"`
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
		it := staffItem{Technician: t, Attendance: nil, CurrentSession: nil, Room: nil, ServiceStatus: "not_checked_in", CheckedIn: false, CheckedInAt: nil, ServiceStartAt: nil, ServiceFinishAt: nil, ElapsedSeconds: 0, RemainSeconds: 0, NextAvailableAt: nil, NextAvailableInSecs: 0, Now: now}

		if a, ok := attByTech[t.ID]; ok {
			it.Attendance = &a
			it.CheckedIn = a.CheckedInAt != nil
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
		}

		out = append(out, it)
	}

	c.JSON(http.StatusOK, gin.H{"data": out})
}
