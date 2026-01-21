package handlers

import (
	"errors"
	"kabao/config"
	"kabao/models"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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
	err := config.DB.Preload("Merchant").Preload("Project").Preload("Technician").
		Where("user_id = ? AND merchant_id = ? AND status IN ('pending', 'confirmed', 'failed')", card.UserID, card.MerchantID).
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

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"appointment":       appointment,
			"queue_before":      queueBefore,
			"estimated_minutes": int(queueBefore) * 30,
		},
	})
}

func CreateAppointment(c *gin.Context) {
	var input struct {
		MerchantID      uint   `json:"merchant_id" binding:"required"`
		UserID          uint   `json:"user_id" binding:"required"`
		ProjectID       *uint  `json:"project_id"`
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
	if input.ProjectID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择项目"})
		return
	}
	serviceMinutes := project.Duration
	serviceEnd := appointmentTime.Add(time.Duration(serviceMinutes) * time.Minute)
	if !isWithinBusinessIntervals(serviceEnd.Add(-1*time.Second), intervals) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "服务必须在营业时间内完成"})
		return
	}

	appointment := models.Appointment{
		MerchantID:      input.MerchantID,
		UserID:          input.UserID,
		ProjectID:       input.ProjectID,
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

	config.DB.Preload("User").Preload("Merchant").Preload("Project").Preload("Technician").First(&appointment, appointment.ID)
	c.JSON(http.StatusOK, gin.H{"data": appointment})
}

func ConfirmAppointment(c *gin.Context) {
	id := c.Param("id")
	var appointment models.Appointment
	if err := config.DB.First(&appointment, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}

	// 权限检查
	technicianIDAny, ok := c.Get("technician_id")
	if ok {
		// 技师登录：只能确认分配给自己的预约
		if technicianID, ok := technicianIDAny.(uint); ok && technicianID > 0 {
			if appointment.TechnicianID == nil || *appointment.TechnicianID != technicianID {
				c.JSON(http.StatusForbidden, gin.H{"error": "无权限：只能确认分配给自己的预约"})
				return
			}
		}
	} else {
		// 商户登录：需要管理权限
		merchantIDAny, ok := c.Get("merchant_id")
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
			return
		}
		merchantID, ok := merchantIDAny.(uint)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
			return
		}
		
		// 检查预约是否属于当前商户
		if appointment.MerchantID != merchantID {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权限：不属于您的商户"})
			return
		}
		
		// 检查管理权限
		if serviceRoleIDAny, ok := c.Get("service_role_id"); ok {
			if serviceRoleID, ok := serviceRoleIDAny.(uint); ok {
				var managePerm models.Permission
				if err := config.DB.Where("`key` = ?", "merchant.appointment.manage").First(&managePerm).Error; err == nil {
					var override models.MerchantRolePermissionOverride
					err := config.DB.Where("merchant_id = ? AND service_role_id = ? AND permission_id = ?", merchantID, serviceRoleID, managePerm.ID).First(&override).Error
					if err == nil {
						if !override.Allowed {
							c.JSON(http.StatusForbidden, gin.H{"error": "无权限：需要预约管理权限"})
							return
						}
					} else {
						var rolePerm models.RolePermission
						err = config.DB.Where("service_role_id = ? AND permission_id = ? AND allowed = ?", serviceRoleID, managePerm.ID, true).First(&rolePerm).Error
						if err != nil {
							c.JSON(http.StatusForbidden, gin.H{"error": "无权限：需要预约管理权限"})
							return
						}
					}
				}
			}
		}
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

	serviceMinutes := getAppointmentServiceMinutes(merchant.ID, appointment)

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

	// 权限检查
	technicianIDAny, ok := c.Get("technician_id")
	if ok {
		// 技师登录：只能取消分配给自己的预约
		if technicianID, ok := technicianIDAny.(uint); ok && technicianID > 0 {
			if appointment.TechnicianID == nil || *appointment.TechnicianID != technicianID {
				c.JSON(http.StatusForbidden, gin.H{"error": "无权限：只能取消分配给自己的预约"})
				return
			}
		}
	} else {
		// 用户登录：只能取消自己的预约
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
	config.DB.Preload("User").Preload("Merchant").Preload("Technician").First(&appointment, id)
	c.JSON(http.StatusOK, gin.H{"data": appointment})
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
		},
	})
}

// GetAvailableTimeSlots 获取商户的可用预约时间段
func GetAvailableTimeSlots(c *gin.Context) {
	merchantID := c.Param("id")
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
	log.Printf("GetAvailableTimeSlots: merchant_id=%s date=%s project_id=%d", merchantID, date, projectID)
	loc, locErr := time.LoadLocation("Asia/Shanghai")
	if locErr != nil {
		loc = time.Local
	}

	now := time.Now().In(loc)
	tomorrowDate := now.Add(24 * time.Hour).Format("2006-01-02")
	if date == "" {
		date = tomorrowDate
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

	if date != tomorrowDate {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持预约明天"})
		return
	}
	if projectID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择项目"})
		return
	}
	var project models.MerchantProject
	if err := config.DB.Where("id = ? AND merchant_id = ? AND is_active = ?", projectID, merchant.ID, true).First(&project).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目"})
		return
	}
	serviceMinutes := project.Duration
	log.Printf("GetAvailableTimeSlots: project_id=%d duration=%d service_minutes=%d", projectID, project.Duration, serviceMinutes)
	if serviceMinutes <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "项目时长无效"})
		return
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
	if merchant.SupportCustomerService {
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
	type TimeSlot struct {
		Time      string `json:"time"`
		Available bool   `json:"available"`
		UserName  string `json:"user_name,omitempty"`
		TechnicianIDs []uint `json:"technician_ids,omitempty"`
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
			availableTechIDs = append(availableTechIDs, allTechIDs...)
			// 逐个预约把冲突的技师剔除
			for _, apt := range appointments {
				if apt.AppointmentTime == nil || apt.TechnicianID == nil {
					continue
				}
				aptTime := *apt.AppointmentTime
				aptMinutes := getAppointmentServiceMinutes(merchant.ID, apt)
				aptEnd := aptTime.Add(time.Duration(aptMinutes) * time.Minute)
				slotEnd := slotTime.Add(time.Duration(serviceMinutes) * time.Minute)
				if slotTime.Before(aptEnd) && aptTime.Before(slotEnd) {
					// remove tech id
					removeID := *apt.TechnicianID
					filtered := make([]uint, 0, len(availableTechIDs))
					for _, id := range availableTechIDs {
						if id != removeID {
							filtered = append(filtered, id)
						}
					}
					availableTechIDs = filtered
				}
			}
		}

		// 仅返回可预约的时间段（排除已被预约/占用的时间段）
		if available {
			timeSlots = append(timeSlots, TimeSlot{Time: slot, Available: true, UserName: userName, TechnicianIDs: availableTechIDs})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"date":                  date,
			"service_minutes":       serviceMinutes,
			"time_slots":            timeSlots,
			"technicians":           technicians,
		},
	})
}
