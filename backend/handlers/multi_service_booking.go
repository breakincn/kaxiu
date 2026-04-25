package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"kabao/config"
	"kabao/models"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	multiServiceBookingStatusBooked   = "booked"
	multiServiceBookingStatusCanceled = "canceled"
	multiServiceBookingStatusAttended = "attended"
	multiServiceBookingStatusNoShow   = "no_show"
)

type multiServiceSlotView struct {
	ProjectID                   uint       `json:"project_id"`
	ProjectName                 string     `json:"project_name"`
	ProjectDuration             int        `json:"project_duration"`
	ServiceCapacity             int        `json:"service_capacity"`
	SlotStartAt                 time.Time  `json:"slot_start_at"`
	SlotEndAt                   time.Time  `json:"slot_end_at"`
	BookingOpenAt               time.Time  `json:"booking_open_at"`
	CancelDeadlineAt            time.Time  `json:"cancel_deadline_at"`
	BookedCount                 int        `json:"booked_count"`
	CheckedInCount              int        `json:"checked_in_count"`
	RemainingCount              int        `json:"remaining_count"`
	IsFull                      bool       `json:"is_full"`
	CurrentUserBooked           bool       `json:"current_user_booked"`
	CurrentUserBookingID        *uint      `json:"current_user_booking_id,omitempty"`
	CurrentUserBookingStatus    string     `json:"current_user_booking_status"`
	CurrentUserCanCancel        bool       `json:"current_user_can_cancel"`
	CurrentUserCancelDeadlineAt *time.Time `json:"current_user_cancel_deadline_at,omitempty"`
}

func merchantProjectMultiServiceCancelDeadlineMinutes(project models.MerchantProject) int {
	if project.MultiServiceBookingCancelDeadlineMinutesBeforeStart > 0 {
		return project.MultiServiceBookingCancelDeadlineMinutesBeforeStart
	}
	return 60
}

func merchantProjectServiceSlotStart(slot models.MerchantProjectServiceTimeSlot, date time.Time) (time.Time, bool) {
	loc := config.ProjectServiceTimeLocation()
	startClock, err := time.ParseInLocation("15:04", strings.TrimSpace(slot.StartTime), loc)
	if err != nil {
		return time.Time{}, false
	}
	value := time.Date(date.Year(), date.Month(), date.Day(), startClock.Hour(), startClock.Minute(), 0, 0, loc)
	return value, true
}

func merchantProjectServiceWindowEnd(project models.MerchantProject, slotStart time.Time) (time.Time, bool) {
	duration := time.Duration(project.Duration) * time.Minute
	if duration <= 0 {
		return time.Time{}, false
	}
	return slotStart.Add(duration).Add(-3 * time.Minute), true
}

func merchantProjectBookingWindowStart(slotStart time.Time) time.Time {
	return slotStart.Add(-24 * time.Hour)
}

func merchantProjectCancelDeadline(project models.MerchantProject, slotStart time.Time) time.Time {
	return slotStart.Add(-time.Duration(merchantProjectMultiServiceCancelDeadlineMinutes(project)) * time.Minute)
}

func collectProjectUpcomingBookableSlots(project models.MerchantProject, now time.Time, limit int) []multiServiceSlotView {
	if project.ServiceCapacity <= 1 || len(project.ServiceTimeSlots) == 0 || limit <= 0 {
		return nil
	}
	loc := config.ProjectServiceTimeLocation()
	localNow := now.In(loc)
	results := make([]multiServiceSlotView, 0, limit)
	seen := make(map[string]struct{})
	for _, slot := range project.ServiceTimeSlots {
		for offset := 0; offset <= 2; offset++ {
			candidateDate := localNow.AddDate(0, 0, offset)
			if !config.MerchantProjectServiceTimeSlotMatchesDate(slot, candidateDate) {
				continue
			}
			slotStart, ok := merchantProjectServiceSlotStart(slot, candidateDate)
			if !ok {
				continue
			}
			if !slotStart.After(localNow) {
				continue
			}
			if slotStart.Sub(localNow) > 24*time.Hour {
				continue
			}
			slotEnd, ok := merchantProjectServiceWindowEnd(project, slotStart)
			if !ok {
				continue
			}
			key := fmt.Sprintf("%d:%s", project.ID, slotStart.Format(time.RFC3339))
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			results = append(results, multiServiceSlotView{
				ProjectID:        project.ID,
				ProjectName:      strings.TrimSpace(project.Name),
				ProjectDuration:  project.Duration,
				ServiceCapacity:  project.ServiceCapacity,
				SlotStartAt:      slotStart,
				SlotEndAt:        slotEnd,
				BookingOpenAt:    merchantProjectBookingWindowStart(slotStart),
				CancelDeadlineAt: merchantProjectCancelDeadline(project, slotStart),
			})
			break
		}
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].SlotStartAt.Equal(results[j].SlotStartAt) {
			return results[i].ProjectID < results[j].ProjectID
		}
		return results[i].SlotStartAt.Before(results[j].SlotStartAt)
	})
	if len(results) > limit {
		results = results[:limit]
	}
	return results
}

