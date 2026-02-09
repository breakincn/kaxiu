package handlers

import (
	"fmt"
	"kabao/config"
	"kabao/models"
	"kabao/queue"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func GetCardUsages(c *gin.Context) {
	cardID := c.Param("id")
	var usages []models.Usage
	config.DB.Preload("Merchant").Preload("Technician").Preload("Technician.ServiceRole").Preload("Project").Where("card_id = ?", cardID).Order("used_at DESC").Find(&usages)
	enrichUsagesWithServiceSession(&usages)
	enrichUsagesWithQueue(&usages)
	autoFixUsages(&usages)
	c.JSON(http.StatusOK, gin.H{"data": usages})
}

func GetMerchantUsages(c *gin.Context) {
	merchantID := c.Param("id")
	var usages []models.Usage

	dateStr := c.Query("date")
	technicianIDStr := c.Query("technician_id")
	onlyWithSessionStr := c.Query("only_with_session")
	limitStr := c.Query("limit")

	query := config.DB.Model(&models.Usage{})
	query = query.Where("usages.merchant_id = ?", merchantID)
	query = query.Order("usages.used_at DESC")

	// 是否需要 join service_sessions
	needJoinSession := false
	if onlyWithSessionStr == "1" || onlyWithSessionStr == "true" {
		needJoinSession = true
	}
	if technicianIDStr != "" {
		needJoinSession = true
	}
	if needJoinSession {
		query = query.Joins("JOIN service_sessions ss ON ss.initial_usage_id = usages.id")
		query = query.Group("usages.id")
		if onlyWithSessionStr == "1" || onlyWithSessionStr == "true" {
			// 已经 inner join，可不加额外 where；保留结构以便未来改为 left join 时仍可用
		}
		if technicianIDStr != "" {
			if tid, err := strconv.ParseUint(technicianIDStr, 10, 64); err == nil && tid > 0 {
				query = query.Where("ss.technician_id = ?", tid)
			}
		}
	}

	// date=YYYY-MM-DD：按 used_at 过滤当天（以服务器本地时区为准）
	if dateStr != "" {
		if d, err := time.ParseInLocation("2006-01-02", dateStr, time.Local); err == nil {
			start := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.Local)
			end := start.Add(24 * time.Hour)
			query = query.Where("usages.used_at >= ? AND usages.used_at < ?", start, end)
		}
	}

	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
			if n > 200 {
				n = 200
			}
			query = query.Limit(n)
		}
	}

	query = query.Preload("Card").Preload("Card.User").Preload("Technician").Preload("Technician.ServiceRole").Preload("Merchant").Preload("Project")
	query.Find(&usages)
	enrichUsagesWithServiceSession(&usages)
	enrichUsagesWithQueue(&usages)
	autoFixUsages(&usages)
	c.JSON(http.StatusOK, gin.H{"data": usages})
}

func enrichUsagesWithQueue(usages *[]models.Usage) {
	if usages == nil || len(*usages) == 0 {
		return
	}

	// 仅填充当天现场叫号队列信息（预约走预约队列，不在使用记录里展示现场叫号）
	now := time.Now()
	today := now.Format("2006-01-02")

	for i := range *usages {
		u := &(*usages)[i]
		u.QueueNo = 0
		u.QueueCalledAt = nil
		u.QueueKind = ""

		if u.MerchantID == 0 {
			continue
		}
		if u.UsedAt == nil {
			continue
		}
		// 仅当天
		if u.UsedAt.Format("2006-01-02") != today {
			continue
		}

		snap := queue.Default.Snapshot(u.MerchantID, today, queue.QueueTypeOnsite)
		if snap.ByID == nil {
			continue
		}
		tk, ok := snap.ByID[u.ID]
		if !ok {
			if os.Getenv("KABAO_QUEUE_DEBUG") == "1" {
				headID := uint(0)
				if len(snap.Tickets) > 0 {
					headID = snap.Tickets[0].ID
				}
				log.Printf("[queue-debug] usage snapshot miss: merchant=%d date=%s usage_id=%d queue_len=%d head_id=%d\n", u.MerchantID, today, u.ID, len(snap.Tickets), headID)
			}
			continue
		}
		u.QueueNo = tk.No
		u.QueueCalledAt = tk.CalledAt
		u.QueueKind = string(queue.QueueTypeOnsite)
	}
}

