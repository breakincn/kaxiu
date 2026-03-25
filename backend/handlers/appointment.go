package handlers

import (
	"errors"
	"fmt"
	"kabao/config"
	"kabao/models"
	"kabao/queue"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

func loadPublishedSchedulePublishings(tx *gorm.DB, merchantID uint, targetDate time.Time) ([]models.TechnicianSchedulePublishing, error) {
	if tx == nil || merchantID == 0 {
		return nil, nil
	}
	dateOnly := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, targetDate.Location())
	rows, err := tx.Table("technician_schedule_publishings").
		Select(technicianSchedulePublishingSelectColumns()).
		Where("merchant_id = ? AND publish_date >= ? AND publish_date < ? AND status = ?", merchantID, dateOnly, dateOnly.Add(24*time.Hour), "published").
		Order("start_at asc, id asc").
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.TechnicianSchedulePublishing, 0)
	for rows.Next() {
		row, err := scanTechnicianSchedulePublishingRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, nil
}

func loadSchedulePublishingsByDate(tx *gorm.DB, merchantID uint, targetDate time.Time) ([]models.TechnicianSchedulePublishing, error) {
	if tx == nil || merchantID == 0 {
		return nil, nil
	}
	dateOnly := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, targetDate.Location())
	rows, err := tx.Table("technician_schedule_publishings").
		Select(technicianSchedulePublishingSelectColumns()).
		Where("merchant_id = ? AND publish_date >= ? AND publish_date < ?", merchantID, dateOnly, dateOnly.Add(24*time.Hour)).
		Order("start_at asc, id asc").
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.TechnicianSchedulePublishing, 0)
	for rows.Next() {
		row, err := scanTechnicianSchedulePublishingRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, nil
}

func loadSchedulePublishingByID(tx *gorm.DB, merchantID, scheduleID uint) (*models.TechnicianSchedulePublishing, error) {
	if tx == nil || merchantID == 0 || scheduleID == 0 {
		return nil, nil
	}
	rows, err := tx.Table("technician_schedule_publishings").
		Select(technicianSchedulePublishingSelectColumns()).
		Where("id = ? AND merchant_id = ?", scheduleID, merchantID).
		Limit(1).
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, nil
	}
	row, err := scanTechnicianSchedulePublishingRow(rows)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func loadLeaveSchedulePublishingByTechnicianAndDate(tx *gorm.DB, merchantID, technicianID uint, targetDate time.Time) (*models.TechnicianSchedulePublishing, error) {
	if tx == nil || merchantID == 0 || technicianID == 0 {
		return nil, nil
	}
	dateOnly := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, targetDate.Location())
	rows, err := tx.Table("technician_schedule_publishings").
		Select(technicianSchedulePublishingSelectColumns()).
		Where("merchant_id = ? AND technician_id = ? AND publish_date >= ? AND publish_date < ? AND status = ?", merchantID, technicianID, dateOnly, dateOnly.Add(24*time.Hour), "leave").
		Order("start_at asc, id asc").
		Limit(1).
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, nil
	}
	row, err := scanTechnicianSchedulePublishingRow(rows)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func loadProtectedRepairSlotsByPublishDate(tx *gorm.DB, merchantID uint, publishDate *time.Time) ([]models.ProtectedRepairSlot, error) {
	if tx == nil || merchantID == 0 {
		return nil, nil
	}
	q := tx.Table("protected_repair_slots").Select(protectedRepairSlotSelectColumns()).Where("merchant_id = ?", merchantID)
	if publishDate != nil {
		dateOnly := time.Date(publishDate.Year(), publishDate.Month(), publishDate.Day(), 0, 0, 0, 0, publishDate.Location())
		q = q.Where("publish_date >= ? AND publish_date < ?", dateOnly, dateOnly.Add(24*time.Hour))
	}
	rows, err := q.Order("id asc").Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.ProtectedRepairSlot, 0)
	for rows.Next() {
		row, err := scanProtectedRepairSlotRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, nil
}

func buildPublishedScheduleIntervals(rows []models.TechnicianSchedulePublishing) ([]businessInterval, bool) {
	intervals := make([]businessInterval, 0, len(rows))
	for _, row := range rows {
		if row.StartAt == nil || row.EndAt == nil || !row.EndAt.After(*row.StartAt) {
			continue
		}
		intervals = append(intervals, businessInterval{Start: *row.StartAt, End: *row.EndAt})
	}
	return intervals, len(intervals) > 0
}

func slotWithinPublishedSchedule(rows []models.TechnicianSchedulePublishing, slotStart, slotEnd time.Time, technicianID *uint) bool {
	if len(rows) == 0 {
		return true
	}
	for _, row := range rows {
		if row.StartAt == nil || row.EndAt == nil || !row.EndAt.After(*row.StartAt) {
			continue
		}
		if technicianID != nil && *technicianID > 0 {
			if row.TechnicianID == nil || *row.TechnicianID != *technicianID {
				continue
			}
		} else if row.TechnicianID != nil {
			continue
		}
		if (slotStart.Equal(*row.StartAt) || slotStart.After(*row.StartAt)) && (slotEnd.Equal(*row.EndAt) || slotEnd.Before(*row.EndAt)) {
			return true
		}
	}
	return false
}

func getAppointmentServiceMinutes(merchantID uint, appt models.Appointment) int {
	serviceMinutes, _ := resolveProjectBookingConfig(config.DB, merchantID, appt.ProjectID, 30, 3)
	return serviceMinutes
}

func getAppointmentGapMinutes(merchantID uint, appt models.Appointment) int {
	_, gapMinutes := resolveProjectBookingConfig(config.DB, merchantID, appt.ProjectID, 30, 3)
	return gapMinutes
}

func getAppointmentOccupiedMinutes(merchantID uint, appt models.Appointment) int {
	serviceMinutes, gapMinutes := resolveProjectBookingConfig(config.DB, merchantID, appt.ProjectID, 30, 3)
	return projectBookingOccupiedMinutes(serviceMinutes, gapMinutes)
}

func appointmentPlacementScore(leftMinutes, rightMinutes int) int {
	if leftMinutes < 0 {
		leftMinutes = 0
	}
	if rightMinutes < 0 {
		rightMinutes = 0
	}
	splitPenalty := leftMinutes
	if rightMinutes < splitPenalty {
		splitPenalty = rightMinutes
	}
	return splitPenalty*10000 + leftMinutes + rightMinutes
}

type appointmentTechnicianCandidate struct {
	TechnicianID         uint   `json:"technician_id"`
	AvailabilityState    string `json:"availability_state"`
	PredictedWaitMinutes int    `json:"predicted_wait_minutes"`
	AvailabilityReason   string `json:"availability_reason,omitempty"`
}

type appointmentTimeSlot struct {
	Time                 string                           `json:"time"`
	Available            bool                             `json:"available"`
	UserName             string                           `json:"user_name,omitempty"`
	TechnicianIDs        []uint                           `json:"technician_ids,omitempty"`
	TechnicianCandidates []appointmentTechnicianCandidate `json:"technician_candidates,omitempty"`
	PlacementScoreValue  int                              `json:"placement_score,omitempty"`
	ComparisonKind       string                           `json:"comparison_kind,omitempty"`
	ComparisonLabel      string                           `json:"comparison_label,omitempty"`
	RecommendationReason string                           `json:"recommendation_reason,omitempty"`
}

type appointmentPlacementDecision struct {
	PredictedWaitMinutes int
	AssignedTechnicianID *uint
	PlacementScore       int
}

type appointmentFragmentEvaluation struct {
	CreatesFragment bool
	Reason          string
	PlacementScore  int
}

type rankedAppointmentTimeSlot struct {
	appointmentTimeSlot
	PlacementScore int
	SlotTime       time.Time
}

type availableTimeSlotsOptions struct {
	ExcludeAppointmentID uint
}

type appointmentOccupiedRange struct {
	Start time.Time
	End   time.Time
}

func appointmentLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.Local
	}
	return loc
}

func loadCoreAppointmentOccupiedMinutes(tx *gorm.DB, merchantID uint) ([]int, error) {
	if tx == nil || merchantID == 0 {
		return []int{projectBookingOccupiedMinutes(30, 3)}, nil
	}
	var projects []models.MerchantProject
	if err := tx.Where("merchant_id = ? AND is_active = ? AND bookable_online = ?", merchantID, true, true).Find(&projects).Error; err != nil {
		return nil, err
	}
	out := make([]int, 0, len(projects))
	seen := make(map[int]struct{}, len(projects))
	for _, p := range projects {
		occupied := projectBookingOccupiedMinutes(p.Duration, p.ServiceGapMinutes)
		if occupied <= 0 {
			continue
		}
		if _, ok := seen[occupied]; ok {
			continue
		}
		seen[occupied] = struct{}{}
		out = append(out, occupied)
	}
	if len(out) == 0 {
		out = append(out, projectBookingOccupiedMinutes(30, 3))
	}
	return out, nil
}

func checkNonTechnicianAppointmentOverlap(tx *gorm.DB, merchantID uint, appointmentStart time.Time, occupiedMinutes int, excludeAppointmentID uint) (bool, error) {
	q := tx.Table("appointments").
		Select(appointmentSelectColumns()).
		Where("merchant_id = ? AND technician_id IS NULL AND appointment_time IS NOT NULL AND status IN ?",
			merchantID, appointmentProtectedStatuses())
	if excludeAppointmentID > 0 {
		q = q.Where("id <> ?", excludeAppointmentID)
	}
	rows, err := q.Rows()
	if err != nil {
		return false, err
	}
	defer rows.Close()
	appointmentEnd := appointmentStart.Add(time.Duration(occupiedMinutes) * time.Minute)
	for rows.Next() {
		apt, err := scanAppointmentRow(rows)
		if err != nil {
			return false, err
		}
		if apt.AppointmentTime == nil {
			continue
		}
		aptStart := *apt.AppointmentTime
		aptEnd := aptStart.Add(time.Duration(getAppointmentOccupiedMinutes(merchantID, apt)) * time.Minute)
		if appointmentStart.Before(aptEnd) && aptStart.Before(appointmentEnd) {
			return true, nil
		}
	}
	return false, nil
}

func listAppointmentBookableTechnicians(tx *gorm.DB, merchant models.Merchant) ([]models.Technician, error) {
	if tx == nil || merchant.ID == 0 {
		return nil, nil
	}
	var technicians []models.Technician
	if err := tx.
		Joins("JOIN service_roles sr ON sr.id = technicians.service_role_id").
		Where("technicians.merchant_id = ? AND technicians.is_active = ? AND (sr.role_type IS NULL OR sr.role_type = '' OR sr.role_type <> ?)", merchant.ID, true, "operational").
		Order("technicians.id asc").
		Find(&technicians).Error; err != nil {
		return nil, err
	}
	return technicians, nil
}

func loadAppointmentOccupiedRangesForFragment(tx *gorm.DB, merchant models.Merchant, targetDate time.Time, technicianID *uint, excludeAppointmentID uint) ([]appointmentOccupiedRange, error) {
	dayStart := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, targetDate.Location())
	dayEnd := dayStart.Add(24 * time.Hour)
	q := tx.Table("appointments").
		Select(appointmentSelectColumns()).
		Where("merchant_id = ? AND appointment_time IS NOT NULL AND appointment_time >= ? AND appointment_time < ? AND status IN ?",
			merchant.ID, dayStart, dayEnd, appointmentProtectedStatuses())
	if excludeAppointmentID > 0 {
		q = q.Where("id <> ?", excludeAppointmentID)
	}
	if technicianID != nil && *technicianID > 0 {
		q = q.Where("technician_id = ?", *technicianID)
	} else if merchant.SupportCustomerServiceMode {
		q = q.Where("technician_id IS NULL")
	}
	rows, err := q.Order("appointment_time asc").Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ranges := make([]appointmentOccupiedRange, 0)
	for rows.Next() {
		apt, err := scanAppointmentRow(rows)
		if err != nil {
			return nil, err
		}
		if apt.AppointmentTime == nil {
			continue
		}
		start := *apt.AppointmentTime
		end := start.Add(time.Duration(getAppointmentOccupiedMinutes(merchant.ID, apt)) * time.Minute)
		if end.After(start) {
			ranges = append(ranges, appointmentOccupiedRange{Start: start, End: end})
		}
	}
	return ranges, nil
}

func evaluateAppointmentFragmentImpact(tx *gorm.DB, merchant models.Merchant, targetDate time.Time, appointmentStart time.Time, occupiedMinutes int, technicianID *uint, excludeAppointmentID uint) (appointmentFragmentEvaluation, error) {
	result := appointmentFragmentEvaluation{}
	if tx == nil {
		return result, nil
	}
	coreOccupied, err := loadCoreAppointmentOccupiedMinutes(tx, merchant.ID)
	if err != nil {
		return result, err
	}
	occupiedRanges, err := loadAppointmentOccupiedRangesForFragment(tx, merchant, targetDate, technicianID, excludeAppointmentID)
	if err != nil {
		return result, err
	}
	intervals, ok := getMerchantBusinessIntervalsForDate(merchant, targetDate)
	if !ok {
		return result, nil
	}
	candidateEnd := appointmentStart.Add(time.Duration(occupiedMinutes) * time.Minute)
	for _, it := range intervals {
		if appointmentStart.Before(it.Start) || candidateEnd.After(it.End) {
			continue
		}
		leftAnchor := it.Start
		rightAnchor := it.End
		hasPrevOccupied := false
		hasNextOccupied := false
		for _, r := range occupiedRanges {
			if !r.End.After(it.Start) || !r.Start.Before(it.End) {
				continue
			}
			if !r.End.After(appointmentStart) && r.End.After(leftAnchor) {
				leftAnchor = r.End
				hasPrevOccupied = true
			}
			if !r.Start.Before(candidateEnd) && r.Start.Before(rightAnchor) {
				rightAnchor = r.Start
				hasNextOccupied = true
			}
		}
		leftMinutes := int(appointmentStart.Sub(leftAnchor).Minutes())
		rightMinutes := int(rightAnchor.Sub(candidateEnd).Minutes())
		result.PlacementScore = appointmentPlacementScore(leftMinutes, rightMinutes)
		if hasPrevOccupied && leftMinutes > 0 {
			fit := false
			for _, core := range coreOccupied {
				if leftMinutes >= core {
					fit = true
					break
				}
			}
			if !fit {
				result.CreatesFragment = true
				result.Reason = "该时段会在前段留下不可复用的碎片时间"
				return result, nil
			}
		}
		if hasNextOccupied && rightMinutes > 0 {
			fit := false
			for _, core := range coreOccupied {
				if rightMinutes >= core {
					fit = true
					break
				}
			}
			if !fit {
				result.CreatesFragment = true
				result.Reason = "该时段会在后段留下不可复用的碎片时间"
				return result, nil
			}
		}
		return result, nil
	}
	return result, nil
}

func validateAppointmentPlacementRules(tx *gorm.DB, merchant models.Merchant, targetDate time.Time, appointmentStart time.Time, occupiedMinutes int, technicianID *uint, excludeAppointmentID uint) (appointmentPlacementDecision, error) {
	decision := appointmentPlacementDecision{}
	publishedRows, err := loadPublishedSchedulePublishings(tx, merchant.ID, targetDate)
	if err != nil {
		return decision, err
	}
	appointmentEnd := appointmentStart.Add(time.Duration(occupiedMinutes) * time.Minute)
	if technicianID != nil && *technicianID > 0 {
		if !slotWithinPublishedSchedule(publishedRows, appointmentStart, appointmentEnd, technicianID) {
			return decision, apiErr{status: http.StatusBadRequest, msg: "该时间不在已发布可预约排班内"}
		}
		availability, err := evaluateBookingTechnicianAvailability(tx, merchant, *technicianID, appointmentStart, occupiedMinutes, excludeAppointmentID)
		if err != nil {
			return decision, err
		}
		if availability.State == appointmentAvailabilityUnavailable {
			return decision, apiErr{status: http.StatusBadRequest, msg: availability.Reason}
		}
		fragment, err := evaluateAppointmentFragmentImpact(tx, merchant, targetDate, appointmentStart, occupiedMinutes, technicianID, excludeAppointmentID)
		if err != nil {
			return decision, err
		}
		if fragment.CreatesFragment {
			return decision, apiErr{status: http.StatusBadRequest, msg: fragment.Reason}
		}
		decision.PredictedWaitMinutes = availability.PredictedWaitMinutes
		decision.AssignedTechnicianID = technicianID
		decision.PlacementScore = fragment.PlacementScore
		return decision, nil
	} else if merchant.SupportCustomerServiceMode {
		technicians, err := listAppointmentBookableTechnicians(tx, merchant)
		if err != nil {
			return decision, err
		}
		if len(technicians) == 0 {
			return decision, apiErr{status: http.StatusBadRequest, msg: "当前无可预约客服"}
		}
		found := false
		for _, tech := range technicians {
			if !slotWithinPublishedSchedule(publishedRows, appointmentStart, appointmentEnd, &tech.ID) {
				continue
			}
			availability, err := evaluateBookingTechnicianAvailability(tx, merchant, tech.ID, appointmentStart, occupiedMinutes, excludeAppointmentID)
			if err != nil {
				return decision, err
			}
			if availability.State == appointmentAvailabilityUnavailable {
				continue
			}
			fragment, err := evaluateAppointmentFragmentImpact(tx, merchant, targetDate, appointmentStart, occupiedMinutes, &tech.ID, excludeAppointmentID)
			if err != nil {
				return decision, err
			}
			if fragment.CreatesFragment {
				continue
			}
			candidateScore := fragment.PlacementScore
			if !found || candidateScore < decision.PlacementScore || (candidateScore == decision.PlacementScore && tech.ID < *decision.AssignedTechnicianID) {
				techID := tech.ID
				decision = appointmentPlacementDecision{
					PredictedWaitMinutes: availability.PredictedWaitMinutes,
					AssignedTechnicianID: &techID,
					PlacementScore:       candidateScore,
				}
				found = true
			}
		}
		if !found {
			return decision, apiErr{status: http.StatusBadRequest, msg: "当前时段无可预约客服"}
		}
		return decision, nil
	}
	if !slotWithinPublishedSchedule(publishedRows, appointmentStart, appointmentEnd, nil) {
		return decision, apiErr{status: http.StatusBadRequest, msg: "该时间不在已发布可预约排班内"}
	}
	conflicted, err := checkNonTechnicianAppointmentOverlap(tx, merchant.ID, appointmentStart, occupiedMinutes, excludeAppointmentID)
	if err != nil {
		return decision, err
	}
	if conflicted {
		return decision, apiErr{status: http.StatusBadRequest, msg: "该时间段已被占用"}
	}
	fragment, err := evaluateAppointmentFragmentImpact(tx, merchant, targetDate, appointmentStart, occupiedMinutes, nil, excludeAppointmentID)
	if err != nil {
		return decision, err
	}
	if fragment.CreatesFragment {
		return decision, apiErr{status: http.StatusBadRequest, msg: fragment.Reason}
	}
	decision.PlacementScore = fragment.PlacementScore
	return decision, nil
}