func findProjectRelevantSummarySlot(project models.MerchantProject, now time.Time) (*multiServiceSlotView, bool) {
	if project.ServiceCapacity <= 1 || len(project.ServiceTimeSlots) == 0 {
		return nil, false
	}
	loc := config.ProjectServiceTimeLocation()
	localNow := now.In(loc)
	var best *multiServiceSlotView
	for _, slot := range project.ServiceTimeSlots {
		for offset := -1; offset <= 2; offset++ {
			candidateDate := localNow.AddDate(0, 0, offset)
			if !config.MerchantProjectServiceTimeSlotMatchesDate(slot, candidateDate) {
				continue
			}
			slotStart, ok := merchantProjectServiceSlotStart(slot, candidateDate)
			if !ok {
				continue
			}
			slotEnd, ok := merchantProjectServiceWindowEnd(project, slotStart)
			if !ok {
				continue
			}
			bookingOpenAt := merchantProjectBookingWindowStart(slotStart)
			if localNow.Before(bookingOpenAt) || localNow.After(slotEnd) {
				continue
			}
			item := &multiServiceSlotView{
				ProjectID:        project.ID,
				ProjectName:      strings.TrimSpace(project.Name),
				ProjectDuration:  project.Duration,
				ServiceCapacity:  project.ServiceCapacity,
				SlotStartAt:      slotStart,
				SlotEndAt:        slotEnd,
				BookingOpenAt:    bookingOpenAt,
				CancelDeadlineAt: merchantProjectCancelDeadline(project, slotStart),
			}
			if best == nil || item.SlotStartAt.Before(best.SlotStartAt) {
				best = item
			}
		}
	}
	if best == nil {
		return nil, false
	}
	return best, true
}

func findProjectCurrentServiceSlot(project models.MerchantProject, now time.Time) (*multiServiceSlotView, bool) {
	if project.ServiceCapacity <= 1 || len(project.ServiceTimeSlots) == 0 {
		return nil, false
	}
	loc := config.ProjectServiceTimeLocation()
	localNow := now.In(loc)
	for _, slot := range project.ServiceTimeSlots {
		for offset := -1; offset <= 1; offset++ {
			candidateDate := localNow.AddDate(0, 0, offset)
			if !config.MerchantProjectServiceTimeSlotMatchesDate(slot, candidateDate) {
				continue
			}
			slotStart, ok := merchantProjectServiceSlotStart(slot, candidateDate)
			if !ok {
				continue
			}
			slotEnd, ok := merchantProjectServiceWindowEnd(project, slotStart)
			if !ok {
				continue
			}
			windowStart := slotStart.Add(-1 * time.Hour)
			if !localNow.Before(windowStart) && !localNow.After(slotEnd) {
				return &multiServiceSlotView{
					ProjectID:        project.ID,
					ProjectName:      strings.TrimSpace(project.Name),
					ProjectDuration:  project.Duration,
					ServiceCapacity:  project.ServiceCapacity,
					SlotStartAt:      slotStart,
					SlotEndAt:        slotEnd,
					BookingOpenAt:    merchantProjectBookingWindowStart(slotStart),
					CancelDeadlineAt: merchantProjectCancelDeadline(project, slotStart),
				}, true
			}
		}
	}
	return nil, false
}

