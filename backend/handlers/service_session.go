package handlers

import (
	"errors"
	"fmt"
	"kabao/config"
	"kabao/models"
	"kabao/queue"
	"kabao/sessionflow"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func technicianStatusConflictText(status string) string {
	switch strings.TrimSpace(status) {
	case "busy":
		return "在服务中"
	case "paused":
		return "在暂停服务中"
	case "rest":
		return "已下班"
	case "idle":
		return "当前空闲"
	default:
		return "状态异常"
	}
}

func shouldBlockCustomerServiceStartByAttendance(status string, hasOtherServingSession bool) bool {
	switch strings.TrimSpace(status) {
	case "idle":
		return false
	case "busy":
		return hasOtherServingSession
	default:
		return true
	}
}

func ensureQueueSessionRoomLocked(merchant *models.Merchant, s *models.ServiceSession) error {
	if merchant == nil || s == nil {
		return nil
	}
	if !merchant.SupportRoom {
		return nil
	}
	if s.RoomID == nil || *s.RoomID == 0 {
		return apiErr{status: http.StatusBadRequest, msg: "当前无可用房间，请继续排队等待房间分配"}
	}
	return nil
}

func lockTechnicianAttendanceForQueueScan(tx *gorm.DB, merchantID uint, technicianID uint, sessionID uint, now time.Time) (*models.TechnicianAttendance, error) {
	if technicianID == 0 {
		return nil, apiErr{status: http.StatusForbidden, msg: "仅工作人员可扫码上号"}
	}
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var att models.TechnicianAttendance
	attRes := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("merchant_id = ? AND technician_id = ? AND checked_in_at >= ? AND checked_out_at IS NULL", merchantID, technicianID, start).
		Order("id desc").
		Limit(1).
		Find(&att)
	if attRes.Error != nil {
		return nil, attRes.Error
	}
	if attRes.RowsAffected == 0 {
		return nil, apiErr{status: http.StatusBadRequest, msg: "未上班签到，扫码上号失败"}
	}
	if att.Status != "idle" && att.Status != "busy" {
		if att.Status == "paused" {
			return nil, apiErr{status: http.StatusBadRequest, msg: "你目前在暂停服务中，请更新服务状态为空闲才可继续上号"}
		}
		return nil, apiErr{status: http.StatusBadRequest, msg: fmt.Sprintf("你目前%s，待服务完成后才可重新上号", technicianStatusConflictText(att.Status))}
	}
	if err := ensureNoOtherActiveServingSessionForTechnician(tx, merchantID, technicianID, sessionID); err != nil {
		return nil, err
	}
	return &att, nil
}

func ensureNoOtherActiveServingSessionForTechnician(tx *gorm.DB, merchantID uint, technicianID uint, currentSessionID uint) error {
	if tx == nil || merchantID == 0 || technicianID == 0 {
		return nil
	}
	var activeCnt int64
	if err := tx.Model(&models.ServiceSession{}).
		Where("merchant_id = ? AND technician_id = ? AND id <> ? AND status IN ?",
			merchantID, technicianID, currentSessionID,
			models.ExpandStatusesWithKnownPrefixes([]string{"serving", "auto_finishing"})).
		Count(&activeCnt).Error; err != nil {
		return err
	}
	if activeCnt > 0 {
		return apiErr{status: http.StatusBadRequest, msg: "你目前在服务中，待服务完成后才可重新上号"}
	}
	return nil
}

func promoteQueueSessionToServing(tx *gorm.DB, s *models.ServiceSession, now time.Time, allowedBaseStatuses []string, clearPendingTimeout bool) error {
	if tx == nil || s == nil {
		return nil
	}
	updates := map[string]interface{}{
		"status":     models.ApplyStatusPrefix(s.Status, "serving"),
		"started_at": now,
	}
	if s.StartConfirmedAt == nil {
		updates["start_confirmed_at"] = now
	}
	if clearPendingTimeout {
		updates["start_pending_timeout_seconds"] = 0
	}
	if s.DurationMinutes > 0 {
		finishAt := now.Add(time.Duration(s.DurationMinutes) * time.Minute)
		updates["scheduled_finish_at"] = finishAt
	}

	res := tx.Model(&models.ServiceSession{}).
		Where("id = ? AND status IN ?", s.ID, models.ExpandStatusesWithKnownPrefixes(allowedBaseStatuses)).
		Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected > 0 {
		if s.SourceType == "appointment" && s.SourceID != nil {
			if err := tx.Model(&models.Appointment{}).Where("id = ? AND actual_start_at IS NULL", *s.SourceID).Update("actual_start_at", now).Error; err != nil {
				return err
			}
		}
		return nil
	}

	var current models.ServiceSession
	if err := tx.Select("id", "status").First(&current, s.ID).Error; err != nil {
		return err
	}
	if models.NormalizeSessionStatus(current.Status) == "serving" {
		return nil
	}
	return apiErr{status: http.StatusBadRequest, msg: "当前状态不可扫码上号"}
}

func handleServiceSessionStartScan(c *gin.Context, raw string) bool {
	code := strings.TrimSpace(raw)
	if !strings.HasPrefix(code, "SS:") {
		return false
	}
	sidStr := strings.TrimSpace(strings.TrimPrefix(code, "SS:"))
	sid64, err := strconv.ParseUint(sidStr, 10, 64)
	if err != nil || sid64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的会话码"})
		return true
	}

	merchantIDAny, ok := c.Get("merchant_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return true
	}
	merchantID, _ := merchantIDAny.(uint)
	if merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return true
	}

	// 查询商户信息判断模式
	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return true
	}

	// 叫号模式：未开启客服模式 + 开启叫号 + 叫号模式（自动或手动）
	isQueueMode := !merchant.SupportCustomerServiceMode && merchant.SupportQueue && (merchant.QueueMode == "auto" || merchant.QueueMode == "manual")

	if isQueueMode {
		// 叫号模式：允许商户或工作人员扫码上号
		// 支持两种状态：
		// 1. delay_pending：单队列串行模式，扫码上号
		// 2. start_pending：多客服模式，工作人员扫码确认后进入 delay_pending，再扫码上号
		return handleQueueModeStartScan(c, uint(sid64), merchantID, &merchant)
	}

	startTerm := resolveStartTerm(&merchant)

	// 客服模式：仅工作人员可执行开始服务操作
	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)
	if authType != "staff" {
		c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("仅工作人员可%s", startTerm)})
		return true
	}
	techIDAny, ok := c.Get("technician_id")
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("仅工作人员可%s", startTerm)})
		return true
	}
	techIDVal, ok := techIDAny.(uint)
	if !ok || techIDVal == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("仅工作人员可%s", startTerm)})
		return true
	}
	techID := &techIDVal

	now := time.Now()
	var out models.ServiceSession
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		var s models.ServiceSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND merchant_id = ?", uint(sid64), merchantID).First(&s).Error; err != nil {
			return err
		}
		// 模式守卫：拒绝跨模式操作（如 queue 会话被 CS 入口处理）
		if err := models.ValidateSessionModeForEntry(&s, &merchant); err != nil {
			return apiErr{status: http.StatusBadRequest, msg: err.Error()}
		}
		baseStatus := models.NormalizeSessionStatus(s.Status)

		if merchant.SupportRoom && s.RoomID == nil {
			if err := assignRoomIfPossible(tx, &s, now); err != nil {
				return err
			}
			if s.RoomID == nil {
				return apiErr{status: http.StatusBadRequest, msg: "无可用房间"}
			}
		}

		defaultTechIDs := []uint(s.ServiceTechnicianIDs)
		if len(defaultTechIDs) == 0 {
			var err error
			defaultTechIDs, err = resolveProjectDefaultServiceTechnicianIDs(tx, merchantID, s.ProjectID)
			if err != nil {
				return err
			}
		}
		if s.TechnicianID != nil && *s.TechnicianID > 0 && *s.TechnicianID != *techID {
			if !uintIDInSlice(defaultTechIDs, *techID) {
				return apiErr{status: http.StatusBadRequest, msg: "该单已选择其他人员服务"}
			}
			s.TechnicianID = techID
		}
		if s.TechnicianID == nil {
			s.TechnicianID = techID
		}

		if s.TechnicianID == nil {
			return apiErr{status: http.StatusBadRequest, msg: "未选择工作人员"}
		}

		// 开始服务前校验：只有当专业客服(技师)为"空闲"才允许继续
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		var att models.TechnicianAttendance
		attRes := tx.
			Select("id", "merchant_id", "technician_id", "status").
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("merchant_id = ? AND technician_id = ? AND checked_in_at >= ? AND checked_out_at IS NULL", merchantID, *s.TechnicianID, start).
			Order("id desc").
			Limit(1).
			Find(&att)
		if attRes.Error != nil {
			return attRes.Error
		}
		if attRes.RowsAffected == 0 {
			startTerm := resolveStartTerm(&merchant)
			return apiErr{status: http.StatusBadRequest, msg: fmt.Sprintf("未上班签到，扫码%s失败", startTerm)}
		}
		hasOtherServingSession := false
		if att.Status == "busy" {
			if err := ensureNoOtherActiveServingSessionForTechnician(tx, merchantID, *s.TechnicianID, s.ID); err != nil {
				hasOtherServingSession = true
			}
		}
		if shouldBlockCustomerServiceStartByAttendance(att.Status, hasOtherServingSession) {
			if att.Status == "paused" {
				startTerm := resolveStartTerm(&merchant)
				return apiErr{status: http.StatusBadRequest, msg: fmt.Sprintf("你目前在暂停服务中，请更新服务状态为空闲才可继续%s", startTerm)}
			}
			startTerm := resolveStartTerm(&merchant)
			return apiErr{status: http.StatusBadRequest, msg: fmt.Sprintf("你目前%s，待服务完成后才可重新%s", technicianStatusConflictText(att.Status), startTerm)}
		}

		if baseStatus == "finished" || baseStatus == "canceled" {
			return apiErr{status: http.StatusBadRequest, msg: fmt.Sprintf("会话状态不可%s", startTerm)}
		}

		startAt := now.Add(time.Duration(s.StartDelaySeconds) * time.Second)
		updates := map[string]interface{}{
			"start_confirmed_at": now,
			"scheduled_start_at": startAt,
			"status":             models.ApplyStatusPrefix(s.Status, "delay_pending"),
			"technician_id":      *s.TechnicianID,
			"last_technician_id": *s.TechnicianID,
		}
		if len(defaultTechIDs) > 0 {
			updates["service_technician_ids"] = models.MerchantProjectDefaultServiceTechnicianIDs(defaultTechIDs)
		}
		if err := tx.Model(&models.ServiceSession{}).Where("id = ?", s.ID).Updates(updates).Error; err != nil {
			return err
		}
		if s.InitialUsageID > 0 {
			if err := tx.Model(&models.Usage{}).Where("id = ?", s.InitialUsageID).Update("technician_id", *s.TechnicianID).Error; err != nil {
				return err
			}
		}

		// 开始服务成功后占用技师：仅允许 idle -> busy，避免并发重复提交
		if att.Status == "idle" {
			res := tx.Model(&models.TechnicianAttendance{}).
				Where("id = ? AND merchant_id = ? AND technician_id = ? AND status = ?", att.ID, merchantID, *s.TechnicianID, "idle").
				Updates(map[string]interface{}{"status": "busy"})
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				startTerm := resolveStartTerm(&merchant)
				return apiErr{status: http.StatusBadRequest, msg: fmt.Sprintf("你目前在服务中，待服务完成后才可重新%s", startTerm)}
			}
		}

		outQuery := tx.Preload("Room").Preload("Technician")
		if tx.Dialector != nil && tx.Dialector.Name() == "sqlite" {
			outQuery = outQuery.Select("id", "merchant_id", "user_id", "card_id", "project_id", "initial_usage_id", "verify_code", "session_mode", "source_type", "source_id", "technician_id", "last_technician_id", "service_technician_ids", "room_id", "status", "start_delay_seconds", "duration_minutes", "auto_finish_delay_seconds", "auto_idle_after_seconds", "start_pending_timeout_seconds", "start_timeout_count")
		}
		if err := outQuery.First(&out, s.ID).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		var ae apiErr
		if errors.As(err, &ae) {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return true
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
			return true
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return true
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"action":     "start",
		"session_id": out.ID,
		"session":    out,
	}})
	return true
}

