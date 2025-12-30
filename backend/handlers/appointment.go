package handlers

import (
	"kabao/config"
	"kabao/models"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func GetMerchantAppointments(c *gin.Context) {
	merchantID := c.Param("id")
	status := c.Query("status")

	var appointments []models.Appointment
	query := config.DB.Preload("User").Where("merchant_id = ?", merchantID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Order("appointment_time ASC").Find(&appointments)
	c.JSON(http.StatusOK, gin.H{"data": appointments})
}

func GetUserAppointments(c *gin.Context) {
	userID := c.Param("id")
	var appointments []models.Appointment
	config.DB.Preload("Merchant").Where("user_id = ?", userID).Order("appointment_time DESC").Find(&appointments)
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
	err := config.DB.Preload("Merchant").
		Where("user_id = ? AND merchant_id = ? AND status IN ('pending', 'confirmed')", card.UserID, card.MerchantID).
		Order("appointment_time ASC").
		First(&appointment).Error

	if err != nil {
		log.Printf("未找到预约: %v", err)
		c.JSON(http.StatusOK, gin.H{"data": nil})
		return
	}

	log.Printf("找到预约: ID=%d, 状态=%s, 时间=%v", appointment.ID, appointment.Status, appointment.AppointmentTime)

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
			"estimated_minutes": int(queueBefore) * merchant.AvgServiceMinutes,
		},
	})
}

func CreateAppointment(c *gin.Context) {
	var input struct {
		MerchantID      uint   `json:"merchant_id" binding:"required"`
		UserID          uint   `json:"user_id" binding:"required"`
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

	// 检查用户在该商户是否已有活跃的预约
	var existingAppointment models.Appointment
	err := config.DB.Where("user_id = ? AND merchant_id = ? AND status IN ('pending', 'confirmed')",
		input.UserID, input.MerchantID).First(&existingAppointment).Error

	if err == nil {
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

	config.DB.Preload("User").Preload("Merchant").First(&appointment, appointment.ID)
	c.JSON(http.StatusOK, gin.H{"data": appointment})
}

func ConfirmAppointment(c *gin.Context) {
	id := c.Param("id")
	var appointment models.Appointment
	if err := config.DB.First(&appointment, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}

	if appointment.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "只能确认待处理的预约"})
		return
	}

	config.DB.Model(&appointment).Update("status", "confirmed")
	config.DB.Preload("User").Preload("Merchant").First(&appointment, id)
	c.JSON(http.StatusOK, gin.H{"data": appointment})
}

func FinishAppointment(c *gin.Context) {
	id := c.Param("id")
	var appointment models.Appointment
	if err := config.DB.First(&appointment, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}

	if appointment.Status != "confirmed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "只能完成已确认的预约"})
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

	config.DB.Model(&appointment).Update("status", "canceled")
	config.DB.Preload("User").Preload("Merchant").First(&appointment, id)
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
			"avg_service_minutes":  merchant.AvgServiceMinutes,
		},
	})
}

// GetAvailableTimeSlots 获取商户的可用预约时间段
func GetAvailableTimeSlots(c *gin.Context) {
	merchantID := c.Param("id")
	date := c.Query("date") // 格式: 2024-01-01

	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	// 验证日期格式
	if _, err := time.Parse("2006-01-02", date); err != nil {
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

	// 获取当天的所有预约（pending和confirmed状态）
	var appointments []models.Appointment
	// 使用字符串前缀匹配来查询日期，避免DATE()函数的兼容性问题
	datePrefix := date + " %"
	config.DB.Preload("User").Where("merchant_id = ? AND appointment_time LIKE ? AND status IN ('pending', 'confirmed')",
		merchantID, datePrefix).Order("appointment_time ASC").Find(&appointments)

	// 生成可用时间段（营业时间 9:00-21:00，每个时间段为服务时长）
	serviceMinutes := merchant.AvgServiceMinutes
	if serviceMinutes == 0 {
		serviceMinutes = 30 // 默认30分钟
	}

	// 解析日期
	targetDate, _ := time.Parse("2006-01-02", date)

	// 营业时间: 9:00 - 21:00
	startHour := 9
	endHour := 21

	// 生成所有可能的时间段
	var allSlots []string
	for hour := startHour; hour < endHour; hour++ {
		for minute := 0; minute < 60; minute += serviceMinutes {
			slotTime := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(),
				hour, minute, 0, 0, time.Local)
			
			// 如果是今天，只显示未来的时间段
			if date == time.Now().Format("2006-01-02") && slotTime.Before(time.Now()) {
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
			aptTime, _ := time.Parse("2006-01-02 15:04:05", apt.AppointmentTime)
			
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

		timeSlots = append(timeSlots, TimeSlot{
			Time:      slot,
			Available: available,
			UserName:  userName,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"date":               date,
			"service_minutes":    serviceMinutes,
			"time_slots":         timeSlots,
			"existing_appointments": appointments,
		},
	})
}
