package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"kabao/config"
	"kabao/middleware"
	"kabao/models"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type apiErr struct {
	status int
	msg    string
}

func (e apiErr) Error() string { return e.msg }

const verifyHandCardUnreturnedGuardWindow = 8*time.Minute + 5*time.Second

func parseDatePtr(v string) (*time.Time, error) {
	if v == "" {
		return nil, nil
	}
	t, err := time.ParseInLocation("2006-01-02", v, time.Local)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func dateOnlyPtr(t time.Time) *time.Time {
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	return &d
}

func nextMerchantCardNo(tx *gorm.DB, merchantID uint) (string, error) {
	var last string
	err := tx.Raw(
		"SELECT card_no FROM cards WHERE merchant_id = ? AND card_no REGEXP '^[0-9]{5}$' ORDER BY card_no DESC LIMIT 1 FOR UPDATE",
		merchantID,
	).Scan(&last).Error
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(last) == "" {
		return fmt.Sprintf("%05d", 1), nil
	}
	seq, err := strconv.Atoi(last)
	if err != nil {
		// 如果历史数据不符合预期，退化到1
		return fmt.Sprintf("%05d", 1), nil
	}
	seq++
	if seq < 1 {
		seq = 1
	}
	return fmt.Sprintf("%05d", seq), nil
}

func verifyHandCardUnreturnedAgeGuard(tx *gorm.DB, merchantID uint, supportHandCard bool, now time.Time) error {
	if !supportHandCard {
		return nil
	}

	cutoff := now.Add(-verifyHandCardUnreturnedGuardWindow)
	var usage models.Usage
	err := tx.
		Select("hand_card_no").
		Where("merchant_id = ?", merchantID).
		Where("hand_card_returned_at IS NULL").
		Where("hand_card_no IS NOT NULL AND hand_card_no <> ''").
		Where("(hand_card_assigned_at IS NOT NULL AND hand_card_assigned_at <= ?) OR (hand_card_assigned_at IS NULL AND used_at IS NOT NULL AND used_at <= ?)", cutoff, cutoff).
		Order("COALESCE(hand_card_assigned_at, used_at) asc, id asc").
		First(&usage).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	no := ""
	if usage.HandCardNo != nil {
		no = strings.TrimSpace(*usage.HandCardNo)
	}
	if no == "" {
		return nil
	}

	return apiErr{
		status: http.StatusBadRequest,
		msg:    fmt.Sprintf("你尚有未归还的手牌%s，请先归还手牌再核销", no),
	}
}

func merchantProjectServiceTimeAllowed(project models.MerchantProject, now time.Time) bool {
	if len(project.ServiceTimeSlots) == 0 {
		return true
	}

	loc := config.ProjectServiceTimeLocation()
	localNow := now.In(loc)
	duration := time.Duration(project.Duration) * time.Minute
	if duration <= 0 {
		return false
	}

	for _, slot := range project.ServiceTimeSlots {
		startClock, err := time.ParseInLocation("15:04", strings.TrimSpace(slot.StartTime), loc)
		if err != nil {
			continue
		}
		for offset := -1; offset <= 1; offset++ {
			candidateDate := localNow.AddDate(0, 0, offset)
			if !config.MerchantProjectServiceTimeSlotMatchesDate(slot, candidateDate) {
				continue
			}
			startAt := time.Date(candidateDate.Year(), candidateDate.Month(), candidateDate.Day(), startClock.Hour(), startClock.Minute(), 0, 0, loc)
			windowStart := startAt.Add(-1 * time.Hour)
			windowEnd := startAt.Add(duration).Add(-3 * time.Minute)
			if !localNow.Before(windowStart) && !localNow.After(windowEnd) {
				return true
			}
		}
	}

	return false
}

func GetCards(c *gin.Context) {
	userID, ok := mustUserID(c)
	if !ok {
		return
	}
	var cards []models.Card
	config.DB.Preload("User").Preload("Merchant").Where("user_id = ?", userID).Order("id desc").Find(&cards)
	c.JSON(http.StatusOK, gin.H{"data": cards})
}

func GetCard(c *gin.Context) {
	userID, ok := mustUserID(c)
	if !ok {
		return
	}
	id := c.Param("id")
	var card models.Card
	if err := config.DB.Preload("User").Preload("Merchant").First(&card, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "卡片不存在"})
		return
	}
	if card.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问此卡"})
		return
	}

	card.StartPendingTimeoutSeconds = int64(config.MerchantQueueWaitingStartSeconds(&card.Merchant))
	card.StartScanTimeoutSeconds = int64(config.StartScanTimeout().Seconds())

	// 查询卡片关联项目
	var projects []models.MerchantProject
	config.DB.Raw(`
		SELECT p.* 
		FROM merchant_projects p
		INNER JOIN card_projects cp ON cp.project_id = p.id
		WHERE cp.card_id = ? AND p.merchant_id = ?
		ORDER BY cp.id ASC
	`, card.ID, card.MerchantID).Scan(&projects)
	if project, err := config.GetDefaultMerchantProject(config.DB, card.MerchantID); err == nil && project != nil {
		card.StartPendingTimeoutSeconds = int64(config.NormalizeRoleStartPendingTimeoutSeconds(project.StartPendingTimeoutSeconds))
		if len(projects) == 0 {
			projects = []models.MerchantProject{*project}
		}
	}
	card.Projects = projects

	c.JSON(http.StatusOK, gin.H{"data": card})
}

