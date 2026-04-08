package handlers

import (
	"fmt"
	"kabao/models"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	appointmentSchedulingModeGrouped       = "technician_grouped"
	appointmentSchedulingModeMixedTimeline = "technician_mixed_timeline"
)

type mixedTimelineCandidate struct {
	TechnicianID         uint
	Start                time.Time
	End                  time.Time
	OccupiedMinutes      int
	GapStart             time.Time
	GapEnd               time.Time
	LeftWasteMinutes     int
	RightWasteMinutes    int
	FragmentCost         int
	ScarcityPenalty      int
	WaitPenalty          int
	StartBias            int
	PlacementScore       int
	FitSummary           string
	PredictedWaitMinutes int
}

type technicianDayTimeline struct {
	MerchantID                    uint
	TechnicianID                  uint
	Date                          string
	GranularityMinutes            int
	SupplyIntervals               []businessInterval
	OccupiedIntervals             []appointmentOccupiedRange
	FreeIntervals                 []businessInterval
	AllowedProjectOccupiedMinutes []int
}

type mixedTimelineEngine struct {
	tx               *gorm.DB
	merchant         models.Merchant
	targetDate       time.Time
	projectID        uint
	selectedOccupied int
	granularity      int
	business         []businessInterval
	publishedRows    []models.TechnicianSchedulePublishing
	excludeID        uint

	allowedProjectsByTech map[uint]map[uint]struct{}
	allowedOccupiedByTech map[uint][]int
	timelineCache         map[uint]*technicianDayTimeline
}

func normalizeAppointmentSchedulingMode(raw string) string {
	mode := strings.TrimSpace(raw)
	if mode == "" {
		return appointmentSchedulingModeGrouped
	}
	if mode != appointmentSchedulingModeGrouped && mode != appointmentSchedulingModeMixedTimeline {
		return appointmentSchedulingModeGrouped
	}
	return mode
}

func merchantUsesMixedTimeline(merchant models.Merchant) bool {
	return merchant.SupportCustomerServiceMode && normalizeAppointmentSchedulingMode(merchant.AppointmentSchedulingMode) == appointmentSchedulingModeMixedTimeline
}

func loadTechnicianAllowedProjects(tx *gorm.DB, merchantID uint, technicianIDs []uint) (map[uint]map[uint]struct{}, error) {
	out := make(map[uint]map[uint]struct{}, len(technicianIDs))
	if tx == nil || merchantID == 0 || len(technicianIDs) == 0 {
		return out, nil
	}
	var rows []models.TechnicianAppointmentProject
	if err := tx.Where("merchant_id = ? AND technician_id IN ?", merchantID, technicianIDs).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		if _, ok := out[row.TechnicianID]; !ok {
			out[row.TechnicianID] = make(map[uint]struct{})
		}
		out[row.TechnicianID][row.ProjectID] = struct{}{}
	}
	return out, nil
}

func loadMerchantBookableProjects(tx *gorm.DB, merchantID uint) ([]models.MerchantProject, error) {
	if tx == nil || merchantID == 0 {
		return nil, nil
	}
	var projects []models.MerchantProject
	if err := tx.Where("merchant_id = ? AND is_active = ? AND bookable_online = ?", merchantID, true, true).Order("id asc").Find(&projects).Error; err != nil {
		return nil, err
	}
	return projects, nil
}

