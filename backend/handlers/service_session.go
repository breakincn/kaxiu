package handlers

import (
	"errors"
	"fmt"
	"kabao/config"
	"kabao/models"
	"kabao/queue"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func technicianServiceStatusText(status string) string {
	switch strings.TrimSpace(status) {
	case "idle":
		return "空闲"
	case "busy":
		return "服务中"
	case "paused":
		return "暂停服务"
	case "rest":
		return "下班"
	default:
		return "未知状态"
	}
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
		// 2. start_pending：多客服模式，工作人员扫码起单后进入 delay_pending，再扫码上号
		return handleQueueModeStartScan(c, uint(sid64), merchantID, &merchant)
	}

	// 客服模式：仅工作人员可起单
	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)
	if authType != "staff" {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅工作人员可起单"})
		return true
	}
	techIDAny, ok := c.Get("technician_id")
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅工作人员可起单"})
		return true
	}
	techIDVal, ok := techIDAny.(uint)
	if !ok || techIDVal == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅工作人员可起单"})
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
		baseStatus := models.NormalizeSessionStatus(s.Status)

		if merchant.SupportRoom && s.RoomID == nil {
			if err := assignRoomIfPossible(tx, &s, now); err != nil {
				return err
			}
			if s.RoomID == nil {
				return apiErr{status: http.StatusBadRequest, msg: "无可用房间"}
			}
		}

		if s.TechnicianID != nil && *s.TechnicianID > 0 && *s.TechnicianID != *techID {
			return apiErr{status: http.StatusBadRequest, msg: "该单已选择其他人员服务"}
		}
		if s.TechnicianID == nil {
			s.TechnicianID = techID
		}

		if s.TechnicianID == nil {
			return apiErr{status: http.StatusBadRequest, msg: "未选择工作人员"}
		}

		// 起单前校验：只有当专业客服(技师)为"空闲"才允许起单
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		var att models.TechnicianAttendance
		attRes := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("merchant_id = ? AND technician_id = ? AND checked_in_at >= ? AND checked_out_at IS NULL", merchantID, *s.TechnicianID, start).
			Order("id desc").
			Limit(1).
			Find(&att)
		if attRes.Error != nil {
			return attRes.Error
		}
		if attRes.RowsAffected == 0 {
			startTerm := "起单"
			if merchant.StartTerm != "" {
				startTerm = merchant.StartTerm
			}
			return apiErr{status: http.StatusBadRequest, msg: fmt.Sprintf("未上班签到，扫码%s失败", startTerm)}
		}
		if att.Status != "idle" {
			if att.Status == "paused" {
				startTerm := "起单"
				if merchant.StartTerm != "" {
					startTerm = merchant.StartTerm
				}
				return apiErr{status: http.StatusBadRequest, msg: fmt.Sprintf("你目前在暂停服务中，请更新服务状态为空闲才可继续%s", startTerm)}
			}
			startTerm := "起单"
			if merchant.StartTerm != "" {
				startTerm = merchant.StartTerm
			}
			return apiErr{status: http.StatusBadRequest, msg: fmt.Sprintf("你目前在%s中，待服务完成后才可重新%s", technicianServiceStatusText(att.Status), startTerm)}
		}

		if baseStatus == "finished" || baseStatus == "canceled" {
			return apiErr{status: http.StatusBadRequest, msg: "会话状态不可起单"}
		}

		startAt := now.Add(time.Duration(s.StartDelaySeconds) * time.Second)
		updates := map[string]interface{}{
			"start_confirmed_at": now,
			"scheduled_start_at": startAt,
			"status":             models.ApplyStatusPrefix(s.Status, "delay_pending"),
		}
		if err := tx.Model(&models.ServiceSession{}).Where("id = ?", s.ID).Updates(updates).Error; err != nil {
			return err
		}

		// 起单成功后占用技师：仅允许 idle -> busy，避免并发重复起单
		res := tx.Model(&models.TechnicianAttendance{}).
			Where("id = ? AND merchant_id = ? AND technician_id = ? AND status = ?", att.ID, merchantID, *s.TechnicianID, "idle").
			Updates(map[string]interface{}{"status": "busy"})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			startTerm := "起单"
			if merchant.StartTerm != "" {
				startTerm = merchant.StartTerm
			}
			return apiErr{status: http.StatusBadRequest, msg: fmt.Sprintf("你目前在服务中，待服务完成后才可重新%s", startTerm)}
		}

		if err := tx.Preload("Room").Preload("Technician").First(&out, s.ID).Error; err != nil {
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
			// 上号前校验：允许技师状态为 idle 或 busy（自动叫号多客服模式分配时已置为 busy，手动叫号模式分配时仍为 idle）
			start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			var att models.TechnicianAttendance
			attRes := tx.
				Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("merchant_id = ? AND technician_id = ? AND checked_in_at >= ? AND checked_out_at IS NULL", merchantID, *s.TechnicianID, start).
				Order("id desc").
				Limit(1).
				Find(&att)
			if attRes.Error != nil {
				return attRes.Error
			}
			if attRes.RowsAffected == 0 {
				return apiErr{status: http.StatusBadRequest, msg: "未上班签到，扫码上号失败"}
			}
			// 允许 idle 或 busy 状态上号（兼容自动/手动叫号模式）
			if att.Status != "idle" && att.Status != "busy" {
				if att.Status == "paused" {
					return apiErr{status: http.StatusBadRequest, msg: "你目前在暂停服务中，请更新服务状态为空闲才可继续上号"}
				}
				return apiErr{status: http.StatusBadRequest, msg: fmt.Sprintf("你目前在%s中，待服务完成后才可重新上号", technicianServiceStatusText(att.Status))}
			}

			// start_pending -> serving（扫码上号），避免二次扫码
			updates := map[string]interface{}{
				"start_confirmed_at":            now,
				"status":                        models.ApplyStatusPrefix(s.Status, "serving"),
				"started_at":                    now,
				"start_pending_timeout_seconds": 0,
			}
			if s.DurationMinutes > 0 {
				finishAt := now.Add(time.Duration(s.DurationMinutes) * time.Minute)
				updates["scheduled_finish_at"] = finishAt
			}
			if err := tx.Model(&models.ServiceSession{}).Where("id = ? AND status IN ?", s.ID, models.ExpandStatusWithKnownPrefixes("start_pending")).Updates(updates).Error; err != nil {
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

			// 记录本次上号/服务人员（用于“今日上钟/起单”展示）
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
				currentNo := 0
				if len(snap.Tickets) > 0 {
					currentNo = snap.Tickets[0].No
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
					updates := map[string]interface{}{
						"status":      models.ApplyStatusPrefix(s.Status, "timeout_failed"),
						"finished_at": now,
					}
					if err := tx.Model(&models.ServiceSession{}).
						Where("id = ? AND status IN ?", s.ID, models.ExpandStatusWithKnownPrefixes("timeout_waiting")).
						Updates(updates).Error; err != nil {
						return err
					}

					if err := tx.Model(&models.Usage{}).
						Where("id = ? AND status = ?", s.InitialUsageID, "in_progress").
						Updates(map[string]interface{}{
							"status":      "failed",
							"finished_at": now,
						}).Error; err != nil {
						return err
					}

					var usage models.Usage
					if err := tx.First(&usage, s.InitialUsageID).Error; err == nil {
						if err := tx.Model(&models.Card{}).
							Where("id = ?", usage.CardID).
							Updates(map[string]interface{}{
								"remain_times": gorm.Expr("remain_times + ?", usage.UsedTimes),
								"used_times":   gorm.Expr("used_times - ?", usage.UsedTimes),
							}).Error; err != nil {
							return err
						}
					}

					return apiErr{status: http.StatusBadRequest, msg: "过号超时，该号已失效"}
				}
			}

			// 自动叫号单窗口：保留原有插队窗口限制（避免无限回补）
			if s.SessionMode == models.SessionModeQueueAutoSingle {
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

			// 撤销 MarkDone/Uncall，让该号重新回到队列（手动叫号回补不做单窗口限制）
			if s.InitialUsageID > 0 {
				queue.Default.UnmarkDone(merchant.ID, date, queue.QueueTypeOnsite, s.InitialUsageID)
				queue.Default.Uncall(merchant.ID, date, queue.QueueTypeOnsite, s.InitialUsageID)
				queue.Default.CallNextUncalled(merchant.ID, date, queue.QueueTypeOnsite, now)
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
				Update("technician_id", scannerTechID).Error; err != nil {
				return err
			}
		} else {
			if *s.TechnicianID != scannerTechID {
				return apiErr{status: http.StatusBadRequest, msg: "该单已分配其他工作人员"}
			}
		}

		// 校验并占用技师：仅允许 idle -> busy，避免仍显示空闲
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		var att models.TechnicianAttendance
		attRes := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("merchant_id = ? AND technician_id = ? AND checked_in_at >= ? AND checked_out_at IS NULL", merchantID, scannerTechID, start).
			Order("id desc").
			Limit(1).
			Find(&att)
		if attRes.Error != nil {
			return attRes.Error
		}
		if attRes.RowsAffected == 0 {
			return apiErr{status: http.StatusBadRequest, msg: "未上班签到，扫码上号失败"}
		}
		if att.Status != "idle" && att.Status != "busy" {
			if att.Status == "paused" {
				return apiErr{status: http.StatusBadRequest, msg: "你目前在暂停服务中，请更新服务状态为空闲才可继续上号"}
			}
			return apiErr{status: http.StatusBadRequest, msg: fmt.Sprintf("你目前在%s中，待服务完成后才可重新上号", technicianServiceStatusText(att.Status))}
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

		// 直接推进到 serving 状态
		updates := map[string]interface{}{
			"status":     models.ApplyStatusPrefix(s.Status, "serving"),
			"started_at": now,
		}
		if s.DurationMinutes > 0 {
			finishAt := now.Add(time.Duration(s.DurationMinutes) * time.Minute)
			updates["scheduled_finish_at"] = finishAt
		}
		if err := tx.Model(&models.ServiceSession{}).
			Where("id = ? AND status IN ?", s.ID, models.ExpandStatusesWithKnownPrefixes([]string{"delay_pending", "timeout_waiting"})).
			Updates(updates).Error; err != nil {
			return err
		}
		// 记录本次上号/服务人员（用于“今日上钟/起单”展示）
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
		message = "起单成功，请再次扫码上号"
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
	if err := config.DB.Preload("Room").Preload("Technician").Preload("Technician.ServiceRole").Preload("Project").Where("id = ? AND merchant_id = ?", id, merchantID).First(&s).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}
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
	q := config.DB.Preload("Room").Preload("Technician").Preload("Technician.ServiceRole").Preload("Project").Where("merchant_id = ?", merchantID)
	if status != "" {
		st := strings.TrimSpace(status)
		if st != "" {
			q = q.Where("status IN ?", models.ExpandStatusWithKnownPrefixes(st))
		}
	}

	var list []models.ServiceSession
	q.Order("id desc").Limit(200).Find(&list)
	c.JSON(http.StatusOK, gin.H{"data": list})
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
		lockedAt := now

		updates := map[string]interface{}{
			"room_id":                 input.RoomID,
			"room_locked_at":          lockedAt,
			"room_select_deadline_at": nil,
		}
		if m.SupportCustomerServiceMode {
			updates["status"] = models.ApplyStatusPrefix(s.Status, "room_locked")
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
		var m models.Merchant
		if err := tx.First(&m, merchantID).Error; err != nil {
			return err
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
		updates := map[string]interface{}{
			"technician_id":                 input.TechnicianID,
			"status":                        models.ApplyStatusPrefix(s.Status, "start_pending"),
			"staff_select_entered_at":       nil,
			"staff_select_cooldown_until":   nil,
			"start_pending_timeout_seconds": int(config.StartPendingTimeout().Seconds()),
		}
		if err := tx.Model(&models.ServiceSession{}).Where("id = ?", s.ID).Updates(updates).Error; err != nil {
			return err
		}
		_ = now
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

		updates := map[string]interface{}{
			"duration_minutes":          gorm.Expr("duration_minutes + ?", addMinutes),
			"auto_finish_delay_seconds": gorm.Expr("auto_finish_delay_seconds + ?", addMinutes*60),
		}
		if s.ScheduledFinishAt != nil {
			updates["scheduled_finish_at"] = s.ScheduledFinishAt.Add(time.Duration(addMinutes) * time.Minute)
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
	c.JSON(http.StatusOK, gin.H{"data": out})
}