func GetCardProjects(c *gin.Context) {
	cardID := c.Param("id")
	var card models.Card
	if err := config.DB.First(&card, cardID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "卡片不存在"})
		return
	}

	if userIDAny, ok := c.Get("user_id"); ok {
		if userID, ok2 := userIDAny.(uint); ok2 && userID > 0 {
			if card.UserID != userID {
				c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此卡"})
				return
			}
		}
	}

	// 规则A：优先从 card_projects 读取卡片已绑定项目
	var boundIDs []uint
	config.DB.Model(&models.CardProject{}).
		Where("card_id = ?", card.ID).
		Order("id asc").
		Pluck("project_id", &boundIDs)
	if len(boundIDs) > 0 {
		var projects []models.MerchantProject
		config.DB.
			Where("merchant_id = ? AND id IN ?", card.MerchantID, boundIDs).
			Order("sort_order asc, id asc").
			Find(&projects)
		c.JSON(http.StatusOK, gin.H{"data": projects})
		return
	}

	// 兼容：老卡没有 card_projects 时，回退到 直购订单 -> 模板 -> 模板项目
	var purchase models.DirectPurchase
	err := config.DB.
		Where("card_id = ?", card.ID).
		Order("id desc").
		First(&purchase).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if project, err2 := config.GetDefaultMerchantProject(config.DB, card.MerchantID); err2 == nil && project != nil {
				c.JSON(http.StatusOK, gin.H{"data": []models.MerchantProject{*project}})
				return
			}
			c.JSON(http.StatusOK, gin.H{"data": []models.MerchantProject{}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	var projectIDs []uint
	config.DB.Model(&models.CardTemplateProject{}).
		Where("card_template_id = ?", purchase.CardTemplateID).
		Order("id asc").
		Pluck("project_id", &projectIDs)
	if len(projectIDs) == 0 {
		if project, err := config.GetDefaultMerchantProject(config.DB, card.MerchantID); err == nil && project != nil {
			c.JSON(http.StatusOK, gin.H{"data": []models.MerchantProject{*project}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": []models.MerchantProject{}})
		return
	}

	var projects []models.MerchantProject
	config.DB.
		Where("merchant_id = ? AND id IN ?", card.MerchantID, projectIDs).
		Order("sort_order asc, id asc").
		Find(&projects)
	if len(projects) == 0 {
		if project, err := config.GetDefaultMerchantProject(config.DB, card.MerchantID); err == nil && project != nil {
			projects = []models.MerchantProject{*project}
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": projects})
}

func GetNextMerchantCardNo(c *gin.Context) {
	merchantIDAny, ok := c.Get("merchant_id")
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅商户可操作"})
		return
	}
	merchantID, ok := merchantIDAny.(uint)
	if !ok || merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var next string
	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		v, err := nextMerchantCardNo(tx, merchantID)
		if err != nil {
			return err
		}
		next = v
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"card_no": next}})
}

func GetUserCards(c *gin.Context) {
	authUserID, ok := mustUserID(c)
	if !ok {
		return
	}
	userID := c.Param("id")
	if v, err := strconv.ParseUint(strings.TrimSpace(userID), 10, 32); err != nil || uint(v) != authUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问其他用户数据"})
		return
	}
	status := c.Query("status")

	var cards []models.Card
	query := config.DB.Preload("Merchant").Where("user_id = ?", userID)

	now := time.Now()
	if status == "active" {
		// 获取所有未过期且有剩余次数的卡片
		query = query.Where("end_date >= ? AND remain_times > 0", dateOnlyPtr(now))
		query.Find(&cards)

		// 添加逻辑：即使已过期或次数为0，但如果有未完成的核销记录，也应该显示在进行中
		var expiredCardsWithInProgressUsage []models.Card
		expiredQuery := config.DB.Preload("Merchant").Where("user_id = ? AND (end_date < ? OR remain_times = 0)", userID, dateOnlyPtr(now))
		expiredQuery.Find(&expiredCardsWithInProgressUsage)

		// 检查每张过期卡片是否有未完成的核销记录
		for _, card := range expiredCardsWithInProgressUsage {
			var lastUsage models.Usage
			err := config.DB.Where("card_id = ?", card.ID).Order("used_at DESC").First(&lastUsage).Error
			if err == nil && lastUsage.Status == "in_progress" {
				// 有未完成的核销记录，添加到进行中列表
				cards = append(cards, card)
			}
		}
	} else if status == "expired" {
		// 获取所有过期或次数为0的卡片
		query = query.Where("end_date < ? OR remain_times = 0", dateOnlyPtr(now))
		query.Find(&cards)

		// 过滤掉有未完成核销记录的卡片（它们应该显示在进行中）
		var filteredCards []models.Card
		for _, card := range cards {
			var lastUsage models.Usage
			err := config.DB.Where("card_id = ?", card.ID).Order("used_at DESC").First(&lastUsage).Error
			if err != nil || lastUsage.Status != "in_progress" {
				// 没有核销记录或最后一次已完成，显示在失效中
				filteredCards = append(filteredCards, card)
			}
		}
		cards = filteredCards
	} else {
		query.Find(&cards)
	}

	c.JSON(http.StatusOK, gin.H{"data": cards})
}

func GetMerchantCards(c *gin.Context) {
	if _, ok := ensureMerchantScope(c, "id"); !ok {
		return
	}
	merchantIDAny, ok := c.Get("merchant_id")
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅商户可查看"})
		return
	}
	merchantID, ok := merchantIDAny.(uint)
	if !ok || merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	phone := c.Query("phone")
	nickname := c.Query("nickname")
	cardNo := c.Query("card_no")
	cardType := c.Query("card_type")
	userIDStr := strings.TrimSpace(c.Query("user_id"))
	userCode := strings.TrimSpace(c.Query("user_code"))

	var cards []models.Card
	query := config.DB.
		Model(&models.Card{}).
		Joins("LEFT JOIN users ON users.id = cards.user_id").
		Where("cards.merchant_id = ?", merchantID)

	if userIDStr != "" {
		uid, err := strconv.ParseUint(userIDStr, 10, 32)
		if err != nil || uid == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id 参数错误"})
			return
		}
		query = query.Where("cards.user_id = ?", uint(uid))
	} else if userCode != "" {
		uid, err := parseUserCodeToUserID(userCode)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		query = query.Where("cards.user_id = ?", uid)
	}

	if phone != "" {
		query = query.Where("users.phone LIKE ?", "%"+phone+"%")
	}
	if nickname != "" {
		query = query.Where("users.nickname LIKE ?", "%"+nickname+"%")
	}
	if cardNo != "" {
		query = query.Where("cards.card_no LIKE ?", "%"+cardNo+"%")
	}
	if cardType != "" {
		query = query.Where("cards.card_type LIKE ?", "%"+cardType+"%")
	}

	query.
		Select("cards.*").
		Preload("User").
		Order("cards.id desc").
		Find(&cards)
	c.JSON(http.StatusOK, gin.H{"data": cards})
}