func buildAvailableTimeSlotsPayload(merchant models.Merchant, merchantID uint, date string, projectID uint, loc *time.Location, opts *availableTimeSlotsOptions) (gin.H, error) {
	now := time.Now().In(loc)
	serviceMinutes := 30
	serviceGapMinutes := 3
	if projectID > 0 {
		var project models.MerchantProject
		if err := config.DB.Where("id = ? AND merchant_id = ? AND is_active = ? AND bookable_online = ?", projectID, merchant.ID, true, true).First(&project).Error; err != nil {
			return nil, apiErr{status: http.StatusBadRequest, msg: "无效的项目"}
		}
		serviceMinutes = project.Duration
		serviceGapMinutes = project.ServiceGapMinutes
		if serviceMinutes <= 0 {
			return nil, apiErr{status: http.StatusBadRequest, msg: "项目时长无效"}
		}
	}
	occupiedMinutes := projectBookingOccupiedMinutes(serviceMinutes, serviceGapMinutes)
	excludeAppointmentID := uint(0)
	if opts != nil {
		excludeAppointmentID = opts.ExcludeAppointmentID
	}

	// 懒更新：自动取消该商户已超时(>=35分钟)但未核销的预约，避免继续占用时间段。
	overdueDeadline := now.Add(-35 * time.Minute)
	config.DB.Model(&models.Appointment{}).
		Where("merchant_id = ? AND status IN ('pending','confirmed') AND appointment_time IS NOT NULL AND appointment_time <= ?", merchantID, overdueDeadline).
		Updates(map[string]interface{}{
			"status":      "canceled",
			"canceled_at": now,
		})

	var technicians []models.Technician
	if merchant.SupportCustomerServiceMode {
		var err error
		technicians, err = listAppointmentBookableTechnicians(config.DB, merchant)
		if err != nil {
			return nil, err
		}
	}

	targetDate, _ := time.ParseInLocation("2006-01-02", date, loc)
	publishedRows, err := loadPublishedSchedulePublishings(config.DB, merchantID, targetDate)
	if err != nil {
		return nil, err
	}
	intervals, ok := buildPublishedScheduleIntervals(publishedRows)
	if !ok {
		intervals, ok = getMerchantBusinessIntervalsForDate(merchant, targetDate)
	}
	if !ok {
		return nil, apiErr{status: http.StatusBadRequest, msg: "商户未设置营业时间"}
	}

	var allSlots []string
	granularity := merchantAppointmentSlotGranularityMinutes(&merchant)
	for _, it := range intervals {
		latestStart := it.End.Add(-time.Duration(occupiedMinutes) * time.Minute)
		for t := it.Start; !t.After(latestStart); t = t.Add(time.Duration(granularity) * time.Minute) {
			allSlots = append(allSlots, t.Format("2006-01-02 15:04:05"))
		}
	}

	allTechIDs := make([]uint, 0)
	if len(technicians) > 0 {
		for _, t := range technicians {
			allTechIDs = append(allTechIDs, t.ID)
		}
	}

	timeSlots := make([]rankedAppointmentTimeSlot, 0, len(allSlots))
	for _, slot := range allSlots {
		slotTime, _ := time.ParseInLocation("2006-01-02 15:04:05", slot, loc)
		slotEnd := slotTime.Add(time.Duration(occupiedMinutes) * time.Minute)
		available := true
		userName := ""
		availableTechIDs := make([]uint, 0)
		candidates := make([]appointmentTechnicianCandidate, 0, len(technicians))
		placementScore := int(^uint(0) >> 1)

		if available && len(allTechIDs) > 0 {
			for _, tech := range technicians {
				if !slotWithinPublishedSchedule(publishedRows, slotTime, slotEnd, &tech.ID) {
					candidates = append(candidates, appointmentTechnicianCandidate{
						TechnicianID:         tech.ID,
						AvailabilityState:    string(appointmentAvailabilityUnavailable),
						PredictedWaitMinutes: 0,
						AvailabilityReason:   "未在已发布排班内",
					})
					continue
				}
				availability, err := evaluateBookingTechnicianAvailability(config.DB, merchant, tech.ID, slotTime, occupiedMinutes, excludeAppointmentID)
				if err != nil {
					log.Printf("evaluate booking availability failed: merchant=%d tech=%d slot=%s err=%v", merchant.ID, tech.ID, slot, err)
					continue
				}
				fragment, err := evaluateAppointmentFragmentImpact(config.DB, merchant, targetDate, slotTime, occupiedMinutes, &tech.ID, excludeAppointmentID)
				if err != nil {
					log.Printf("evaluate booking fragments failed: merchant=%d tech=%d slot=%s err=%v", merchant.ID, tech.ID, slot, err)
					continue
				}
				reason := availability.Reason
				state := availability.State
				if fragment.CreatesFragment {
					state = appointmentAvailabilityUnavailable
					reason = fragment.Reason
				}
				if state != appointmentAvailabilityUnavailable && fragment.PlacementScore < placementScore {
					placementScore = fragment.PlacementScore
				}
				candidates = append(candidates, appointmentTechnicianCandidate{
					TechnicianID:         tech.ID,
					AvailabilityState:    string(state),
					PredictedWaitMinutes: availability.PredictedWaitMinutes,
					AvailabilityReason:   reason,
				})
				if state != appointmentAvailabilityUnavailable {
					availableTechIDs = append(availableTechIDs, tech.ID)
				}
			}
		}

		if available && len(allTechIDs) == 0 {
			if !slotWithinPublishedSchedule(publishedRows, slotTime, slotEnd, nil) {
				available = false
			}
			conflicted, err := checkNonTechnicianAppointmentOverlap(config.DB, merchantID, slotTime, occupiedMinutes, excludeAppointmentID)
			if err != nil {
				log.Printf("check booking overlap failed: merchant=%d slot=%s err=%v", merchant.ID, slot, err)
				continue
			}
			if conflicted {
				available = false
			}
			if available {
				fragment, err := evaluateAppointmentFragmentImpact(config.DB, merchant, targetDate, slotTime, occupiedMinutes, nil, excludeAppointmentID)
				if err != nil {
					log.Printf("evaluate booking fragments failed: merchant=%d slot=%s err=%v", merchant.ID, slot, err)
					continue
				}
				if fragment.CreatesFragment {
					available = false
				} else {
					placementScore = fragment.PlacementScore
				}
			}
		}

		if available && (len(allTechIDs) == 0 || len(availableTechIDs) > 0) {
			if placementScore == int(^uint(0)>>1) {
				placementScore = 0
			}
			timeSlots = append(timeSlots, rankedAppointmentTimeSlot{
				appointmentTimeSlot: appointmentTimeSlot{
					Time:                 slot,
					Available:            true,
					UserName:             userName,
					TechnicianIDs:        availableTechIDs,
					TechnicianCandidates: candidates,
					PlacementScoreValue:  placementScore,
				},
				PlacementScore: placementScore,
				SlotTime:       slotTime,
			})
		}
	}

	sort.SliceStable(timeSlots, func(i, j int) bool {
		if timeSlots[i].PlacementScore != timeSlots[j].PlacementScore {
			return timeSlots[i].PlacementScore < timeSlots[j].PlacementScore
		}
		return timeSlots[i].SlotTime.Before(timeSlots[j].SlotTime)
	})

	orderedSlots := make([]appointmentTimeSlot, 0, len(timeSlots))
	for _, slot := range timeSlots {
		orderedSlots = append(orderedSlots, slot.appointmentTimeSlot)
	}

	return gin.H{
		"date":            date,
		"service_minutes": serviceMinutes,
		"time_slots":      orderedSlots,
		"technicians":     technicians,
	}, nil
}

func appointmentCompensationPreload(db *gorm.DB) *gorm.DB {
	return db.Order("id DESC")
}

func appointmentRescheduleRequestPreload(db *gorm.DB) *gorm.DB {
	// 改签提议在用户端/商户端都要展示改签后的客服信息，因此这里一并预加载新客服资料，
	// 避免前端为了一个昵称和账号额外再发请求组装展示文本。
	return db.Preload("NewTechnician").Order("id DESC")
}

func appointmentCancelRequestPreload(db *gorm.DB) *gorm.DB {
	return db.Order("id DESC")
}

func appointmentForceMajeureReliefRequestPreload(db *gorm.DB) *gorm.DB {
	return db.Order("id DESC")
}

func computeAppointmentCurrentPlacementScore(tx *gorm.DB, merchant models.Merchant, appt models.Appointment) (int, error) {
	if tx == nil || appt.AppointmentTime == nil {
		return 0, nil
	}
	occupiedMinutes := getAppointmentOccupiedMinutes(appt.MerchantID, appt)
	decision, err := validateAppointmentPlacementRules(tx, merchant, appt.AppointmentTime.In(appointmentLocation()), *appt.AppointmentTime, occupiedMinutes, appt.TechnicianID, appt.ID)
	if err != nil {
		return 0, err
	}
	return decision.PlacementScore, nil
}

func compareAppointmentPlacementScore(currentScore, targetScore int) (string, string) {
	switch {
	case targetScore < currentScore:
		return "better", "更优于当前"
	case targetScore == currentScore:
		return "not_worse", "不劣于当前"
	default:
		return "worse", ""
	}
}

func filterAppointmentRescheduleSlotsByComparison(slots []appointmentTimeSlot, eligibility appointmentRescheduleEligibility, currentScore int) []appointmentTimeSlot {
	if len(slots) == 0 {
		return slots
	}
	betterOnly := eligibility.RuleMode != appointmentRescheduleTodayOrTomorrow
	filtered := make([]appointmentTimeSlot, 0, len(slots))
	for _, slot := range slots {
		kind, label := compareAppointmentPlacementScore(currentScore, slot.PlacementScoreValue)
		slot.ComparisonKind = kind
		slot.ComparisonLabel = label
		if kind == "worse" {
			continue
		}
		if betterOnly && kind != "better" {
			continue
		}
		filtered = append(filtered, slot)
	}
	return filtered
}

func buildAppointmentRecommendationSlots(slots []appointmentTimeSlot, eligibility appointmentRescheduleEligibility, currentScore int) []appointmentTimeSlot {
	filtered := filterAppointmentRescheduleSlotsByComparison(slots, eligibility, currentScore)
	for i := range filtered {
		switch filtered[i].ComparisonKind {
		case "better":
			filtered[i].RecommendationReason = "该时段排布优于当前预约，适合优先推荐改签"
		case "not_worse":
			filtered[i].RecommendationReason = "该时段排布不劣于当前预约，可作为备选推荐"
		}
	}
	return filtered
}

func hydrateAppointmentRelations(tx *gorm.DB, appt *models.Appointment) error {
	if tx == nil || appt == nil || appt.ID == 0 {
		return nil
	}
	// 预约主记录统一通过 raw loader 读取，避免 sqlite 在测试环境里把 datetime 字符串直接扫进 *time.Time。
	// 这里单独补齐关联关系，保证接口返回结构与原来保持一致。
	if err := tx.Where("id = ?", appt.UserID).First(&appt.User).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err := tx.Where("id = ?", appt.CardID).First(&appt.Card).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err := tx.Where("id = ?", appt.MerchantID).First(&appt.Merchant).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if appt.ProjectID != nil && *appt.ProjectID > 0 {
		var project models.MerchantProject
		if err := tx.Where("id = ?", *appt.ProjectID).First(&project).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		} else {
			appt.Project = &project
		}
	}
	if appt.TechnicianID != nil && *appt.TechnicianID > 0 {
		var technician models.Technician
		if err := tx.Preload("ServiceRole").Where("id = ?", *appt.TechnicianID).First(&technician).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		} else {
			appt.Technician = &technician
		}
	}
	var compensations []models.AppointmentCompensation
	if err := tx.Scopes(appointmentCompensationPreload).Where("appointment_id = ?", appt.ID).Find(&compensations).Error; err != nil {
		return err
	}
	appt.Compensations = compensations
	var requests []models.AppointmentRescheduleRequest
	if err := tx.Scopes(appointmentRescheduleRequestPreload).Where("appointment_id = ?", appt.ID).Find(&requests).Error; err != nil {
		return err
	}
	appt.RescheduleRequests = requests
	var cancelRequests []models.AppointmentCancelRequest
	if err := tx.Scopes(appointmentCancelRequestPreload).Where("appointment_id = ?", appt.ID).Find(&cancelRequests).Error; err != nil {
		return err
	}
	appt.CancelRequests = cancelRequests
	var reliefRequests []models.ForceMajeureReliefRequest
	if err := tx.Scopes(appointmentForceMajeureReliefRequestPreload).Where("appointment_id = ?", appt.ID).Find(&reliefRequests).Error; err != nil {
		return err
	}
	appt.ForceMajeureReliefRequests = reliefRequests
	return nil
}

func latestPendingRescheduleRequest(appt *models.Appointment) *models.AppointmentRescheduleRequest {
	if appt == nil {
		return nil
	}
	for i := range appt.RescheduleRequests {
		status := strings.TrimSpace(appt.RescheduleRequests[i].Status)
		if status == "pending_user" || status == "pending_merchant" {
			return &appt.RescheduleRequests[i]
		}
	}
	return nil
}

func loadAppointmentRescheduleRequestByID(tx *gorm.DB, requestID uint) (*models.AppointmentRescheduleRequest, error) {
	if tx == nil || requestID == 0 {
		return nil, nil
	}
	rows, err := tx.Table("appointment_reschedule_requests").
		Select("id, appointment_id, merchant_id, user_id, current_project_id, current_technician_id, current_appointment_time, new_project_id, new_technician_id, new_appointment_time, status, reason, proposed_by_type, proposed_by_id, confirmed_by_type, confirmed_by_id, confirmed_at, result_appointment_id, created_at").
		Where("id = ?", requestID).
		Limit(1).
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, nil
	}

	var (
		req                       models.AppointmentRescheduleRequest
		currentProjectIDRaw       interface{}
		currentTechnicianIDRaw    interface{}
		currentAppointmentTimeRaw interface{}
		newProjectIDRaw           interface{}
		newTechnicianIDRaw        interface{}
		newTimeRaw                interface{}
		proposedByIDRaw           interface{}
		confirmedByIDRaw          interface{}
		confirmedAtRaw            interface{}
		resultAppointmentIDRaw    interface{}
		createdAtRaw              interface{}
	)
	if err := rows.Scan(
		&req.ID,
		&req.AppointmentID,
		&req.MerchantID,
		&req.UserID,
		&currentProjectIDRaw,
		&currentTechnicianIDRaw,
		&currentAppointmentTimeRaw,
		&newProjectIDRaw,
		&newTechnicianIDRaw,
		&newTimeRaw,
		&req.Status,
		&req.Reason,
		&req.ProposedByType,
		&proposedByIDRaw,
		&req.ConfirmedByType,
		&confirmedByIDRaw,
		&confirmedAtRaw,
		&resultAppointmentIDRaw,
		&createdAtRaw,
	); err != nil {
		return nil, err
	}
	if v, ok := gormValueToUint(currentProjectIDRaw); ok {
		req.CurrentProjectID = &v
	}
	if v, ok := gormValueToUint(currentTechnicianIDRaw); ok {
		req.CurrentTechnicianID = &v
	}
	if v, ok := parseDBTimeValue(currentAppointmentTimeRaw); ok {
		req.CurrentAppointmentTime = &v
	}
	if v, ok := gormValueToUint(newProjectIDRaw); ok {
		req.NewProjectID = &v
	}
	if v, ok := gormValueToUint(newTechnicianIDRaw); ok {
		req.NewTechnicianID = &v
	}
	if v, ok := parseDBTimeValue(newTimeRaw); ok {
		req.NewAppointmentTime = &v
	}
	if v, ok := gormValueToUint(proposedByIDRaw); ok {
		req.ProposedByID = &v
	}
	if v, ok := gormValueToUint(confirmedByIDRaw); ok {
		req.ConfirmedByID = &v
	}
	if v, ok := parseDBTimeValue(confirmedAtRaw); ok {
		req.ConfirmedAt = &v
	}
	if v, ok := gormValueToUint(resultAppointmentIDRaw); ok {
		req.ResultAppointmentID = &v
	}
	if v, ok := parseDBTimeValue(createdAtRaw); ok {
		req.CreatedAt = &v
	}
	return &req, nil
}

func loadAppointmentCancelRequestByID(tx *gorm.DB, requestID uint) (*models.AppointmentCancelRequest, error) {
	if tx == nil || requestID == 0 {
		return nil, nil
	}
	rows, err := tx.Table("appointment_cancel_requests").
		Select("id, appointment_id, merchant_id, user_id, status, reason, proposed_by_type, proposed_by_id, confirmed_by_type, confirmed_by_id, confirmed_at, objection_note, created_at").
		Where("id = ?", requestID).
		Limit(1).
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, nil
	}
	var (
		req              models.AppointmentCancelRequest
		proposedByIDRaw  interface{}
		confirmedByIDRaw interface{}
		confirmedAtRaw   interface{}
		createdAtRaw     interface{}
	)
	if err := rows.Scan(&req.ID, &req.AppointmentID, &req.MerchantID, &req.UserID, &req.Status, &req.Reason, &req.ProposedByType, &proposedByIDRaw, &req.ConfirmedByType, &confirmedByIDRaw, &confirmedAtRaw, &req.ObjectionNote, &createdAtRaw); err != nil {
		return nil, err
	}
	if v, ok := gormValueToUint(proposedByIDRaw); ok {
		req.ProposedByID = &v
	}
	if v, ok := gormValueToUint(confirmedByIDRaw); ok {
		req.ConfirmedByID = &v
	}
	if v, ok := parseDBTimeValue(confirmedAtRaw); ok {
		req.ConfirmedAt = &v
	}
	if v, ok := parseDBTimeValue(createdAtRaw); ok {
		req.CreatedAt = &v
	}
	return &req, nil
}

func loadForceMajeureReliefRequestByID(tx *gorm.DB, requestID uint) (*models.ForceMajeureReliefRequest, error) {
	if tx == nil || requestID == 0 {
		return nil, nil
	}
	rows, err := tx.Table("force_majeure_relief_requests").
		Select("id, appointment_id, merchant_id, user_id, proposed_by_type, proposed_by_id, reason, evidence_note, status, confirmed_by_type, confirmed_by_id, confirmed_at, created_at").
		Where("id = ?", requestID).
		Limit(1).
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, nil
	}
	var (
		req              models.ForceMajeureReliefRequest
		proposedByIDRaw  interface{}
		confirmedByIDRaw interface{}
		confirmedAtRaw   interface{}
		createdAtRaw     interface{}
	)
	if err := rows.Scan(&req.ID, &req.AppointmentID, &req.MerchantID, &req.UserID, &req.ProposedByType, &proposedByIDRaw, &req.Reason, &req.EvidenceNote, &req.Status, &req.ConfirmedByType, &confirmedByIDRaw, &confirmedAtRaw, &createdAtRaw); err != nil {
		return nil, err
	}
	if v, ok := gormValueToUint(proposedByIDRaw); ok {
		req.ProposedByID = &v
	}
	if v, ok := gormValueToUint(confirmedByIDRaw); ok {
		req.ConfirmedByID = &v
	}
	if v, ok := parseDBTimeValue(confirmedAtRaw); ok {
		req.ConfirmedAt = &v
	}
	if v, ok := parseDBTimeValue(createdAtRaw); ok {
		req.CreatedAt = &v
	}
	return &req, nil
}