func buildAllowedOccupiedByTechnician(projects []models.MerchantProject, allowed map[uint]map[uint]struct{}, technicianIDs []uint) map[uint][]int {
	projectOccupied := make(map[uint]int, len(projects))
	merchantWide := make([]int, 0, len(projects))
	merchantSeen := make(map[int]struct{}, len(projects))
	for _, project := range projects {
		occupied := projectBookingOccupiedMinutes(project.Duration, project.ServiceGapMinutes)
		projectOccupied[project.ID] = occupied
		if _, ok := merchantSeen[occupied]; ok {
			continue
		}
		merchantSeen[occupied] = struct{}{}
		merchantWide = append(merchantWide, occupied)
	}
	sort.Ints(merchantWide)
	out := make(map[uint][]int, len(technicianIDs))
	for _, techID := range technicianIDs {
		projectIDs := allowed[techID]
		if len(projectIDs) == 0 {
			out[techID] = append([]int(nil), merchantWide...)
			continue
		}
		seen := make(map[int]struct{}, len(projectIDs))
		values := make([]int, 0, len(projectIDs))
		for projectID := range projectIDs {
			occupied, ok := projectOccupied[projectID]
			if !ok || occupied <= 0 {
				continue
			}
			if _, dup := seen[occupied]; dup {
				continue
			}
			seen[occupied] = struct{}{}
			values = append(values, occupied)
		}
		if len(values) == 0 {
			values = append(values, merchantWide...)
		}
		sort.Ints(values)
		out[techID] = values
	}
	return out
}

func newMixedTimelineEngine(tx *gorm.DB, merchant models.Merchant, targetDate time.Time, projectID uint, selectedOccupied int, publishedRows []models.TechnicianSchedulePublishing, excludeAppointmentID uint) (*mixedTimelineEngine, error) {
	business, ok := getMerchantBusinessIntervalsForDate(merchant, targetDate)
	if !ok {
		return nil, apiErr{status: 400, msg: "商户未设置营业时间"}
	}
	technicians, err := listAppointmentBookableTechnicians(tx, merchant)
	if err != nil {
		return nil, err
	}
	if len(technicians) > 0 {
		technicians = filterTechniciansByPublishedScheduleRows(technicians, publishedRows)
	}
	technicianIDs := make([]uint, 0, len(technicians))
	for _, tech := range technicians {
		technicianIDs = append(technicianIDs, tech.ID)
	}
	allowedProjects, err := loadTechnicianAllowedProjects(tx, merchant.ID, technicianIDs)
	if err != nil {
		return nil, err
	}
	projects, err := loadMerchantBookableProjects(tx, merchant.ID)
	if err != nil {
		return nil, err
	}
	return &mixedTimelineEngine{
		tx:                    tx,
		merchant:              merchant,
		targetDate:            targetDate,
		projectID:             projectID,
		selectedOccupied:      selectedOccupied,
		granularity:           merchantAppointmentSlotGranularityMinutes(&merchant),
		business:              business,
		publishedRows:         publishedRows,
		excludeID:             excludeAppointmentID,
		allowedProjectsByTech: allowedProjects,
		allowedOccupiedByTech: buildAllowedOccupiedByTechnician(projects, allowedProjects, technicianIDs),
		timelineCache:         make(map[uint]*technicianDayTimeline, len(technicianIDs)),
	}, nil
}

func (e *mixedTimelineEngine) technicianAllowsProject(technicianID uint) bool {
	if technicianID == 0 || e.projectID == 0 {
		return true
	}
	allowed := e.allowedProjectsByTech[technicianID]
	if len(allowed) == 0 {
		return true
	}
	_, ok := allowed[e.projectID]
	return ok
}

func intersectIntervals(a []businessInterval, b []businessInterval) []businessInterval {
	out := make([]businessInterval, 0)
	for _, ai := range a {
		for _, bi := range b {
			start := ai.Start
			if bi.Start.After(start) {
				start = bi.Start
			}
			end := ai.End
			if bi.End.Before(end) {
				end = bi.End
			}
			if end.After(start) {
				out = append(out, businessInterval{Start: start, End: end})
			}
		}
	}
	return out
}