func parseUserCodeToUserID(code string) (uint, error) {
	v := strings.TrimSpace(code)
	if v == "" {
		return 0, errors.New("用户码无效")
	}
	if !strings.HasPrefix(v, "kabao-user:") {
		return 0, errors.New("用户码格式不正确")
	}
	parts := strings.Split(v, ":")
	if len(parts) != 4 {
		return 0, errors.New("用户码格式不正确")
	}
	uidStr := strings.TrimSpace(parts[1])
	expStr := strings.TrimSpace(parts[2])
	sig := strings.TrimSpace(parts[3])
	uid64, err := strconv.ParseUint(uidStr, 10, 32)
	if err != nil || uid64 == 0 {
		return 0, errors.New("用户码无效")
	}
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || exp <= 0 {
		return 0, errors.New("用户码无效")
	}
	if time.Now().Unix() > exp {
		return 0, errors.New("用户码已过期，请用户刷新后重试")
	}

	msg := uidStr + ":" + expStr
	secret := config.UserCodeSecret()
	if secret == "" {
		return 0, errors.New("用户码服务未配置")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(msg))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(strings.ToLower(expected)), []byte(strings.ToLower(sig))) {
		return 0, errors.New("用户码无效")
	}
	return uint(uid64), nil
}

func GetMerchantCard(c *gin.Context) {
	merchantIDAny, ok := c.Get("merchant_id")
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅商户可查看"})
		return
	}
	merchantID, ok := merchantIDAny.(uint)
	if !ok || merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	cardID := c.Param("id")
	var card models.Card
	if err := config.DB.Preload("User").Preload("Merchant").First(&card, cardID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "卡片不存在"})
		return
	}

	if card.MerchantID != merchantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权查看此卡"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": card})
}