func loadAppointmentSettlementByID(tx *gorm.DB, settlementID uint) (*models.AppointmentSettlement, error) {
	if tx == nil || settlementID == 0 {
		return nil, nil
	}
	var settlement models.AppointmentSettlement
	if err := tx.Where("id = ?", settlementID).First(&settlement).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &settlement, nil
}

func hasPendingRescheduleRequest(tx *gorm.DB, appointmentID uint) (bool, error) {
	if tx == nil || appointmentID == 0 {
		return false, nil
	}
	var count int64
	if err := tx.Table("appointment_reschedule_requests").
		Where("appointment_id = ? AND status IN ?", appointmentID, []string{"pending_user", "pending_merchant"}).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func hasPendingCancelRequest(tx *gorm.DB, appointmentID uint) (bool, error) {
	if tx == nil || appointmentID == 0 {
		return false, nil
	}
	var count int64
	if err := tx.Table("appointment_cancel_requests").
		Where("appointment_id = ? AND status = ?", appointmentID, "pending_user").
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func hasPendingForceMajeureReliefRequest(tx *gorm.DB, appointmentID uint) (bool, error) {
	if tx == nil || appointmentID == 0 {
		return false, nil
	}
	var count int64
	if err := tx.Table("force_majeure_relief_requests").
		Where("appointment_id = ? AND status = ?", appointmentID, "pending").
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func forceMajeureEligibleCompensationSource(sourceType string) bool {
	switch strings.TrimSpace(sourceType) {
	case "merchant_breach", "merchant_failure_offset":
		return true
	default:
		return false
	}
}

func listActiveForceMajeureEligibleCompensations(tx *gorm.DB, appointmentID uint) ([]models.AppointmentCompensation, error) {
	if tx == nil || appointmentID == 0 {
		return nil, nil
	}
	var compensations []models.AppointmentCompensation
	err := tx.Where("appointment_id = ? AND status = ? AND source_type IN ?", appointmentID, "applied", []string{"merchant_breach", "merchant_failure_offset"}).
		Order("id ASC").
		Find(&compensations).Error
	return compensations, err
}

func reverseForceMajeureCompensation(tx *gorm.DB, compensation *models.AppointmentCompensation) error {
	if tx == nil || compensation == nil || compensation.ID == 0 {
		return nil
	}
	switch strings.TrimSpace(compensation.Type) {
	case "extra_times":
		if compensation.Value <= 0 {
			return nil
		}
		var card models.Card
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", compensation.CardID).First(&card).Error; err != nil {
			return err
		}
		if card.TotalTimes < compensation.Value || card.RemainTimes < compensation.Value {
			return apiErr{status: http.StatusBadRequest, msg: "补偿已被部分使用，当前不能直接撤销"}
		}
		return tx.Model(&models.Card{}).Where("id = ?", card.ID).Updates(map[string]interface{}{
			"total_times":  gorm.Expr("total_times - ?", compensation.Value),
			"remain_times": gorm.Expr("remain_times - ?", compensation.Value),
		}).Error
	default:
		return nil
	}
}

func getAppointmentActor(c *gin.Context) (string, *uint) {
	authType := strings.TrimSpace(c.GetString("auth_type"))
	switch authType {
	case "merchant":
		if merchantID, ok := c.Get("merchant_id"); ok {
			if id, ok := merchantID.(uint); ok && id > 0 {
				return "merchant", &id
			}
		}
	case "staff":
		if technicianID, ok := c.Get("technician_id"); ok {
			if id, ok := technicianID.(uint); ok && id > 0 {
				return "staff", &id
			}
		}
	case "user":
		if userID, ok := c.Get("user_id"); ok {
			if id, ok := userID.(uint); ok && id > 0 {
				return "user", &id
			}
		}
	}
	return authType, nil
}

func appointmentAllowsReschedule(status string) bool {
	switch normalizeAppointmentStatus(status) {
	case "pending", "confirmed", "failed":
		return true
	default:
		return false
	}
}

func appendAppointmentResolutionNote(current, next string) string {
	left := strings.TrimSpace(current)
	right := strings.TrimSpace(next)
	if left == "" {
		return right
	}
	if right == "" {
		return left
	}
	return left + "\n" + right
}

func initializeAppointmentSettlement(tx *gorm.DB, appt *models.Appointment, latestReason string) (*models.AppointmentSettlement, error) {
	if tx == nil || appt == nil || appt.ID == 0 {
		return nil, nil
	}
	settlement := models.AppointmentSettlement{
		AppointmentID:                   appt.ID,
		BookingRootID:                   appt.BookingRootID,
		MerchantID:                      appt.MerchantID,
		UserID:                          appt.UserID,
		CardID:                          appt.CardID,
		ProjectID:                       appt.ProjectID,
		Status:                          "pending",
		AssetMode:                       "deduct",
		SettlementStatusSnapshot:        "pending",
		MerchantBreachPending:           appt.MerchantBreachPending,
		BreachDecisionAt:                appt.BreachDecisionAt,
		LiabilityLevel:                  appt.LiabilityLevel,
		SalarySettlementReferenceStatus: appt.SalarySettlementReferenceStatus,
		LatestReason:                    strings.TrimSpace(latestReason),
	}
	if err := tx.Create(&settlement).Error; err != nil {
		return nil, err
	}
	if err := tx.Model(&models.Appointment{}).Where("id = ?", appt.ID).Updates(map[string]interface{}{
		"appointment_settlement_id":  settlement.ID,
		"settlement_status_snapshot": settlement.SettlementStatusSnapshot,
	}).Error; err != nil {
		return nil, err
	}
	appt.AppointmentSettlementID = &settlement.ID
	appt.SettlementStatusSnapshot = settlement.SettlementStatusSnapshot
	return &settlement, nil
}

func updateAppointmentSettlementSnapshot(tx *gorm.DB, appt *models.Appointment, status string, latestReason string) error {
	if tx == nil || appt == nil || appt.ID == 0 {
		return nil
	}
	cleanStatus := strings.TrimSpace(status)
	cleanReason := strings.TrimSpace(latestReason)
	if appt.AppointmentSettlementID != nil && *appt.AppointmentSettlementID > 0 {
		updates := map[string]interface{}{
			"status":                     cleanStatus,
			"settlement_status_snapshot": cleanStatus,
		}
		if cleanReason != "" {
			updates["latest_reason"] = cleanReason
		}
		if err := tx.Model(&models.AppointmentSettlement{}).Where("id = ?", *appt.AppointmentSettlementID).Updates(updates).Error; err != nil {
			return err
		}
	}
	if err := tx.Model(&models.Appointment{}).Where("id = ?", appt.ID).Update("settlement_status_snapshot", cleanStatus).Error; err != nil {
		return err
	}
	appt.SettlementStatusSnapshot = cleanStatus
	return nil
}

func transferAppointmentSettlement(tx *gorm.DB, oldAppt *models.Appointment, newAppt *models.Appointment, latestReason string) error {
	if tx == nil || oldAppt == nil || newAppt == nil || oldAppt.ID == 0 || newAppt.ID == 0 {
		return nil
	}
	if _, err := initializeAppointmentSettlement(tx, newAppt, latestReason); err != nil {
		return err
	}
	if oldAppt.AppointmentSettlementID == nil || *oldAppt.AppointmentSettlementID == 0 || newAppt.AppointmentSettlementID == nil || *newAppt.AppointmentSettlementID == 0 {
		return nil
	}
	if err := tx.Model(&models.AppointmentSettlement{}).Where("id = ?", *oldAppt.AppointmentSettlementID).Updates(map[string]interface{}{
		"status":                       "transferred",
		"settlement_status_snapshot":   "transferred",
		"transferred_to_settlement_id": *newAppt.AppointmentSettlementID,
		"latest_reason":                strings.TrimSpace(latestReason),
	}).Error; err != nil {
		return err
	}
	if err := tx.Model(&models.Appointment{}).Where("id = ?", oldAppt.ID).Update("settlement_status_snapshot", "transferred").Error; err != nil {
		return err
	}
	oldAppt.SettlementStatusSnapshot = "transferred"
	return nil
}

func cancelUnstartedAppointmentArrival(tx *gorm.DB, appt *models.Appointment, now time.Time) error {
	if appt == nil || appt.UsageID == nil || appt.ServiceSessionID == nil {
		return apiErr{status: http.StatusBadRequest, msg: "当前预约未生成可撤回的到店记录"}
	}

	var usage models.Usage
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", *appt.UsageID).First(&usage).Error; err != nil {
		return err
	}
	if usage.Status != "in_progress" {
		return apiErr{status: http.StatusBadRequest, msg: "当前预约已进入不可改签阶段"}
	}

	var session models.ServiceSession
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", *appt.ServiceSessionID).First(&session).Error; err != nil {
		return err
	}
	baseStatus := models.NormalizeSessionStatus(session.Status)
	switch baseStatus {
	case "room_selecting", "room_locked", "staff_selecting", "appointment_waiting", "start_pending", "delay_pending":
	default:
		return apiErr{status: http.StatusBadRequest, msg: "当前预约已开始服务，不能改签"}
	}

	var card models.Card
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&card, appt.CardID).Error; err != nil {
		return err
	}
	used := usage.UsedTimes
	if used <= 0 {
		used = 1
	}
	if err := tx.Model(&models.Card{}).Where("id = ?", card.ID).Updates(map[string]interface{}{
		"remain_times": gorm.Expr("remain_times + ?", used),
		"used_times":   gorm.Expr("CASE WHEN used_times >= ? THEN used_times - ? ELSE 0 END", used, used),
	}).Error; err != nil {
		return err
	}
	if err := tx.Model(&models.Usage{}).Where("id = ? AND status = ?", usage.ID, "in_progress").Updates(map[string]interface{}{
		"status":        "failed",
		"finished_at":   &now,
		"technician_id": nil,
	}).Error; err != nil {
		return err
	}

	updates := map[string]interface{}{
		"status":                  models.ApplyStatusPrefix(session.Status, "canceled"),
		"technician_id":           nil,
		"room_id":                 nil,
		"room_locked_at":          nil,
		"room_select_deadline_at": nil,
		"start_confirmed_at":      nil,
		"predicted_ready_at":      nil,
	}
	if err := tx.Model(&models.ServiceSession{}).
		Where("id = ? AND status NOT IN ?", session.ID, models.ExpandStatusesWithKnownPrefixes([]string{"finished", "canceled"})).
		Updates(updates).Error; err != nil {
		return err
	}
	return nil
}

func autoCloseCrossDayUnfinishedAppointment(tx *gorm.DB, appt *models.Appointment, now time.Time, actorType string, actorID *uint, resolutionReason string) error {
	if tx == nil || appt == nil || appt.ID == 0 {
		return nil
	}
	if !appointmentIsCrossDayUnfinished(appt, now) || !appointmentHasArrivalEvidence(appt) {
		return nil
	}

	currentPtr, err := loadAppointmentByID(tx.Clauses(clause.Locking{Strength: "UPDATE"}), appt.ID)
	if err != nil {
		return err
	}
	if currentPtr == nil {
		return gorm.ErrRecordNotFound
	}
	current := *currentPtr
	if !appointmentIsCrossDayUnfinished(&current, now) || !appointmentHasArrivalEvidence(&current) {
		*appt = current
		return nil
	}

	if current.UsageID != nil && current.ServiceSessionID != nil {
		if err := cancelUnstartedAppointmentArrival(tx, &current, now); err != nil {
			return err
		}
	}
	if _, err := initializeAppointmentSettlement(tx, &current, "service_unclosed_cross_day"); err != nil {
		return err
	}

	resolutionNote := appendAppointmentResolutionNote(current.ResolutionNote, resolutionReason)
	updates := map[string]interface{}{
		"status":                             "failed",
		"failed_at":                          &now,
		"failed_reason":                      "service_unclosed_cross_day",
		"disruption_status":                  "closed",
		"disruption_reason":                  "service_unclosed_cross_day",
		"liability_level":                    "merchant",
		"merchant_breach_pending":            false,
		"breach_decision_at":                 &now,
		"salary_settlement_reference_status": "refund",
		"closed_reason":                      "auto_exception_closed",
		"closed_by_type":                     actorType,
		"closed_by_id":                       actorID,
		"resolution_note":                    resolutionNote,
		"settlement_status_snapshot":         "pending",
	}
	if current.ActualArrivedAt == nil && current.ArrivedAt != nil {
		updates["actual_arrived_at"] = current.ArrivedAt
	}
	if err := tx.Model(&models.Appointment{}).Where("id = ? AND status = ?", current.ID, current.Status).Updates(updates).Error; err != nil {
		return err
	}
	if current.AppointmentSettlementID != nil && *current.AppointmentSettlementID > 0 {
		if err := tx.Model(&models.AppointmentSettlement{}).Where("id = ?", *current.AppointmentSettlementID).Updates(map[string]interface{}{
			"status":                             "pending",
			"settlement_status_snapshot":         "pending",
			"merchant_breach_pending":            false,
			"breach_decision_at":                 &now,
			"liability_level":                    "merchant",
			"salary_settlement_reference_status": "refund",
			"latest_reason":                      "service_unclosed_cross_day",
		}).Error; err != nil {
			return err
		}
	}

	current.Status = "failed"
	current.FailedAt = &now
	current.FailedReason = "service_unclosed_cross_day"
	current.DisruptionStatus = "closed"
	current.DisruptionReason = "service_unclosed_cross_day"
	current.LiabilityLevel = "merchant"
	current.MerchantBreachPending = false
	current.BreachDecisionAt = &now
	current.SalarySettlementReferenceStatus = "refund"
	current.ClosedReason = "auto_exception_closed"
	current.ClosedByType = actorType
	current.ClosedByID = actorID
	current.ResolutionNote = resolutionNote
	current.SettlementStatusSnapshot = "pending"
	if current.ActualArrivedAt == nil && current.ArrivedAt != nil {
		current.ActualArrivedAt = current.ArrivedAt
	}
	*appt = current
	return nil
}

type CrossDayUnfinishedAppointmentRepairResult struct {
	AppointmentID uint   `json:"appointment_id"`
	Status        string `json:"status"`
	Reason        string `json:"reason"`
}

func RepairCrossDayUnfinishedAppointments(db *gorm.DB, now time.Time, limit int, dryRun bool, resolutionReason string) ([]CrossDayUnfinishedAppointmentRepairResult, error) {
	if db == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 200
	}
	if strings.TrimSpace(resolutionReason) == "" {
		resolutionReason = "系统批量巡检自动结案：跨日未开始服务"
	}

	loc := appointmentLocation()
	cutoff := now.In(loc).Truncate(24 * time.Hour)
	var candidateIDs []uint
	if err := db.Model(&models.Appointment{}).
		Select("id").
		Where("status = ? AND appointment_time IS NOT NULL AND appointment_time < ? AND actual_start_at IS NULL", "arrived", cutoff).
		Where("(actual_arrived_at IS NOT NULL OR arrived_at IS NOT NULL OR usage_id IS NOT NULL OR service_session_id IS NOT NULL)").
		Order("appointment_time ASC").
		Limit(limit).
		Find(&candidateIDs).Error; err != nil {
		return nil, err
	}

	results := make([]CrossDayUnfinishedAppointmentRepairResult, 0, len(candidateIDs))
	for _, appointmentID := range candidateIDs {
		apptPtr, err := loadAppointmentByID(db, appointmentID)
		if err != nil {
			return results, err
		}
		if apptPtr == nil {
			continue
		}
		appt := *apptPtr
		if !appointmentIsCrossDayUnfinished(&appt, now) || !appointmentHasArrivalEvidence(&appt) {
			continue
		}
		if dryRun {
			results = append(results, CrossDayUnfinishedAppointmentRepairResult{
				AppointmentID: appt.ID,
				Status:        "dry_run",
				Reason:        "eligible_for_auto_close",
			})
			continue
		}
		if err := db.Transaction(func(tx *gorm.DB) error {
			return autoCloseCrossDayUnfinishedAppointment(tx, &appt, now, "system", nil, resolutionReason)
		}); err != nil {
			results = append(results, CrossDayUnfinishedAppointmentRepairResult{
				AppointmentID: appt.ID,
				Status:        "error",
				Reason:        err.Error(),
			})
			continue
		}
		results = append(results, CrossDayUnfinishedAppointmentRepairResult{
			AppointmentID: appt.ID,
			Status:        "closed",
			Reason:        "service_unclosed_cross_day",
		})
	}
	return results, nil
}

func normalizeAppointmentForRead(appt *models.Appointment, now time.Time) error {
	if appt == nil {
		return nil
	}
	appt.Status = normalizeAppointmentStatus(appt.Status)
	if appointmentIsCrossDayUnfinished(appt, now) && appointmentHasArrivalEvidence(appt) {
		if err := config.DB.Transaction(func(tx *gorm.DB) error {
			return autoCloseCrossDayUnfinishedAppointment(tx, appt, now, "system", nil, "系统自动结案：读取时发现跨日未开始服务")
		}); err != nil {
			return err
		}
	}
	appt.Status = normalizeAppointmentStatus(appt.Status)
	return nil
}

func compensationDisplayText(t string, value int, remark string) string {
	switch t {
	case "extra_times":
		return fmt.Sprintf("补偿次数 +%d", value)
	case "extend_minutes":
		return fmt.Sprintf("补偿时长 +%d分钟", value)
	case "discount_note":
		if strings.TrimSpace(remark) != "" {
			return "补偿优惠：" + strings.TrimSpace(remark)
		}
		return "补偿优惠"
	default:
		return strings.TrimSpace(remark)
	}
}

func cancelAppointmentWithTime(appointment *models.Appointment, canceledAt time.Time) error {
	if appointment == nil || appointment.ID == 0 {
		return nil
	}
	return config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Appointment{}).Where("id = ?", appointment.ID).Updates(map[string]interface{}{
			"status":                     "canceled",
			"canceled_at":                canceledAt,
			"settlement_status_snapshot": "refunded",
		}).Error; err != nil {
			return err
		}
		appointment.Status = "canceled"
		appointment.CanceledAt = &canceledAt
		return updateAppointmentSettlementSnapshot(tx, appointment, "refunded", "appointment_canceled")
	})
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

func getMerchantAppointmentPermissionState(c *gin.Context) (canView bool, canManage bool, err error) {
	canView, err = hasMerchantPermissionInHandler(c, "merchant.appointment.view")
	if err != nil {
		return false, false, err
	}
	canManage, err = hasMerchantPermissionInHandler(c, "merchant.appointment.manage")
	if err != nil {
		return false, false, err
	}
	return canView, canManage, nil
}

func getCurrentTechnicianID(c *gin.Context) uint {
	technicianIDAny, ok := c.Get("technician_id")
	if !ok {
		return 0
	}
	technicianID, ok := technicianIDAny.(uint)
	if !ok {
		return 0
	}
	return technicianID
}

func requireMerchantAppointmentAccess(c *gin.Context) (canManage bool, technicianID uint, ok bool) {
	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)
	if authType != "merchant" && authType != "staff" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return false, 0, false
	}

	canView, canManage, err := getMerchantAppointmentPermissionState(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "权限校验失败"})
		return false, 0, false
	}
	if !canView && !canManage {
		c.JSON(http.StatusForbidden, gin.H{"error": "无预约权限"})
		return false, 0, false
	}

	if authType == "staff" {
		technicianID = getCurrentTechnicianID(c)
		if technicianID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
			return false, 0, false
		}
	}

	return canManage, technicianID, true
}