func loadCardBoundProjects(tx *gorm.DB, card models.Card) ([]models.MerchantProject, error) {
	if tx == nil {
		return nil, nil
	}
	var projects []models.MerchantProject
	if err := tx.Raw(`
		SELECT p.*
		FROM merchant_projects p
		INNER JOIN card_projects cp ON cp.project_id = p.id
		WHERE cp.card_id = ? AND p.merchant_id = ?
		ORDER BY cp.id ASC
	`, card.ID, card.MerchantID).Scan(&projects).Error; err != nil {
		return nil, err
	}
	if len(projects) == 0 {
		project, err := config.GetDefaultMerchantProject(tx, card.MerchantID)
		if err != nil {
			return nil, err
		}
		if project != nil {
			projects = append(projects, *project)
		}
	}
	return projects, nil
}

func loadOwnedCardWithProjects(tx *gorm.DB, cardID uint, userID uint) (*models.Card, []models.MerchantProject, error) {
	if tx == nil || cardID == 0 || userID == 0 {
		return nil, nil, gorm.ErrRecordNotFound
	}
	var card models.Card
	if err := tx.Preload("Merchant").First(&card, cardID).Error; err != nil {
		return nil, nil, err
	}
	if card.UserID != userID {
		return nil, nil, apiErr{status: http.StatusForbidden, msg: "无权操作此卡"}
	}
	projects, err := loadCardBoundProjects(tx, card)
	if err != nil {
		return nil, nil, err
	}
	return &card, projects, nil
}

func findCardProject(projects []models.MerchantProject, projectID uint) (*models.MerchantProject, bool) {
	for i := range projects {
		if projects[i].ID == projectID {
			return &projects[i], true
		}
	}
	return nil, false
}

func loadSlotBookings(tx *gorm.DB, projectID uint, slotStart time.Time) ([]models.MultiServiceBooking, error) {
	if tx == nil || projectID == 0 {
		return nil, nil
	}
	parseNullableTime := func(value sql.NullString) *time.Time {
		if !value.Valid {
			return nil
		}
		raw := strings.TrimSpace(value.String)
		if raw == "" {
			return nil
		}
		layouts := []string{
			"2006-01-02 15:04:05.999999999-07:00",
			"2006-01-02 15:04:05.999999999",
			"2006-01-02 15:04:05.999",
			"2006-01-02 15:04:05",
			time.RFC3339Nano,
			time.RFC3339,
		}
		loc := config.ProjectServiceTimeLocation()
		for _, layout := range layouts {
			if parsed, err := time.ParseInLocation(layout, raw, loc); err == nil {
				return &parsed
			}
		}
		return nil
	}
	type bookingRow struct {
		ID            uint           `gorm:"column:id"`
		MerchantID    uint           `gorm:"column:merchant_id"`
		ProjectID     uint           `gorm:"column:project_id"`
		CardID        uint           `gorm:"column:card_id"`
		UserID        uint           `gorm:"column:user_id"`
		SlotStartAt   sql.NullString `gorm:"column:slot_start_at"`
		SlotEndAt     sql.NullString `gorm:"column:slot_end_at"`
		Status        string         `gorm:"column:status"`
		BookedAt      sql.NullString `gorm:"column:booked_at"`
		CanceledAt    sql.NullString `gorm:"column:canceled_at"`
		CancelReason  string         `gorm:"column:cancel_reason"`
		CancelPenalty bool           `gorm:"column:cancel_penalty"`
		AttendedAt    sql.NullString `gorm:"column:attended_at"`
		NoShowAt      sql.NullString `gorm:"column:no_show_at"`
		UsageID       *uint          `gorm:"column:usage_id"`
		CreatedAt     sql.NullString `gorm:"column:created_at"`
		UpdatedAt     sql.NullString `gorm:"column:updated_at"`
	}
	var rows []bookingRow
	err := tx.Table("multi_service_bookings").
		Select("id, merchant_id, project_id, card_id, user_id, slot_start_at, slot_end_at, status, booked_at, canceled_at, cancel_reason, cancel_penalty, attended_at, no_show_at, usage_id, created_at, updated_at").
		Where("project_id = ? AND slot_start_at = ?", projectID, slotStart).
		Order("booked_at asc, id asc").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	bookings := make([]models.MultiServiceBooking, 0, len(rows))
	for _, row := range rows {
		bookings = append(bookings, models.MultiServiceBooking{
			ID:            row.ID,
			MerchantID:    row.MerchantID,
			ProjectID:     row.ProjectID,
			CardID:        row.CardID,
			UserID:        row.UserID,
			SlotStartAt:   parseNullableTime(row.SlotStartAt),
			SlotEndAt:     parseNullableTime(row.SlotEndAt),
			Status:        row.Status,
			BookedAt:      parseNullableTime(row.BookedAt),
			CanceledAt:    parseNullableTime(row.CanceledAt),
			CancelReason:  row.CancelReason,
			CancelPenalty: row.CancelPenalty,
			AttendedAt:    parseNullableTime(row.AttendedAt),
			NoShowAt:      parseNullableTime(row.NoShowAt),
			UsageID:       row.UsageID,
			CreatedAt:     parseNullableTime(row.CreatedAt),
			UpdatedAt:     parseNullableTime(row.UpdatedAt),
		})
	}
	return bookings, nil
}