func CreateCard(c *gin.Context) {
	merchantIDAny, ok := c.Get("merchant_id")
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅商户可发卡"})
		return
	}
	merchantID, ok := merchantIDAny.(uint)
	if !ok || merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var input struct {
		UserID         uint   `json:"user_id" binding:"required"`
		CardNo         string `json:"card_no"`
		CardType       string `json:"card_type" binding:"required"`
		TotalTimes     int    `json:"total_times"`
		RechargeAmount int    `json:"recharge_amount"`
		StartDate      string `json:"start_date"`
		EndDate        string `json:"end_date" binding:"required"`
		ProjectIDs     []uint `json:"project_ids"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.TotalTimes <= 0 && input.RechargeAmount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "总次数与充值金额不能同时为空"})
		return
	}

	var user models.User
	if err := config.DB.First(&user, input.UserID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户不存在"})
		return
	}

	startDate, err := parseDatePtr(input.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "开始日期格式错误"})
		return
	}
	endDate, err := parseDatePtr(input.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "结束日期格式错误"})
		return
	}
	if endDate == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "结束日期不能为空"})
		return
	}

	now := time.Now()
	var card models.Card
	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		// 校验项目ID（可选）
		var validProjectIDs []uint
		if len(input.ProjectIDs) > 0 {
			uniq := make(map[uint]struct{}, len(input.ProjectIDs))
			for _, pid := range input.ProjectIDs {
				if pid == 0 {
					continue
				}
				uniq[pid] = struct{}{}
			}
			for pid := range uniq {
				validProjectIDs = append(validProjectIDs, pid)
			}
			var cnt int64
			if err := tx.Model(&models.MerchantProject{}).
				Where("merchant_id = ? AND id IN ?", merchantID, validProjectIDs).
				Count(&cnt).Error; err != nil {
				return err
			}
			if cnt != int64(len(validProjectIDs)) {
				return apiErr{status: http.StatusBadRequest, msg: "包含无效的项目"}
			}
		}

		cardNo := strings.TrimSpace(input.CardNo)
		if cardNo == "" {
			v, err := nextMerchantCardNo(tx, merchantID)
			if err != nil {
				return err
			}
			cardNo = v
		}

		remain := 0
		if input.TotalTimes > 0 {
			remain = input.TotalTimes
		}

		card = models.Card{
			UserID:         input.UserID,
			MerchantID:     merchantID,
			CardNo:         cardNo,
			CardType:       input.CardType,
			TotalTimes:     input.TotalTimes,
			RemainTimes:    remain,
			UsedTimes:      0,
			RechargeAmount: input.RechargeAmount,
			RechargeAt:     dateOnlyPtr(now),
			StartDate:      startDate,
			EndDate:        endDate,
		}
		if card.StartDate == nil {
			card.StartDate = dateOnlyPtr(now)
		}

		if err := tx.Create(&card).Error; err != nil {
			return err
		}

		// 写入 card_projects
		if len(validProjectIDs) > 0 {
			for _, pid := range validProjectIDs {
				cp := models.CardProject{CardID: card.ID, ProjectID: pid}
				if err := tx.Create(&cp).Error; err != nil {
					return err
				}
			}
		}

		return nil
	}); err != nil {
		var ae apiErr
		if errors.As(err, &ae) {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	config.DB.Preload("User").Preload("Merchant").First(&card, card.ID)
	c.JSON(http.StatusOK, gin.H{"data": card})
}

func UpdateCard(c *gin.Context) {
	id := c.Param("id")
	var card models.Card
	if err := config.DB.First(&card, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "卡片不存在"})
		return
	}

	var input struct {
		TotalTimes     *int   `json:"total_times"`
		RemainTimes    *int   `json:"remain_times"`
		RechargeAmount *int   `json:"recharge_amount"`
		EndDate        string `json:"end_date"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := make(map[string]interface{})
	if input.TotalTimes != nil {
		updates["total_times"] = *input.TotalTimes
	}
	if input.RemainTimes != nil {
		updates["remain_times"] = *input.RemainTimes
	}
	if input.RechargeAmount != nil {
		updates["recharge_amount"] = *input.RechargeAmount
		updates["recharge_at"] = dateOnlyPtr(time.Now())
	}
	if input.EndDate != "" {
		endDate, err := parseDatePtr(input.EndDate)
		if err != nil || endDate == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "结束日期格式错误"})
			return
		}
		updates["end_date"] = endDate
	}

	config.DB.Model(&card).Updates(updates)
	config.DB.Preload("User").Preload("Merchant").First(&card, id)
	c.JSON(http.StatusOK, gin.H{"data": card})
}