func checkMerchantAppointmentOwnership(c *gin.Context, appointment models.Appointment) (canManage bool, technicianID uint, ok bool) {
	merchantIDAny, exists := c.Get("merchant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return false, 0, false
	}
	merchantID, okCast := merchantIDAny.(uint)
	if !okCast || merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return false, 0, false
	}
	if appointment.MerchantID != merchantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限：不属于您的商户"})
		return false, 0, false
	}

	canManage, technicianID, ok = requireMerchantAppointmentAccess(c)
	if !ok {
		return false, 0, false
	}

	if technicianID > 0 && !canManage {
		if appointment.TechnicianID == nil || *appointment.TechnicianID != technicianID {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权限：只能操作分配给自己的预约"})
			return false, 0, false
		}
	}

	return canManage, technicianID, true
}

func GetMerchantAppointments(c *gin.Context) {
	merchantID, ok := ensureMerchantScope(c, "id")
	if !ok {
		return
	}

	canManage, technicianID, ok := requireMerchantAppointmentAccess(c)
	if !ok {
		return
	}
	status := c.Query("status")

	var appointments []models.Appointment
	query := config.DB.Preload("User").Preload("Card").Preload("Merchant").Preload("Project").Preload("Technician").Preload("Technician.ServiceRole").Preload("Compensations", appointmentCompensationPreload).Preload("RescheduleRequests", appointmentRescheduleRequestPreload).Preload("CancelRequests", appointmentCancelRequestPreload).Preload("ForceMajeureReliefRequests", appointmentForceMajeureReliefRequestPreload).Where("merchant_id = ?", merchantID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if technicianID > 0 && !canManage {
		query = query.Where("(technician_id = ? OR technician_id IS NULL)", technicianID)
	}

	query.Order("appointment_time ASC").Find(&appointments)
	now := time.Now().In(appointmentLocation())
	for i := range appointments {
		if err := normalizeAppointmentForRead(&appointments[i], now); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "自动收口跨日异常预约失败"})
			return
		}
		decorateAppointmentDisplay(&appointments[i], now)
	}
	c.JSON(http.StatusOK, gin.H{"data": appointments})
}

func GetUserAppointments(c *gin.Context) {
	authUserID, ok := mustUserID(c)
	if !ok {
		return
	}
	userIDStr := strings.TrimSpace(c.Param("id"))
	uid, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil || uint(uid) != authUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问其他用户数据"})
		return
	}
	var appointments []models.Appointment
	config.DB.Preload("Merchant").Preload("Technician").Preload("Technician.ServiceRole").Preload("Compensations", appointmentCompensationPreload).Preload("RescheduleRequests", appointmentRescheduleRequestPreload).Preload("CancelRequests", appointmentCancelRequestPreload).Preload("ForceMajeureReliefRequests", appointmentForceMajeureReliefRequestPreload).Where("user_id = ?", authUserID).Order("appointment_time DESC").Find(&appointments)
	now := time.Now().In(appointmentLocation())
	for i := range appointments {
		if err := normalizeAppointmentForRead(&appointments[i], now); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "自动收口跨日异常预约失败"})
			return
		}
		decorateAppointmentDisplay(&appointments[i], now)
	}
	c.JSON(http.StatusOK, gin.H{"data": appointments})
}

func GetCardAppointment(c *gin.Context) {
	authUserID, ok := mustUserID(c)
	if !ok {
		return
	}
	cardID := c.Param("id")

	var card models.Card
	if err := config.DB.First(&card, cardID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "卡片不存在"})
		return
	}
	if card.UserID != authUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问此卡"})
		return
	}

	// 添加调试日志
	log.Printf("查询预约: 卡片ID=%s, 用户ID=%d, 商户ID=%d", cardID, card.UserID, card.MerchantID)

	var appointment models.Appointment
	err := config.DB.Preload("Merchant").Preload("Project").Preload("Technician").Preload("Technician.ServiceRole").Preload("Compensations", appointmentCompensationPreload).Preload("RescheduleRequests", appointmentRescheduleRequestPreload).Preload("CancelRequests", appointmentCancelRequestPreload).Preload("ForceMajeureReliefRequests", appointmentForceMajeureReliefRequestPreload).
		Where("card_id = ? AND merchant_id = ? AND user_id = ? AND status IN ('pending', 'confirmed', 'arrived', 'failed')", card.ID, card.MerchantID, card.UserID).
		Order("CASE WHEN status IN ('pending','confirmed','arrived') THEN 0 ELSE 1 END ASC").
		Order("appointment_time DESC").
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
	if err := normalizeAppointmentForRead(&appointment, now.In(appointmentLocation())); err != nil {
		log.Printf("自动收口跨日异常预约失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "自动收口跨日异常预约失败"})
		return
	}
	decorateAppointmentDisplay(&appointment, now.In(appointmentLocation()))

	// 计算预约队列排队信息（内存队列）
	queueBefore := int64(0)
	if appointment.AppointmentTime != nil {
		date := appointment.AppointmentTime.Format("2006-01-02")
		snap := queue.Default.Snapshot(card.MerchantID, date, queue.QueueTypeAppointment)
		if snap.ByID != nil {
			if tk, ok := snap.ByID[appointment.ID]; ok {
				queueBefore = int64(tk.No - 1)
				if queueBefore < 0 {
					queueBefore = 0
				}
			}
		}
	}

	var merchant models.Merchant
	config.DB.First(&merchant, card.MerchantID)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"appointment":                    appointment,
			"queue_before":                   queueBefore,
			"estimated_minutes":              int(queueBefore) * 30,
			"service_session_id":             appointment.ServiceSessionID,
			"predicted_wait_minutes":         appointment.PredictedWaitMinutes,
			"display_wait_state":             appointment.DisplayWaitState,
			"display_wait_message":           appointment.DisplayWaitMessage,
			"current_estimated_wait_minutes": appointment.CurrentEstimatedWaitMinutes,
			"can_arrive_now":                 canArriveForAppointment(appointment, &merchant, now),
		},
	})
}

func CreateAppointment(c *gin.Context) {
	var input struct {
		CardID          uint   `json:"card_id" binding:"required"`
		MerchantID      uint   `json:"merchant_id" binding:"required"`
		UserID          *uint  `json:"user_id"`
		ProjectID       *uint  `json:"project_id"`
		TechnicianID    *uint  `json:"technician_id"`
		AppointmentTime string `json:"appointment_time" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	authUserID, ok := mustUserID(c)
	if !ok {
		return
	}
	if input.UserID != nil && *input.UserID != authUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "user_id 与登录态不一致"})
		return
	}

	// 校验卡片归属（避免同商户多卡串数据）
	var card models.Card
	if err := config.DB.First(&card, input.CardID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "卡片不存在"})
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
	if card.UserID != authUserID || card.MerchantID != input.MerchantID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的卡片"})
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
		if err := config.DB.Where("id = ? AND merchant_id = ? AND is_active = ? AND bookable_online = ?", *input.ProjectID, input.MerchantID, true, true).First(&project).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目"})
			return
		}
		if project.Duration <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "项目时长无效"})
			return
		}
	}

	// 检查该卡片在该商户是否已有活跃的预约
	var existingAppointment models.Appointment
	err := config.DB.Where("card_id = ? AND merchant_id = ? AND user_id = ? AND status IN ('pending', 'confirmed')",
		input.CardID, input.MerchantID, authUserID).First(&existingAppointment).Error

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

	log.Printf("创建新预约: 用户ID=%d, 商户ID=%d, 时间=%s", authUserID, input.MerchantID, input.AppointmentTime)

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
	serviceMinutes := 30
	serviceGapMinutes := 3
	if input.ProjectID != nil {
		serviceMinutes = project.Duration
		serviceGapMinutes = project.ServiceGapMinutes
	}
	occupiedMinutes := projectBookingOccupiedMinutes(serviceMinutes, serviceGapMinutes)
	reservedEndAt := appointmentTime.Add(time.Duration(serviceMinutes) * time.Minute)
	serviceEnd := appointmentTime.Add(time.Duration(occupiedMinutes) * time.Minute)
	if !isWithinBusinessIntervals(serviceEnd.Add(-1*time.Second), intervals) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "服务必须在营业时间内完成"})
		return
	}
	publishedRows, err := loadPublishedSchedulePublishings(config.DB, merchant.ID, targetDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取排班发布失败"})
		return
	}
	if !slotWithinPublishedSchedule(publishedRows, appointmentTime, serviceEnd, input.TechnicianID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该时间不在已发布可预约排班内"})
		return
	}
	cancelDeadline := time.Date(now.Year(), now.Month(), now.Day(), 16, 0, 0, 0, loc)
	bookingCloseDeadline := time.Date(now.Year(), now.Month(), now.Day(), 18, 0, 0, 0, loc)
	lateArrivalMinServiceMinutes := serviceMinutes / 2
	confirmedAt := now

	appointment := models.Appointment{
		CardID:                       input.CardID,
		MerchantID:                   input.MerchantID,
		UserID:                       authUserID,
		ProjectID:                    input.ProjectID,
		TechnicianID:                 input.TechnicianID,
		AppointmentTime:              &appointmentTime,
		ReservedStartAt:              &appointmentTime,
		ReservedEndAt:                &reservedEndAt,
		OccupiedEndAt:                &serviceEnd,
		CancelDeadlineAt:             &cancelDeadline,
		BookingCloseDeadlineAt:       &bookingCloseDeadline,
		LateArrivalMinServiceMinutes: lateArrivalMinServiceMinutes,
		Status:                       "confirmed",
		ConfirmedAt:                  &confirmedAt,
	}

	// 预约创建必须做事务级校验：用户手选客服时，要在落库前重新检查“未来预约占产能”规则，
	// 避免前端时段缓存与并发提交之间出现超卖或越过最大等待上限。
	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		decision, err := validateAppointmentPlacementRules(tx, merchant, targetDate, appointmentTime, occupiedMinutes, input.TechnicianID, 0)
		if err != nil {
			return err
		}
		appointment.PredictedWaitMinutes = decision.PredictedWaitMinutes
		appointment.TechnicianID = decision.AssignedTechnicianID
		if err := tx.Create(&appointment).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.Appointment{}).Where("id = ?", appointment.ID).Update("booking_root_id", appointment.ID).Error; err != nil {
			return err
		}
		appointment.BookingRootID = &appointment.ID
		if _, err := initializeAppointmentSettlement(tx, &appointment, "appointment_created"); err != nil {
			return err
		}
		return nil
	}); err != nil {
		log.Printf("创建预约失败: %v", err)
		var ae apiErr
		if errors.As(err, &ae) {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建预约失败"})
		return
	}

	log.Printf("预约创建成功: ID=%d, 状态=%s", appointment.ID, appointment.Status)
	if reloaded, err := loadAppointmentByID(config.DB, appointment.ID); err == nil && reloaded != nil {
		appointment = *reloaded
		_ = hydrateAppointmentRelations(config.DB, &appointment)
	}
	c.JSON(http.StatusOK, gin.H{"data": appointment})
}

func ConfirmAppointment(c *gin.Context) {
	id := c.Param("id")
	appointmentID64, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的预约ID"})
		return
	}
	appointmentPtr, err := loadAppointmentByID(config.DB, uint(appointmentID64))
	if err != nil || appointmentPtr == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}
	appointment := *appointmentPtr

	if _, _, ok := checkMerchantAppointmentOwnership(c, appointment); !ok {
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

	var merchant models.Merchant
	if err := config.DB.First(&merchant, appointment.MerchantID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "商户不存在"})
		return
	}
	occupiedMinutes := getAppointmentOccupiedMinutes(appointment.MerchantID, appointment)
	targetDate := appointment.AppointmentTime.In(appointmentLocation())
	confirmedAt := time.Now()
	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		currentPtr, err := loadAppointmentByID(tx.Clauses(clause.Locking{Strength: "UPDATE"}), appointment.ID)
		if err != nil {
			return err
		}
		if currentPtr == nil {
			return gorm.ErrRecordNotFound
		}
		current := *currentPtr
		if current.Status != "pending" {
			return apiErr{status: http.StatusBadRequest, msg: "只能确认待处理的预约"}
		}
		decision, err := validateAppointmentPlacementRules(tx, merchant, targetDate, *current.AppointmentTime, occupiedMinutes, current.TechnicianID, current.ID)
		if err != nil {
			return err
		}
		return tx.Model(&current).Updates(map[string]interface{}{
			"status":                 "confirmed",
			"confirmed_at":           &confirmedAt,
			"predicted_wait_minutes": decision.PredictedWaitMinutes,
			"technician_id":          decision.AssignedTechnicianID,
		}).Error
	}); err != nil {
		var ae apiErr
		if errors.As(err, &ae) {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "确认预约失败"})
		return
	}
	if reloaded, err := loadAppointmentByID(config.DB, appointment.ID); err == nil && reloaded != nil {
		appointment = *reloaded
		_ = hydrateAppointmentRelations(config.DB, &appointment)
	}
	appointment.Status = normalizeAppointmentStatus(appointment.Status)
	// 入预约队列（内存队列）：仅 confirmed 才进入预约排队
	if appointment.AppointmentTime != nil && queue.Default != nil {
		date := appointment.AppointmentTime.Format("2006-01-02")
		now2 := time.Now()
		queue.Default.Enqueue(appointment.MerchantID, date, queue.QueueTypeAppointment, appointment.ID, 1, true, now2)
	}
	c.JSON(http.StatusOK, gin.H{"data": appointment})
}

func CheckInAppointment(c *gin.Context) {
	id := c.Param("id")
	appointmentID64, err := strconv.ParseUint(strings.TrimSpace(id), 10, 64)
	if err != nil || appointmentID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的预约ID"})
		return
	}
	appointment, err := loadAppointmentByID(config.DB, uint(appointmentID64))
	if err != nil || appointment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}
	if _, _, ok := checkMerchantAppointmentOwnership(c, *appointment); !ok {
		return
	}
	status := normalizeAppointmentStatus(appointment.Status)
	if status != "confirmed" && status != "arrived" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前预约状态不可签到"})
		return
	}
	var merchant models.Merchant
	if err := config.DB.First(&merchant, appointment.MerchantID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取商户信息失败"})
		return
	}
	now := time.Now().In(appointmentLocation())
	if !canArriveForAppointment(*appointment, &merchant, now) && status != "arrived" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前不在可签到时间窗口内"})
		return
	}

	var result verifyCommitResult
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		currentAppt, err := loadAppointmentByID(tx, uint(appointmentID64))
		if err != nil {
			return err
		}
		if currentAppt == nil {
			return gorm.ErrRecordNotFound
		}
		currentStatus := normalizeAppointmentStatus(currentAppt.Status)
		if currentStatus != "confirmed" && currentStatus != "arrived" {
			return apiErr{status: http.StatusBadRequest, msg: "当前预约状态不可签到"}
		}
		if currentAppt.UsageID != nil && currentAppt.ServiceSessionID != nil && *currentAppt.UsageID > 0 && *currentAppt.ServiceSessionID > 0 {
			result = verifyCommitResult{
				Merchant:          merchant,
				Card:              models.Card{ID: currentAppt.CardID, UserID: currentAppt.UserID, MerchantID: currentAppt.MerchantID},
				UsageID:           *currentAppt.UsageID,
				SessionID:         *currentAppt.ServiceSessionID,
				UsedAt:            now,
				Action:            "appointment_checkin",
				AppointmentStatus: "arrived",
			}
			return nil
		}
		var card models.Card
		if err := tx.First(&card, currentAppt.CardID).Error; err != nil {
			return err
		}
		verifyCode := models.VerifyCode{
			CardID:    card.ID,
			ProjectID: currentAppt.ProjectID,
			Code:      fmt.Sprintf("APPTCI-%d-%d", currentAppt.ID, now.UnixNano()),
			ExpireAt:  now.Add(10 * time.Minute).Unix(),
			Used:      true,
			UsedAt:    &now,
		}
		if err := tx.Create(&verifyCode).Error; err != nil {
			return err
		}
		usage := models.Usage{
			CardID:             card.ID,
			MerchantID:         card.MerchantID,
			ProjectID:          currentAppt.ProjectID,
			UsedTimes:          1,
			UsedAt:             &now,
			VerifyCode:         verifyCode.Code,
			VerifyCodeExpireAt: verifyCode.ExpireAt,
			Status:             "in_progress",
		}
		if err := tx.Create(&usage).Error; err != nil {
			return err
		}
		session, nextStep, shouldEnqueueOnsite, err := createServiceSessionForUsage(tx, merchant, card, verifyCode, usage, now)
		if err != nil {
			return err
		}
		actualArrivedAt := now
		if err := tx.Model(&models.Appointment{}).
			Where("id = ? AND status IN ?", currentAppt.ID, []string{"confirmed", "arrived"}).
			Updates(map[string]interface{}{
				"status":                 "arrived",
				"arrived_at":             &actualArrivedAt,
				"actual_arrived_at":      &actualArrivedAt,
				"usage_id":               usage.ID,
				"service_session_id":     session.ID,
				"predicted_wait_minutes": session.PredictedAppointmentDelayMinutes,
			}).Error; err != nil {
			return err
		}
		result = verifyCommitResult{
			Merchant:             merchant,
			Card:                 card,
			UsedAt:               now,
			RemainTimes:          card.RemainTimes,
			SessionID:            session.ID,
			NextStep:             nextStep,
			UsageID:              usage.ID,
			ShouldEnqueueOnsite:  shouldEnqueueOnsite,
			Action:               "appointment_checkin",
			AppointmentStatus:    "arrived",
			PredictedWaitMinutes: session.PredictedAppointmentDelayMinutes,
			SessionWaitState:     models.NormalizeSessionStatus(session.Status),
		}
		if session.TechnicianID != nil {
			result.BoundTechnicianID = *session.TechnicianID
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	enqueueVerifyUsageIfNeeded(result.Merchant, result.Card, result.UsageID, result.ShouldEnqueueOnsite)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"appointment_id":         appointment.ID,
		"checkin_type":           "appointment_checkin",
		"actual_arrived_at":      result.UsedAt.Format("2006-01-02 15:04:05"),
		"service_session_id":     result.SessionID,
		"usage_id":               result.UsageID,
		"appointment_status":     result.AppointmentStatus,
		"predicted_wait_minutes": result.PredictedWaitMinutes,
		"session_wait_state":     result.SessionWaitState,
		"bound_technician_id":    result.BoundTechnicianID,
		"skip_asset_deduction":   true,
	}})
}

func FinishAppointment(c *gin.Context) {
	c.JSON(http.StatusBadRequest, gin.H{"error": "预约完成已改为随核销/服务会话自动闭环"})
}

func UpdateAppointmentResolution(c *gin.Context) {
	id := c.Param("id")
	var appointment models.Appointment
	if err := config.DB.First(&appointment, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}

	if _, _, ok := checkMerchantAppointmentOwnership(c, appointment); !ok {
		return
	}

	var input struct {
		ResolutionNote string `json:"resolution_note" binding:"required,max=255"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	note := strings.TrimSpace(input.ResolutionNote)
	if note == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "处理备注不能为空"})
		return
	}

	// 改签/补偿首版先把人工处理动作落到结构化备注，避免预约冲突只能靠线下口头同步。
	if err := config.DB.Model(&appointment).Update("resolution_note", note).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存处理备注失败"})
		return
	}
	appointment.ResolutionNote = note
	appointment.Status = normalizeAppointmentStatus(appointment.Status)
	c.JSON(http.StatusOK, gin.H{"data": appointment})
}