func enrichUsagesWithServiceSession(usages *[]models.Usage) {
	if usages == nil || len(*usages) == 0 {
		return
	}

	ids := make([]uint, 0, len(*usages))
	for i := range *usages {
		u := (*usages)[i]
		if u.ID > 0 {
			ids = append(ids, u.ID)
		}
	}
	if len(ids) == 0 {
		return
	}

	type sessLite struct {
		ID                         uint       `gorm:"column:id"`
		InitialUsageID             uint       `gorm:"column:initial_usage_id"`
		ProjectID                  *uint      `gorm:"column:project_id"`
		Status                     string     `gorm:"column:status"`
		RoomID                     *uint      `gorm:"column:room_id"`
		TechnicianID               *uint      `gorm:"column:technician_id"`
		StartTimeoutCount          int        `gorm:"column:start_timeout_count"`
		StartConfirmedAt           *time.Time `gorm:"column:start_confirmed_at"`
		StartedAt                  *time.Time `gorm:"column:started_at"`
		ScheduledFinishAt          *time.Time `gorm:"column:scheduled_finish_at"`
		FinishedAt                 *time.Time `gorm:"column:finished_at"`
		DurationMinutes            int        `gorm:"column:duration_minutes"`
		UpdatedAt                  *time.Time `gorm:"column:updated_at"`
		StartPendingTimeoutSeconds int        `gorm:"column:start_pending_timeout_seconds"`
		RoomSelectDeadlineAt       *time.Time `gorm:"column:room_select_deadline_at"`
		RoomLockedAt               *time.Time `gorm:"column:room_locked_at"`
		StaffSelectCooldownUntil   *time.Time `gorm:"column:staff_select_cooldown_until"`
		StaffSelectEnteredAt       *time.Time `gorm:"column:staff_select_entered_at"`
	}

	var sessions []sessLite
	if err := config.DB.
		Table("service_sessions").
		Select("id, initial_usage_id, project_id, status, room_id, technician_id, start_timeout_count, start_confirmed_at, started_at, scheduled_finish_at, finished_at, duration_minutes, updated_at, start_pending_timeout_seconds, room_select_deadline_at, room_locked_at, staff_select_cooldown_until, staff_select_entered_at").
		Where("initial_usage_id IN ?", ids).
		Order("id desc").
		Find(&sessions).Error; err != nil {
		return
	}

	// 同一个 usage 可能存在多条会话（例如取消/重试/重建）。这里按 id desc 查询后，
	// 仅保留每个 usage 最新的一条，避免旧会话覆盖导致状态展示错乱。
	byUsageID := make(map[uint]sessLite, len(sessions))
	for i := range sessions {
		s := sessions[i]
		if s.InitialUsageID == 0 {
			continue
		}
		if _, exists := byUsageID[s.InitialUsageID]; exists {
			continue
		}
		byUsageID[s.InitialUsageID] = s
	}

	// 若 usage.Project 为空，尝试用 service_session.project_id 兜底补齐
	needProjectIDs := make([]uint, 0, len(sessions))
	for i := range *usages {
		u := &(*usages)[i]
		if u.ProjectID != nil || u.Project != nil {
			continue
		}
		if s, ok := byUsageID[u.ID]; ok {
			if s.ProjectID != nil && *s.ProjectID > 0 {
				pid := *s.ProjectID
				u.ProjectID = &pid
				needProjectIDs = append(needProjectIDs, pid)
				if u.ID == 77 {
					fmt.Printf("[DEBUG] usage.id=77 found service_session.project_id=%d, set usage.ProjectID\n", pid)
				}
			}
		}
	}

	byProjectID := make(map[uint]*models.MerchantProject)
	if len(needProjectIDs) > 0 {
		var projects []models.MerchantProject
		if err := config.DB.Where("id IN ?", needProjectIDs).Find(&projects).Error; err == nil {
			for i := range projects {
				p := projects[i]
				byProjectID[p.ID] = &projects[i]
			}
		}
		for i := range *usages {
			u := &(*usages)[i]
			if u.Project != nil || u.ProjectID == nil {
				continue
			}
			if p, ok := byProjectID[*u.ProjectID]; ok {
				u.Project = p
				if u.ID == 77 {
					fmt.Printf("[DEBUG] usage.id=77 set usage.Project.name=%s\n", p.Name)
				}
			}
		}
	}

	roomIDs := make([]uint, 0, len(sessions))
	techIDs := make([]uint, 0, len(sessions))
	for i := range sessions {
		s := sessions[i]
		if s.RoomID != nil && *s.RoomID > 0 {
			roomIDs = append(roomIDs, *s.RoomID)
		}
		if s.TechnicianID != nil && *s.TechnicianID > 0 {
			techIDs = append(techIDs, *s.TechnicianID)
		}
	}

	byRoomID := make(map[uint]*models.Room, len(roomIDs))
	if len(roomIDs) > 0 {
		var rooms []models.Room
		if err := config.DB.Where("id IN ?", roomIDs).Find(&rooms).Error; err == nil {
			for i := range rooms {
				r := rooms[i]
				byRoomID[r.ID] = &rooms[i]
			}
		}
	}

	byTechID := make(map[uint]*models.Technician, len(techIDs))
	if len(techIDs) > 0 {
		var techs []models.Technician
		if err := config.DB.Preload("ServiceRole").Where("id IN ?", techIDs).Find(&techs).Error; err == nil {
			for i := range techs {
				t := techs[i]
				byTechID[t.ID] = &techs[i]
			}
		}
	}

	for i := range *usages {
		u := &(*usages)[i]
		if s, ok := byUsageID[u.ID]; ok {
			sid := s.ID
			u.ServiceSessionID = &sid
			u.ServiceSessionStatus = s.Status
			u.ServiceSessionStartConfirmedAt = s.StartConfirmedAt
			u.ServiceSessionStartedAt = s.StartedAt
			u.ServiceSessionScheduledFinishAt = s.ScheduledFinishAt
			u.ServiceSessionFinishedAt = s.FinishedAt
			u.ServiceSessionDurationMinutes = s.DurationMinutes
			u.ServiceSessionUpdatedAt = s.UpdatedAt
			u.ServiceSessionStartPendingTimeoutSeconds = s.StartPendingTimeoutSeconds
			u.StartTimeoutCount = s.StartTimeoutCount
			u.RoomSelectDeadlineAt = s.RoomSelectDeadlineAt
			u.RoomLockedAt = s.RoomLockedAt
			u.StaffSelectCooldownUntil = s.StaffSelectCooldownUntil
			u.StaffSelectEnteredAt = s.StaffSelectEnteredAt
			if s.RoomID != nil {
				if r, okR := byRoomID[*s.RoomID]; okR {
					u.ServiceRoom = r
				}
			}
			if s.TechnicianID != nil {
				if t, okT := byTechID[*s.TechnicianID]; okT {
					u.ServiceTechnician = t
				}
			}
		}
	}

	// 计算撤销能力（仅用户展示字段，不落库）
	now := time.Now()
	for i := range *usages {
		u := &(*usages)[i]
		u.CanRevoke = false
		u.RevokeDeadlineAt = nil
		if u.UsedAt == nil {
			continue
		}
		if u.Merchant.SupportCustomerServiceMode == false {
			continue
		}
		if u.Status != "in_progress" {
			continue
		}
		// 起单成功/结单成功不可撤销
		if u.ServiceSessionStartConfirmedAt != nil {
			continue
		}
		dl := u.UsedAt.Add(12 * time.Hour)
		u.RevokeDeadlineAt = &dl
		if !now.Before(dl) {
			continue
		}
		// 仅当起单超时累计达到2次，才允许撤销
		if u.StartTimeoutCount < 2 {
			continue
		}
		u.CanRevoke = true
	}
}

