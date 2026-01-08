package handlers

import (
	"kabao/config"
	"kabao/models"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

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

func getCooldownUntil(userID uint, merchantID uint) (*time.Time, error) {
	var lastCanceled models.Appointment
	err := config.DB.
		Where("user_id = ? AND merchant_id = ? AND status = 'canceled'", userID, merchantID).
		Order("canceled_at DESC").
		Limit(1).
		First(&lastCanceled).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	if lastCanceled.CanceledAt == nil {
		return nil, nil
	}
	until := lastCanceled.CanceledAt.Add(1 * time.Hour)
	return &until, nil
}

func GetMerchantAppointments(c *gin.Context) {
	merchantID := c.Param("id")
	status := c.Query("status")

	var appointments []models.Appointment
	query := config.DB.Preload("User").Preload("Technician").Where("merchant_id = ?", merchantID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 如果是技师登录，且只有预约权限（没有管理预约权限），则只显示预约自己的预约
	authType, _ := c.Get("auth_type")
	if authType == "technician" {
		technicianIDAny, ok := c.Get("technician_id")
		if ok {
			if technicianID, ok := technicianIDAny.(uint); ok && technicianID > 0 {
				// 检查是否有管理预约权限
				hasManagePermission := false
				if merchantIDAny, ok := c.Get("merchant_id"); ok {
					if mID, ok := merchantIDAny.(uint); ok {
						if serviceRoleIDAny, ok := c.Get("service_role_id"); ok {
							if serviceRoleID, ok := serviceRoleIDAny.(uint); ok {
								// 查找管理预约权限
								var managePerm models.Permission
								if err := config.DB.Where("`key` = ?", "merchant.appointment.manage").First(&managePerm).Error; err == nil {
									// 检查商户级别的权限覆盖
									var override models.MerchantRolePermissionOverride
									err := config.DB.Where("merchant_id = ? AND service_role_id = ? AND permission_id = ?", mID, serviceRoleID, managePerm.ID).First(&override).Error
									if err == nil {
										hasManagePermission = override.Allowed
									} else {
										// 检查全局角色权限
										var rolePerm models.RolePermission
										err = config.DB.Where("service_role_id = ? AND permission_id = ? AND allowed = ?", serviceRoleID, managePerm.ID, true).First(&rolePerm).Error
										if err == nil {
											hasManagePermission = true
										}
									}
								}
							}
						}
					}
				}

				// 如果没有管理预约权限，则只显示预约自己的预约
				if !hasManagePermission {
					query = query.Where("technician_id = ?", technicianID)
				}
			}
		}
	}

	query.Order("appointment_time ASC").Find(&appointments)
	c.JSON(http.StatusOK, gin.H{"data": appointments})
}

func GetUserAppointments(c *gin.Context) {
	userID := c.Param("id")
	var appointments []models.Appointment
	config.DB.Preload("Merchant").Preload("Technician").Where("user_id = ?", userID).Order("appointment_time DESC").Find(&appointments)
	c.JSON(http.StatusOK, gin.H{"data": appointments})
}

func GetCardAppointment(c *gin.Context) {
	cardID := c.Param("id")

	var card models.Card
	if err := config.DB.First(&card, cardID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "卡片不存在"})
		return
	}

	// 添加调试日志
	log.Printf("查询预约: 卡片ID=%s, 用户ID=%d, 商户ID=%d", cardID, card.UserID, card.MerchantID)

	var appointment models.Appointment
	err := config.DB.Preload("Merchant").Preload("Technician").
		Where("user_id = ? AND merchant_id = ? AND status IN ('pending', 'confirmed')", card.UserID, card.MerchantID).
		Order("appointment_time ASC").
		First(&appointment).Error

	if err != nil {
		log.Printf("未找到预约: %v", err)
		cooldownUntil, cooldownErr := getCooldownUntil(card.UserID, card.MerchantID)
		if cooldownErr != nil {
			log.Printf("查询冷却时间失败: %v", cooldownErr)
			cooldownUntil = nil
		}
		c.JSON(http.StatusOK, gin.H{"data": gin.H{
			"appointment":       nil,
			"queue_before":      0,
			"estimated_minutes": 0,
			"cooldown_until":    cooldownUntil,
		}})
		return
	}

	log.Printf("找到预约: ID=%d, 状态=%s, 时间=%v", appointment.ID, appointment.Status, appointment.AppointmentTime)

	now := time.Now()
	autoCanceled, autoCancelErr := autoCancelAppointmentIfOverdue(&appointment, now)
	if autoCancelErr != nil {
		log.Printf("自动取消预约失败: %v", autoCancelErr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "自动取消预约失败"})
		return
	}
	if autoCanceled {
		cooldownUntil := now.Add(1 * time.Hour)
		c.JSON(http.StatusOK, gin.H{"data": gin.H{
			"appointment":       nil,
			"queue_before":      0,
			"estimated_minutes": 0,
			"cooldown_until":    &cooldownUntil,
		}})
		return
	}

	// 计算排队信息
	var queueBefore int64
	if appointment.AppointmentTime != nil {
		config.DB.Model(&models.Appointment{}).
			Where("merchant_id = ? AND status IN ('pending', 'confirmed') AND appointment_time < ?",
				card.MerchantID, appointment.AppointmentTime).
			Count(&queueBefore)
	}

	var merchant models.Merchant
	config.DB.First(&merchant, card.MerchantID)

	cooldownUntil, cooldownErr := getCooldownUntil(card.UserID, card.MerchantID)
	if cooldownErr != nil {
		cooldownUntil = nil
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"appointment":       appointment,
			"queue_before":      queueBefore,
			"estimated_minutes": int(queueBefore) * merchant.AvgServiceMinutes,
			"cooldown_until":    cooldownUntil,
		},
	})
}