func GenerateVerifyCode(c *gin.Context) {
	cardID := c.Param("id")
	var card models.Card
	if err := config.DB.First(&card, cardID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "卡片不存在"})
		return
	}
	if card.Locked {
		msg := "卡片已锁定"
		if strings.TrimSpace(card.LockedReason) != "" {
			msg = card.LockedReason
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	var input struct {
		ProjectID     *uint `json:"project_id"`
		AppointmentID *uint `json:"appointment_id"`
	}
	_ = c.ShouldBindJSON(&input)

	// 用户端仅允许生成自己的卡片核销码
	if userIDAny, ok := c.Get("user_id"); ok {
		if userID, ok2 := userIDAny.(uint); ok2 && userID > 0 {
			if card.UserID != userID {
				c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此卡"})
				return
			}
		}
	}

	now := time.Now()
	var appointment *models.Appointment
	if input.AppointmentID != nil && *input.AppointmentID > 0 {
		appt, err := loadAppointmentByID(config.DB, *input.AppointmentID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询预约失败"})
			return
		}
		if appt == nil || appt.CardID != card.ID || appt.UserID != card.UserID || appt.MerchantID != card.MerchantID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "预约不存在或不属于当前卡片"})
			return
		}

		status := normalizeAppointmentStatus(appt.Status)
		if status != "confirmed" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "当前预约不可生成签到码"})
			return
		}

		var merchant models.Merchant
		if err := config.DB.First(&merchant, card.MerchantID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询商户失败"})
			return
		}
		if !canArriveForAppointment(*appt, &merchant, now.In(appointmentLocation())) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "当前不在可到店签到时间窗口内"})
			return
		}

		if appt.ProjectID != nil {
			if input.ProjectID == nil || *input.ProjectID == 0 {
				input.ProjectID = appt.ProjectID
			} else if *input.ProjectID != *appt.ProjectID {
				c.JSON(http.StatusBadRequest, gin.H{"error": "预约项目与签到项目不一致"})
				return
			}
		}
		appointment = appt
	}

	// 检查卡片是否有效
	if card.EndDate != nil && now.After(*card.EndDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "卡片已过期"})
		return
	}

	if appointment == nil && card.RemainTimes <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "剩余次数不足"})
		return
	}

	// 规则A：优先使用 card_projects 作为可选项目来源
	var ids []uint
	config.DB.Model(&models.CardProject{}).
		Where("card_id = ?", card.ID).
		Order("id asc").
		Pluck("project_id", &ids)
	if len(ids) == 0 {
		// 兼容：老卡没有 card_projects 时，回退到 直购订单 -> 模板 -> 模板项目
		var purchase models.DirectPurchase
		err := config.DB.
			Where("card_id = ?", card.ID).
			Order("id desc").
			First(&purchase).Error
		if err == nil {
			config.DB.Model(&models.CardTemplateProject{}).
				Where("card_template_id = ?", purchase.CardTemplateID).
				Order("id asc").
				Pluck("project_id", &ids)
		}
	}
	// 多项目时必须选项目；单项目时默认该项目
	if len(ids) > 1 {
		if input.ProjectID == nil || *input.ProjectID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请先选择项目"})
			return
		}
		ok := false
		for _, v := range ids {
			if v == *input.ProjectID {
				ok = true
				break
			}
		}
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目"})
			return
		}
	} else if len(ids) == 1 {
		if input.ProjectID == nil || *input.ProjectID == 0 {
			pid := ids[0]
			input.ProjectID = &pid
		}
	}

	if appointment == nil && input.ProjectID != nil && *input.ProjectID > 0 {
		var project models.MerchantProject
		if err := config.DB.Where("id = ? AND merchant_id = ?", *input.ProjectID, card.MerchantID).First(&project).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "查询项目失败"})
				return
			}
		} else if !merchantProjectServiceTimeAllowed(project, now) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "当前不在卡片项目服务时间"})
			return
		}
	}

	// 恢复为默认短有效期核销码：每次生成都强制生成新码，并使旧未使用码立即失效
	config.DB.Model(&models.VerifyCode{}).
		Where("card_id = ? AND used = ? AND expire_at > ?", card.ID, false, now.Unix()).
		Update("expire_at", now.Unix())

	code := uuid.New().String()[:8]
	if appointment != nil {
		code = "APPT-" + strings.ToUpper(uuid.New().String()[:8])
	}
	expireAt := now.Add(5 * time.Minute).Unix()

	verifyCode := models.VerifyCode{CardID: card.ID, ProjectID: input.ProjectID, Code: code, ExpireAt: expireAt, Used: false}
	config.DB.Create(&verifyCode)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"code":           code,
			"expire_at":      expireAt,
			"card_id":        card.ID,
			"project_id":     input.ProjectID,
			"appointment_id": input.AppointmentID,
			"verify_mode": func() string {
				if appointment != nil {
					return "appointment_checkin"
				}
				return "verify"
			}(),
		},
	})
}