func uintIDInSlice(ids []uint, id uint) bool {
	for _, item := range ids {
		if item == id {
			return true
		}
	}
	return false
}

// handleQueueModeStartScan 叫号模式下的扫码上号处理
func handleQueueModeStartScan(c *gin.Context, sessionID uint, merchantID uint, merchant *models.Merchant) bool {
	now := time.Now()
	var out models.ServiceSession

	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)
	var scannerTechID uint
	if authType == "staff" {
		techIDAny, ok := c.Get("technician_id")
		if ok {
			v, _ := techIDAny.(uint)
			scannerTechID = v
		}
	}

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var s models.ServiceSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND merchant_id = ?", sessionID, merchantID).First(&s).Error; err != nil {
			return err
		}
		// 模式守卫：拒绝跨模式操作（如 CS 会话被叫号入口处理）
		if err := models.ValidateSessionModeForEntry(&s, merchant); err != nil {
			return apiErr{status: http.StatusBadRequest, msg: err.Error()}
		}

		// 叫号模式下，支持两种状态：
		// 1. start_pending：多窗口叫号，工作人员扫码后直接上号进入 serving（移除二次扫码）
		// 2. delay_pending：扫码上号 -> serving
		if models.NormalizeSessionStatus(s.Status) == "start_pending" {
			// 多窗口叫号：必须由被分配的工作人员扫码上号
			if scannerTechID == 0 {
				return apiErr{status: http.StatusForbidden, msg: "仅工作人员可扫码上号"}
			}
			// 多客服模式：工作人员扫码上号
			if s.TechnicianID == nil || *s.TechnicianID == 0 {
				return apiErr{status: http.StatusBadRequest, msg: "该单还未分配工作人员"}
			}
			if *s.TechnicianID != scannerTechID {
				return apiErr{status: http.StatusBadRequest, msg: "该单已分配其他工作人员"}
			}
			att, err := lockTechnicianAttendanceForQueueScan(tx, merchantID, *s.TechnicianID, s.ID, now)
			if err != nil {
				return err
			}

			if err := promoteQueueSessionToServing(tx, &s, now, []string{"start_pending"}, true); err != nil {
				return err
			}

			// 扫码上号成功后占用技师：仅在 idle 状态时更新为 busy，已是 busy 则跳过
			if att.Status == "idle" {
				if err := tx.Model(&models.TechnicianAttendance{}).
					Where("id = ? AND merchant_id = ? AND technician_id = ? AND status = ?", att.ID, merchantID, *s.TechnicianID, "idle").
					Updates(map[string]interface{}{"status": "busy"}).Error; err != nil {
					return err
				}
			}

			// 记录本次上号/服务人员（用于“今日上钟/服务记录”展示）
			if s.InitialUsageID > 0 {
				_ = tx.Model(&models.Usage{}).
					Where("id = ? AND merchant_id = ?", s.InitialUsageID, merchantID).
					Update("technician_id", scannerTechID).Error
			}

			if err := tx.First(&out, s.ID).Error; err != nil {
				return err
			}
			return nil
		}

		// delay_pending 状态：扫码上号
		baseStatus := models.NormalizeSessionStatus(s.Status)
		if baseStatus != "delay_pending" && baseStatus != "timeout_waiting" {
			if baseStatus == "serving" {
				return apiErr{status: http.StatusBadRequest, msg: "已开始服务，无需重复扫码"}
			}
			if baseStatus == "finished" {
				return apiErr{status: http.StatusBadRequest, msg: "服务已完成"}
			}
			if baseStatus == "canceled" {
				return apiErr{status: http.StatusBadRequest, msg: "服务已取消"}
			}
			if baseStatus == "staff_selecting" {
				return apiErr{status: http.StatusBadRequest, msg: "该号还在排队中，请等待叫号"}
			}
			return apiErr{status: http.StatusBadRequest, msg: "当前状态不可扫码上号"}
		}

		// qs_ 单窗口：timeout_waiting 状态下允许在插队窗口内再次扫码上号
		if baseStatus == "timeout_waiting" {
			if merchant == nil || !merchant.SupportQueue {
				return apiErr{status: http.StatusBadRequest, msg: "该号已被跳过"}
			}
			if queue.Default == nil {
				return apiErr{status: http.StatusBadRequest, msg: "该号已被跳过"}
			}
			date := now.Format("2006-01-02")

			// 手动叫号：固定窗口（3个号段）+ 最小宽延时间（15分钟），两者同时超过才失效退回
			if merchant.QueueMode == "manual" {
				if s.InitialUsageID == 0 {
					return apiErr{status: http.StatusBadRequest, msg: "该号已被跳过"}
				}
				snap := queue.Default.Snapshot(merchant.ID, date, queue.QueueTypeOnsite)
				minNo := 0
				if len(snap.Tickets) > 0 {
					minNo = snap.Tickets[0].No
				}
				maxCalledNo := snap.MaxCalledNo
				currentNo := maxCalledNo
				if currentNo <= 0 {
					currentNo = minNo
				}
				myNo, ok := queue.Default.GetNo(merchant.ID, date, queue.QueueTypeOnsite, s.InitialUsageID)
				if !ok || myNo <= 0 || currentNo <= 0 {
					return apiErr{status: http.StatusBadRequest, msg: "该号已被跳过"}
				}

				endNo := myNo + 3
				exceedNoWindow := currentNo >= endNo+1

				baseAt := s.StartTimeoutLastAt
				if baseAt == nil {
					baseAt = s.UpdatedAt
				}
				if baseAt == nil {
					baseAt = s.CreatedAt
				}
				exceedTimeWindow := false
				if baseAt != nil {
					exceedTimeWindow = now.Sub(*baseAt) > 15*time.Minute
				}

				if exceedNoWindow && exceedTimeWindow {
					if err := sessionflow.FailServiceSessionAndRefund(tx, &s, merchant, now, sessionflow.FailOptions{
						AllowedBaseStatuses: []string{"timeout_waiting"},
						TargetStatus:        "timeout_failed",
						MarkQueueDone:       true,
					}); err != nil {
						return err
					}
					return apiErr{status: http.StatusBadRequest, msg: "过号超时，该号已失效"}
				}
			}

			// 自动叫号单窗口：保留原有插队窗口限制（避免无限回补）。
			// NormalizeLegacySessionMode 统一处理历史空 session_mode 的回退逻辑。
			isQueueAutoSingleMode := models.NormalizeLegacySessionMode(&s, merchant) == models.SessionModeQueueAutoSingle
			if isQueueAutoSingleMode {
				if merchant.QueueMode != "auto" || merchant.SupportMultiCustomerService {
					return apiErr{status: http.StatusBadRequest, msg: "该号已被跳过"}
				}
				snap := queue.Default.Snapshot(merchant.ID, date, queue.QueueTypeOnsite)
				currentNo := 0
				if len(snap.Tickets) > 0 {
					currentNo = snap.Tickets[0].No
				}
				myNo, ok := queue.Default.GetNo(merchant.ID, date, queue.QueueTypeOnsite, s.InitialUsageID)
				if !ok || myNo <= 0 || currentNo <= 0 {
					return apiErr{status: http.StatusBadRequest, msg: "该号已被跳过"}
				}
				cnt := s.StartTimeoutCount
				if models.QsTimeoutWaitingExpired(currentNo, myNo, cnt) {
					return apiErr{status: http.StatusBadRequest, msg: "过号超时，该号已失效"}
				}
			}

			// 撤销 MarkDone/Uncall，让该号重新回到队列（手动叫号回补不做单窗口限制）。
			// 这里不主动 CallNextUncalled，避免将队列 current 推进到其它号码。
			// 该会话马上会进入 serving，队列推进由后续服务结束/调度处理。
			if s.InitialUsageID > 0 {
				queue.Default.UnmarkDone(merchant.ID, date, queue.QueueTypeOnsite, s.InitialUsageID)
				queue.Default.Uncall(merchant.ID, date, queue.QueueTypeOnsite, s.InitialUsageID)
			}
		}
		// 扫码上号：优先要求工作人员扫码；若尚未绑定工作人员，则绑定为当前扫码工作人员
		if scannerTechID == 0 {
			return apiErr{status: http.StatusForbidden, msg: "仅工作人员可扫码上号"}
		}
		if s.TechnicianID == nil || *s.TechnicianID == 0 {
			v := scannerTechID
			s.TechnicianID = &v
			if err := tx.Model(&models.ServiceSession{}).
				Where("id = ? AND merchant_id = ? AND status IN ? AND technician_id IS NULL", s.ID, merchantID, models.ExpandStatusesWithKnownPrefixes([]string{"delay_pending", "timeout_waiting"})).
				Updates(map[string]interface{}{"technician_id": scannerTechID, "last_technician_id": scannerTechID}).Error; err != nil {
				return err
			}
		} else {
			if *s.TechnicianID != scannerTechID {
				return apiErr{status: http.StatusBadRequest, msg: "该单已分配其他工作人员"}
			}
		}

		att, err := lockTechnicianAttendanceForQueueScan(tx, merchantID, scannerTechID, s.ID, now)
		if err != nil {
			return err
		}
		if att.Status == "idle" {
			if err := tx.Model(&models.TechnicianAttendance{}).
				Where("id = ? AND merchant_id = ? AND technician_id = ? AND status = ?", att.ID, merchantID, scannerTechID, "idle").
				Updates(map[string]interface{}{"status": "busy"}).Error; err != nil {
				return err
			}
		}

		// delay_pending 状态：超时检查；timeout_waiting 已由 qs_ 窗口判断处理
		if baseStatus == "delay_pending" {
			if s.ScheduledStartAt != nil {
				timeoutAt := s.ScheduledStartAt.Add(config.StartScanTimeout())
				if now.After(timeoutAt) {
					return apiErr{status: http.StatusBadRequest, msg: "上号超时，该号已被跳过"}
				}
			}
		}

		if err := promoteQueueSessionToServing(tx, &s, now, []string{"delay_pending", "timeout_waiting"}, false); err != nil {
			return err
		}
		// 记录本次上号/服务人员（用于“今日上钟/服务记录”展示）
		if s.InitialUsageID > 0 {
			_ = tx.Model(&models.Usage{}).
				Where("id = ? AND merchant_id = ?", s.InitialUsageID, merchantID).
				Update("technician_id", scannerTechID).Error
		}

		if err := tx.First(&out, s.ID).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		var ae apiErr
		if errors.As(err, &ae) {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return true
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
			return true
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return true
	}

	// 根据会话状态返回不同的消息
	message := "上号成功，服务已开始"
	if models.NormalizeSessionStatus(out.Status) == "delay_pending" {
		message = resolveStartTerm(merchant) + "成功，请再次扫码上号"
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"action":     "start",
		"session_id": out.ID,
		"session":    out,
		"message":    message,
	}})
	return true
}

