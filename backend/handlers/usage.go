package handlers

import (
	"fmt"
	"kabao/config"
	"kabao/models"
	"kabao/queue"
	"kabao/sessionflow"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func GetCardUsages(c *gin.Context) {
	userID, ok := mustUserID(c)
	if !ok {
		return
	}
	cardID := c.Param("id")
	var card models.Card
	if err := config.DB.First(&card, cardID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "卡片不存在"})
		return
	}
	if card.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问此卡"})
		return
	}
	var usages []models.Usage
	config.DB.Preload("Merchant").Preload("Technician").Preload("Technician.ServiceRole").Preload("Project").Where("card_id = ?", cardID).Order("used_at DESC").Find(&usages)
	enrichUsagesWithCardSnapshots(usages)
	enrichUsagesWithServiceSession(&usages)
	enrichUsagesWithServiceParticipants(&usages)
	attachLatestExtendRequestsToUsages(usages)
	enrichUsagesWithQueue(&usages)
	c.JSON(http.StatusOK, gin.H{"data": usages})
}

func resolveSessionStartConfirmedAt(status string, startConfirmedAt *time.Time, startedAt *time.Time) *time.Time {
	if startConfirmedAt != nil {
		return startConfirmedAt
	}
	if startedAt == nil {
		return nil
	}
	baseStatus := models.NormalizeSessionStatus(status)
	if baseStatus == "serving" || baseStatus == "auto_finishing" || baseStatus == "finished" {
		return startedAt
	}
	return nil
}

func shouldFallbackServiceTechnicianToLast(status string, startConfirmedAt *time.Time, startedAt *time.Time, finishedAt *time.Time) bool {
	if startConfirmedAt != nil || startedAt != nil || finishedAt != nil {
		return true
	}
	switch models.NormalizeSessionStatus(status) {
	case "timeout_waiting", "serving", "auto_finishing", "finished":
		return true
	default:
		return false
	}
}

func GetMerchantUsages(c *gin.Context) {
	merchantID, ok := ensureMerchantScope(c, "id")
	if !ok {
		return
	}
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
	enrichUsagesWithCardSnapshots(usages)
	enrichUsagesWithServiceSession(&usages)
	enrichUsagesWithServiceParticipants(&usages)
	attachLatestExtendRequestsToUsages(usages)
	enrichUsagesWithQueue(&usages)
	c.JSON(http.StatusOK, gin.H{"data": usages})
}

func enrichUsagesWithServiceParticipants(usages *[]models.Usage) {
	if usages == nil || len(*usages) == 0 {
		return
	}

	projectIDs := make([]uint, 0)
	projectByID := make(map[uint]*models.MerchantProject)
	merchantIDs := make(map[uint]struct{})
	for i := range *usages {
		u := &(*usages)[i]
		if u.MerchantID > 0 {
			merchantIDs[u.MerchantID] = struct{}{}
		}
		if u.Project == nil || u.Project.ID == 0 {
			continue
		}
		if u.Project.ServiceCapacity <= 1 || !u.Project.ShowParticipants {
			continue
		}
		if _, ok := projectByID[u.Project.ID]; ok {
			continue
		}
		projectByID[u.Project.ID] = u.Project
		projectIDs = append(projectIDs, u.Project.ID)
	}
	if len(projectIDs) == 0 || len(merchantIDs) == 0 {
		return
	}

	merchantIDList := make([]uint, 0, len(merchantIDs))
	for id := range merchantIDs {
		merchantIDList = append(merchantIDList, id)
	}

	type participantRow struct {
		ProjectID uint   `gorm:"column:project_id"`
		UserID    uint   `gorm:"column:user_id"`
		Nickname  string `gorm:"column:nickname"`
	}
	var rows []participantRow
	activeStatuses := models.ExpandStatusesWithKnownPrefixes([]string{
		"room_selecting",
		"room_locked",
		"staff_selecting",
		"start_pending",
		"delay_pending",
		"serving",
		"auto_finishing",
		"timeout_waiting",
	})
	if err := config.DB.
		Table("service_sessions ss").
		Select("ss.project_id, ss.user_id, COALESCE(NULLIF(users.nickname, ''), users.username) AS nickname").
		Joins("JOIN users ON users.id = ss.user_id").
		Where("ss.merchant_id IN ?", merchantIDList).
		Where("ss.project_id IN ?", projectIDs).
		Where("ss.status IN ?", activeStatuses).
		Order("ss.created_at ASC, ss.id ASC").
		Scan(&rows).Error; err != nil {
		return
	}

	byProjectID := make(map[uint][]models.UsageParticipantUser)
	seenByProjectID := make(map[uint]map[uint]struct{})
	for _, row := range rows {
		if row.ProjectID == 0 || row.UserID == 0 {
			continue
		}
		nickname := strings.TrimSpace(row.Nickname)
		if nickname == "" {
			continue
		}
		if seenByProjectID[row.ProjectID] == nil {
			seenByProjectID[row.ProjectID] = make(map[uint]struct{})
		}
		if _, seen := seenByProjectID[row.ProjectID][row.UserID]; seen {
			continue
		}
		seenByProjectID[row.ProjectID][row.UserID] = struct{}{}
		byProjectID[row.ProjectID] = append(byProjectID[row.ProjectID], models.UsageParticipantUser{
			UserID:   row.UserID,
			Nickname: nickname,
		})
	}

	for i := range *usages {
		u := &(*usages)[i]
		if u.Project == nil || u.Project.ServiceCapacity <= 1 || !u.Project.ShowParticipants {
			continue
		}
		u.ServiceParticipantUsers = byProjectID[u.Project.ID]
	}
}