func VerifyCard(c *gin.Context) {
	merchantIDAny, ok := c.Get("merchant_id")
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅商户可核销"})
		return
	}
	merchantID, ok := merchantIDAny.(uint)
	if !ok || merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var input struct {
		Code string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var currentMerchant models.Merchant
	if err := config.DB.Select("id, support_hand_card").First(&currentMerchant, merchantID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if currentMerchant.SupportHandCard {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该商户已启用手牌，请使用新的扫码核销流程"})
		return
	}

	var result verifyCommitResult
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		merchant, verifyCode, card, _, err := loadVerifyPrepareContext(tx, merchantID, input.Code, now)
		if err != nil {
			return err
		}
		result, err = performVerifyCommit(tx, c, merchant, verifyCode, card, "")
		return err
	})
	if err != nil {
		var ae apiErr
		if errors.As(err, &ae) {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "核销成功",
		"data": gin.H{
			"usage_id":                result.UsageID,
			"card_id":                 result.Card.ID,
			"remain_times":            result.RemainTimes,
			"used_at":                 result.UsedAt.Format("2006-01-02 15:04:05"),
			"session_id":              result.SessionID,
			"next_step":               result.NextStep,
			"appointment_status":      result.AppointmentStatus,
			"predicted_delay_minutes": result.PredictedWaitMinutes,
			"session_wait_state":      result.SessionWaitState,
			"bound_technician_id":     result.BoundTechnicianID,
		},
	})
	enqueueVerifyUsageIfNeeded(result.Merchant, result.Card, result.UsageID, result.ShouldEnqueueOnsite)
}

func FinishVerifyCard(c *gin.Context) {
	merchantIDAny, ok := c.Get("merchant_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	merchantID, ok := merchantIDAny.(uint)
	if !ok || merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	var merchant models.Merchant
	if err := config.DB.Select("id", "support_queue", "support_customer_service_mode", "queue_mode", "finish_term").First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取商户信息失败"})
		return
	}
	finishTerm := resolveFinishTerm(&merchant)

	// 获取账号类型
	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)

	// 商户老板号拥有全部权限，不需要检查
	if authType != "merchant" {
		// 技师账号需要检查结单权限
		okFinish, err := middleware.HasPermission(c, "merchant.card.finish")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "权限检查失败"})
			return
		}
		if !okFinish {
			c.JSON(http.StatusForbidden, gin.H{"error": "无" + finishTerm + "权限"})
			return
		}
	}

	var techID uint
	var hasTechnicianID bool
	if authType == "staff" {
		techIDAny, ok := c.Get("technician_id")
		if ok {
			techID, hasTechnicianID = techIDAny.(uint)
			hasTechnicianID = hasTechnicianID && techID > 0
		}
	}

	var input struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 已取消手动结束能力：仅支持开始服务后自动结束。
	c.JSON(http.StatusBadRequest, gin.H{"error": "已取消手动" + finishTerm + "，请等待系统自动" + finishTerm})
}

