package handlers

import (
	"errors"
	"kabao/config"
	"kabao/models"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func ListMerchantProjects(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	var list []models.MerchantProject
	config.DB.Where("merchant_id = ?", merchantID).Order("sort_order asc, id desc").Find(&list)
	c.JSON(http.StatusOK, gin.H{"data": list})
}

func normalizeMerchantProjectServiceTimeSlots(slots models.MerchantProjectServiceTimeSlots) (models.MerchantProjectServiceTimeSlots, error) {
	if len(slots) == 0 {
		return models.MerchantProjectServiceTimeSlots{}, nil
	}

	normalized := make(models.MerchantProjectServiceTimeSlots, 0, len(slots))
	seen := make(map[string]struct{}, len(slots))
	for _, slot := range slots {
		startTime := strings.TrimSpace(slot.StartTime)
		if _, err := time.Parse("15:04", startTime); err != nil {
			return nil, errors.New("服务时间必须为 HH:mm 格式")
		}

		recurrenceType := strings.TrimSpace(slot.RecurrenceType)
		if recurrenceType == "" {
			recurrenceType = "weekly"
		}
		switch recurrenceType {
		case "weekly":
			if slot.Weekday < 1 || slot.Weekday > 7 {
				return nil, errors.New("服务时间星期范围应为 1-7")
			}
			slot.MonthDay = 0
		case "monthly":
			if slot.MonthDay < 1 || slot.MonthDay > 31 {
				return nil, errors.New("服务时间月日期范围应为 1-31")
			}
			slot.Weekday = 0
		default:
			return nil, errors.New("服务时间类型必须为 weekly 或 monthly")
		}

		key := recurrenceType + "|" + strconv.Itoa(slot.Weekday) + "|" + strconv.Itoa(slot.MonthDay) + "|" + startTime
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, models.MerchantProjectServiceTimeSlot{
			RecurrenceType: recurrenceType,
			Weekday:        slot.Weekday,
			MonthDay:       slot.MonthDay,
			StartTime:      startTime,
		})
	}
	sort.Slice(normalized, func(i, j int) bool {
		if normalized[i].RecurrenceType != normalized[j].RecurrenceType {
			return normalized[i].RecurrenceType < normalized[j].RecurrenceType
		}
		if normalized[i].MonthDay != normalized[j].MonthDay {
			return normalized[i].MonthDay < normalized[j].MonthDay
		}
		if normalized[i].Weekday != normalized[j].Weekday {
			return normalized[i].Weekday < normalized[j].Weekday
		}
		return normalized[i].StartTime < normalized[j].StartTime
	})
	return normalized, nil
}