func countSlotBookedOccupancy(bookings []models.MultiServiceBooking) int {
	total := 0
	for _, booking := range bookings {
		switch strings.TrimSpace(booking.Status) {
		case multiServiceBookingStatusBooked, multiServiceBookingStatusAttended, multiServiceBookingStatusNoShow:
			total++
		}
	}
	return total
}

func countSlotCheckedIn(bookings []models.MultiServiceBooking) int {
	total := 0
	for _, booking := range bookings {
		if strings.TrimSpace(booking.Status) == multiServiceBookingStatusAttended {
			total++
		}
	}
	return total
}

func findUserSlotBooking(bookings []models.MultiServiceBooking, userID uint) *models.MultiServiceBooking {
	for i := range bookings {
		if bookings[i].UserID == userID {
			return &bookings[i]
		}
	}
	return nil
}

func listRecentNoShowCountsByUser(tx *gorm.DB, merchantID uint, userIDs []uint, now time.Time) map[uint]int {
	result := make(map[uint]int)
	if tx == nil || merchantID == 0 || len(userIDs) == 0 {
		return result
	}
	since := now.AddDate(0, -6, 0)
	type row struct {
		UserID uint `gorm:"column:user_id"`
		Count  int  `gorm:"column:cnt"`
	}
	var rows []row
	if err := tx.Table("multi_service_penalty_ledgers").
		Select("user_id, COUNT(*) AS cnt").
		Where("merchant_id = ? AND user_id IN ? AND counts_toward_no_show = ? AND penalty_at >= ?", merchantID, userIDs, true, since).
		Group("user_id").
		Scan(&rows).Error; err != nil {
		return result
	}
	for _, row := range rows {
		result[row.UserID] = row.Count
	}
	return result
}

func fillSlotViewCounts(slot *multiServiceSlotView, bookings []models.MultiServiceBooking, userID uint) {
	if slot == nil {
		return
	}
	slot.BookedCount = countSlotBookedOccupancy(bookings)
	slot.CheckedInCount = countSlotCheckedIn(bookings)
	slot.RemainingCount = slot.ServiceCapacity - slot.BookedCount
	if slot.RemainingCount < 0 {
		slot.RemainingCount = 0
	}
	slot.IsFull = slot.BookedCount >= slot.ServiceCapacity && slot.ServiceCapacity > 0
	currentUserBooking := findUserSlotBooking(bookings, userID)
	if currentUserBooking != nil {
		slot.CurrentUserBooked = strings.TrimSpace(currentUserBooking.Status) != multiServiceBookingStatusCanceled
		slot.CurrentUserBookingID = &currentUserBooking.ID
		slot.CurrentUserBookingStatus = strings.TrimSpace(currentUserBooking.Status)
		if slot.CurrentUserBooked && currentUserBooking.Status == multiServiceBookingStatusBooked {
			deadline := slot.CancelDeadlineAt
			slot.CurrentUserCancelDeadlineAt = &deadline
			slot.CurrentUserCanCancel = time.Now().In(config.ProjectServiceTimeLocation()).Before(slot.SlotStartAt)
		}
	}
}