func ScanVerifyCard(c *gin.Context) {
	merchantIDAny, ok := c.Get("merchant_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	merchantID, ok := merchantIDAny.(uint)
	if !ok || merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	// 获取账号类型
	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)

	var input struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	code := strings.TrimSpace(input.Code)
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code 不能为空"})
		return
	}

	if !strings.HasPrefix(code, "SS:") {
		var currentMerchant models.Merchant
		if err := config.DB.Select("id, support_hand_card").First(&currentMerchant, merchantID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if currentMerchant.SupportHandCard {
			c.JSON(http.StatusBadRequest, gin.H{"error": "该商户已启用手牌，请升级到新的扫码核销流程"})
			return
		}
	}

	// 仅“服务结束权限”账号（无核销权限）只能扫码开始服务（SS:<session_id>），禁止扫码核销。
	// 说明：路由层允许 RequireAnyPermission(verify, finish)，这里做更细粒度校验。
	if !strings.HasPrefix(code, "SS:") {
		// 商户老板号默认拥有全部权限，不做限制
		if authType != "merchant" {
			okVerify, err := middleware.HasPermission(c, "merchant.card.verify")
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "权限检查失败"})
				return
			}
			if !okVerify {
				c.JSON(http.StatusForbidden, gin.H{"error": "无核销权限"})
				return
			}
		}
	}

	// B方案：服务二维码（SS:<session_id>）走服务会话开始逻辑
	if handled := handleServiceSessionStartScan(c, code); handled {
		return
	}

	var result verifyCommitResult
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		var merchant models.Merchant
		if err := loadVerifyMerchant(tx, merchantID, &merchant); err != nil {
			return err
		}
		var verifyCode models.VerifyCode
		if err := loadVerifyCodeForUpdate(tx, code, &verifyCode); err != nil {
			return err
		}
		if verifyCode.Used {
			return apiErr{status: http.StatusBadRequest, msg: "已取消手动" + resolveFinishTerm(&merchant) + "，请等待系统自动" + resolveFinishTerm(&merchant)}
		}
		if now.Unix() > verifyCode.ExpireAt {
			return apiErr{status: http.StatusBadRequest, msg: "核销码已过期"}
		}
		var card models.Card
		if err := loadVerifyCardContext(tx, merchant, verifyCode, now, &card); err != nil {
			return err
		}
		var err error
		result, err = performVerifyCommit(tx, c, merchant, verifyCode, card, "")
		return err
	})
	if err != nil {
		var ae apiErr
		if errors.As(err, &ae) {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := gin.H{
		"action":                  result.Action,
		"card_id":                 result.Card.ID,
		"usage_id":                result.UsageID,
		"remain_times":            result.RemainTimes,
		"used_at":                 result.UsedAt.Format("2006-01-02 15:04:05"),
		"session_id":              result.SessionID,
		"next_step":               result.NextStep,
		"appointment_status":      result.AppointmentStatus,
		"predicted_delay_minutes": result.PredictedWaitMinutes,
		"session_wait_state":      result.SessionWaitState,
		"bound_technician_id":     result.BoundTechnicianID,
	}
	c.JSON(http.StatusOK, gin.H{"message": "核销成功", "data": resp})
	enqueueVerifyUsageIfNeeded(result.Merchant, result.Card, result.UsageID, result.ShouldEnqueueOnsite)
	return
}

func GetTodayVerify(c *gin.Context) {
	merchantID, ok := ensureMerchantScope(c, "id")
	if !ok {
		return
	}
	today := time.Now().Format("2006-01-02")

	var count int64
	config.DB.Model(&models.Usage{}).
		Where("merchant_id = ? AND DATE(used_at) = ?", merchantID, today).
		Count(&count)

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"count": count}})
}