func CreateMerchantProject(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	var input struct {
		Name                             string                                 `json:"name" binding:"required"`
		Duration                         int                                    `json:"duration" binding:"required,min=1"`
		BookableOnline                   *bool                                  `json:"bookable_online"`
		ServiceGapMinutes                *int                                   `json:"service_gap_minutes"`
		StartDelaySeconds                *int                                   `json:"start_delay_seconds"`
		RoomSelectTimeoutSeconds         *int                                   `json:"room_select_timeout_seconds"`
		StartPendingTimeoutSeconds       *int                                   `json:"start_pending_timeout_seconds"`
		ServiceCapacity                  *int                                   `json:"service_capacity"`
		ShowParticipants                 *bool                                  `json:"show_participants"`
		ServiceTimeSlots                 models.MerchantProjectServiceTimeSlots `json:"service_time_slots"`
		AutoAssignTechnicianDelayMinutes *int                                   `json:"auto_assign_technician_delay_minutes"`
		DelayToleranceMinutes            *int                                   `json:"delay_tolerance_minutes"`
		DelayCompensationMode            *string                                `json:"delay_compensation_mode"`
		DelayRedeemThresholdPercent      *int                                   `json:"delay_redeem_threshold_percent"`
		DelayFixedUnitValue              *int                                   `json:"delay_fixed_unit_value"`
		IsDefault                        *bool                                  `json:"is_default"`
		Price                            float64                                `json:"price"`
		Description                      string                                 `json:"description"`
		IsActive                         *bool                                  `json:"is_active"`
		SortOrder                        *int                                   `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "项目名称不能为空"})
		return
	}
	if input.Duration < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "服务时长必须大于等于1"})
		return
	}
	if input.Price < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "项目价格不能为负数"})
		return
	}
	startDelaySeconds := 60
	if input.StartDelaySeconds != nil {
		if *input.StartDelaySeconds < 0 || *input.StartDelaySeconds > 3600 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "服务开始延迟时间范围应为 0-3600 秒"})
			return
		}
		startDelaySeconds = *input.StartDelaySeconds
	}
	serviceGapMinutes := 3
	if input.ServiceGapMinutes != nil {
		if *input.ServiceGapMinutes < 0 || *input.ServiceGapMinutes > 60 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "服务间歇时间范围应为 0-60 分钟"})
			return
		}
		serviceGapMinutes = *input.ServiceGapMinutes
	}
	roomSelectTimeoutSeconds := 90
	if input.RoomSelectTimeoutSeconds != nil {
		if *input.RoomSelectTimeoutSeconds < 1 || *input.RoomSelectTimeoutSeconds > 3600 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "自动分配房间延迟时间范围应为 1-3600 秒"})
			return
		}
		roomSelectTimeoutSeconds = *input.RoomSelectTimeoutSeconds
	}
	startPendingTimeoutSeconds := config.DefaultRoleStartPendingTimeoutSeconds
	if input.StartPendingTimeoutSeconds != nil {
		if *input.StartPendingTimeoutSeconds < 1 || *input.StartPendingTimeoutSeconds > 3600 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "待开始服务倒计时时间范围应为 1-3600 秒"})
			return
		}
		startPendingTimeoutSeconds = *input.StartPendingTimeoutSeconds
	}
	serviceCapacity := 1
	if input.ServiceCapacity != nil {
		if *input.ServiceCapacity < 1 || *input.ServiceCapacity > 999 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "服务人数范围应为 1-999"})
			return
		}
		serviceCapacity = *input.ServiceCapacity
	}
	showParticipants := true
	if input.ShowParticipants != nil {
		showParticipants = *input.ShowParticipants
	}
	serviceTimeSlots, err := normalizeMerchantProjectServiceTimeSlots(input.ServiceTimeSlots)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	autoAssignTechnicianDelayMinutes := 5
	if input.AutoAssignTechnicianDelayMinutes != nil {
		if *input.AutoAssignTechnicianDelayMinutes < 0 || *input.AutoAssignTechnicianDelayMinutes > 180 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "自动分配客服延迟时间范围应为 0-180 分钟"})
			return
		}
		autoAssignTechnicianDelayMinutes = *input.AutoAssignTechnicianDelayMinutes
	}
	delayToleranceMinutes := 1
	if input.DelayToleranceMinutes != nil {
		if *input.DelayToleranceMinutes < 0 || *input.DelayToleranceMinutes > 180 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "拖堂容忍分钟数范围应为 0-180 分钟"})
			return
		}
		delayToleranceMinutes = *input.DelayToleranceMinutes
	}
	delayCompensationMode := "minutes_bucket"
	if input.DelayCompensationMode != nil {
		switch strings.TrimSpace(*input.DelayCompensationMode) {
		case "", "minutes_bucket":
			delayCompensationMode = "minutes_bucket"
		case "amount_bucket", "fixed_unit":
			delayCompensationMode = strings.TrimSpace(*input.DelayCompensationMode)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "拖堂补偿模式必须为 minutes_bucket、amount_bucket 或 fixed_unit"})
			return
		}
	}
	delayRedeemThresholdPercent := 100
	if input.DelayRedeemThresholdPercent != nil {
		if *input.DelayRedeemThresholdPercent < 1 || *input.DelayRedeemThresholdPercent > 1000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "拖堂兑现阈值百分比范围应为 1-1000"})
			return
		}
		delayRedeemThresholdPercent = *input.DelayRedeemThresholdPercent
	}
	delayFixedUnitValue := 0
	if input.DelayFixedUnitValue != nil {
		if *input.DelayFixedUnitValue < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "拖堂固定补偿值不能为负数"})
			return
		}
		delayFixedUnitValue = *input.DelayFixedUnitValue
	}
	bookableOnline := true
	if input.BookableOnline != nil {
		bookableOnline = *input.BookableOnline
	}

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}
	sortOrder := 0
	if input.SortOrder != nil {
		sortOrder = *input.SortOrder
	}
	var existingCount int64
	if err := config.DB.Model(&models.MerchantProject{}).Where("merchant_id = ?", merchantID).Count(&existingCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询项目失败"})
		return
	}
	isDefault := existingCount == 0
	if input.IsDefault != nil {
		isDefault = *input.IsDefault
	}

	p := models.MerchantProject{
		MerchantID:                       merchantID,
		Name:                             name,
		Duration:                         input.Duration,
		BookableOnline:                   bookableOnline,
		ServiceGapMinutes:                serviceGapMinutes,
		StartDelaySeconds:                startDelaySeconds,
		RoomSelectTimeoutSeconds:         roomSelectTimeoutSeconds,
		StartPendingTimeoutSeconds:       startPendingTimeoutSeconds,
		ServiceCapacity:                  serviceCapacity,
		ShowParticipants:                 showParticipants,
		ServiceTimeSlots:                 serviceTimeSlots,
		AutoAssignTechnicianDelayMinutes: autoAssignTechnicianDelayMinutes,
		DelayToleranceMinutes:            delayToleranceMinutes,
		DelayCompensationMode:            delayCompensationMode,
		DelayRedeemThresholdPercent:      delayRedeemThresholdPercent,
		DelayFixedUnitValue:              delayFixedUnitValue,
		IsDefault:                        isDefault,
		Price:                            input.Price,
		Description:                      strings.TrimSpace(input.Description),
		IsActive:                         isActive,
		SortOrder:                        sortOrder,
	}
	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		if p.IsDefault {
			if err := tx.Model(&models.MerchantProject{}).Where("merchant_id = ?", merchantID).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		if err := tx.Create(&p).Error; err != nil {
			return err
		}
		return config.EnsureMerchantDefaultProject(tx, merchantID)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": p})
}

func UpdateMerchantProject(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	id := c.Param("id")
	var p models.MerchantProject
	if err := config.DB.Where("id = ? AND merchant_id = ?", id, merchantID).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "项目不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	var input struct {
		Name                             *string                                 `json:"name"`
		Duration                         *int                                    `json:"duration"`
		BookableOnline                   *bool                                   `json:"bookable_online"`
		ServiceGapMinutes                *int                                    `json:"service_gap_minutes"`
		StartDelaySeconds                *int                                    `json:"start_delay_seconds"`
		RoomSelectTimeoutSeconds         *int                                    `json:"room_select_timeout_seconds"`
		StartPendingTimeoutSeconds       *int                                    `json:"start_pending_timeout_seconds"`
		ServiceCapacity                  *int                                    `json:"service_capacity"`
		ShowParticipants                 *bool                                   `json:"show_participants"`
		ServiceTimeSlots                 *models.MerchantProjectServiceTimeSlots `json:"service_time_slots"`
		AutoAssignTechnicianDelayMinutes *int                                    `json:"auto_assign_technician_delay_minutes"`
		DelayToleranceMinutes            *int                                    `json:"delay_tolerance_minutes"`
		DelayCompensationMode            *string                                 `json:"delay_compensation_mode"`
		DelayRedeemThresholdPercent      *int                                    `json:"delay_redeem_threshold_percent"`
		DelayFixedUnitValue              *int                                    `json:"delay_fixed_unit_value"`
		IsDefault                        *bool                                   `json:"is_default"`
		Price                            *float64                                `json:"price"`
		Description                      *string                                 `json:"description"`
		IsActive                         *bool                                   `json:"is_active"`
		SortOrder                        *int                                    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if input.Name != nil {
		n := strings.TrimSpace(*input.Name)
		if n == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "项目名称不能为空"})
			return
		}
		updates["name"] = n
	}
	if input.Duration != nil {
		if *input.Duration < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "服务时长必须大于等于1"})
			return
		}
		updates["duration"] = *input.Duration
	}
	if input.StartDelaySeconds != nil {
		if *input.StartDelaySeconds < 0 || *input.StartDelaySeconds > 3600 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "服务开始延迟时间范围应为 0-3600 秒"})
			return
		}
		updates["start_delay_seconds"] = *input.StartDelaySeconds
	}
	if input.ServiceGapMinutes != nil {
		if *input.ServiceGapMinutes < 0 || *input.ServiceGapMinutes > 60 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "服务间歇时间范围应为 0-60 分钟"})
			return
		}
		updates["service_gap_minutes"] = *input.ServiceGapMinutes
	}
	if input.RoomSelectTimeoutSeconds != nil {
		if *input.RoomSelectTimeoutSeconds < 1 || *input.RoomSelectTimeoutSeconds > 3600 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "自动分配房间延迟时间范围应为 1-3600 秒"})
			return
		}
		updates["room_select_timeout_seconds"] = *input.RoomSelectTimeoutSeconds
	}
	if input.StartPendingTimeoutSeconds != nil {
		if *input.StartPendingTimeoutSeconds < 1 || *input.StartPendingTimeoutSeconds > 3600 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "待开始服务倒计时时间范围应为 1-3600 秒"})
			return
		}
		updates["start_pending_timeout_seconds"] = *input.StartPendingTimeoutSeconds
	}
	if input.ServiceCapacity != nil {
		if *input.ServiceCapacity < 1 || *input.ServiceCapacity > 999 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "服务人数范围应为 1-999"})
			return
		}
		updates["service_capacity"] = *input.ServiceCapacity
	}
	if input.ShowParticipants != nil {
		updates["show_participants"] = *input.ShowParticipants
	}
	if input.ServiceTimeSlots != nil {
		serviceTimeSlots, err := normalizeMerchantProjectServiceTimeSlots(*input.ServiceTimeSlots)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		updates["service_time_slots"] = serviceTimeSlots
	}
	if input.AutoAssignTechnicianDelayMinutes != nil {
		if *input.AutoAssignTechnicianDelayMinutes < 0 || *input.AutoAssignTechnicianDelayMinutes > 180 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "自动分配客服延迟时间范围应为 0-180 分钟"})
			return
		}
		updates["auto_assign_technician_delay_minutes"] = *input.AutoAssignTechnicianDelayMinutes
	}
	if input.DelayToleranceMinutes != nil {
		if *input.DelayToleranceMinutes < 0 || *input.DelayToleranceMinutes > 180 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "拖堂容忍分钟数范围应为 0-180 分钟"})
			return
		}
		updates["delay_tolerance_minutes"] = *input.DelayToleranceMinutes
	}
	if input.DelayCompensationMode != nil {
		mode := strings.TrimSpace(*input.DelayCompensationMode)
		switch mode {
		case "", "minutes_bucket":
			updates["delay_compensation_mode"] = "minutes_bucket"
		case "amount_bucket", "fixed_unit":
			updates["delay_compensation_mode"] = mode
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "拖堂补偿模式必须为 minutes_bucket、amount_bucket 或 fixed_unit"})
			return
		}
	}
	if input.DelayRedeemThresholdPercent != nil {
		if *input.DelayRedeemThresholdPercent < 1 || *input.DelayRedeemThresholdPercent > 1000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "拖堂兑现阈值百分比范围应为 1-1000"})
			return
		}
		updates["delay_redeem_threshold_percent"] = *input.DelayRedeemThresholdPercent
	}
	if input.DelayFixedUnitValue != nil {
		if *input.DelayFixedUnitValue < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "拖堂固定补偿值不能为负数"})
			return
		}
		updates["delay_fixed_unit_value"] = *input.DelayFixedUnitValue
	}
	if input.BookableOnline != nil {
		updates["bookable_online"] = *input.BookableOnline
	}
	if input.Price != nil {
		if *input.Price < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "项目价格不能为负数"})
			return
		}
		updates["price"] = *input.Price
	}
	if input.Description != nil {
		updates["description"] = strings.TrimSpace(*input.Description)
	}
	if input.IsActive != nil {
		updates["is_active"] = *input.IsActive
	}
	if input.SortOrder != nil {
		updates["sort_order"] = *input.SortOrder
	}
	if input.IsDefault != nil {
		updates["is_default"] = *input.IsDefault
	}

	if len(updates) == 0 {
		c.JSON(http.StatusOK, gin.H{"data": p})
		return
	}

	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		if nextDefault, ok := updates["is_default"].(bool); ok && nextDefault {
			if err := tx.Model(&models.MerchantProject{}).Where("merchant_id = ?", merchantID).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&p).Updates(updates).Error; err != nil {
			return err
		}
		return config.EnsureMerchantDefaultProject(tx, merchantID)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}
	config.DB.Where("id = ? AND merchant_id = ?", p.ID, merchantID).First(&p)
	c.JSON(http.StatusOK, gin.H{"data": p})
}

func DeleteMerchantProject(c *gin.Context) {
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}

	id := c.Param("id")
	var project models.MerchantProject
	if err := config.DB.Where("id = ? AND merchant_id = ?", id, merchantID).First(&project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "项目不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&models.MerchantProject{}).Where("merchant_id = ?", merchantID).Count(&count).Error; err != nil {
			return err
		}
		if count <= 1 {
			return errors.New("merchant_project_last_one")
		}
		if err := tx.Where("id = ? AND merchant_id = ?", id, merchantID).Delete(&models.MerchantProject{}).Error; err != nil {
			return err
		}
		return config.EnsureMerchantDefaultProject(tx, merchantID)
	}); err != nil {
		if err.Error() == "merchant_project_last_one" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "项目列表至少保留一个项目"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