func ListCardMultiServiceBookingSlots(c *gin.Context) {
	userID, ok := mustUserID(c)
	if !ok {
		return
	}
	cardID64, err := strconv.ParseUint(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || cardID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的卡片ID"})
		return
	}
	now := time.Now()
	card, projects, err := loadOwnedCardWithProjects(config.DB, uint(cardID64), userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "卡片不存在"})
			return
		}
		var ae apiErr
		if errors.As(err, &ae) {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取卡片失败"})
		return
	}
	_ = card
	response := make([]multiServiceSlotView, 0)
	for _, project := range projects {
		if project.ServiceCapacity <= 1 {
			continue
		}
		slots := collectProjectUpcomingBookableSlots(project, now, 8)
		for _, slot := range slots {
			bookings, err := loadSlotBookings(config.DB, project.ID, slot.SlotStartAt)
			if err != nil {
				continue
			}
			fillSlotViewCounts(&slot, bookings, userID)
			response = append(response, slot)
		}
	}
	sort.Slice(response, func(i, j int) bool {
		if response[i].SlotStartAt.Equal(response[j].SlotStartAt) {
			return response[i].ProjectID < response[j].ProjectID
		}
		return response[i].SlotStartAt.Before(response[j].SlotStartAt)
	})
	c.JSON(http.StatusOK, gin.H{"data": response})
}