func assignRoomIfPossible(tx *gorm.DB, s *models.ServiceSession, now time.Time) error {
	var rooms []models.Room
	if err := tx.Where("merchant_id = ? AND is_active = ?", s.MerchantID, true).Order("id asc").Find(&rooms).Error; err != nil {
		return err
	}
	for i := range rooms {
		r := rooms[i]
		var cnt int64
		if err := tx.Model(&models.ServiceSession{}).
			Where("merchant_id = ? AND room_id = ? AND status IN ?", s.MerchantID, r.ID, models.ExpandStatusesWithKnownPrefixes([]string{"room_locked", "staff_selecting", "start_pending", "delay_pending", "serving", "auto_finishing"})).
			Count(&cnt).Error; err != nil {
			return err
		}
		if cnt == 0 {
			lockedAt := now
			s.RoomID = &r.ID
			s.RoomLockedAt = &lockedAt
			return tx.Model(&models.ServiceSession{}).Where("id = ?", s.ID).Updates(map[string]interface{}{
				"room_id":        r.ID,
				"room_locked_at": lockedAt,
				"status":         models.ApplyStatusPrefix(s.Status, "room_locked"),
			}).Error
		}
	}
	return nil
}

func GetServiceSession(c *gin.Context) {
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

	id := c.Param("id")
	var s models.ServiceSession
	if err := config.DB.Preload("Room").Preload("Technician").Preload("Technician.ServiceRole").Preload("LastTechnician").Preload("LastTechnician.ServiceRole").Preload("Project").Preload("Card").Preload("InitialUsage").Preload("InitialUsage.Technician").Preload("Appointment").Preload("Merchant").Where("id = ? AND merchant_id = ?", id, merchantID).First(&s).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}
	snapshotSessions := []models.ServiceSession{s}
	enrichServiceSessionsWithCardSnapshots(snapshotSessions)
	s = snapshotSessions[0]
	s.StartPendingRemainingSeconds = computeStartPendingRemainingSecondsForSession(&s, time.Now())
	attachLatestExtendRequestToSession(&s)
	serviceSessions := []models.ServiceSession{s}
	enrichServiceSessionsWithServiceTechnicians(serviceSessions)
	s = serviceSessions[0]
	c.JSON(http.StatusOK, gin.H{"data": s})
}