func CloseAppointmentException(c *gin.Context) {
	id := c.Param("id")
	appointmentID64, err := strconv.ParseUint(strings.TrimSpace(id), 10, 64)
	if err != nil || appointmentID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的预约ID"})
		return
	}
	appointment, err := loadAppointmentByID(config.DB, uint(appointmentID64))
	if err != nil || appointment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}
	if _, _, ok := checkMerchantAppointmentOwnership(c, *appointment); !ok {
		return
	}

	var input struct {
		Reason string `json:"reason" binding:"required,max=255"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "异常结案说明不能为空"})
		return
	}

	now := time.Now().In(appointmentLocation())
	if !appointmentIsCrossDayUnfinished(appointment, now) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前预约不属于跨日未闭环异常，不能直接异常结案"})
		return
	}

	actorType, actorID := getAppointmentActor(c)
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		currentPtr, err := loadAppointmentByID(tx.Clauses(clause.Locking{Strength: "UPDATE"}), appointment.ID)
		if err != nil {
			return err
		}
		if currentPtr == nil {
			return gorm.ErrRecordNotFound
		}
		current := *currentPtr
		if !appointmentIsCrossDayUnfinished(&current, now) {
			return apiErr{status: http.StatusBadRequest, msg: "当前预约不属于跨日未闭环异常，不能直接异常结案"}
		}

		if err := autoCloseCrossDayUnfinishedAppointment(tx, &current, now, actorType, actorID, "跨日未闭环异常结案："+reason); err != nil {
			return err
		}
		appointment = &current
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "异常结案失败"})
		return
	}
	appointment.Status = normalizeAppointmentStatus(appointment.Status)
	decorateAppointmentDisplay(appointment, now)
	c.JSON(http.StatusOK, gin.H{"data": appointment})
}

type appointmentReschedulePayload struct {
	AppointmentTime string `json:"appointment_time" binding:"required"`
	TechnicianID    *uint  `json:"technician_id"`
	ProjectID       *uint  `json:"project_id"`
	Reason          string `json:"reason" binding:"required,max=255"`
}

func parseAppointmentReschedulePayload(input appointmentReschedulePayload) (time.Time, string, *time.Location, error) {
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		return time.Time{}, "", nil, apiErr{status: http.StatusBadRequest, msg: "改签原因不能为空"}
	}
	loc := appointmentLocation()
	newAppointmentTime, err := time.ParseInLocation("2006-01-02 15:04:05", input.AppointmentTime, loc)
	if err != nil {
		return time.Time{}, "", nil, apiErr{status: http.StatusBadRequest, msg: "预约时间格式错误"}
	}
	if newAppointmentTime.Before(time.Now().In(loc).Add(-1 * time.Minute)) {
		return time.Time{}, "", nil, apiErr{status: http.StatusBadRequest, msg: "不能改签到过去时间"}
	}
	return newAppointmentTime, reason, loc, nil
}

func appointmentDateAllowed(date string, allowedDates []string) bool {
	for _, item := range allowedDates {
		if strings.TrimSpace(item) == strings.TrimSpace(date) {
			return true
		}
	}
	return false
}

func loadRescheduleTargetAppointment(c *gin.Context) (*models.Appointment, string, error) {
	id := c.Param("id")
	appointmentID64, err := strconv.ParseUint(strings.TrimSpace(id), 10, 64)
	if err != nil || appointmentID64 == 0 {
		return nil, "", apiErr{status: http.StatusBadRequest, msg: "无效的预约ID"}
	}
	appointment, err := loadAppointmentByID(config.DB, uint(appointmentID64))
	if err != nil || appointment == nil {
		return nil, "", apiErr{status: http.StatusNotFound, msg: "预约不存在"}
	}

	authType := strings.TrimSpace(c.GetString("auth_type"))
	switch authType {
	case "merchant", "staff":
		if _, _, ok := checkMerchantAppointmentOwnership(c, *appointment); !ok {
			return nil, authType, apiErr{status: 0, msg: ""}
		}
	case "user":
		authUserID, ok := mustUserID(c)
		if !ok {
			return nil, authType, apiErr{status: 0, msg: ""}
		}
		if appointment.UserID != authUserID {
			return nil, authType, apiErr{status: http.StatusForbidden, msg: "无权操作此预约"}
		}
	default:
		return nil, authType, apiErr{status: http.StatusUnauthorized, msg: "未授权"}
	}
	return appointment, authType, nil
}

func getAppointmentRescheduleEligibilityData(tx *gorm.DB, appointment models.Appointment) (models.Merchant, appointmentRescheduleEligibility, error) {
	var merchant models.Merchant
	if err := tx.First(&merchant, appointment.MerchantID).Error; err != nil {
		return merchant, appointmentRescheduleEligibility{}, err
	}
	eligibility, err := computeAppointmentRescheduleEligibility(tx, appointment, merchant, time.Now().In(appointmentLocation()))
	return merchant, eligibility, err
}

type appointmentRepairEvaluation struct {
	AffectedByLeave         bool
	HasHighQualityCandidate bool
	CandidateTime           *time.Time
	CandidateTechnicianID   *uint
}

type appointmentRepairInspection struct {
	Appointment             models.Appointment    `json:"appointment"`
	AffectedByLeave         bool                  `json:"affected_by_leave"`
	HasHighQualityCandidate bool                  `json:"has_high_quality_candidate"`
	Decision                string                `json:"decision"`
	Reason                  string                `json:"reason"`
	CandidateTime           string                `json:"candidate_time,omitempty"`
	CandidateTechnicianID   *uint                 `json:"candidate_technician_id,omitempty"`
	Recommendations         []appointmentTimeSlot `json:"recommendations"`
}

func buildAppointmentSchedulingSnapshot(appointmentTime time.Time, serviceMinutes, serviceGapMinutes int, loc *time.Location) (time.Time, time.Time, time.Time, time.Time, int) {
	if loc == nil {
		loc = appointmentTime.Location()
	}
	reservedEnd := appointmentTime.Add(time.Duration(serviceMinutes) * time.Minute)
	occupiedEnd := appointmentTime.Add(time.Duration(projectBookingOccupiedMinutes(serviceMinutes, serviceGapMinutes)) * time.Minute)
	baseDate := time.Date(appointmentTime.Year(), appointmentTime.Month(), appointmentTime.Day(), 0, 0, 0, 0, loc).Add(-24 * time.Hour)
	cancelDeadline := time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(), 16, 0, 0, 0, loc)
	bookingCloseDeadline := time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(), 18, 0, 0, 0, loc)
	return reservedEnd, occupiedEnd, cancelDeadline, bookingCloseDeadline, serviceMinutes / 2
}

func appointmentHasProtectedRepairSlot(tx *gorm.DB, merchantID, appointmentID uint) (bool, error) {
	if tx == nil || merchantID == 0 || appointmentID == 0 {
		return false, nil
	}
	var count int64
	if err := tx.Table("protected_repair_slots").Where("merchant_id = ? AND appointment_id = ? AND status = ?", merchantID, appointmentID, "reserved").Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func evaluateAppointmentRepairOptions(tx *gorm.DB, merchant models.Merchant, appt models.Appointment) (appointmentRepairEvaluation, error) {
	out := appointmentRepairEvaluation{}
	affected, err := appointmentHasProtectedRepairSlot(tx, appt.MerchantID, appt.ID)
	if err != nil {
		return out, err
	}
	out.AffectedByLeave = affected
	if !affected || appt.AppointmentTime == nil {
		return out, nil
	}
	loc := appointmentLocation()
	appointmentDay := appt.AppointmentTime.In(loc)
	targetDate := time.Date(appointmentDay.Year(), appointmentDay.Month(), appointmentDay.Day(), 0, 0, 0, 0, loc)
	publishedRows, err := loadPublishedSchedulePublishings(tx, merchant.ID, targetDate)
	if err != nil {
		return out, err
	}
	intervals, ok := buildPublishedScheduleIntervals(publishedRows)
	if !ok {
		return out, nil
	}
	serviceMinutes := getAppointmentServiceMinutes(appt.MerchantID, appt)
	serviceGapMinutes := getAppointmentGapMinutes(appt.MerchantID, appt)
	occupiedMinutes := projectBookingOccupiedMinutes(serviceMinutes, serviceGapMinutes)
	granularity := merchantAppointmentSlotGranularityMinutes(&merchant)
	tryCandidate := func(slotTime time.Time, technicianID *uint) bool {
		decision, err := validateAppointmentPlacementRules(tx, merchant, targetDate, slotTime, occupiedMinutes, technicianID, appt.ID)
		if err != nil {
			return false
		}
		out.HasHighQualityCandidate = true
		candidateTime := slotTime
		out.CandidateTime = &candidateTime
		out.CandidateTechnicianID = decision.AssignedTechnicianID
		return true
	}
	for _, it := range intervals {
		latestStart := it.End.Add(-time.Duration(occupiedMinutes) * time.Minute)
		for slotTime := it.Start; !slotTime.After(latestStart); slotTime = slotTime.Add(time.Duration(granularity) * time.Minute) {
			if appt.TechnicianID != nil && *appt.TechnicianID > 0 {
				techID := *appt.TechnicianID
				if tryCandidate(slotTime, &techID) {
					return out, nil
				}
			}
			if merchant.SupportCustomerServiceMode {
				if tryCandidate(slotTime, nil) {
					return out, nil
				}
			} else if appt.TechnicianID == nil {
				if tryCandidate(slotTime, nil) {
					return out, nil
				}
			}
		}
	}
	return out, nil
}

func buildAppointmentRepairInspection(tx *gorm.DB, merchant models.Merchant, appt models.Appointment) (appointmentRepairInspection, error) {
	item := appointmentRepairInspection{
		Appointment:     appt,
		Recommendations: []appointmentTimeSlot{},
	}
	if err := hydrateAppointmentRelations(tx, &item.Appointment); err != nil {
		return item, err
	}
	eval, err := evaluateAppointmentRepairOptions(tx, merchant, appt)
	if err != nil {
		return item, err
	}
	item.AffectedByLeave = eval.AffectedByLeave
	item.HasHighQualityCandidate = eval.HasHighQualityCandidate
	if !eval.AffectedByLeave {
		item.Decision = "not_affected"
		item.Reason = "当前预约未命中请假异常修复范围"
		return item, nil
	}
	if eval.CandidateTime != nil {
		item.CandidateTime = eval.CandidateTime.In(appointmentLocation()).Format("2006-01-02 15:04:05")
	}
	item.CandidateTechnicianID = eval.CandidateTechnicianID
	if !eval.HasHighQualityCandidate || appt.AppointmentTime == nil {
		item.Decision = "cancel_only"
		item.Reason = "当前无高质量修复方案，请转取消/补偿分流"
		return item, nil
	}
	item.Decision = "repairable"
	item.Reason = "存在高质量修复方案，可优先发起保护性改签"

	projectID := uint(0)
	if appt.ProjectID != nil {
		projectID = *appt.ProjectID
	}
	date := appt.AppointmentTime.In(appointmentLocation()).Format("2006-01-02")
	payload, err := buildAvailableTimeSlotsPayload(merchant, merchant.ID, date, projectID, appointmentLocation(), &availableTimeSlotsOptions{ExcludeAppointmentID: appt.ID})
	if err != nil {
		return item, err
	}
	currentScore, err := computeAppointmentCurrentPlacementScore(tx, merchant, appt)
	if err != nil {
		return item, err
	}
	if slots, ok := payload["time_slots"].([]appointmentTimeSlot); ok {
		recommendations := buildAppointmentRecommendationSlots(slots, appointmentRescheduleEligibility{
			Allowed:  true,
			RuleMode: appointmentRescheduleTodayOrTomorrow,
		}, currentScore)
		if len(recommendations) > 3 {
			recommendations = recommendations[:3]
		}
		item.Recommendations = recommendations
	}
	return item, nil
}

func buildAppointmentRepairInspections(tx *gorm.DB, merchantID uint, appts []models.Appointment) ([]appointmentRepairInspection, error) {
	if tx == nil || merchantID == 0 || len(appts) == 0 {
		return []appointmentRepairInspection{}, nil
	}
	var merchant models.Merchant
	if err := tx.First(&merchant, merchantID).Error; err != nil {
		return nil, err
	}
	items := make([]appointmentRepairInspection, 0, len(appts))
	for _, appt := range appts {
		item, err := buildAppointmentRepairInspection(tx, merchant, appt)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func executeAppointmentReschedule(tx *gorm.DB, appointmentID uint, proposal models.AppointmentRescheduleRequest, actorType string, actorID *uint, now time.Time) (models.Appointment, models.Appointment, error) {
	var oldAppointment models.Appointment
	var newAppointment models.Appointment
	currentPtr, err := loadAppointmentByID(tx, appointmentID)
	if err != nil {
		return oldAppointment, newAppointment, err
	}
	if currentPtr == nil {
		return oldAppointment, newAppointment, gorm.ErrRecordNotFound
	}
	current := *currentPtr
	if !appointmentAllowsReschedule(current.Status) {
		return oldAppointment, newAppointment, apiErr{status: http.StatusBadRequest, msg: "当前预约状态不可改签"}
	}
	if proposal.NewAppointmentTime == nil {
		return oldAppointment, newAppointment, apiErr{status: http.StatusBadRequest, msg: "改签时间不能为空"}
	}

	var merchant models.Merchant
	if err := tx.First(&merchant, current.MerchantID).Error; err != nil {
		return oldAppointment, newAppointment, err
	}
	targetDate := proposal.NewAppointmentTime.In(appointmentLocation()).Format("2006-01-02")
	repairEval, err := evaluateAppointmentRepairOptions(tx, merchant, current)
	if err != nil {
		return oldAppointment, newAppointment, err
	}
	protectiveByMerchant := repairEval.AffectedByLeave && (proposal.ProposedByType == "merchant" || proposal.ProposedByType == "staff")
	if !protectiveByMerchant {
		eligibility, err := computeAppointmentRescheduleEligibility(tx, current, merchant, now.In(appointmentLocation()))
		if err != nil {
			return oldAppointment, newAppointment, err
		}
		if !eligibility.Allowed || !appointmentDateAllowed(targetDate, eligibility.AllowedDates) {
			msg := eligibility.Reason
			if strings.TrimSpace(msg) == "" {
				msg = "目标日期不在允许的改签范围内"
			}
			return oldAppointment, newAppointment, apiErr{status: http.StatusBadRequest, msg: msg}
		}
	} else if current.AppointmentTime != nil && targetDate != current.AppointmentTime.In(appointmentLocation()).Format("2006-01-02") {
		return oldAppointment, newAppointment, apiErr{status: http.StatusBadRequest, msg: "保护性改签当前仅支持原服务日修复"}
	}
	intervals, ok := getMerchantBusinessIntervalsForDate(merchant, proposal.NewAppointmentTime.In(appointmentLocation()))
	if !ok || !isWithinBusinessIntervals(*proposal.NewAppointmentTime, intervals) {
		return oldAppointment, newAppointment, apiErr{status: http.StatusBadRequest, msg: "改签时间不在营业时间内"}
	}

	projectID := current.ProjectID
	serviceMinutes := getAppointmentServiceMinutes(current.MerchantID, current)
	serviceGapMinutes := getAppointmentGapMinutes(current.MerchantID, current)
	if proposal.NewProjectID != nil && *proposal.NewProjectID > 0 {
		var project models.MerchantProject
		if err := tx.Where("id = ? AND merchant_id = ? AND is_active = ? AND bookable_online = ?", *proposal.NewProjectID, current.MerchantID, true, true).First(&project).Error; err != nil {
			return oldAppointment, newAppointment, apiErr{status: http.StatusBadRequest, msg: "无效的项目"}
		}
		projectID = proposal.NewProjectID
		serviceMinutes = project.Duration
		serviceGapMinutes = project.ServiceGapMinutes
	}
	if serviceMinutes <= 0 {
		serviceMinutes = 30
	}
	occupiedMinutes := projectBookingOccupiedMinutes(serviceMinutes, serviceGapMinutes)

	technicianID := current.TechnicianID
	if proposal.NewTechnicianID != nil {
		if *proposal.NewTechnicianID == 0 {
			technicianID = nil
		} else {
			technicianID = proposal.NewTechnicianID
		}
	}

	decision, err := validateAppointmentPlacementRules(tx, merchant, proposal.NewAppointmentTime.In(appointmentLocation()), *proposal.NewAppointmentTime, occupiedMinutes, technicianID, current.ID)
	if err != nil {
		return oldAppointment, newAppointment, err
	}
	current.PredictedWaitMinutes = decision.PredictedWaitMinutes
	technicianID = decision.AssignedTechnicianID

	if normalizeAppointmentStatus(current.Status) == "arrived" {
		// 到店后接受改签，需要先回滚未开始服务的占用，再关闭旧预约。
		if err := cancelUnstartedAppointmentArrival(tx, &current, now); err != nil {
			return oldAppointment, newAppointment, err
		}
	}

	newStatus := "pending"
	newConfirmedAt := (*time.Time)(nil)
	if normalizeAppointmentStatus(current.Status) == "confirmed" || normalizeAppointmentStatus(current.Status) == "arrived" {
		newStatus = "confirmed"
		newConfirmedAt = &now
	}
	bookingRootID := current.ID
	if current.BookingRootID != nil && *current.BookingRootID > 0 {
		bookingRootID = *current.BookingRootID
	}
	reservedEndAt, occupiedEndAt, cancelDeadlineAt, bookingCloseDeadlineAt, lateArrivalMinServiceMinutes := buildAppointmentSchedulingSnapshot(*proposal.NewAppointmentTime, serviceMinutes, serviceGapMinutes, appointmentLocation())
	newAppointment = models.Appointment{
		CardID:                       current.CardID,
		MerchantID:                   current.MerchantID,
		UserID:                       current.UserID,
		BookingRootID:                &bookingRootID,
		ProjectID:                    projectID,
		TechnicianID:                 technicianID,
		AppointmentTime:              proposal.NewAppointmentTime,
		ReservedStartAt:              proposal.NewAppointmentTime,
		ReservedEndAt:                &reservedEndAt,
		OccupiedEndAt:                &occupiedEndAt,
		CancelDeadlineAt:             &cancelDeadlineAt,
		BookingCloseDeadlineAt:       &bookingCloseDeadlineAt,
		LateArrivalMinServiceMinutes: lateArrivalMinServiceMinutes,
		Status:                       newStatus,
		ConfirmedAt:                  newConfirmedAt,
		PredictedWaitMinutes:         current.PredictedWaitMinutes,
		ReplacesAppointmentID:        &current.ID,
		RescheduleReason:             proposal.Reason,
		ResolutionNote:               appendAppointmentResolutionNote("", "改签说明："+proposal.Reason),
	}
	if err := tx.Create(&newAppointment).Error; err != nil {
		return oldAppointment, newAppointment, err
	}

	closeUpdates := map[string]interface{}{
		"status":                     "canceled",
		"closed_reason":              "rescheduled",
		"closed_by_type":             actorType,
		"closed_by_id":               actorID,
		"reschedule_reason":          proposal.Reason,
		"replaced_by_appointment_id": newAppointment.ID,
		"canceled_at":                &now,
		"resolution_note":            appendAppointmentResolutionNote(current.ResolutionNote, fmt.Sprintf("改签到 %s", proposal.NewAppointmentTime.Format("2006-01-02 15:04"))),
		"arrived_at":                 nil,
		"service_session_id":         nil,
		"usage_id":                   nil,
		"predicted_wait_minutes":     0,
	}
	if err := tx.Model(&models.Appointment{}).Where("id = ?", current.ID).Updates(closeUpdates).Error; err != nil {
		return oldAppointment, newAppointment, err
	}
	if err := transferAppointmentSettlement(tx, &current, &newAppointment, "appointment_rescheduled"); err != nil {
		return oldAppointment, newAppointment, err
	}

	oldAppointment = current
	oldAppointment.Status = "canceled"
	oldAppointment.ClosedReason = "rescheduled"
	oldAppointment.ClosedByType = actorType
	oldAppointment.ClosedByID = actorID
	oldAppointment.RescheduleReason = proposal.Reason
	oldAppointment.ReplacedByAppointmentID = &newAppointment.ID
	oldAppointment.CanceledAt = &now
	return oldAppointment, newAppointment, nil
}

func GetAppointmentRescheduleEligibility(c *gin.Context) {
	appointment, authType, err := loadRescheduleTargetAppointment(c)
	if err != nil {
		var ae apiErr
		if errors.As(err, &ae) && ae.status > 0 {
			c.JSON(ae.status, gin.H{"error": ae.msg})
		}
		return
	}
	merchant, eligibility, err := getAppointmentRescheduleEligibilityData(config.DB, *appointment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取改签资格失败"})
		return
	}
	repairEval, err := evaluateAppointmentRepairOptions(config.DB, merchant, *appointment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取异常修复信息失败"})
		return
	}
	if repairEval.AffectedByLeave {
		if authType == "user" {
			eligibility.Allowed = false
			eligibility.RuleMode = appointmentRescheduleForbidden
			eligibility.AllowedDates = []string{}
			eligibility.DefaultDate = ""
			eligibility.Reason = "该预约受请假影响，需由商户发起保护性改签"
		} else if appointment.AppointmentTime != nil {
			serviceDate := appointment.AppointmentTime.In(appointmentLocation()).Format("2006-01-02")
			eligibility.Allowed = repairEval.HasHighQualityCandidate
			eligibility.AllowedDates = []string{serviceDate}
			eligibility.DefaultDate = serviceDate
			if repairEval.HasHighQualityCandidate {
				eligibility.Reason = ""
			} else {
				eligibility.RuleMode = appointmentRescheduleForbidden
				eligibility.Reason = "当前无高质量修复方案，请改走取消分流"
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": eligibility})
}

func GetAppointmentRescheduleSlots(c *gin.Context) {
	appointment, authType, err := loadRescheduleTargetAppointment(c)
	if err != nil {
		var ae apiErr
		if errors.As(err, &ae) && ae.status > 0 {
			c.JSON(ae.status, gin.H{"error": ae.msg})
		}
		return
	}
	merchant, eligibility, err := getAppointmentRescheduleEligibilityData(config.DB, *appointment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取改签资格失败"})
		return
	}
	repairEval, err := evaluateAppointmentRepairOptions(config.DB, merchant, *appointment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取异常修复信息失败"})
		return
	}
	if repairEval.AffectedByLeave {
		if authType == "user" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "该预约受请假影响，需由商户发起保护性改签"})
			return
		}
		if !repairEval.HasHighQualityCandidate {
			c.JSON(http.StatusBadRequest, gin.H{"error": "当前无高质量修复方案，请改走取消分流"})
			return
		}
		if appointment.AppointmentTime != nil {
			eligibility.Allowed = true
			eligibility.AllowedDates = []string{appointment.AppointmentTime.In(appointmentLocation()).Format("2006-01-02")}
			eligibility.DefaultDate = eligibility.AllowedDates[0]
		}
	}
	if !eligibility.Allowed {
		c.JSON(http.StatusBadRequest, gin.H{"error": eligibility.Reason})
		return
	}
	loc := appointmentLocation()
	date := strings.TrimSpace(c.Query("date"))
	if date == "" {
		date = eligibility.DefaultDate
	}
	if !appointmentDateAllowed(date, eligibility.AllowedDates) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "目标日期不在允许的改签范围内"})
		return
	}
	if _, err := time.ParseInLocation("2006-01-02", date, loc); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "日期格式错误，应为 YYYY-MM-DD"})
		return
	}
	projectID := uint(0)
	if appointment.ProjectID != nil {
		projectID = *appointment.ProjectID
	}
	payload, err := buildAvailableTimeSlotsPayload(merchant, merchant.ID, date, projectID, loc, &availableTimeSlotsOptions{ExcludeAppointmentID: appointment.ID})
	if err != nil {
		var ae apiErr
		if errors.As(err, &ae) {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取可改签时间失败"})
		return
	}
	if !repairEval.AffectedByLeave {
		currentScore, err := computeAppointmentCurrentPlacementScore(config.DB, merchant, *appointment)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "计算当前改签基准失败"})
			return
		}
		if slots, ok := payload["time_slots"].([]appointmentTimeSlot); ok {
			payload["time_slots"] = filterAppointmentRescheduleSlotsByComparison(slots, eligibility, currentScore)
		}
	}
	payload["eligibility"] = eligibility
	c.JSON(http.StatusOK, gin.H{"data": payload})
}

func GetAppointmentRescheduleRecommendations(c *gin.Context) {
	appointment, authType, err := loadRescheduleTargetAppointment(c)
	if err != nil {
		var ae apiErr
		if errors.As(err, &ae) && ae.status > 0 {
			c.JSON(ae.status, gin.H{"error": ae.msg})
		}
		return
	}
	if authType != "user" {
		c.JSON(http.StatusForbidden, gin.H{"error": "当前仅支持用户查看推荐时段"})
		return
	}
	merchant, eligibility, err := getAppointmentRescheduleEligibilityData(config.DB, *appointment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取改签资格失败"})
		return
	}
	if !merchant.AppointmentRescheduleRecommendationEnabled {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{
			"enabled":         false,
			"reason":          "系统优化性改签推荐未开启",
			"eligibility":     eligibility,
			"recommendations": []appointmentTimeSlot{},
		}})
		return
	}
	repairEval, err := evaluateAppointmentRepairOptions(config.DB, merchant, *appointment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取异常修复信息失败"})
		return
	}
	if repairEval.AffectedByLeave {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{
			"enabled":         true,
			"reason":          "当前处于异常修复模式，系统优化性改签推荐已自动降级",
			"eligibility":     eligibility,
			"recommendations": []appointmentTimeSlot{},
		}})
		return
	}
	if !eligibility.Allowed {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{
			"enabled":         true,
			"reason":          eligibility.Reason,
			"eligibility":     eligibility,
			"recommendations": []appointmentTimeSlot{},
		}})
		return
	}
	loc := appointmentLocation()
	date := strings.TrimSpace(c.Query("date"))
	if date == "" {
		date = eligibility.DefaultDate
	}
	if !appointmentDateAllowed(date, eligibility.AllowedDates) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "目标日期不在允许的推荐范围内"})
		return
	}
	if _, err := time.ParseInLocation("2006-01-02", date, loc); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "日期格式错误，应为 YYYY-MM-DD"})
		return
	}
	projectID := uint(0)
	if appointment.ProjectID != nil {
		projectID = *appointment.ProjectID
	}
	payload, err := buildAvailableTimeSlotsPayload(merchant, merchant.ID, date, projectID, loc, &availableTimeSlotsOptions{ExcludeAppointmentID: appointment.ID})
	if err != nil {
		var ae apiErr
		if errors.As(err, &ae) {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取推荐时段失败"})
		return
	}
	currentScore, err := computeAppointmentCurrentPlacementScore(config.DB, merchant, *appointment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "计算当前改签基准失败"})
		return
	}
	recommendations := []appointmentTimeSlot{}
	if slots, ok := payload["time_slots"].([]appointmentTimeSlot); ok {
		recommendations = buildAppointmentRecommendationSlots(slots, eligibility, currentScore)
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"enabled":         true,
		"reason":          "",
		"eligibility":     eligibility,
		"recommendations": recommendations,
	}})
}

func CreateAppointmentRescheduleRequest(c *gin.Context) {
	appointment, authType, err := loadRescheduleTargetAppointment(c)
	if err != nil {
		var ae apiErr
		if errors.As(err, &ae) && ae.status > 0 {
			c.JSON(ae.status, gin.H{"error": ae.msg})
		}
		return
	}

	var input appointmentReschedulePayload
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	newAppointmentTime, reason, _, err := parseAppointmentReschedulePayload(input)
	if err != nil {
		var ae apiErr
		if errors.As(err, &ae) {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	merchant, eligibility, err := getAppointmentRescheduleEligibilityData(config.DB, *appointment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取改签资格失败"})
		return
	}
	repairEval, err := evaluateAppointmentRepairOptions(config.DB, merchant, *appointment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取异常修复信息失败"})
		return
	}
	if repairEval.AffectedByLeave && authType == "user" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该预约受请假影响，需由商户发起保护性改签"})
		return
	}
	if repairEval.AffectedByLeave && (authType == "merchant" || authType == "staff") && !repairEval.HasHighQualityCandidate {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前无高质量修复方案，请改走取消分流"})
		return
	}
	targetDate := newAppointmentTime.In(appointmentLocation()).Format("2006-01-02")
	if !repairEval.AffectedByLeave && (!eligibility.Allowed || !appointmentDateAllowed(targetDate, eligibility.AllowedDates)) {
		msg := eligibility.Reason
		if strings.TrimSpace(msg) == "" {
			msg = "目标日期不在允许的改签范围内"
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	if repairEval.AffectedByLeave && appointment.AppointmentTime != nil && targetDate != appointment.AppointmentTime.In(appointmentLocation()).Format("2006-01-02") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "保护性改签当前仅支持原服务日修复"})
		return
	}
	serviceMinutes := getAppointmentServiceMinutes(appointment.MerchantID, *appointment)
	serviceGapMinutes := getAppointmentGapMinutes(appointment.MerchantID, *appointment)
	if input.ProjectID != nil && *input.ProjectID > 0 {
		var project models.MerchantProject
		if err := config.DB.Where("id = ? AND merchant_id = ? AND is_active = ? AND bookable_online = ?", *input.ProjectID, appointment.MerchantID, true, true).First(&project).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目"})
			return
		}
		serviceMinutes = project.Duration
		serviceGapMinutes = project.ServiceGapMinutes
	}
	occupiedMinutes := projectBookingOccupiedMinutes(serviceMinutes, serviceGapMinutes)
	placementDecision, err := validateAppointmentPlacementRules(config.DB, merchant, newAppointmentTime.In(appointmentLocation()), newAppointmentTime, occupiedMinutes, input.TechnicianID, appointment.ID)
	if err != nil {
		var ae apiErr
		if errors.As(err, &ae) {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "校验改签时间失败"})
		return
	}

	actorType, actorID := getAppointmentActor(c)
	requestStatus := "pending_user"
	if authType == "user" {
		requestStatus = "pending_merchant"
	}

	var proposal models.AppointmentRescheduleRequest
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		currentPtr, err := loadAppointmentByID(tx, appointment.ID)
		if err != nil {
			return err
		}
		if currentPtr == nil {
			return gorm.ErrRecordNotFound
		}
		current := *currentPtr
		if !appointmentAllowsReschedule(current.Status) {
			return apiErr{status: http.StatusBadRequest, msg: "当前预约状态不可改签"}
		}
		if exists, err := hasPendingRescheduleRequest(tx, current.ID); err != nil {
			return err
		} else if exists {
			return apiErr{status: http.StatusBadRequest, msg: "当前预约已有待确认的改签提议"}
		}

		proposal = models.AppointmentRescheduleRequest{
			AppointmentID:          current.ID,
			MerchantID:             current.MerchantID,
			UserID:                 current.UserID,
			CurrentProjectID:       current.ProjectID,
			CurrentTechnicianID:    current.TechnicianID,
			CurrentAppointmentTime: current.AppointmentTime,
			NewProjectID:           current.ProjectID,
			NewTechnicianID:        current.TechnicianID,
			NewAppointmentTime:     &newAppointmentTime,
			Status:                 requestStatus,
			Reason:                 reason,
			ProposedByType:         actorType,
			ProposedByID:           actorID,
		}
		if repairEval.AffectedByLeave && (authType == "merchant" || authType == "staff") {
			proposal.Status = "pending_user"
		}
		if input.ProjectID != nil {
			proposal.NewProjectID = input.ProjectID
		}
		if placementDecision.AssignedTechnicianID != nil {
			proposal.NewTechnicianID = placementDecision.AssignedTechnicianID
		}
		return tx.Create(&proposal).Error
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
	c.JSON(http.StatusOK, gin.H{"data": proposal})
}

func AcceptAppointmentRescheduleRequest(c *gin.Context) {
	requestID64, err := strconv.ParseUint(strings.TrimSpace(c.Param("request_id")), 10, 64)
	if err != nil || requestID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的改签提议ID"})
		return
	}
	req, err := loadAppointmentRescheduleRequestByID(config.DB, uint(requestID64))
	if err != nil || req == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "改签提议不存在"})
		return
	}
	appointment, err := loadAppointmentByID(config.DB, req.AppointmentID)
	if err != nil || appointment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}
	authType := strings.TrimSpace(c.GetString("auth_type"))
	switch authType {
	case "merchant", "staff":
		if _, _, ok := checkMerchantAppointmentOwnership(c, *appointment); !ok {
			return
		}
		if req.Status != "pending_merchant" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "当前改签提议无需商户确认"})
			return
		}
	case "user":
		authUserID, ok := mustUserID(c)
		if !ok {
			return
		}
		if appointment.UserID != authUserID {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此改签提议"})
			return
		}
		if req.Status != "pending_user" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "当前改签提议无需用户确认"})
			return
		}
	default:
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	actorType, actorID := getAppointmentActor(c)
	now := time.Now()
	var oldAppointment, newAppointment models.Appointment
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		currentReq, err := loadAppointmentRescheduleRequestByID(tx, req.ID)
		if err != nil {
			return err
		}
		if currentReq == nil {
			return gorm.ErrRecordNotFound
		}
		if strings.TrimSpace(currentReq.Status) != strings.TrimSpace(req.Status) {
			return apiErr{status: http.StatusBadRequest, msg: "改签提议状态已变化，请刷新后重试"}
		}
		oldAppointment, newAppointment, err = executeAppointmentReschedule(tx, currentReq.AppointmentID, *currentReq, actorType, actorID, now)
		if err != nil {
			return err
		}
		return tx.Model(&models.AppointmentRescheduleRequest{}).Where("id = ?", currentReq.ID).Updates(map[string]interface{}{
			"status":                "accepted",
			"confirmed_by_type":     actorType,
			"confirmed_by_id":       actorID,
			"confirmed_at":          &now,
			"result_appointment_id": newAppointment.ID,
		}).Error
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
	if reloadedAppointment, err := loadAppointmentByID(config.DB, newAppointment.ID); err == nil && reloadedAppointment != nil {
		newAppointment = *reloadedAppointment
		_ = hydrateAppointmentRelations(config.DB, &newAppointment)
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"old_appointment": oldAppointment,
		"new_appointment": newAppointment,
	}})
}

func RejectAppointmentRescheduleRequest(c *gin.Context) {
	requestID64, err := strconv.ParseUint(strings.TrimSpace(c.Param("request_id")), 10, 64)
	if err != nil || requestID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的改签提议ID"})
		return
	}
	req, err := loadAppointmentRescheduleRequestByID(config.DB, uint(requestID64))
	if err != nil || req == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "改签提议不存在"})
		return
	}
	appointment, err := loadAppointmentByID(config.DB, req.AppointmentID)
	if err != nil || appointment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}
	authType := strings.TrimSpace(c.GetString("auth_type"))
	switch authType {
	case "merchant", "staff":
		if _, _, ok := checkMerchantAppointmentOwnership(c, *appointment); !ok {
			return
		}
		if req.Status != "pending_merchant" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "当前改签提议无需商户处理"})
			return
		}
	case "user":
		authUserID, ok := mustUserID(c)
		if !ok {
			return
		}
		if appointment.UserID != authUserID {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此改签提议"})
			return
		}
		if req.Status != "pending_user" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "当前改签提议无需用户处理"})
			return
		}
	default:
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	actorType, actorID := getAppointmentActor(c)
	now := time.Now()
	if err := config.DB.Model(&models.AppointmentRescheduleRequest{}).
		Where("id = ? AND status = ?", req.ID, req.Status).
		Updates(map[string]interface{}{
			"status":            "rejected",
			"confirmed_by_type": actorType,
			"confirmed_by_id":   actorID,
			"confirmed_at":      &now,
		}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "拒绝改签提议失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"id": req.ID, "status": "rejected"}})
}

func CancelAppointmentRescheduleRequest(c *gin.Context) {
	requestID64, err := strconv.ParseUint(strings.TrimSpace(c.Param("request_id")), 10, 64)
	if err != nil || requestID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的改签提议ID"})
		return
	}
	req, err := loadAppointmentRescheduleRequestByID(config.DB, uint(requestID64))
	if err != nil || req == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "改签提议不存在"})
		return
	}
	appointment, err := loadAppointmentByID(config.DB, req.AppointmentID)
	if err != nil || appointment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}
	authType := strings.TrimSpace(c.GetString("auth_type"))
	switch authType {
	case "merchant", "staff":
		if _, _, ok := checkMerchantAppointmentOwnership(c, *appointment); !ok {
			return
		}
		proposer := strings.TrimSpace(req.ProposedByType)
		if proposer != "merchant" && proposer != "staff" {
			c.JSON(http.StatusForbidden, gin.H{"error": "只有发起改签的一方可以撤销"})
			return
		}
	case "user":
		authUserID, ok := mustUserID(c)
		if !ok {
			return
		}
		if appointment.UserID != authUserID {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此改签提议"})
			return
		}
		if strings.TrimSpace(req.ProposedByType) != "user" {
			c.JSON(http.StatusForbidden, gin.H{"error": "只有发起改签的一方可以撤销"})
			return
		}
	default:
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	if req.Status != "pending_user" && req.Status != "pending_merchant" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前改签提议不可撤销"})
		return
	}
	actorType, actorID := getAppointmentActor(c)
	now := time.Now()
	result := config.DB.Model(&models.AppointmentRescheduleRequest{}).
		Where("id = ? AND status = ?", req.ID, req.Status).
		Updates(map[string]interface{}{
			"status":            "canceled",
			"confirmed_by_type": actorType,
			"confirmed_by_id":   actorID,
			"confirmed_at":      &now,
		})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "撤销改签提议失败"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "改签提议状态已变化，请刷新后重试"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"id": req.ID, "status": "canceled"}})
}

func CreateAppointmentCancelRequest(c *gin.Context) {
	appointmentID64, err := strconv.ParseUint(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || appointmentID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的预约ID"})
		return
	}
	appointment, err := loadAppointmentByID(config.DB, uint(appointmentID64))
	if err != nil || appointment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}
	authType := strings.TrimSpace(c.GetString("auth_type"))
	if authType != "merchant" && authType != "staff" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	if _, _, ok := checkMerchantAppointmentOwnership(c, *appointment); !ok {
		return
	}
	if appointment.Status != "pending" && appointment.Status != "confirmed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前预约状态不可发起取消申请"})
		return
	}
	if start, _, ok := appointmentReservedWindow(*appointment); !ok || time.Now().In(appointmentLocation()).Before(start.Add(-5*time.Hour)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "距预约开始超过5小时，当前可直接取消，无需发起取消申请"})
		return
	}
	var input struct {
		Reason string `json:"reason" binding:"required,max=255"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "取消原因不能为空"})
		return
	}
	actorType, actorID := getAppointmentActor(c)
	var req models.AppointmentCancelRequest
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		current, err := loadAppointmentByID(tx, appointment.ID)
		if err != nil {
			return err
		}
		if current == nil {
			return gorm.ErrRecordNotFound
		}
		if current.Status != "pending" && current.Status != "confirmed" {
			return apiErr{status: http.StatusBadRequest, msg: "当前预约状态不可发起取消申请"}
		}
		if exists, err := hasPendingCancelRequest(tx, current.ID); err != nil {
			return err
		} else if exists {
			return apiErr{status: http.StatusBadRequest, msg: "当前预约已有待处理的取消申请"}
		}
		req = models.AppointmentCancelRequest{
			AppointmentID:  current.ID,
			MerchantID:     current.MerchantID,
			UserID:         current.UserID,
			Status:         "pending_user",
			Reason:         reason,
			ProposedByType: actorType,
			ProposedByID:   actorID,
		}
		return tx.Create(&req).Error
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
	c.JSON(http.StatusOK, gin.H{"data": req})
}