func normalizeOccupiedRanges(ranges []appointmentOccupiedRange) []appointmentOccupiedRange {
	if len(ranges) == 0 {
		return nil
	}
	sort.Slice(ranges, func(i, j int) bool {
		if ranges[i].Start.Equal(ranges[j].Start) {
			return ranges[i].End.Before(ranges[j].End)
		}
		return ranges[i].Start.Before(ranges[j].Start)
	})
	out := make([]appointmentOccupiedRange, 0, len(ranges))
	for _, r := range ranges {
		if !r.End.After(r.Start) {
			continue
		}
		if len(out) == 0 {
			out = append(out, r)
			continue
		}
		last := &out[len(out)-1]
		if !r.Start.After(last.End) {
			if r.End.After(last.End) {
				last.End = r.End
			}
			continue
		}
		out = append(out, r)
	}
	return out
}

func subtractOccupiedFromIntervals(supply []businessInterval, occupied []appointmentOccupiedRange) []businessInterval {
	if len(supply) == 0 {
		return nil
	}
	if len(occupied) == 0 {
		return supply
	}
	out := make([]businessInterval, 0, len(supply))
	for _, src := range supply {
		cursor := src.Start
		for _, occ := range occupied {
			if !occ.End.After(src.Start) || !occ.Start.Before(src.End) {
				continue
			}
			if occ.Start.After(cursor) {
				out = append(out, businessInterval{Start: cursor, End: minTime(occ.Start, src.End)})
			}
			if occ.End.After(cursor) {
				cursor = occ.End
			}
			if !cursor.Before(src.End) {
				break
			}
		}
		if cursor.Before(src.End) {
			out = append(out, businessInterval{Start: cursor, End: src.End})
		}
	}
	filtered := out[:0]
	for _, it := range out {
		if it.End.After(it.Start) {
			filtered = append(filtered, it)
		}
	}
	return filtered
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func (e *mixedTimelineEngine) loadProtectedRepairRanges(technicianID uint) ([]appointmentOccupiedRange, error) {
	if e.tx == nil || technicianID == 0 {
		return nil, nil
	}
	dayStart := time.Date(e.targetDate.Year(), e.targetDate.Month(), e.targetDate.Day(), 0, 0, 0, 0, e.targetDate.Location())
	dayEnd := dayStart.Add(24 * time.Hour)
	rows, err := e.tx.Table("protected_repair_slots").
		Select("start_at, end_at").
		Where("merchant_id = ? AND technician_id = ? AND status = ? AND start_at IS NOT NULL AND end_at IS NOT NULL AND start_at >= ? AND start_at < ?", e.merchant.ID, technicianID, "reserved", dayStart, dayEnd).
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]appointmentOccupiedRange, 0)
	for rows.Next() {
		var start, end time.Time
		if err := rows.Scan(&start, &end); err != nil {
			return nil, err
		}
		if end.After(start) {
			out = append(out, appointmentOccupiedRange{Start: start, End: end})
		}
	}
	return out, nil
}

func (e *mixedTimelineEngine) timelineForTechnician(technicianID uint) (*technicianDayTimeline, error) {
	if cached, ok := e.timelineCache[technicianID]; ok {
		return cached, nil
	}
	schedule := filterScheduleRowsByTechnician(e.publishedRows, technicianID)
	supplyRows := make([]businessInterval, 0, len(schedule))
	for _, row := range schedule {
		if row.StartAt == nil || row.EndAt == nil {
			continue
		}
		supplyRows = append(supplyRows, businessInterval{Start: *row.StartAt, End: *row.EndAt})
	}
	supply := intersectIntervals(e.business, supplyRows)
	occupied, err := loadAppointmentOccupiedRangesForFragment(e.tx, e.merchant, e.targetDate, &technicianID, e.excludeID)
	if err != nil {
		return nil, err
	}
	repairRanges, err := e.loadProtectedRepairRanges(technicianID)
	if err != nil {
		return nil, err
	}
	occupied = append(occupied, repairRanges...)
	occupied = normalizeOccupiedRanges(occupied)
	timeline := &technicianDayTimeline{
		MerchantID:                    e.merchant.ID,
		TechnicianID:                  technicianID,
		Date:                          e.targetDate.Format("2006-01-02"),
		GranularityMinutes:            e.granularity,
		SupplyIntervals:               supply,
		OccupiedIntervals:             occupied,
		FreeIntervals:                 subtractOccupiedFromIntervals(supply, occupied),
		AllowedProjectOccupiedMinutes: append([]int(nil), e.allowedOccupiedByTech[technicianID]...),
	}
	e.timelineCache[technicianID] = timeline
	return timeline, nil
}