func ListServiceSessions(c *gin.Context) {
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

	status := c.Query("status")
	dateText := strings.TrimSpace(c.Query("date"))
	selfOnly := strings.TrimSpace(c.Query("self_only")) == "1"
	q := config.DB.Preload("Room").Preload("Technician").Preload("Technician.ServiceRole").Preload("LastTechnician").Preload("LastTechnician.ServiceRole").Preload("Project").Preload("Card").Preload("InitialUsage").Preload("InitialUsage.Technician").Preload("Appointment").Preload("Merchant").Where("merchant_id = ?", merchantID)
	if selfOnly {
		authTypeAny, _ := c.Get("auth_type")
		authType, _ := authTypeAny.(string)
		technicianIDAny, _ := c.Get("technician_id")
		technicianID, _ := technicianIDAny.(uint)
		if authType == "staff" && technicianID > 0 {
			q = q.Where("(technician_id = ? OR last_technician_id = ?)", technicianID, technicianID)
		}
	}
	if status != "" {
		st := strings.TrimSpace(status)
		if st != "" {
			q = q.Where("status IN ?", models.ExpandStatusWithKnownPrefixes(st))
		}
	}
	if dateText != "" {
		loc := appointmentLocation()
		targetDate, err := time.ParseInLocation("2006-01-02", dateText, loc)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "日期格式错误"})
			return
		}
		dayStart := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, loc)
		dayEnd := dayStart.Add(24 * time.Hour)
		q = q.Where("created_at >= ? AND created_at < ?", dayStart, dayEnd)
	}

	var list []models.ServiceSession
	q.Order("id desc").Limit(200).Find(&list)
	enrichServiceSessionsWithCardSnapshots(list)
	now := time.Now()
	for i := range list {
		list[i].StartPendingRemainingSeconds = computeStartPendingRemainingSecondsForSession(&list[i], now)
	}
	enrichServiceSessionsWithServiceTechnicians(list)
	attachLatestExtendRequestsToSessions(list)
	c.JSON(http.StatusOK, gin.H{"data": list})
}