func AcceptAppointmentCancelRequest(c *gin.Context) {
	requestID64, err := strconv.ParseUint(strings.TrimSpace(c.Param("request_id")), 10, 64)
	if err != nil || requestID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的取消申请ID"})
		return
	}
	req, err := loadAppointmentCancelRequestByID(config.DB, uint(requestID64))
	if err != nil || req == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "取消申请不存在"})
		return
	}
	appointment, err := loadAppointmentByID(config.DB, req.AppointmentID)
	if err != nil || appointment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}
	authUserID, ok := mustUserID(c)
	if !ok {
		return
	}
	if appointment.UserID != authUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此取消申请"})
		return
	}
	if req.Status != "pending_user" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前取消申请无需用户确认"})
		return
	}
	actorType, actorID := getAppointmentActor(c)
	now := time.Now().In(appointmentLocation())
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		currentReq, err := loadAppointmentCancelRequestByID(tx, req.ID)
		if err != nil {
			return err
		}
		if currentReq == nil {
			return gorm.ErrRecordNotFound
		}
		if currentReq.Status != req.Status {
			return apiErr{status: http.StatusBadRequest, msg: "取消申请状态已变化，请刷新后重试"}
		}
		currentAppt, err := loadAppointmentByID(tx, currentReq.AppointmentID)
		if err != nil {
			return err
		}
		if currentAppt == nil {
			return gorm.ErrRecordNotFound
		}
		if err := tx.Model(&models.Appointment{}).Where("id = ?", currentAppt.ID).Updates(map[string]interface{}{
			"status":                     "canceled",
			"canceled_at":                &now,
			"closed_reason":              "canceled",
			"closed_by_type":             actorType,
			"closed_by_id":               actorID,
			"settlement_status_snapshot": "refunded",
			"resolution_note":            appendAppointmentResolutionNote(currentAppt.ResolutionNote, "取消申请已同意："+currentReq.Reason),
		}).Error; err != nil {
			return err
		}
		currentAppt.Status = "canceled"
		currentAppt.CanceledAt = &now
		if err := updateAppointmentSettlementSnapshot(tx, currentAppt, "refunded", "appointment_cancel_request_accepted"); err != nil {
			return err
		}
		return tx.Model(&models.AppointmentCancelRequest{}).Where("id = ? AND status = ?", currentReq.ID, currentReq.Status).Updates(map[string]interface{}{
			"status":            "accepted",
			"confirmed_by_type": actorType,
			"confirmed_by_id":   actorID,
			"confirmed_at":      &now,
		}).Error
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
	if reloaded, err := loadAppointmentByID(config.DB, appointment.ID); err == nil && reloaded != nil {
		appointment = reloaded
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"id": req.ID, "status": "accepted", "appointment": appointment}})
}