// autoFixUsages 对超过12小时的使用记录自动置为完成并清空技师ID
func autoFixUsages(usages *[]models.Usage) {
	now := time.Now()
	for i := range *usages {
		u := &(*usages)[i]
		if u.UsedAt == nil || u.Status == "failed" {
			continue
		}
		// 超过12小时：自动置为完成，并清空服务人员
		if now.Sub(*u.UsedAt) <= 12*time.Hour {
			continue
		}

		// 支持客服流程：仅当始终未起单（start_confirmed_at 为空）时，12小时后自动完成，且不计入客服业绩（technician_id 置空）。
		if u.Merchant.SupportCustomerServiceMode {
			var s struct {
				ID               uint       `gorm:"column:id"`
				Status           string     `gorm:"column:status"`
				RoomID           *uint      `gorm:"column:room_id"`
				TechnicianID     *uint      `gorm:"column:technician_id"`
				StartConfirmedAt *time.Time `gorm:"column:start_confirmed_at"`
			}
			err := config.DB.Table("service_sessions").
				Select("id,status,room_id,technician_id,start_confirmed_at").
				Where("initial_usage_id = ?", u.ID).
				Order("id desc").
				First(&s).Error
			if err == nil {
				if s.StartConfirmedAt != nil {
					continue
				}
				finishedAt := u.UsedAt.Add(12 * time.Hour)
				// 若会话仍保留技师，释放技师状态（避免 busy 残留）
				if s.TechnicianID != nil && *s.TechnicianID > 0 {
					_ = config.DB.Model(&models.TechnicianAttendance{}).
						Where("merchant_id = ? AND technician_id = ? AND status = ?", u.MerchantID, *s.TechnicianID, "busy").
						Updates(map[string]interface{}{"status": "idle"}).Error
				}
				// 结束会话并释放资源（房间/技师）
				config.DB.Table("service_sessions").
					Where("id = ? AND status NOT IN ?", s.ID, models.ExpandStatusesWithKnownPrefixes([]string{"finished", "canceled"})).
					Updates(map[string]interface{}{
						"status":                  models.ApplyStatusPrefix(s.Status, "finished"),
						"finished_at":             finishedAt,
						"technician_id":           nil,
						"room_id":                 nil,
						"room_locked_at":          nil,
						"room_select_deadline_at": nil,
					})

				config.DB.Model(u).Updates(map[string]interface{}{
					"status":        "success",
					"technician_id": nil,
					"finished_at":   finishedAt,
				})
				u.Status = "success"
				u.TechnicianID = nil
				u.FinishedAt = &finishedAt
				continue
			}
			// 未找到会话也视为可自动完成
			finishedAt := u.UsedAt.Add(12 * time.Hour)
			config.DB.Model(u).Updates(map[string]interface{}{
				"status":        "success",
				"technician_id": nil,
				"finished_at":   finishedAt,
			})
			u.Status = "success"
			u.TechnicianID = nil
			u.FinishedAt = &finishedAt
			continue
		}

		// 非客服流程：保留历史逻辑（超过12小时自动完成）
		if u.Status != "success" || u.TechnicianID != nil {
			// 优先使用项目时长，如果没有则用默认15分钟
			durationMinutes := 15
			if u.Project != nil && u.Project.Duration > 0 {
				durationMinutes = u.Project.Duration
			}
			finishedAt := u.UsedAt.Add(time.Duration(durationMinutes+5) * time.Minute)
			config.DB.Model(u).Updates(map[string]interface{}{
				"status":        "success",
				"technician_id": nil,
				"finished_at":   finishedAt,
			})
			u.Status = "success"
			u.TechnicianID = nil
			u.FinishedAt = &finishedAt
		}
	}
}