func enrichServiceSessionsWithServiceTechnicians(sessions []models.ServiceSession) {
	ids := make([]uint, 0)
	for i := range sessions {
		for _, id := range sessions[i].ServiceTechnicianIDs {
			if id > 0 {
				ids = append(ids, id)
			}
		}
		if len(sessions[i].ServiceTechnicianIDs) == 0 && sessions[i].TechnicianID != nil && *sessions[i].TechnicianID > 0 {
			ids = append(ids, *sessions[i].TechnicianID)
		}
	}
	if len(ids) == 0 {
		return
	}
	var techs []models.Technician
	if err := config.DB.Preload("ServiceRole").Where("id IN ?", ids).Find(&techs).Error; err != nil {
		return
	}
	byID := make(map[uint]*models.Technician, len(techs))
	for i := range techs {
		byID[techs[i].ID] = &techs[i]
	}
	for i := range sessions {
		orderedIDs := []uint(sessions[i].ServiceTechnicianIDs)
		if len(orderedIDs) == 0 && sessions[i].TechnicianID != nil && *sessions[i].TechnicianID > 0 {
			orderedIDs = []uint{*sessions[i].TechnicianID}
		}
		for _, id := range orderedIDs {
			if tech, ok := byID[id]; ok {
				sessions[i].ServiceTechnicians = append(sessions[i].ServiceTechnicians, tech)
			}
		}
	}
}