func RejectAppointmentCancelRequest(c *gin.Context) {
	requestID64, err := strconv.ParseUint(strings.TrimSpace(c.Param("request_id")), 10, 64)
	if err != nil || requestID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的取消申请ID"})
		return
	}
	req, err := loadAppointmentCancelRequestByID(config.DB, uint(requestID64))
	if err != nil || req == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "取消申请不存在"})
		return
	}
	appointment, err := loadAppointmentByID(config.DB, req.AppointmentID)
	if err != nil || appointment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}
	authUserID, ok := mustUserID(c)
	if !ok {
		return
	}
	if appointment.UserID != authUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此取消申请"})
		return
	}
	if req.Status != "pending_user" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前取消申请无需用户处理"})
		return
	}
	var input struct {
		ObjectionNote string `json:"objection_note" binding:"max=255"`
	}
	_ = c.ShouldBindJSON(&input)
	actorType, actorID := getAppointmentActor(c)
	now := time.Now().In(appointmentLocation())
	result := config.DB.Model(&models.AppointmentCancelRequest{}).
		Where("id = ? AND status = ?", req.ID, req.Status).
		Updates(map[string]interface{}{
			"status":            "rejected",
			"confirmed_by_type": actorType,
			"confirmed_by_id":   actorID,
			"confirmed_at":      &now,
			"objection_note":    strings.TrimSpace(input.ObjectionNote),
		})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "拒绝取消申请失败"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "取消申请状态已变化，请刷新后重试"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"id": req.ID, "status": "rejected"}})
}

func CreateForceMajeureReliefRequest(c *gin.Context) {
	appointmentID64, err := strconv.ParseUint(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || appointmentID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的预约ID"})
		return
	}
	appointment, err := loadAppointmentByID(config.DB, uint(appointmentID64))
	if err != nil || appointment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}
	authType := strings.TrimSpace(c.GetString("auth_type"))
	switch authType {
	case "merchant", "staff":
		if _, _, ok := checkMerchantAppointmentOwnership(c, *appointment); !ok {
			return
		}
	case "user":
		authUserID, ok := mustUserID(c)
		if !ok {
			return
		}
		if appointment.UserID != authUserID {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此预约"})
			return
		}
	default:
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	var input struct {
		Reason       string `json:"reason" binding:"required,max=255"`
		EvidenceNote string `json:"evidence_note" binding:"max=255"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.Reason = strings.TrimSpace(input.Reason)
	input.EvidenceNote = strings.TrimSpace(input.EvidenceNote)
	if input.Reason == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不可抗力原因不能为空"})
		return
	}

	actorType, actorID := getAppointmentActor(c)
	var req models.ForceMajeureReliefRequest
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		current, err := loadAppointmentByID(tx, appointment.ID)
		if err != nil {
			return err
		}
		if current == nil {
			return gorm.ErrRecordNotFound
		}
		if exists, err := hasPendingForceMajeureReliefRequest(tx, current.ID); err != nil {
			return err
		} else if exists {
			return apiErr{status: http.StatusBadRequest, msg: "当前预约已有待处理的不可抗力申请"}
		}
		compensations, err := listActiveForceMajeureEligibleCompensations(tx, current.ID)
		if err != nil {
			return err
		}
		if len(compensations) == 0 {
			return apiErr{status: http.StatusBadRequest, msg: "当前预约没有可撤销的违约补偿"}
		}
		req = models.ForceMajeureReliefRequest{
			AppointmentID:  current.ID,
			MerchantID:     current.MerchantID,
			UserID:         current.UserID,
			ProposedByType: actorType,
			ProposedByID:   actorID,
			Reason:         input.Reason,
			EvidenceNote:   input.EvidenceNote,
			Status:         "pending",
		}
		return tx.Create(&req).Error
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
	c.JSON(http.StatusOK, gin.H{"data": req})
}

func AcceptForceMajeureReliefRequest(c *gin.Context) {
	requestID64, err := strconv.ParseUint(strings.TrimSpace(c.Param("request_id")), 10, 64)
	if err != nil || requestID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的不可抗力申请ID"})
		return
	}
	req, err := loadForceMajeureReliefRequestByID(config.DB, uint(requestID64))
	if err != nil || req == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "不可抗力申请不存在"})
		return
	}
	appointment, err := loadAppointmentByID(config.DB, req.AppointmentID)
	if err != nil || appointment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}
	authType := strings.TrimSpace(c.GetString("auth_type"))
	switch authType {
	case "merchant", "staff":
		if _, _, ok := checkMerchantAppointmentOwnership(c, *appointment); !ok {
			return
		}
		if strings.TrimSpace(req.ProposedByType) == "merchant" || strings.TrimSpace(req.ProposedByType) == "staff" {
			c.JSON(http.StatusForbidden, gin.H{"error": "发起方不能确认自己的不可抗力申请"})
			return
		}
	case "user":
		authUserID, ok := mustUserID(c)
		if !ok {
			return
		}
		if appointment.UserID != authUserID {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此不可抗力申请"})
			return
		}
		if strings.TrimSpace(req.ProposedByType) == "user" {
			c.JSON(http.StatusForbidden, gin.H{"error": "发起方不能确认自己的不可抗力申请"})
			return
		}
	default:
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	if req.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前不可抗力申请不可确认"})
		return
	}

	actorType, actorID := getAppointmentActor(c)
	now := time.Now().In(appointmentLocation())
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		currentReq, err := loadForceMajeureReliefRequestByID(tx, req.ID)
		if err != nil {
			return err
		}
		if currentReq == nil {
			return gorm.ErrRecordNotFound
		}
		if currentReq.Status != req.Status {
			return apiErr{status: http.StatusBadRequest, msg: "不可抗力申请状态已变化，请刷新后重试"}
		}
		compensations, err := listActiveForceMajeureEligibleCompensations(tx, currentReq.AppointmentID)
		if err != nil {
			return err
		}
		if len(compensations) == 0 {
			return apiErr{status: http.StatusBadRequest, msg: "当前预约没有可撤销的违约补偿"}
		}
		for i := range compensations {
			if !forceMajeureEligibleCompensationSource(compensations[i].SourceType) {
				continue
			}
			if err := reverseForceMajeureCompensation(tx, &compensations[i]); err != nil {
				return err
			}
			if err := tx.Model(&models.AppointmentCompensation{}).Where("id = ? AND status = ?", compensations[i].ID, "applied").Updates(map[string]interface{}{
				"status":                          "canceled",
				"force_majeure_relief_request_id": currentReq.ID,
				"remark":                          appendAppointmentResolutionNote(compensations[i].Remark, "不可抗力撤销"),
			}).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&models.ForceMajeureReliefRequest{}).Where("id = ? AND status = ?", currentReq.ID, currentReq.Status).Updates(map[string]interface{}{
			"status":            "accepted",
			"confirmed_by_type": actorType,
			"confirmed_by_id":   actorID,
			"confirmed_at":      &now,
		}).Error; err != nil {
			return err
		}
		currentAppt, err := loadAppointmentByID(tx, currentReq.AppointmentID)
		if err != nil {
			return err
		}
		if currentAppt != nil {
			return tx.Model(&models.Appointment{}).Where("id = ?", currentAppt.ID).Updates(map[string]interface{}{
				"resolution_note": appendAppointmentResolutionNote(currentAppt.ResolutionNote, "不可抗力撤销违约补偿已确认："+currentReq.Reason),
			}).Error
		}
		return nil
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
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"id": req.ID, "status": "accepted"}})
}

func RejectForceMajeureReliefRequest(c *gin.Context) {
	requestID64, err := strconv.ParseUint(strings.TrimSpace(c.Param("request_id")), 10, 64)
	if err != nil || requestID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的不可抗力申请ID"})
		return
	}
	req, err := loadForceMajeureReliefRequestByID(config.DB, uint(requestID64))
	if err != nil || req == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "不可抗力申请不存在"})
		return
	}
	appointment, err := loadAppointmentByID(config.DB, req.AppointmentID)
	if err != nil || appointment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}
	authType := strings.TrimSpace(c.GetString("auth_type"))
	switch authType {
	case "merchant", "staff":
		if _, _, ok := checkMerchantAppointmentOwnership(c, *appointment); !ok {
			return
		}
		if strings.TrimSpace(req.ProposedByType) == "merchant" || strings.TrimSpace(req.ProposedByType) == "staff" {
			c.JSON(http.StatusForbidden, gin.H{"error": "发起方不能拒绝自己的不可抗力申请"})
			return
		}
	case "user":
		authUserID, ok := mustUserID(c)
		if !ok {
			return
		}
		if appointment.UserID != authUserID {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此不可抗力申请"})
			return
		}
		if strings.TrimSpace(req.ProposedByType) == "user" {
			c.JSON(http.StatusForbidden, gin.H{"error": "发起方不能拒绝自己的不可抗力申请"})
			return
		}
	default:
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	if req.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前不可抗力申请不可拒绝"})
		return
	}
	actorType, actorID := getAppointmentActor(c)
	now := time.Now().In(appointmentLocation())
	result := config.DB.Model(&models.ForceMajeureReliefRequest{}).
		Where("id = ? AND status = ?", req.ID, req.Status).
		Updates(map[string]interface{}{
			"status":            "rejected",
			"confirmed_by_type": actorType,
			"confirmed_by_id":   actorID,
			"confirmed_at":      &now,
		})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "拒绝不可抗力申请失败"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不可抗力申请状态已变化，请刷新后重试"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"id": req.ID, "status": "rejected"}})
}

func RescheduleAppointment(c *gin.Context) {
	CreateAppointmentRescheduleRequest(c)
}

func GetMerchantAppointmentSettlement(c *gin.Context) {
	appointmentID64, err := strconv.ParseUint(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || appointmentID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的预约ID"})
		return
	}
	appointment, err := loadAppointmentByID(config.DB, uint(appointmentID64))
	if err != nil || appointment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}
	if _, _, ok := checkMerchantAppointmentOwnership(c, *appointment); !ok {
		return
	}
	if appointment.AppointmentSettlementID == nil || *appointment.AppointmentSettlementID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约结算不存在"})
		return
	}
	settlement, err := loadAppointmentSettlementByID(config.DB, *appointment.AppointmentSettlementID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取预约结算失败"})
		return
	}
	if settlement == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约结算不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": settlement})
}

func GetUserAppointmentSettlement(c *gin.Context) {
	appointmentID64, err := strconv.ParseUint(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || appointmentID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的预约ID"})
		return
	}
	appointment, err := loadAppointmentByID(config.DB, uint(appointmentID64))
	if err != nil || appointment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}
	authUserID, ok := mustUserID(c)
	if !ok {
		return
	}
	if appointment.UserID != authUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权查看此预约结算"})
		return
	}
	if appointment.AppointmentSettlementID == nil || *appointment.AppointmentSettlementID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约结算不存在"})
		return
	}
	settlement, err := loadAppointmentSettlementByID(config.DB, *appointment.AppointmentSettlementID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取预约结算失败"})
		return
	}
	if settlement == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约结算不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": settlement})
}

func ListMerchantAppointmentDelayLedgers(c *gin.Context) {
	appointmentID64, err := strconv.ParseUint(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || appointmentID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的预约ID"})
		return
	}
	appointment, err := loadAppointmentByID(config.DB, uint(appointmentID64))
	if err != nil || appointment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}
	if _, _, ok := checkMerchantAppointmentOwnership(c, *appointment); !ok {
		return
	}
	var ledgers []models.AppointmentDelayLedger
	if err := config.DB.Where("appointment_id = ?", appointment.ID).Order("id DESC").Find(&ledgers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取拖堂账本失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": ledgers})
}

func ListUserCardDelayLedgers(c *gin.Context) {
	cardID64, err := strconv.ParseUint(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || cardID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的卡ID"})
		return
	}
	authUserID, ok := mustUserID(c)
	if !ok {
		return
	}
	var card models.Card
	if err := config.DB.First(&card, uint(cardID64)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "卡片不存在"})
		return
	}
	if card.UserID != authUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权查看此卡的拖堂账本"})
		return
	}
	var ledgers []models.AppointmentDelayLedger
	if err := config.DB.Where("card_id = ?", card.ID).Order("id DESC").Find(&ledgers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取拖堂账本失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": ledgers})
}

func GetMerchantAppointmentCompensationSummary(c *gin.Context) {
	appointmentID64, err := strconv.ParseUint(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || appointmentID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的预约ID"})
		return
	}
	appointment, err := loadAppointmentByID(config.DB, uint(appointmentID64))
	if err != nil || appointment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}
	if _, _, ok := checkMerchantAppointmentOwnership(c, *appointment); !ok {
		return
	}
	var ledgers []models.AppointmentDelayLedger
	if err := config.DB.Where("appointment_id = ?", appointment.ID).Order("id DESC").Find(&ledgers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取拖堂账本失败"})
		return
	}
	var compensations []models.AppointmentCompensation
	if err := config.DB.Where("appointment_id = ?", appointment.ID).Order("id DESC").Find(&compensations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取补偿记录失败"})
		return
	}
	totalDelayMinutes := 0
	totalCreditedMinutes := 0
	for _, ledger := range ledgers {
		totalDelayMinutes += ledger.DelayMinutes
		totalCreditedMinutes += ledger.CreditedMinutes
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"appointment_id":         appointment.ID,
		"delay_ledgers":          ledgers,
		"compensations":          compensations,
		"total_delay_minutes":    totalDelayMinutes,
		"total_credited_minutes": totalCreditedMinutes,
		"settlement_status":      appointment.SettlementStatusSnapshot,
		"liability_level":        appointment.LiabilityLevel,
	}})
}

func ListAppointmentCompensations(c *gin.Context) {
	id := c.Param("id")
	appointmentID64, err := strconv.ParseUint(strings.TrimSpace(id), 10, 64)
	if err != nil || appointmentID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的预约ID"})
		return
	}
	appointment, err := loadAppointmentByID(config.DB, uint(appointmentID64))
	if err != nil || appointment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}
	if _, _, ok := checkMerchantAppointmentOwnership(c, *appointment); !ok {
		return
	}
	var compensations []models.AppointmentCompensation
	if err := config.DB.Where("appointment_id = ?", appointment.ID).Order("id DESC").Find(&compensations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取补偿记录失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": compensations})
}

func CreateAppointmentCompensation(c *gin.Context) {
	id := c.Param("id")
	appointmentID64, err := strconv.ParseUint(strings.TrimSpace(id), 10, 64)
	if err != nil || appointmentID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的预约ID"})
		return
	}
	appointment, err := loadAppointmentByID(config.DB, uint(appointmentID64))
	if err != nil || appointment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}
	if _, _, ok := checkMerchantAppointmentOwnership(c, *appointment); !ok {
		return
	}

	var input struct {
		Type   string `json:"type" binding:"required"`
		Value  int    `json:"value"`
		Reason string `json:"reason" binding:"required,max=255"`
		Remark string `json:"remark" binding:"max=255"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input.Type = strings.TrimSpace(input.Type)
	input.Reason = strings.TrimSpace(input.Reason)
	input.Remark = strings.TrimSpace(input.Remark)
	if input.Reason == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "补偿原因不能为空"})
		return
	}

	switch input.Type {
	case "extra_times":
		if input.Value <= 0 || input.Value > 20 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "补偿次数范围应为 1-20"})
			return
		}
	case "extend_minutes":
		if input.Value < 5 || input.Value > 180 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "补偿时长范围应为 5-180 分钟"})
			return
		}
	case "discount_note", "other_note":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的补偿类型"})
		return
	}

	actorType, actorID := getAppointmentActor(c)
	now := time.Now()
	var compensation models.AppointmentCompensation
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		currentPtr, err := loadAppointmentByID(tx, appointment.ID)
		if err != nil {
			return err
		}
		if currentPtr == nil {
			return gorm.ErrRecordNotFound
		}
		current := *currentPtr
		compensation = models.AppointmentCompensation{
			AppointmentID:    current.ID,
			MerchantID:       current.MerchantID,
			UserID:           current.UserID,
			CardID:           current.CardID,
			ServiceSessionID: current.ServiceSessionID,
			Type:             input.Type,
			Value:            input.Value,
			Status:           "pending",
			Reason:           input.Reason,
			Remark:           input.Remark,
			CreatedByType:    actorType,
			CreatedByID:      actorID,
		}
		if err := tx.Create(&compensation).Error; err != nil {
			return err
		}

		// 不同补偿类型直接驱动对应资源：
		// 1. extra_times 立刻回充卡次数
		// 2. extend_minutes 直接延长当前服务会话
		// 3. discount_note / other_note 先形成正式补偿记录，供线下履约和后续稽核
		switch input.Type {
		case "extra_times":
			if err := tx.Model(&models.Card{}).Where("id = ?", current.CardID).Updates(map[string]interface{}{
				"total_times":  gorm.Expr("total_times + ?", input.Value),
				"remain_times": gorm.Expr("remain_times + ?", input.Value),
			}).Error; err != nil {
				return err
			}
		case "extend_minutes":
			if current.ServiceSessionID == nil {
				return apiErr{status: http.StatusBadRequest, msg: "当前预约没有关联服务会话，不能补时"}
			}
			var session models.ServiceSession
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", *current.ServiceSessionID).First(&session).Error; err != nil {
				return err
			}
			baseStatus := models.NormalizeSessionStatus(session.Status)
			if baseStatus != "serving" && baseStatus != "start_pending" && baseStatus != "delay_pending" && baseStatus != "appointment_waiting" {
				return apiErr{status: http.StatusBadRequest, msg: "当前阶段不能补时"}
			}
			updates := map[string]interface{}{
				"duration_minutes": gorm.Expr("duration_minutes + ?", input.Value),
			}
			if session.ScheduledFinishAt != nil {
				updates["scheduled_finish_at"] = session.ScheduledFinishAt.Add(time.Duration(input.Value) * time.Minute)
			}
			if err := tx.Model(&models.ServiceSession{}).Where("id = ?", session.ID).Updates(updates).Error; err != nil {
				return err
			}
		}

		noteText := compensationDisplayText(input.Type, input.Value, input.Remark)
		if err := tx.Model(&models.Appointment{}).Where("id = ?", current.ID).Updates(map[string]interface{}{
			"resolution_note": appendAppointmentResolutionNote(current.ResolutionNote, noteText),
		}).Error; err != nil {
			return err
		}

		compensation.Status = "applied"
		compensation.AppliedAt = &now
		return tx.Model(&models.AppointmentCompensation{}).Where("id = ?", compensation.ID).Updates(map[string]interface{}{
			"status":     "applied",
			"applied_at": &now,
		}).Error
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
	c.JSON(http.StatusOK, gin.H{"data": compensation})
}