func (e *mixedTimelineEngine) locateFreeGap(technicianID uint, start time.Time, occupiedMinutes int) (businessInterval, bool, error) {
	timeline, err := e.timelineForTechnician(technicianID)
	if err != nil {
		return businessInterval{}, false, err
	}
	end := start.Add(time.Duration(occupiedMinutes) * time.Minute)
	for _, gap := range timeline.FreeIntervals {
		if (start.Equal(gap.Start) || start.After(gap.Start)) && (end.Equal(gap.End) || end.Before(gap.End)) {
			return gap, true, nil
		}
	}
	return businessInterval{}, false, nil
}

func nextGridTime(base, current time.Time, granularity int) time.Time {
	if granularity <= 0 {
		return current
	}
	if !current.After(base) {
		return base
	}
	diffMinutes := int(current.Sub(base).Minutes())
	if diffMinutes%granularity == 0 {
		return current
	}
	step := diffMinutes/granularity + 1
	return base.Add(time.Duration(step*granularity) * time.Minute)
}

func bestFillSegment(segment businessInterval, durations []int, granularity int) int {
	if !segment.End.After(segment.Start) || len(durations) == 0 || granularity <= 0 {
		return 0
	}
	points := make([]time.Time, 0)
	for p := segment.Start; !p.After(segment.End); p = p.Add(time.Duration(granularity) * time.Minute) {
		points = append(points, p)
	}
	if len(points) == 0 {
		return 0
	}
	dp := make([]int, len(points)+1)
	for i := len(points) - 1; i >= 0; i-- {
		best := dp[i+1]
		start := points[i]
		for _, d := range durations {
			if d <= 0 {
				continue
			}
			end := start.Add(time.Duration(d) * time.Minute)
			if end.After(segment.End) {
				continue
			}
			next := nextGridTime(segment.Start, end, granularity)
			j := len(points)
			for idx := i + 1; idx < len(points); idx++ {
				if !points[idx].Before(next) {
					j = idx
					break
				}
			}
			value := d + dp[j]
			if value > best {
				best = value
			}
		}
		dp[i] = best
	}
	return dp[0]
}

func countStartsInIntervals(intervals []businessInterval, duration, granularity int) int {
	if duration <= 0 || granularity <= 0 {
		return 0
	}
	total := 0
	for _, it := range intervals {
		latest := it.End.Add(-time.Duration(duration) * time.Minute)
		for start := it.Start; !start.After(latest); start = start.Add(time.Duration(granularity) * time.Minute) {
			total++
		}
	}
	return total
}

func removeCandidateFromIntervals(intervals []businessInterval, start time.Time, occupiedMinutes int) []businessInterval {
	end := start.Add(time.Duration(occupiedMinutes) * time.Minute)
	out := make([]businessInterval, 0, len(intervals)+1)
	for _, it := range intervals {
		if !end.After(it.Start) || !start.Before(it.End) {
			out = append(out, it)
			continue
		}
		if start.After(it.Start) {
			out = append(out, businessInterval{Start: it.Start, End: start})
		}
		if end.Before(it.End) {
			out = append(out, businessInterval{Start: end, End: it.End})
		}
	}
	return out
}

func longestDuration(durations []int) int {
	longest := 0
	for _, d := range durations {
		if d > longest {
			longest = d
		}
	}
	return longest
}