func ChooseServiceSessionRoom(c *gin.Context) {
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

	var input struct {
		RoomID uint `json:"room_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sid := c.Param("id")
	now := time.Now()
	var out models.ServiceSession
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var s models.ServiceSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND merchant_id = ?", sid, merchantID).First(&s).Error; err != nil {
			return err
		}
		baseStatus := models.NormalizeSessionStatus(s.Status)
		if baseStatus == "finished" || baseStatus == "canceled" {
			return apiErr{status: http.StatusBadRequest, msg: "会话已结束"}
		}

		var m models.Merchant
		if err := tx.First(&m, merchantID).Error; err != nil {
			return err
		}
		// 模式守卫：拒绝跨模式操作
		if err := models.ValidateSessionModeForEntry(&s, &m); err != nil {
			return apiErr{status: http.StatusBadRequest, msg: err.Error()}
		}
		lockedAt := now

		updates := map[string]interface{}{
			"room_id":                 input.RoomID,
			"room_locked_at":          lockedAt,
			"room_select_deadline_at": nil,
		}
		if m.SupportCustomerServiceMode {
			defaultTechIDs := []uint(s.ServiceTechnicianIDs)
			if len(defaultTechIDs) == 0 {
				var err error
				defaultTechIDs, err = resolveProjectDefaultServiceTechnicianIDs(tx, merchantID, s.ProjectID)
				if err != nil {
					return err
				}
			}
			if len(defaultTechIDs) > 0 {
				techID := defaultTechIDs[0]
				updates["technician_id"] = techID
				updates["last_technician_id"] = techID
				updates["service_technician_ids"] = models.MerchantProjectDefaultServiceTechnicianIDs(defaultTechIDs)
				updates["status"] = models.ApplyStatusPrefix(s.Status, "start_pending")
				updates["start_pending_timeout_seconds"] = config.ResolveServiceSessionStartPendingTimeoutSeconds(tx, merchantID, techID, s.ProjectID)
				updates["staff_select_entered_at"] = nil
				updates["staff_select_cooldown_until"] = nil
			} else {
				updates["status"] = models.ApplyStatusPrefix(s.Status, "room_locked")
			}
		} else {
			delaySeconds := s.StartDelaySeconds
			if delaySeconds <= 0 {
				delaySeconds = 60
			}
			startAt := now.Add(time.Duration(delaySeconds) * time.Second)
			updates["status"] = models.ApplyStatusPrefix(s.Status, "delay_pending")
			updates["start_confirmed_at"] = now
			updates["scheduled_start_at"] = startAt
		}
		if err := tx.Model(&models.ServiceSession{}).Where("id = ?", s.ID).Updates(updates).Error; err != nil {
			return err
		}
		outQuery := tx.Preload("Room").Preload("Technician")
		if tx.Dialector != nil && tx.Dialector.Name() == "sqlite" {
			outQuery = outQuery.Select("id", "merchant_id", "user_id", "card_id", "project_id", "initial_usage_id", "verify_code", "session_mode", "source_type", "source_id", "technician_id", "last_technician_id", "service_technician_ids", "room_id", "status", "start_delay_seconds", "duration_minutes", "auto_finish_delay_seconds", "auto_idle_after_seconds", "start_pending_timeout_seconds", "start_timeout_count")
		}
		return outQuery.First(&out, s.ID).Error
	})
	if err != nil {
		var ae apiErr
		if errors.As(err, &ae) {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": out})
}

func ChooseServiceSessionTechnician(c *gin.Context) {
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

	var input struct {
		TechnicianID uint `json:"technician_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sid := c.Param("id")
	now := time.Now()
	var out models.ServiceSession
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var s models.ServiceSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND merchant_id = ?", sid, merchantID).First(&s).Error; err != nil {
			return err
		}
		baseStatus := models.NormalizeSessionStatus(s.Status)
		if baseStatus == "finished" || baseStatus == "canceled" {
			return apiErr{status: http.StatusBadRequest, msg: "会话已结束"}
		}
		// 商户端手动改派只允许发生在“还没正式开始服务”的阶段。
		if baseStatus != "room_locked" && baseStatus != "staff_selecting" {
			return apiErr{status: http.StatusBadRequest, msg: "当前会话状态不支持改派客服"}
		}
		var m models.Merchant
		if err := tx.First(&m, merchantID).Error; err != nil {
			return err
		}
		// 模式守卫：拒绝跨模式操作
		if err := models.ValidateSessionModeForEntry(&s, &m); err != nil {
			return apiErr{status: http.StatusBadRequest, msg: err.Error()}
		}
		if m.SupportRoom && s.RoomID == nil {
			return apiErr{status: http.StatusBadRequest, msg: "请先选择房间"}
		}
		var tech models.Technician
		if err := tx.Where("id = ? AND merchant_id = ?", input.TechnicianID, merchantID).First(&tech).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apiErr{status: http.StatusBadRequest, msg: "无效的工作人员"}
			}
			return err
		}
		var activeServingCnt int64
		if err := tx.Model(&models.ServiceSession{}).
			Where("merchant_id = ? AND technician_id = ? AND status IN ?",
				merchantID, input.TechnicianID,
				models.ExpandStatusesWithKnownPrefixes([]string{"serving", "auto_finishing"})).
			Count(&activeServingCnt).Error; err != nil {
			return err
		}
		if activeServingCnt > 0 {
			return apiErr{status: http.StatusBadRequest, msg: "该工作人员当前正在服务中，请先完成当前服务再分配"}
		}
		timeoutSeconds := config.ResolveServiceSessionStartPendingTimeoutSeconds(tx, merchantID, input.TechnicianID, s.ProjectID)
		updates := map[string]interface{}{
			"technician_id":                 input.TechnicianID,
			"last_technician_id":            input.TechnicianID,
			"service_technician_ids":        models.MerchantProjectDefaultServiceTechnicianIDs{input.TechnicianID},
			"status":                        models.ApplyStatusPrefix(s.Status, "start_pending"),
			"staff_select_entered_at":       nil,
			"staff_select_cooldown_until":   nil,
			"predicted_ready_at":            nil,
			"start_pending_timeout_seconds": timeoutSeconds,
		}
		if err := tx.Model(&models.ServiceSession{}).Where("id = ?", s.ID).Updates(updates).Error; err != nil {
			return err
		}
		_ = now
		outQuery := tx.Preload("Room").Preload("Technician")
		if tx.Dialector != nil && tx.Dialector.Name() == "sqlite" {
			outQuery = outQuery.Select("id", "merchant_id", "user_id", "card_id", "project_id", "initial_usage_id", "verify_code", "session_mode", "source_type", "source_id", "technician_id", "last_technician_id", "service_technician_ids", "room_id", "status", "start_delay_seconds", "duration_minutes", "auto_finish_delay_seconds", "auto_idle_after_seconds", "start_pending_timeout_seconds", "start_timeout_count")
		}
		return outQuery.First(&out, s.ID).Error
	})
	if err != nil {
		var ae apiErr
		if errors.As(err, &ae) {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

func ExtendServiceSession(c *gin.Context) {
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

	sid := c.Param("id")
	now := time.Now()
	addMinutes := 50
	var remainTimes int
	var out models.ServiceSession

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var s models.ServiceSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND merchant_id = ?", sid, merchantID).First(&s).Error; err != nil {
			return err
		}
		if models.NormalizeSessionStatus(s.Status) == "finished" || models.NormalizeSessionStatus(s.Status) == "canceled" {
			return apiErr{status: http.StatusBadRequest, msg: "会话已结束"}
		}
		var card models.Card
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&card, s.CardID).Error; err != nil {
			return err
		}
		if card.RemainTimes <= 0 {
			return apiErr{status: http.StatusBadRequest, msg: "剩余次数不足"}
		}
		if err := tx.Model(&models.Card{}).Where("id = ? AND remain_times > 0", card.ID).Updates(map[string]interface{}{
			"remain_times": gorm.Expr("remain_times - ?", 1),
			"used_times":   gorm.Expr("used_times + ?", 1),
			"last_used_at": now,
		}).Error; err != nil {
			return err
		}
		if err := tx.First(&card, card.ID).Error; err != nil {
			return err
		}
		remainTimes = card.RemainTimes

		extCode := fmt.Sprintf("EXT-%d-%d", s.ID, now.Unix())
		u := models.Usage{
			CardID:     card.ID,
			MerchantID: card.MerchantID,
			UsedTimes:  1,
			UsedAt:     &now,
			VerifyCode: extCode,
			Status:     "success",
			FinishedAt: &now,
		}
		applyUsageCardSnapshotFromCard(&u, card)
		if err := tx.Create(&u).Error; err != nil {
			return err
		}

		newDuration := s.DurationMinutes + addMinutes
		updates := map[string]interface{}{"duration_minutes": newDuration}
		if s.ScheduledFinishAt != nil {
			updates["scheduled_finish_at"] = s.ScheduledFinishAt.Add(time.Duration(addMinutes) * time.Minute)
		} else if s.StartedAt != nil {
			updates["scheduled_finish_at"] = s.StartedAt.Add(time.Duration(newDuration) * time.Minute)
		}
		if err := tx.Model(&models.ServiceSession{}).Where("id = ?", s.ID).Updates(updates).Error; err != nil {
			return err
		}
		return tx.Preload("Room").Preload("Technician").First(&out, s.ID).Error
	})
	if err != nil {
		var ae apiErr
		if errors.As(err, &ae) {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": out, "remain_times": remainTimes})
}

func ExtendServiceSessionDuration(c *gin.Context) {
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

	var input struct {
		AddMinutes *int `json:"add_minutes"`
		Minutes    *int `json:"minutes"`
	}
	_ = c.ShouldBindJSON(&input)

	addMinutes := 50
	if input.AddMinutes != nil && *input.AddMinutes > 0 {
		addMinutes = *input.AddMinutes
	} else if input.Minutes != nil && *input.Minutes > 0 {
		addMinutes = *input.Minutes
	}
	if addMinutes < 5 || addMinutes > 180 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "minutes 范围应为 5-180"})
		return
	}

	sid := c.Param("id")
	var out models.ServiceSession
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var s models.ServiceSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND merchant_id = ?", sid, merchantID).First(&s).Error; err != nil {
			return err
		}
		if models.NormalizeSessionStatus(s.Status) != "serving" {
			return apiErr{status: http.StatusBadRequest, msg: "仅服务中可延长"}
		}

		if err := applyServiceSessionExtension(tx, &s, addMinutes); err != nil {
			return err
		}
		return tx.Preload("Room").Preload("Technician").First(&out, s.ID).Error
	})
	if err != nil {
		var ae apiErr
		if errors.As(err, &ae) {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}