func CreateAppointment(c *gin.Context) {
	var input struct {
		MerchantID      uint   `json:"merchant_id" binding:"required"`
		UserID          uint   `json:"user_id" binding:"required"`
		TechnicianID    *uint  `json:"technician_id"`
		AppointmentTime string `json:"appointment_time" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	if input.TechnicianID != nil {
		var tech models.Technician
		if err := config.DB.Where("id = ? AND merchant_id = ?", *input.TechnicianID, input.MerchantID).First(&tech).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的技师"})
			return
		}
	}

	// 检查取消后冷却(1小时)
	var lastCanceled models.Appointment
	lastCanceledErr := config.DB.
		Where("user_id = ? AND merchant_id = ? AND status = 'canceled' AND canceled_at IS NOT NULL", input.UserID, input.MerchantID).
		Order("canceled_at DESC").
		Limit(1).
		First(&lastCanceled).Error
	if lastCanceledErr == nil && lastCanceled.CanceledAt != nil {
		cooldownUntil := lastCanceled.CanceledAt.Add(1 * time.Hour)
		if time.Now().Before(cooldownUntil) {
			remaining := int64(time.Until(cooldownUntil).Seconds())
			c.JSON(http.StatusBadRequest, gin.H{
				"error":            "取消预约后1小时内不可再次预约",
				"cooldown_until":   cooldownUntil,
				"cooldown_seconds": remaining,
			})
			return
		}
	}

	// 检查用户在该商户是否已有活跃的预约
	var existingAppointment models.Appointment
	err := config.DB.Where("user_id = ? AND merchant_id = ? AND status IN ('pending', 'confirmed')",
		input.UserID, input.MerchantID).First(&existingAppointment).Error

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

	log.Printf("创建新预约: 用户ID=%d, 商户ID=%d, 时间=%s", input.UserID, input.MerchantID, input.AppointmentTime)

	appointmentTime, err := time.ParseInLocation("2006-01-02 15:04:05", input.AppointmentTime, time.Local)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "预约时间格式错误"})
		return
	}

	appointment := models.Appointment{
		MerchantID:      input.MerchantID,
		UserID:          input.UserID,
		TechnicianID:    input.TechnicianID,
		AppointmentTime: &appointmentTime,
		Status:          "pending",
	}
	result := config.DB.Create(&appointment)
	if result.Error != nil {
		log.Printf("创建预约失败: %v", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建预约失败"})
		return
	}

	log.Printf("预约创建成功: ID=%d, 状态=%s", appointment.ID, appointment.Status)

	config.DB.Preload("User").Preload("Merchant").Preload("Technician").First(&appointment, appointment.ID)
	c.JSON(http.StatusOK, gin.H{"data": appointment})
}

func ConfirmAppointment(c *gin.Context) {
	id := c.Param("id")
	var appointment models.Appointment
	if err := config.DB.First(&appointment, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
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

	config.DB.Model(&appointment).Update("status", "confirmed")
	config.DB.Preload("User").Preload("Merchant").Preload("Technician").First(&appointment, id)
	c.JSON(http.StatusOK, gin.H{"data": appointment})
}

func FinishAppointment(c *gin.Context) {
	id := c.Param("id")
	var appointment models.Appointment
	if err := config.DB.First(&appointment, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}

	if autoCanceled, autoCancelErr := autoCancelAppointmentIfOverdue(&appointment, time.Now()); autoCancelErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "自动取消预约失败"})
		return
	} else if autoCanceled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "预约已超时取消，无法核销"})
		return
	}

	if appointment.Status != "confirmed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "只能完成已确认的预约"})
		return
	}

	if appointment.AppointmentTime == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "预约时间为空"})
		return
	}

	var merchant models.Merchant
	if err := config.DB.First(&merchant, appointment.MerchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}

	serviceMinutes := merchant.AvgServiceMinutes
	if serviceMinutes <= 0 {
		serviceMinutes = 30
	}

	now := time.Now()
	finishDeadline := appointment.AppointmentTime.Add(time.Duration(serviceMinutes+30) * time.Minute)
	if now.After(finishDeadline) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "已过服务时间，无法核销"})
		return
	}

	config.DB.Model(&appointment).Update("status", "finished")
	config.DB.Preload("User").Preload("Merchant").First(&appointment, id)
	c.JSON(http.StatusOK, gin.H{"data": appointment})
}

func CancelAppointment(c *gin.Context) {
	id := c.Param("id")
	var appointment models.Appointment
	if err := config.DB.First(&appointment, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
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
	config.DB.Preload("User").Preload("Merchant").First(&appointment, id)
	c.JSON(http.StatusOK, gin.H{"data": appointment})
}

func CancelOverdueAppointments(c *gin.Context) {
	merchantIDStr := c.Query("merchant_id")
	var merchantID uint64
	var err error
	if merchantIDStr != "" {
		merchantID, err = strconv.ParseUint(merchantIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "merchant_id 参数错误"})
			return
		}
	}

	now := time.Now()
	deadline := now.Add(-35 * time.Minute)

	q := config.DB.Model(&models.Appointment{}).
		Where("status IN ('pending','confirmed') AND appointment_time IS NOT NULL AND appointment_time <= ?", deadline)
	if merchantIDStr != "" {
		q = q.Where("merchant_id = ?", merchantID)
	}

	result := q.Updates(map[string]interface{}{
		"status":      "canceled",
		"canceled_at": now,
	})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "批量取消失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"canceled": result.RowsAffected}})
}

func GetQueueStatus(c *gin.Context) {
	merchantID := c.Param("id")

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
			"avg_service_minutes":  merchant.AvgServiceMinutes,
		},
	})
}

// GetAvailableTimeSlots 获取商户的可用预约时间段
func GetAvailableTimeSlots(c *gin.Context) {
	merchantID := c.Param("id")
	date := c.Query("date") // 格式: 2024-01-01
	loc, locErr := time.LoadLocation("Asia/Shanghai")
	if locErr != nil {
		loc = time.Local
	}

	if date == "" {
		date = time.Now().In(loc).Format("2006-01-02")
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

	// 懒更新：自动取消该商户已超时(>=35分钟)但未核销的预约，避免继续占用时间段
	now := time.Now().In(loc)
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

	// 生成可用时间段（营业时间 9:00-21:00，每个时间段为服务时长）
	serviceMinutes := merchant.AvgServiceMinutes
	if serviceMinutes == 0 {
		serviceMinutes = 30
	}

	// 判断是否为今天
	isToday := date == time.Now().In(loc).Format("2006-01-02")
	var minStartTime time.Time
	if isToday {
		// 今天仅展示从当前时间之后的时间段，且需要满足：now + (avg_service_minutes + 5分钟)
		// 若商户未配置 avg_service_minutes，则按 now + 1小时
		now := time.Now().In(loc)
		if merchant.AvgServiceMinutes == 0 {
			minStartTime = now.Add(1 * time.Hour)
		} else {
			minStartTime = now.Add(time.Duration(merchant.AvgServiceMinutes)*time.Minute + 5*time.Minute)
		}
	}

	// 解析日期
	targetDate, _ := time.ParseInLocation("2006-01-02", date, loc)

	// 营业时间: 9:00 - 21:00
	startHour := 9
	endHour := 21

	// 生成所有可能的时间段
	var allSlots []string
	for hour := startHour; hour < endHour; hour++ {
		for minute := 0; minute < 60; minute += serviceMinutes {
			slotTime := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(),
				hour, minute, 0, 0, loc)

			// 如果是今天，仅展示满足最小开始时间后的时间段
			if isToday && slotTime.Before(minStartTime) {
				continue
			}

			allSlots = append(allSlots, slotTime.Format("2006-01-02 15:04:05"))
		}
	}

	// 标记已被占用的时间段
	type TimeSlot struct {
		Time      string `json:"time"`
		Available bool   `json:"available"`
		UserName  string `json:"user_name,omitempty"`
	}

	var timeSlots []TimeSlot
	for _, slot := range allSlots {
		slotTime, _ := time.Parse("2006-01-02 15:04:05", slot)
		available := true
		userName := ""

		// 检查这个时间段是否与现有预约冲突
		for _, apt := range appointments {
			if apt.AppointmentTime == nil {
				continue
			}
			aptTime := *apt.AppointmentTime

			// 如果预约时间在当前时间段内，或者当前时间段在预约的服务时长内
			if slotTime.Equal(aptTime) ||
				(slotTime.After(aptTime) && slotTime.Before(aptTime.Add(time.Duration(serviceMinutes)*time.Minute))) {
				available = false
				if apt.User.Nickname != "" {
					userName = apt.User.Nickname
				}
				break
			}
		}

		// 仅返回可预约的时间段（排除已被预约/占用的时间段）
		if available {
			timeSlots = append(timeSlots, TimeSlot{
				Time:      slot,
				Available: available,
				UserName:  userName,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"date":                  date,
			"service_minutes":       serviceMinutes,
			"time_slots":            timeSlots,
			"existing_appointments": appointments,
		},
	})
}