func (e *mixedTimelineEngine) scoreCandidate(technicianID uint, start time.Time, occupiedMinutes int, waitPenalty int) (*mixedTimelineCandidate, error) {
	gap, ok, err := e.locateFreeGap(technicianID, start, occupiedMinutes)
	if err != nil || !ok {
		return nil, err
	}
	timeline, err := e.timelineForTechnician(technicianID)
	if err != nil {
		return nil, err
	}
	left := businessInterval{Start: gap.Start, End: start}
	right := businessInterval{Start: start.Add(time.Duration(occupiedMinutes) * time.Minute), End: gap.End}
	leftLength := int(left.End.Sub(left.Start).Minutes())
	rightLength := int(right.End.Sub(right.Start).Minutes())
	leftBest := bestFillSegment(left, timeline.AllowedProjectOccupiedMinutes, e.granularity)
	rightBest := bestFillSegment(right, timeline.AllowedProjectOccupiedMinutes, e.granularity)
	leftWaste := leftLength - leftBest
	if leftWaste < 0 {
		leftWaste = 0
	}
	rightWaste := rightLength - rightBest
	if rightWaste < 0 {
		rightWaste = 0
	}
	fragmentCost := leftWaste + rightWaste
	beforeIntervals := timeline.FreeIntervals
	afterIntervals := removeCandidateFromIntervals(beforeIntervals, start, occupiedMinutes)
	longest := longestDuration(timeline.AllowedProjectOccupiedMinutes)
	scarcityPenalty := 0
	hideForUsers := false
	for _, d := range timeline.AllowedProjectOccupiedMinutes {
		beforeCount := countStartsInIntervals(beforeIntervals, d, e.granularity)
		afterCount := countStartsInIntervals(afterIntervals, d, e.granularity)
		if beforeCount > 2 && afterCount <= 2 {
			scarcityPenalty += 5
		}
		if beforeCount >= 1 && afterCount == 0 {
			scarcityPenalty += 20
			if d == longest {
				hideForUsers = true
			}
		}
	}
	startBias := start.Hour()*60 + start.Minute()
	score := fragmentCost*1000 + scarcityPenalty*100 + waitPenalty*10 + startBias
	fitSummary := fmt.Sprintf("gap=%s~%s,left_waste=%d,right_waste=%d", gap.Start.Format("15:04"), gap.End.Format("15:04"), leftWaste, rightWaste)
	if hideForUsers {
		fitSummary += ",hide_longest"
	}
	return &mixedTimelineCandidate{
		TechnicianID:         technicianID,
		Start:                start,
		End:                  start.Add(time.Duration(occupiedMinutes) * time.Minute),
		OccupiedMinutes:      occupiedMinutes,
		GapStart:             gap.Start,
		GapEnd:               gap.End,
		LeftWasteMinutes:     leftWaste,
		RightWasteMinutes:    rightWaste,
		FragmentCost:         fragmentCost,
		ScarcityPenalty:      scarcityPenalty,
		WaitPenalty:          waitPenalty,
		StartBias:            startBias,
		PlacementScore:       score,
		FitSummary:           fitSummary,
		PredictedWaitMinutes: waitPenalty,
	}, nil
}

func (e *mixedTimelineEngine) evaluateTechnicianCandidate(technicianID uint, start time.Time, occupiedMinutes int) (*mixedTimelineCandidate, string, error) {
	if !e.technicianAllowsProject(technicianID) {
		return nil, "当前客服未开放该预约项目", nil
	}
	end := start.Add(time.Duration(occupiedMinutes) * time.Minute)
	if !slotWithinBusinessIntervals(e.business, start, end) {
		return nil, "不在营业时间内", nil
	}
	if !slotWithinPublishedSchedule(e.publishedRows, start, end, &technicianID) {
		return nil, "未在已发布排班内", nil
	}
	availability, err := evaluateBookingTechnicianAvailability(e.tx, e.merchant, technicianID, start, occupiedMinutes, e.excludeID)
	if err != nil {
		return nil, "", err
	}
	if availability.State == appointmentAvailabilityUnavailable {
		return nil, availability.Reason, nil
	}
	candidate, err := e.scoreCandidate(technicianID, start, occupiedMinutes, availability.PredictedWaitMinutes)
	if err != nil {
		return nil, "", err
	}
	if strings.Contains(candidate.FitSummary, "hide_longest") {
		return nil, "该时段会占用当天最后一个长时长窗口", nil
	}
	return candidate, "", nil
}