func CancelAppointment(c *gin.Context) {
	id := c.Param("id")
	appointmentID64, err := strconv.ParseUint(strings.TrimSpace(id), 10, 64)
	if err != nil || appointmentID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的预约ID"})
		return
	}
	appointment, err := loadAppointmentByID(config.DB, uint(appointmentID64))
	if err != nil || appointment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}

	authTypeAny, _ := c.Get("auth_type")
	authType, _ := authTypeAny.(string)
	if authType == "merchant" || authType == "staff" {
		if _, _, ok := checkMerchantAppointmentOwnership(c, *appointment); !ok {
			return
		}
	} else {
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

	now := time.Now().In(appointmentLocation())
	merchantCancelReason := ""
	actorType, actorID := getAppointmentActor(c)
	if authType == "merchant" || authType == "staff" {
		if start, _, ok := appointmentReservedWindow(*appointment); ok && !now.Before(start.Add(-5*time.Hour)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "距预约开始不足5小时，商户当前只能发起取消申请"})
			return
		}
		var merchant models.Merchant
		if err := config.DB.First(&merchant, appointment.MerchantID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取商户信息失败"})
			return
		}
		repairEval, err := evaluateAppointmentRepairOptions(config.DB, merchant, *appointment)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取异常修复信息失败"})
			return
		}
		if repairEval.AffectedByLeave {
			if repairEval.HasHighQualityCandidate {
				c.JSON(http.StatusBadRequest, gin.H{"error": "该预约存在高质量修复方案，需优先发起保护性改签"})
				return
			}
		}
		var input struct {
			Reason string `json:"reason" binding:"required,max=255"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		merchantCancelReason = strings.TrimSpace(input.Reason)
		if merchantCancelReason == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "商户取消原因不能为空"})
			return
		}
	}
	if err := cancelAppointmentWithTime(appointment, now); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "取消预约失败"})
		return
	}
	if merchantCancelReason != "" {
		resolutionNote := appendAppointmentResolutionNote(appointment.ResolutionNote, "商户取消预约："+merchantCancelReason)
		if err := config.DB.Model(&models.Appointment{}).Where("id = ?", appointment.ID).Updates(map[string]interface{}{
			"merchant_cancel_reason": merchantCancelReason,
			"closed_reason":          "canceled",
			"closed_by_type":         actorType,
			"closed_by_id":           actorID,
			"resolution_note":        resolutionNote,
		}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存取消原因失败"})
			return
		}
		appointment.MerchantCancelReason = merchantCancelReason
		appointment.ClosedReason = "canceled"
		appointment.ClosedByType = actorType
		appointment.ClosedByID = actorID
		appointment.ResolutionNote = resolutionNote
		if appointment.AppointmentSettlementID != nil && *appointment.AppointmentSettlementID > 0 {
			_ = config.DB.Model(&models.AppointmentSettlement{}).Where("id = ?", *appointment.AppointmentSettlementID).Update("latest_reason", "merchant_direct_cancel").Error
		}
	}
	appointment.Status = "canceled"
	appointment.CanceledAt = &now
	c.JSON(http.StatusOK, gin.H{"data": appointment})
}

func appointmentReservedWindow(appt models.Appointment) (time.Time, time.Time, bool) {
	if appt.AppointmentTime == nil {
		return time.Time{}, time.Time{}, false
	}
	start := *appt.AppointmentTime
	if appt.ReservedStartAt != nil {
		start = *appt.ReservedStartAt
	}
	end := start.Add(time.Duration(getAppointmentServiceMinutes(appt.MerchantID, appt)) * time.Minute)
	if appt.ReservedEndAt != nil && appt.ReservedEndAt.After(start) {
		end = *appt.ReservedEndAt
	}
	return start, end, end.After(start)
}

func appointmentOccupiedWindow(appt models.Appointment) (time.Time, time.Time, bool) {
	start, _, ok := appointmentReservedWindow(appt)
	if !ok {
		return time.Time{}, time.Time{}, false
	}
	end := start.Add(time.Duration(getAppointmentOccupiedMinutes(appt.MerchantID, appt)) * time.Minute)
	if appt.OccupiedEndAt != nil && appt.OccupiedEndAt.After(start) {
		end = *appt.OccupiedEndAt
	}
	return start, end, end.After(start)
}

func loadAffectedAppointmentsForPublishing(tx *gorm.DB, publishing models.TechnicianSchedulePublishing) ([]models.Appointment, error) {
	if tx == nil || publishing.MerchantID == 0 || publishing.StartAt == nil || publishing.EndAt == nil || !publishing.EndAt.After(*publishing.StartAt) {
		return nil, nil
	}
	dayStart := time.Date(publishing.StartAt.Year(), publishing.StartAt.Month(), publishing.StartAt.Day(), 0, 0, 0, 0, publishing.StartAt.Location())
	dayEnd := dayStart.Add(24 * time.Hour)
	q := tx.Table("appointments").
		Select(appointmentSelectColumns()).
		Where("merchant_id = ? AND appointment_time IS NOT NULL AND appointment_time >= ? AND appointment_time < ? AND status IN ?", publishing.MerchantID, dayStart, dayEnd, appointmentProtectedStatuses())
	if publishing.TechnicianID != nil && *publishing.TechnicianID > 0 {
		q = q.Where("technician_id = ?", *publishing.TechnicianID)
	}
	rows, err := q.Order("appointment_time asc").Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]models.Appointment, 0)
	for rows.Next() {
		appt, err := scanAppointmentRow(rows)
		if err != nil {
			return nil, err
		}
		start, end, ok := appointmentOccupiedWindow(appt)
		if !ok {
			continue
		}
		if start.Before(*publishing.EndAt) && publishing.StartAt.Before(end) {
			list = append(list, appt)
		}
	}
	return list, nil
}

func createProtectedRepairSlotForAppointment(tx *gorm.DB, publishing models.TechnicianSchedulePublishing, appt models.Appointment) error {
	if tx == nil || publishing.MerchantID == 0 || appt.ID == 0 {
		return nil
	}
	start, end, ok := appointmentReservedWindow(appt)
	if !ok {
		return nil
	}
	var count int64
	if err := tx.Model(&models.ProtectedRepairSlot{}).
		Where("merchant_id = ? AND appointment_id = ? AND status = ?", publishing.MerchantID, appt.ID, "reserved").
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	row := models.ProtectedRepairSlot{
		MerchantID:    publishing.MerchantID,
		AppointmentID: appt.ID,
		TechnicianID:  appt.TechnicianID,
		PublishDate:   publishing.PublishDate,
		StartAt:       &start,
		EndAt:         &end,
		Status:        "reserved",
		SourceType:    "leave",
	}
	return tx.Create(&row).Error
}

func PublishNextDaySchedule(c *gin.Context) {
	if !requireAnyMerchantPermissionInHandler(c, "merchant.appointment.manage", "merchant.service.manage", "merchant.cs.manage") {
		return
	}
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}
	var merchant models.Merchant
	if err := config.DB.First(&merchant, merchantID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}
	loc := appointmentLocation()
	now := time.Now().In(loc)
	nextDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).Add(24 * time.Hour)
	intervals, ok := getMerchantBusinessIntervalsForDate(merchant, nextDate)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "商户未设置营业时间"})
		return
	}
	publishedAt := now
	rows := make([]models.TechnicianSchedulePublishing, 0)
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("merchant_id = ? AND publish_date >= ? AND publish_date < ?", merchantID, nextDate, nextDate.Add(24*time.Hour)).Delete(&models.TechnicianSchedulePublishing{}).Error; err != nil {
			return err
		}
		technicians := make([]models.Technician, 0)
		if merchant.SupportCustomerServiceMode {
			list, err := listAppointmentBookableTechnicians(tx, merchant)
			if err != nil {
				return err
			}
			technicians = list
		}
		if len(technicians) == 0 {
			for _, it := range intervals {
				rows = append(rows, models.TechnicianSchedulePublishing{MerchantID: merchantID, PublishDate: &nextDate, StartAt: &it.Start, EndAt: &it.End, Status: "published", PublishedAt: &publishedAt})
			}
		} else {
			for _, tech := range technicians {
				for _, it := range intervals {
					techID := tech.ID
					rows = append(rows, models.TechnicianSchedulePublishing{MerchantID: merchantID, TechnicianID: &techID, PublishDate: &nextDate, StartAt: &it.Start, EndAt: &it.End, Status: "published", PublishedAt: &publishedAt})
				}
			}
		}
		if len(rows) == 0 {
			return apiErr{status: http.StatusBadRequest, msg: "没有可发布的排班"}
		}
		return tx.Create(&rows).Error
	})
	if err != nil {
		var ae apiErr
		if errors.As(err, &ae) {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "发布次日排班失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"publish_date": nextDate.Format("2006-01-02"), "items": rows}})
}

func MarkScheduleLeave(c *gin.Context) {
	if !requireAnyMerchantPermissionInHandler(c, "merchant.appointment.manage", "merchant.service.manage", "merchant.cs.manage") {
		return
	}
	id64, err := strconv.ParseUint(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的排班ID"})
		return
	}
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}
	publishing, err := loadSchedulePublishingByID(config.DB, merchantID, uint(id64))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取排班失败"})
		return
	}
	if publishing == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "排班不存在"})
		return
	}
	var affected []models.Appointment
	var inspections []appointmentRepairInspection
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.TechnicianSchedulePublishing{}).Where("id = ? AND merchant_id = ?", publishing.ID, merchantID).Update("status", "leave").Error; err != nil {
			return err
		}
		publishing.Status = "leave"
		list, err := loadAffectedAppointmentsForPublishing(tx, *publishing)
		if err != nil {
			return err
		}
		affected = list
		for _, appt := range affected {
			if err := tx.Model(&models.Appointment{}).Where("id = ?", appt.ID).Updates(map[string]interface{}{
				"disruption_status": "pending",
				"disruption_reason": "technician_leave",
			}).Error; err != nil {
				return err
			}
			if err := createProtectedRepairSlotForAppointment(tx, *publishing, appt); err != nil {
				return err
			}
		}
		repairItems, err := buildAppointmentRepairInspections(tx, merchantID, affected)
		if err != nil {
			return err
		}
		inspections = repairItems
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "排班请假标记失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"schedule":                     publishing,
		"affected_appointments":        affected,
		"affected_appointment_repairs": inspections,
	}})
}

func ListSchedulePublishings(c *gin.Context) {
	if !requireAnyMerchantPermissionInHandler(c, "merchant.appointment.view", "merchant.appointment.manage", "merchant.service.manage", "merchant.cs.manage") {
		return
	}
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}
	loc := appointmentLocation()
	targetDate := strings.TrimSpace(c.Query("date"))
	var date time.Time
	var err error
	if targetDate == "" {
		now := time.Now().In(loc)
		date = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).Add(24 * time.Hour)
	} else {
		date, err = time.ParseInLocation("2006-01-02", targetDate, loc)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "日期格式错误，应为 YYYY-MM-DD"})
			return
		}
	}
	rows, err := loadSchedulePublishingsByDate(config.DB, merchantID, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取排班发布列表失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"date":        date.Format("2006-01-02"),
		"publishings": rows,
	}})
}

func GetScheduleAffectedAppointments(c *gin.Context) {
	if !requireAnyMerchantPermissionInHandler(c, "merchant.appointment.view", "merchant.appointment.manage", "merchant.service.manage", "merchant.cs.manage") {
		return
	}
	scheduleID64, err := strconv.ParseUint(strings.TrimSpace(c.Query("schedule_id")), 10, 64)
	if err != nil || scheduleID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少有效的 schedule_id"})
		return
	}
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}
	publishing, err := loadSchedulePublishingByID(config.DB, merchantID, uint(scheduleID64))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取排班失败"})
		return
	}
	if publishing == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "排班不存在"})
		return
	}
	affected, err := loadAffectedAppointmentsForPublishing(config.DB, *publishing)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取受影响预约失败"})
		return
	}
	repairSlots, err := loadProtectedRepairSlotsByPublishDate(config.DB, merchantID, publishing.PublishDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取保护修复槽失败"})
		return
	}
	inspections, err := buildAppointmentRepairInspections(config.DB, merchantID, affected)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取异常修复判定结果失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"schedule":                     publishing,
		"affected_appointments":        affected,
		"protected_repair_slots":       repairSlots,
		"affected_appointment_repairs": inspections,
	}})
}

func GetAppointmentRepairOverview(c *gin.Context) {
	appointmentID64, err := strconv.ParseUint(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || appointmentID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的预约ID"})
		return
	}
	appointment, err := loadAppointmentByID(config.DB, uint(appointmentID64))
	if err != nil || appointment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}
	if _, _, ok := checkMerchantAppointmentOwnership(c, *appointment); !ok {
		return
	}
	if appointment.AppointmentTime == nil || appointment.TechnicianID == nil || *appointment.TechnicianID == 0 {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{
			"schedule":                     nil,
			"affected_appointments":        []models.Appointment{},
			"protected_repair_slots":       []models.ProtectedRepairSlot{},
			"affected_appointment_repairs": []appointmentRepairInspection{},
			"reason":                       "当前预约未绑定客服排班，请假异常修复详情不可用",
		}})
		return
	}
	publishDate := appointment.AppointmentTime.In(appointmentLocation())
	target, err := loadLeaveSchedulePublishingByTechnicianAndDate(config.DB, appointment.MerchantID, *appointment.TechnicianID, publishDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取排班失败"})
		return
	}
	if target == nil {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{
			"schedule":                     nil,
			"affected_appointments":        []models.Appointment{},
			"protected_repair_slots":       []models.ProtectedRepairSlot{},
			"affected_appointment_repairs": []appointmentRepairInspection{},
			"reason":                       "当前预约未命中已确认请假的排班发布记录",
		}})
		return
	}
	affected, err := loadAffectedAppointmentsForPublishing(config.DB, *target)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取受影响预约失败"})
		return
	}
	repairSlots, err := loadProtectedRepairSlotsByPublishDate(config.DB, appointment.MerchantID, target.PublishDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取保护修复槽失败"})
		return
	}
	inspections, err := buildAppointmentRepairInspections(config.DB, appointment.MerchantID, affected)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取异常修复判定结果失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"schedule":                     target,
		"affected_appointments":        affected,
		"protected_repair_slots":       repairSlots,
		"affected_appointment_repairs": inspections,
		"reason":                       "",
	}})
}

func GetQueueStatus(c *gin.Context) {
	merchantID, ok := ensureMerchantScope(c, "id")
	if !ok {
		return
	}

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
	authType := c.GetString("auth_type")
	var merchantID uint
	var ok bool
	if authType == "user" {
		merchantID, ok = merchantIDFromRouteParam(c, "id")
	} else {
		merchantID, ok = ensureMerchantScope(c, "id")
	}
	if !ok {
		return
	}

	date := c.Query("date")
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

	loc := appointmentLocation()
	now := time.Now().In(loc)
	todayDate := now.Format("2006-01-02")
	tomorrowDate := now.Add(24 * time.Hour).Format("2006-01-02")
	if date == "" {
		if authType == "user" {
			date = tomorrowDate
		} else {
			date = todayDate
		}
	}
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
	if authType == "user" {
		if date != tomorrowDate {
			c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持预约明天"})
			return
		}
	} else {
		selectedDate, _ := time.ParseInLocation("2006-01-02", date, loc)
		if selectedDate.Before(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "不能查询过去的日期"})
			return
		}
	}

	payload, err := buildAvailableTimeSlotsPayload(merchant, merchantID, date, projectID, loc, nil)
	if err != nil {
		var ae apiErr
		if errors.As(err, &ae) {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取时间段失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": payload})
}