func enrichUsagesWithQueue(usages *[]models.Usage) {
	if usages == nil || len(*usages) == 0 {
		return
	}
	if queue.Default == nil {
		return
	}

	// 仅填充当天现场叫号队列信息；预约履约不再维护独立排队号。
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
			// MarkDone 的记录会在 Snapshot 中被过滤，但队列 list 仍保留该 id（用于维持号码稳定）。
			// 因此这里兜底用 GetNo 取号码，让 timeout_waiting/过号等待态也能展示叫号信息。
			if no, ok2 := queue.Default.GetNo(u.MerchantID, today, queue.QueueTypeOnsite, u.ID); ok2 && no > 0 {
				u.QueueNo = no
				u.QueueCalledAt = nil
				u.QueueKind = string(queue.QueueTypeOnsite)
			}
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
		ID                         uint                                              `gorm:"column:id"`
		InitialUsageID             uint                                              `gorm:"column:initial_usage_id"`
		ProjectID                  *uint                                             `gorm:"column:project_id"`
		SourceType                 string                                            `gorm:"column:source_type"`
		SourceID                   *uint                                             `gorm:"column:source_id"`
		Status                     string                                            `gorm:"column:status"`
		RoomID                     *uint                                             `gorm:"column:room_id"`
		TechnicianID               *uint                                             `gorm:"column:technician_id"`
		LastTechnicianID           *uint                                             `gorm:"column:last_technician_id"`
		ServiceTechnicianIDs       models.MerchantProjectDefaultServiceTechnicianIDs `gorm:"column:service_technician_ids"`
		StartTimeoutCount          int                                               `gorm:"column:start_timeout_count"`
		StartConfirmedAt           *time.Time                                        `gorm:"column:start_confirmed_at"`
		ScheduledStartAt           *time.Time                                        `gorm:"column:scheduled_start_at"`
		StartedAt                  *time.Time                                        `gorm:"column:started_at"`
		ScheduledFinishAt          *time.Time                                        `gorm:"column:scheduled_finish_at"`
		FinishedAt                 *time.Time                                        `gorm:"column:finished_at"`
		DurationMinutes            int                                               `gorm:"column:duration_minutes"`
		CreatedAt                  *time.Time                                        `gorm:"column:created_at"`
		UpdatedAt                  *time.Time                                        `gorm:"column:updated_at"`
		StartPendingTimeoutSeconds int                                               `gorm:"column:start_pending_timeout_seconds"`
		RoomSelectDeadlineAt       *time.Time                                        `gorm:"column:room_select_deadline_at"`
		RoomLockedAt               *time.Time                                        `gorm:"column:room_locked_at"`
		StaffSelectCooldownUntil   *time.Time                                        `gorm:"column:staff_select_cooldown_until"`
		StaffSelectEnteredAt       *time.Time                                        `gorm:"column:staff_select_entered_at"`
	}

	var sessions []sessLite
	if err := config.DB.
		Table("service_sessions").
		Select("id, initial_usage_id, project_id, source_type, source_id, status, room_id, technician_id, last_technician_id, service_technician_ids, start_timeout_count, start_confirmed_at, scheduled_start_at, started_at, scheduled_finish_at, finished_at, duration_minutes, created_at, updated_at, start_pending_timeout_seconds, room_select_deadline_at, room_locked_at, staff_select_cooldown_until, staff_select_entered_at").
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

	// 若 usage.Project 为空，尝试用 service_session.project_id 或默认项目兜底补齐
	needProjectIDs := make([]uint, 0, len(sessions))
	defaultProjectByMerchant := make(map[uint]*models.MerchantProject)
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
		if u.ProjectID == nil && u.MerchantID > 0 {
			if _, seen := defaultProjectByMerchant[u.MerchantID]; !seen {
				project, err := config.GetDefaultMerchantProject(config.DB, u.MerchantID)
				if err == nil && project != nil {
					defaultProjectByMerchant[u.MerchantID] = project
				} else {
					defaultProjectByMerchant[u.MerchantID] = nil
				}
			}
			if project := defaultProjectByMerchant[u.MerchantID]; project != nil {
				pid := project.ID
				u.ProjectID = &pid
				u.Project = project
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
	defaultServiceTechIDsByUsage := make(map[uint][]uint)
	for i := range sessions {
		s := sessions[i]
		if s.RoomID != nil && *s.RoomID > 0 {
			roomIDs = append(roomIDs, *s.RoomID)
		}
		if s.TechnicianID != nil && *s.TechnicianID > 0 {
			techIDs = append(techIDs, *s.TechnicianID)
		}
		if s.LastTechnicianID != nil && *s.LastTechnicianID > 0 {
			techIDs = append(techIDs, *s.LastTechnicianID)
		}
	}
	for i := range *usages {
		u := &(*usages)[i]
		ids := models.MerchantProjectDefaultServiceTechnicianIDs{}
		if s, ok := byUsageID[u.ID]; ok && len(s.ServiceTechnicianIDs) > 0 {
			ids = s.ServiceTechnicianIDs
		} else if u.Project != nil && u.Project.ServiceCapacity > 1 && len(u.Project.DefaultServiceTechnicianIDs) > 0 {
			ids = u.Project.DefaultServiceTechnicianIDs
		}
		if len(ids) == 0 {
			continue
		}
		resolvedIDs := make([]uint, 0, len(ids))
		seen := make(map[uint]struct{}, len(ids))
		for _, id := range ids {
			if id == 0 {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			resolvedIDs = append(resolvedIDs, id)
			techIDs = append(techIDs, id)
		}
		if len(resolvedIDs) > 0 {
			defaultServiceTechIDsByUsage[u.ID] = resolvedIDs
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

	remainNow := time.Now()

	for i := range *usages {
		u := &(*usages)[i]
		if s, ok := byUsageID[u.ID]; ok {
			sid := s.ID
			startConfirmedAt := resolveSessionStartConfirmedAt(s.Status, s.StartConfirmedAt, s.StartedAt)
			u.ServiceSessionID = &sid
			u.ServiceSessionStatus = s.Status
			u.ServiceSessionStartConfirmedAt = startConfirmedAt
			u.ServiceSessionScheduledStartAt = s.ScheduledStartAt
			u.ServiceSessionStartedAt = s.StartedAt
			u.ServiceSessionScheduledFinishAt = s.ScheduledFinishAt
			u.ServiceSessionFinishedAt = s.FinishedAt
			u.ServiceSessionDurationMinutes = s.DurationMinutes
			u.ServiceSessionUpdatedAt = s.UpdatedAt
			if strings.TrimSpace(s.SourceType) == "appointment" && s.SourceID != nil && *s.SourceID > 0 {
				appointmentID := *s.SourceID
				u.AppointmentID = &appointmentID
			}
			u.ServiceSessionStartPendingTimeoutSeconds = s.StartPendingTimeoutSeconds
			sessForRemain := models.ServiceSession{
				Status:                     s.Status,
				StartConfirmedAt:           startConfirmedAt,
				StartPendingTimeoutSeconds: s.StartPendingTimeoutSeconds,
				CreatedAt:                  s.CreatedAt,
				UpdatedAt:                  s.UpdatedAt,
			}
			u.ServiceSessionStartPendingRemainingSeconds = computeStartPendingRemainingSecondsForSession(&sessForRemain, remainNow)
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
			} else if s.LastTechnicianID != nil && shouldFallbackServiceTechnicianToLast(s.Status, startConfirmedAt, s.StartedAt, s.FinishedAt) {
				if t, okT := byTechID[*s.LastTechnicianID]; okT {
					u.ServiceTechnician = t
				}
			}
			if ids := defaultServiceTechIDsByUsage[u.ID]; len(ids) > 0 {
				seenTechIDs := make(map[uint]struct{}, len(ids)+1)
				for _, id := range ids {
					if t, okT := byTechID[id]; okT {
						u.ServiceTechnicians = append(u.ServiceTechnicians, t)
						seenTechIDs[id] = struct{}{}
					}
				}
				if u.ServiceTechnician != nil && u.ServiceTechnician.ID > 0 {
					if _, ok := seenTechIDs[u.ServiceTechnician.ID]; !ok {
						u.ServiceTechnicians = append(u.ServiceTechnicians, u.ServiceTechnician)
					}
				}
			} else if u.ServiceTechnician != nil && u.ServiceTechnician.ID > 0 {
				u.ServiceTechnicians = []*models.Technician{u.ServiceTechnician}
			}
		}
	}

	// 计算撤销能力（仅用户展示字段，不落库）
	now := time.Now()
	for i := range *usages {
		u := &(*usages)[i]
		u.CanRevoke = false
		u.RevokeDeadlineAt = nil
		dl := revokeDeadlineAt(u.UsedAt)
		if dl != nil {
			u.RevokeDeadlineAt = dl
		}
		if s, ok := byUsageID[u.ID]; ok {
			sessionForRevoke := models.ServiceSession{
				Status:            s.Status,
				RoomID:            s.RoomID,
				TechnicianID:      s.TechnicianID,
				StartTimeoutCount: s.StartTimeoutCount,
				StartConfirmedAt:  resolveSessionStartConfirmedAt(s.Status, s.StartConfirmedAt, s.StartedAt),
				StartedAt:         s.StartedAt,
			}
			u.CanRevoke = canRevokeUsageWithSession(u, &sessionForRevoke, now)
		}
	}

	techIDsForAvailability := make([]uint, 0, len(*usages))
	for i := range *usages {
		u := &(*usages)[i]
		u.ServiceTechnicianAvailable = true
		u.ServiceTechnicianUnavailableReason = ""
		if models.NormalizeSessionStatus(u.ServiceSessionStatus) != "start_pending" {
			continue
		}
		if u.ServiceSessionStartConfirmedAt != nil {
			continue
		}
		if u.ServiceTechnician == nil || u.ServiceTechnician.ID == 0 {
			continue
		}
		techIDsForAvailability = append(techIDsForAvailability, u.ServiceTechnician.ID)
	}

	if len(techIDsForAvailability) == 0 {
		return
	}

	techMeta := listQueuePendingTechMeta((*usages)[0].MerchantID, techIDsForAvailability)
	for i := range *usages {
		u := &(*usages)[i]
		if models.NormalizeSessionStatus(u.ServiceSessionStatus) != "start_pending" || u.ServiceSessionStartConfirmedAt != nil || u.ServiceTechnician == nil || u.ServiceTechnician.ID == 0 {
			continue
		}
		meta, ok := techMeta[u.ServiceTechnician.ID]
		if !ok {
			u.ServiceTechnicianAvailable = false
			u.ServiceTechnicianUnavailableReason = "当前客服不可服务"
			continue
		}
		reason := resolveQueuePendingTechUnavailableReason(meta)
		u.ServiceTechnicianAvailable = reason == ""
		u.ServiceTechnicianUnavailableReason = reason
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
			finishedAt := u.UsedAt.Add(12 * time.Hour)
			handled, err := sessionflow.CompleteUnstartedServiceSession(config.DB, u.ID, u.MerchantID, finishedAt)
			if err != nil {
				continue
			}
			if !handled {
				config.DB.Model(u).Updates(map[string]interface{}{
					"status":        "success",
					"technician_id": nil,
					"finished_at":   finishedAt,
				})
			}
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