func (e *mixedTimelineEngine) chooseBestAtTime(technicians []models.Technician, start time.Time, occupiedMinutes int, specifiedTechnicianID *uint) (appointmentPlacementDecision, []appointmentTechnicianCandidate, []uint, error) {
	decision := appointmentPlacementDecision{}
	candidates := make([]appointmentTechnicianCandidate, 0, len(technicians))
	availableTechIDs := make([]uint, 0, len(technicians))
	bestScore := int(^uint(0) >> 1)
	if specifiedTechnicianID != nil && *specifiedTechnicianID > 0 {
		candidate, reason, err := e.evaluateTechnicianCandidate(*specifiedTechnicianID, start, occupiedMinutes)
		if err != nil {
			return decision, nil, nil, err
		}
		if candidate == nil {
			return decision, []appointmentTechnicianCandidate{{
				TechnicianID:         *specifiedTechnicianID,
				AvailabilityState:    string(appointmentAvailabilityUnavailable),
				PredictedWaitMinutes: 0,
				AvailabilityReason:   reason,
			}}, nil, apiErr{status: 400, msg: reason}
		}
		techID := *specifiedTechnicianID
		return appointmentPlacementDecision{
				PredictedWaitMinutes: candidate.PredictedWaitMinutes,
				AssignedTechnicianID: &techID,
				PlacementScore:       candidate.PlacementScore,
			}, []appointmentTechnicianCandidate{{
				TechnicianID:         techID,
				AvailabilityState:    string(appointmentAvailabilitySafe),
				PredictedWaitMinutes: candidate.PredictedWaitMinutes,
			}}, []uint{techID}, nil
	}
	for _, tech := range technicians {
		candidate, reason, err := e.evaluateTechnicianCandidate(tech.ID, start, occupiedMinutes)
		if err != nil {
			return decision, nil, nil, err
		}
		if candidate == nil {
			candidates = append(candidates, appointmentTechnicianCandidate{
				TechnicianID:         tech.ID,
				AvailabilityState:    string(appointmentAvailabilityUnavailable),
				PredictedWaitMinutes: 0,
				AvailabilityReason:   reason,
			})
			continue
		}
		candidates = append(candidates, appointmentTechnicianCandidate{
			TechnicianID:         tech.ID,
			AvailabilityState:    string(appointmentAvailabilitySafe),
			PredictedWaitMinutes: candidate.PredictedWaitMinutes,
		})
		if candidate.PlacementScore < bestScore {
			bestScore = candidate.PlacementScore
			availableTechIDs = []uint{tech.ID}
			techID := tech.ID
			decision = appointmentPlacementDecision{
				PredictedWaitMinutes: candidate.PredictedWaitMinutes,
				AssignedTechnicianID: &techID,
				PlacementScore:       candidate.PlacementScore,
			}
			continue
		}
		if candidate.PlacementScore == bestScore {
			availableTechIDs = append(availableTechIDs, tech.ID)
			if decision.AssignedTechnicianID != nil && tech.ID < *decision.AssignedTechnicianID {
				techID := tech.ID
				decision.AssignedTechnicianID = &techID
				decision.PredictedWaitMinutes = candidate.PredictedWaitMinutes
			}
		}
	}
	if len(availableTechIDs) == 0 || decision.AssignedTechnicianID == nil {
		return appointmentPlacementDecision{}, candidates, nil, apiErr{status: 400, msg: "当前时段无可预约客服"}
	}
	return decision, candidates, availableTechIDs, nil
}