func CreateCardMultiServiceBooking(c *gin.Context) {
	userID, ok := mustUserID(c)
	if !ok {
		return
	}
	cardID64, err := strconv.ParseUint(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || cardID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的卡片ID"})
		return
	}
	var input struct {
		ProjectID   uint   `json:"project_id" binding:"required"`
		SlotStartAt string `json:"slot_start_at" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	loc := config.ProjectServiceTimeLocation()
	slotStart, err := time.ParseInLocation("2006-01-02 15:04:05", strings.TrimSpace(input.SlotStartAt), loc)
	if err != nil {
		slotStart, err = time.ParseInLocation(time.RFC3339, strings.TrimSpace(input.SlotStartAt), loc)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "场次开始时间格式错误"})
			return
		}
	}
	now := time.Now().In(loc)
	var created models.MultiServiceBooking
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		card, projects, err := loadOwnedCardWithProjects(tx, uint(cardID64), userID)
		if err != nil {
			return err
		}
		project, ok := findCardProject(projects, input.ProjectID)
		if !ok || project == nil {
			return apiErr{status: http.StatusBadRequest, msg: "无效的项目"}
		}
		if project.ServiceCapacity <= 1 {
			return apiErr{status: http.StatusBadRequest, msg: "当前项目不支持多人课程预约"}
		}
		slot, ok := findProjectSlotByStart(*project, slotStart)
		if !ok {
			return apiErr{status: http.StatusBadRequest, msg: "无效的项目场次"}
		}
		if now.Before(slot.BookingOpenAt) || !now.Before(slot.SlotStartAt) {
			return apiErr{status: http.StatusBadRequest, msg: "当前不在可预约时间窗口内"}
		}
		var existing models.MultiServiceBooking
		err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("project_id = ? AND slot_start_at = ? AND user_id = ?", project.ID, slot.SlotStartAt, userID).
			First(&existing).Error
		if err == nil {
			if strings.TrimSpace(existing.Status) == multiServiceBookingStatusCanceled {
				return apiErr{status: http.StatusBadRequest, msg: "当前场次预约已取消，请勿重复预约"}
			}
			return apiErr{status: http.StatusBadRequest, msg: "你已预约该场次"}
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		bookings, err := loadSlotBookings(tx, project.ID, slot.SlotStartAt)
		if err != nil {
			return err
		}
		if countSlotBookedOccupancy(bookings) >= project.ServiceCapacity {
			return apiErr{status: http.StatusBadRequest, msg: "当前场次预约已满"}
		}
		created = models.MultiServiceBooking{
			MerchantID:  card.MerchantID,
			ProjectID:   project.ID,
			CardID:      card.ID,
			UserID:      userID,
			SlotStartAt: &slot.SlotStartAt,
			SlotEndAt:   &slot.SlotEndAt,
			Status:      multiServiceBookingStatusBooked,
			BookedAt:    &now,
		}
		return tx.Create(&created).Error
	})
	if err != nil {
		var ae apiErr
		if errors.As(err, &ae) {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "卡片不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "预约失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": created})
}

func CancelCardMultiServiceBooking(c *gin.Context) {
	userID, ok := mustUserID(c)
	if !ok {
		return
	}
	cardID64, err := strconv.ParseUint(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || cardID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的卡片ID"})
		return
	}
	bookingID64, err := strconv.ParseUint(strings.TrimSpace(c.Param("booking_id")), 10, 64)
	if err != nil || bookingID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的预约ID"})
		return
	}
	var input struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&input)
	now := time.Now().In(config.ProjectServiceTimeLocation())
	var updated models.MultiServiceBooking
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		card, projects, err := loadOwnedCardWithProjects(tx, uint(cardID64), userID)
		if err != nil {
			return err
		}
		var booking models.MultiServiceBooking
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&booking, uint(bookingID64)).Error; err != nil {
			return err
		}
		if booking.CardID != card.ID || booking.UserID != userID {
			return apiErr{status: http.StatusForbidden, msg: "无权操作该预约"}
		}
		if strings.TrimSpace(booking.Status) != multiServiceBookingStatusBooked {
			return apiErr{status: http.StatusBadRequest, msg: "当前预约不可取消"}
		}
		project, ok := findCardProject(projects, booking.ProjectID)
		if !ok || project == nil {
			return apiErr{status: http.StatusBadRequest, msg: "项目不存在"}
		}
		if booking.SlotStartAt == nil || !now.Before(*booking.SlotStartAt) {
			return apiErr{status: http.StatusBadRequest, msg: "课程已开始，当前不可取消"}
		}
		cancelPenalty := !now.Before(merchantProjectCancelDeadline(*project, *booking.SlotStartAt))
		updates := map[string]interface{}{
			"status":         multiServiceBookingStatusCanceled,
			"canceled_at":    &now,
			"cancel_reason":  strings.TrimSpace(input.Reason),
			"cancel_penalty": cancelPenalty,
		}
		if err := tx.Model(&models.MultiServiceBooking{}).Where("id = ?", booking.ID).Updates(updates).Error; err != nil {
			return err
		}
		updated = booking
		updated.Status = multiServiceBookingStatusCanceled
		updated.CanceledAt = &now
		updated.CancelReason = strings.TrimSpace(input.Reason)
		updated.CancelPenalty = cancelPenalty
		if cancelPenalty {
			if _, err := recordMultiServicePenaltyLedger(tx, *card, *project, &updated, "late_cancel", true, 0, 0, "临近开课取消", now, nil); err != nil {
				return err
			}
			if err := applyMultiServiceOverLimitPenaltyIfNeeded(tx, *card, *project, &updated, now); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		var ae apiErr
		if errors.As(err, &ae) {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "取消预约失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": updated})
}

func findProjectSlotByStart(project models.MerchantProject, slotStart time.Time) (*multiServiceSlotView, bool) {
	loc := config.ProjectServiceTimeLocation()
	localStart := slotStart.In(loc)
	for _, slot := range project.ServiceTimeSlots {
		if !config.MerchantProjectServiceTimeSlotMatchesDate(slot, localStart) {
			continue
		}
		startAt, ok := merchantProjectServiceSlotStart(slot, localStart)
		if !ok || !startAt.Equal(localStart) {
			continue
		}
		endAt, ok := merchantProjectServiceWindowEnd(project, startAt)
		if !ok {
			continue
		}
		return &multiServiceSlotView{
			ProjectID:        project.ID,
			ProjectName:      strings.TrimSpace(project.Name),
			ProjectDuration:  project.Duration,
			ServiceCapacity:  project.ServiceCapacity,
			SlotStartAt:      startAt,
			SlotEndAt:        endAt,
			BookingOpenAt:    merchantProjectBookingWindowStart(startAt),
			CancelDeadlineAt: merchantProjectCancelDeadline(project, startAt),
		}, true
	}
	return nil, false
}

func recordMultiServicePenaltyLedger(tx *gorm.DB, card models.Card, project models.MerchantProject, booking *models.MultiServiceBooking, penaltyType string, countsToward bool, chargedTimes int, chargedAmount int, remark string, now time.Time, usageID *uint) (*models.MultiServicePenaltyLedger, error) {
	if tx == nil {
		return nil, nil
	}
	ledger := &models.MultiServicePenaltyLedger{
		MerchantID:         card.MerchantID,
		ProjectID:          project.ID,
		CardID:             card.ID,
		UserID:             card.UserID,
		PenaltyType:        penaltyType,
		CountsTowardNoShow: countsToward,
		ChargedTimes:       chargedTimes,
		ChargedAmount:      chargedAmount,
		Remark:             strings.TrimSpace(remark),
		PenaltyAt:          &now,
		UsageID:            usageID,
	}
	if booking != nil && booking.ID > 0 {
		ledger.BookingID = &booking.ID
	}
	if err := tx.Create(ledger).Error; err != nil {
		return nil, err
	}
	return ledger, nil
}

func applyMultiServicePenaltyUsage(tx *gorm.DB, cardID uint, project models.MerchantProject, note string, now time.Time) (*models.Usage, models.Card, int, int, error) {
	var zeroCard models.Card
	if tx == nil || cardID == 0 {
		return nil, zeroCard, 0, 0, gorm.ErrRecordNotFound
	}
	var card models.Card
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&card, cardID).Error; err != nil {
		return nil, zeroCard, 0, 0, err
	}
	chargedTimes := 0
	chargedAmount := 0
	cardUpdates := map[string]interface{}{
		"last_used_at": now,
	}
	if card.TotalTimes > 0 {
		chargedTimes = 1
		res := tx.Model(&models.Card{}).
			Where("id = ? AND remain_times > 0", card.ID).
			Updates(map[string]interface{}{
				"remain_times": gorm.Expr("remain_times - 1"),
				"used_times":   gorm.Expr("used_times + 1"),
				"last_used_at": now,
			})
		if res.Error != nil {
			return nil, zeroCard, 0, 0, res.Error
		}
		if res.RowsAffected == 0 {
			return nil, zeroCard, 0, 0, apiErr{status: http.StatusBadRequest, msg: "卡片剩余次数不足"}
		}
		card.RemainTimes--
		card.UsedTimes++
	} else {
		chargedAmount = int(math.Ceil(project.Price))
		if chargedAmount <= 0 {
			return nil, zeroCard, 0, 0, apiErr{status: http.StatusBadRequest, msg: "项目价格未配置，无法扣减额度"}
		}
		res := tx.Model(&models.Card{}).
			Where("id = ? AND recharge_amount >= ?", card.ID, chargedAmount).
			Updates(map[string]interface{}{
				"recharge_amount": gorm.Expr("recharge_amount - ?", chargedAmount),
				"last_used_at":    now,
			})
		if res.Error != nil {
			return nil, zeroCard, 0, 0, res.Error
		}
		if res.RowsAffected == 0 {
			return nil, zeroCard, 0, 0, apiErr{status: http.StatusBadRequest, msg: "卡片额度不足"}
		}
		card.RechargeAmount -= chargedAmount
		_ = cardUpdates
	}
	card.LastUsedAt = &now
	usage := &models.Usage{
		CardID:             card.ID,
		MerchantID:         card.MerchantID,
		ProjectID:          &project.ID,
		UsedTimes:          chargedTimes,
		UsedAt:             &now,
		VerifyCode:         "",
		VerifyCodeExpireAt: now.Unix(),
		Status:             "success",
		SourceType:         "multi_service_booking_penalty",
		SourceNote:         strings.TrimSpace(note),
	}
	applyUsageCardSnapshotFromCard(usage, card)
	if err := tx.Create(usage).Error; err != nil {
		return nil, zeroCard, 0, 0, err
	}
	return usage, card, chargedTimes, chargedAmount, nil
}

func applyMultiServiceOverLimitPenaltyIfNeeded(tx *gorm.DB, card models.Card, project models.MerchantProject, booking *models.MultiServiceBooking, now time.Time) error {
	if tx == nil {
		return nil
	}
	since := now.AddDate(0, -6, 0)
	var count int64
	if err := tx.Model(&models.MultiServicePenaltyLedger{}).
		Where("merchant_id = ? AND user_id = ? AND counts_toward_no_show = ? AND penalty_at >= ?", card.MerchantID, card.UserID, true, since).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 || count%3 != 0 {
		return nil
	}
	var existing int64
	if err := tx.Model(&models.MultiServicePenaltyLedger{}).
		Where("merchant_id = ? AND user_id = ? AND penalty_type = ? AND booking_id = ?", card.MerchantID, card.UserID, "over_limit", booking.ID).
		Count(&existing).Error; err != nil {
		return err
	}
	if existing > 0 {
		return nil
	}
	usage, _, chargedTimes, chargedAmount, err := applyMultiServicePenaltyUsage(tx, card.ID, project, "失约超限", now)
	if err != nil {
		return err
	}
	var usageID *uint
	if usage != nil {
		usageID = &usage.ID
	}
	_, err = recordMultiServicePenaltyLedger(tx, card, project, booking, "over_limit", false, chargedTimes, chargedAmount, "失约超限", now, usageID)
	return err
}

func validateMultiServiceVerifyEligibility(tx *gorm.DB, card models.Card, project models.MerchantProject, now time.Time) (*multiServiceSlotView, *models.MultiServiceBooking, error) {
	if tx == nil || project.ID == 0 || project.ServiceCapacity <= 1 {
		return nil, nil, nil
	}
	slot, ok := findProjectCurrentServiceSlot(project, now)
	if !ok || slot == nil {
		return nil, nil, nil
	}
	bookings, err := loadSlotBookings(tx, project.ID, slot.SlotStartAt)
	if err != nil {
		return nil, nil, err
	}
	fillSlotViewCounts(slot, bookings, card.UserID)
	currentUserBooking := findUserSlotBooking(bookings, card.UserID)
	if slot.IsFull && (currentUserBooking == nil || strings.TrimSpace(currentUserBooking.Status) == multiServiceBookingStatusCanceled) {
		return nil, nil, apiErr{status: http.StatusBadRequest, msg: "当前课程预约已满，未预约用户不能核销"}
	}
	return slot, currentUserBooking, nil
}

func markMultiServiceBookingAttendance(tx *gorm.DB, card models.Card, project models.MerchantProject, slot *multiServiceSlotView, usageID uint, now time.Time) error {
	if tx == nil || slot == nil || usageID == 0 {
		return nil
	}
	var booking models.MultiServiceBooking
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("project_id = ? AND slot_start_at = ? AND user_id = ?", project.ID, slot.SlotStartAt, card.UserID).
		First(&booking).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err == nil {
		if strings.TrimSpace(booking.Status) == multiServiceBookingStatusCanceled {
			return apiErr{status: http.StatusBadRequest, msg: "当前课程预约已取消，不能核销"}
		}
		return tx.Model(&models.MultiServiceBooking{}).
			Where("id = ?", booking.ID).
			Updates(map[string]interface{}{
				"status":      multiServiceBookingStatusAttended,
				"attended_at": &now,
				"usage_id":    usageID,
			}).Error
	}
	created := models.MultiServiceBooking{
		MerchantID:  card.MerchantID,
		ProjectID:   project.ID,
		CardID:      card.ID,
		UserID:      card.UserID,
		SlotStartAt: &slot.SlotStartAt,
		SlotEndAt:   &slot.SlotEndAt,
		Status:      multiServiceBookingStatusAttended,
		BookedAt:    &now,
		AttendedAt:  &now,
		UsageID:     &usageID,
	}
	return tx.Create(&created).Error
}
