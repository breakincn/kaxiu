package handlers

import (
	"database/sql"
	"kabao/config"
	"kabao/models"
	"kabao/queue"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var liveServiceStatusActiveStatuses = models.ExpandStatusesWithKnownPrefixes([]string{
	"room_selecting",
	"room_locked",
	"staff_selecting",
	"start_pending",
	"delay_pending",
	"timeout_waiting",
	"serving",
	"auto_finishing",
})

type liveServiceStatusResponse struct {
	MerchantID         uint                          `json:"merchant_id"`
	Enabled            bool                          `json:"enabled"`
	Mode               string                        `json:"mode"`
	ModeLabel          string                        `json:"mode_label"`
	ModeVariantLabel   string                        `json:"mode_variant_label"`
	BusinessOpen       bool                          `json:"business_open"`
	QueuePaused        bool                          `json:"queue_paused"`
	StatusLevel        string                        `json:"status_level"`
	StatusText         string                        `json:"status_text"`
	SummaryText        string                        `json:"summary_text"`
	RecommendationText string                        `json:"recommendation_text"`
	GeneratedAt        time.Time                     `json:"generated_at"`
	Confidence         string                        `json:"confidence"`
	ConfidenceText     string                        `json:"confidence_text"`
	AnomalyCount       int                           `json:"anomaly_count"`
	Counts             liveServiceStatusCounts       `json:"counts"`
	Stages             liveServiceStatusStages       `json:"stages"`
	Estimate           liveServiceStatusEstimate     `json:"estimate"`
	Queue              *liveServiceStatusQueueStatus `json:"queue,omitempty"`
}

type liveServiceStatusCounts struct {
	ArrivedCount          int `json:"arrived_count"`
	WaitingCount          int `json:"waiting_count"`
	ServingCount          int `json:"serving_count"`
	CalledCount           int `json:"called_count"`
	CheckedInStaffCount   int `json:"checked_in_staff_count"`
	ServiceableStaffCount int `json:"serviceable_staff_count"`
	BusyStaffCount        int `json:"busy_staff_count"`
	IdleStaffCount        int `json:"idle_staff_count"`
	PausedStaffCount      int `json:"paused_staff_count"`
	ActiveRoomCount       int `json:"active_room_count"`
	OccupiedRoomCount     int `json:"occupied_room_count"`
}

type liveServiceStatusStages struct {
	RoomSelecting  int `json:"room_selecting"`
	RoomLocked     int `json:"room_locked"`
	StaffSelecting int `json:"staff_selecting"`
	StartPending   int `json:"start_pending"`
	DelayPending   int `json:"delay_pending"`
	TimeoutWaiting int `json:"timeout_waiting"`
	Serving        int `json:"serving"`
	AutoFinishing  int `json:"auto_finishing"`
}

type liveServiceStatusEstimate struct {
	WaitMinutes            *int       `json:"wait_minutes,omitempty"`
	WaitText               string     `json:"wait_text"`
	WaitingAheadCount      int        `json:"waiting_ahead_count"`
	ShouldVisitNow         bool       `json:"should_visit_now"`
	SuggestAppointment     bool       `json:"suggest_appointment"`
	RecommendedArrivalAt   *time.Time `json:"recommended_arrival_at,omitempty"`
	RecommendedArrivalText string     `json:"recommended_arrival_text"`
}

type liveServiceStatusQueueStatus struct {
	QueuePrefix       string `json:"queue_prefix"`
	CurrentCalledNo   int    `json:"current_called_no"`
	WaitingQueueCount int    `json:"waiting_queue_count"`
}

type liveServiceProjectedSession struct {
	Session         *models.ServiceSession
	BaseStatus      string
	DurationMinutes int
	QueueNo         int
}

type liveServiceTechSnapshot struct {
	ServiceableCount int
	CheckedInCount   int
	PausedCount      int
	BusyCount        int
	IdleCount        int
	SlotAvailableAt  []time.Time
	AnomalyCount     int
}

func GetMerchantLiveServiceStatus(c *gin.Context) {
	merchantID64, err := strconv.ParseUint(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || merchantID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的商户ID"})
		return
	}

	var merchant models.Merchant
	if err := config.DB.First(&merchant, uint(merchantID64)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商户不存在"})
		return
	}

	out, err := buildMerchantLiveServiceStatus(&merchant, time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": out})
}

func buildMerchantLiveServiceStatus(merchant *models.Merchant, now time.Time) (liveServiceStatusResponse, error) {
	out := liveServiceStatusResponse{
		GeneratedAt: now,
	}
	if merchant == nil {
		return out, nil
	}

	mode := models.ResolveSessionMode(merchant)
	out.MerchantID = merchant.ID
	out.Mode = mode
	out.ModeLabel = liveServiceModeLabel(mode)
	out.ModeVariantLabel = liveServiceModeVariantLabel(mode)
	out.BusinessOpen = isMerchantInBusinessHours(merchant, now)
	out.QueuePaused = models.IsQueueMode(mode) && merchant.QueuePaused
	out.Enabled = models.IsCSMode(mode) || models.IsQueueMode(mode)
	out.Confidence = "medium"
	out.ConfidenceText = "基于当前服务流转状态估算"

	if !out.Enabled {
		out.StatusLevel = "disabled"
		out.StatusText = "当前门店未启用实时服务状态展示"
		out.SummaryText = "门店当前未处于客服模式或叫号模式"
		out.RecommendationText = "请以门店现场安排为准"
		out.Estimate.WaitText = "暂未启用"
		out.Estimate.RecommendedArrivalText = "请联系门店确认"
		out.Confidence = "low"
		out.ConfidenceText = "当前门店未开启客服模式或叫号模式"
		return out, nil
	}

	sessions, err := liveServiceLoadActiveSessions(merchant.ID)
	if err != nil {
		return out, err
	}

	var snap queue.Snapshot
	queueNumbers := make(map[uint]int)
	if models.IsQueueMode(mode) && queue.Default != nil {
		snap = queue.Default.Snapshot(merchant.ID, now.Format("2006-01-02"), queue.QueueTypeOnsite)
	}

	projected := make([]liveServiceProjectedSession, 0, len(sessions))
	for i := range sessions {
		session := &sessions[i]
		baseStatus := liveServiceProjectedStatus(merchant, session, now)
		item := liveServiceProjectedSession{
			Session:         session,
			BaseStatus:      baseStatus,
			DurationMinutes: liveServiceSessionDurationMinutes(session),
		}
		if models.IsQueueMode(mode) && session.InitialUsageID > 0 {
			item.QueueNo = liveServiceLookupQueueNo(merchant.ID, now, session.InitialUsageID, snap)
			queueNumbers[session.InitialUsageID] = item.QueueNo
		}
		projected = append(projected, item)
		liveServiceAddStageCount(&out.Stages, baseStatus)
	}

	out.Counts = liveServiceBuildSessionCounts(projected)

	techs, err := liveServiceLoadEligibleTechnicians(merchant.ID)
	if err != nil {
		return out, err
	}
	attMap, err := liveServiceLoadOpenAttendances(merchant.ID, liveServiceTechnicianIDs(techs), now)
	if err != nil {
		return out, err
	}
	techSnapshot := liveServiceBuildTechSnapshot(merchant, mode, techs, attMap, projected, now)
	out.Counts.CheckedInStaffCount = techSnapshot.CheckedInCount
	out.Counts.ServiceableStaffCount = techSnapshot.ServiceableCount
	out.Counts.BusyStaffCount = techSnapshot.BusyCount
	out.Counts.IdleStaffCount = techSnapshot.IdleCount
	out.Counts.PausedStaffCount = techSnapshot.PausedCount
	out.AnomalyCount += techSnapshot.AnomalyCount

	roomAvailableAt := []time.Time(nil)
	if models.IsCSMode(mode) && merchant.SupportRoom {
		roomAvailableAt, out.Counts.ActiveRoomCount, out.Counts.OccupiedRoomCount, out.AnomalyCount, err = liveServiceBuildRoomAvailability(merchant.ID, projected, now, out.AnomalyCount)
		if err != nil {
			return out, err
		}
	}

	slotAvailableAt := liveServiceBuildSlotAvailability(merchant, mode, projected, techSnapshot.SlotAvailableAt, roomAvailableAt, now)

	waitMinutes, waitingAheadCount := liveServiceEstimateWaitMinutes(merchant, mode, projected, slotAvailableAt, now)
	if waitMinutes != nil {
		out.Estimate.WaitMinutes = waitMinutes
		out.Estimate.WaitText = liveServiceFormatWaitText(*waitMinutes)
	} else {
		out.Estimate.WaitText = "暂无法估算"
	}
	out.Estimate.WaitingAheadCount = waitingAheadCount

	if models.IsQueueMode(mode) {
		out.Queue = &liveServiceStatusQueueStatus{
			QueuePrefix:       strings.TrimSpace(merchant.QueuePrefix),
			CurrentCalledNo:   liveServiceCurrentCalledNo(projected, snap),
			WaitingQueueCount: liveServiceQueueWaitingCount(projected),
		}
	}

	liveServiceFinalizePresentation(&out, merchant, now, len(slotAvailableAt))
	return out, nil
}

func liveServiceLoadActiveSessions(merchantID uint) ([]models.ServiceSession, error) {
	type sessionRow struct {
		ID                        uint           `gorm:"column:id"`
		MerchantID                uint           `gorm:"column:merchant_id"`
		ProjectID                 *uint          `gorm:"column:project_id"`
		InitialUsageID            uint           `gorm:"column:initial_usage_id"`
		SessionMode               string         `gorm:"column:session_mode"`
		RoomID                    *uint          `gorm:"column:room_id"`
		TechnicianID              *uint          `gorm:"column:technician_id"`
		Status                    string         `gorm:"column:status"`
		StartConfirmedAtText      sql.NullString `gorm:"column:start_confirmed_at"`
		ScheduledStartAtText      sql.NullString `gorm:"column:scheduled_start_at"`
		StartedAtText             sql.NullString `gorm:"column:started_at"`
		ScheduledFinishAtText     sql.NullString `gorm:"column:scheduled_finish_at"`
		StartPendingTimeoutSecond int            `gorm:"column:start_pending_timeout_seconds"`
		DurationMinutes           int            `gorm:"column:duration_minutes"`
		AutoIdleAfterSeconds      int            `gorm:"column:auto_idle_after_seconds"`
		CreatedAtText             sql.NullString `gorm:"column:created_at"`
		UpdatedAtText             sql.NullString `gorm:"column:updated_at"`
	}

	var rows []sessionRow
	if err := config.DB.
		Table("service_sessions").
		Select([]string{
			"id",
			"merchant_id",
			"project_id",
			"initial_usage_id",
			"session_mode",
			"room_id",
			"technician_id",
			"status",
			"start_confirmed_at",
			"scheduled_start_at",
			"started_at",
			"scheduled_finish_at",
			"start_pending_timeout_seconds",
			"duration_minutes",
			"auto_idle_after_seconds",
			"created_at",
			"updated_at",
		}).
		Where("merchant_id = ? AND status IN ?", merchantID, liveServiceStatusActiveStatuses).
		Order("created_at asc, id asc").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	projectIDs := make([]uint, 0)
	projectSeen := map[uint]struct{}{}
	for _, row := range rows {
		if row.ProjectID == nil || *row.ProjectID == 0 {
			continue
		}
		if _, ok := projectSeen[*row.ProjectID]; ok {
			continue
		}
		projectSeen[*row.ProjectID] = struct{}{}
		projectIDs = append(projectIDs, *row.ProjectID)
	}

	projectMap := make(map[uint]models.MerchantProject, len(projectIDs))
	if len(projectIDs) > 0 {
		var projects []models.MerchantProject
		if err := config.DB.Where("id IN ?", projectIDs).Find(&projects).Error; err != nil {
			return nil, err
		}
		for _, project := range projects {
			projectMap[project.ID] = project
		}
	}

	sessions := make([]models.ServiceSession, 0, len(rows))
	for _, row := range rows {
		session := models.ServiceSession{
			ID:                         row.ID,
			MerchantID:                 row.MerchantID,
			ProjectID:                  row.ProjectID,
			InitialUsageID:             row.InitialUsageID,
			SessionMode:                row.SessionMode,
			RoomID:                     row.RoomID,
			TechnicianID:               row.TechnicianID,
			Status:                     row.Status,
			StartConfirmedAt:           liveServiceParseNullableTime(row.StartConfirmedAtText),
			ScheduledStartAt:           liveServiceParseNullableTime(row.ScheduledStartAtText),
			StartedAt:                  liveServiceParseNullableTime(row.StartedAtText),
			ScheduledFinishAt:          liveServiceParseNullableTime(row.ScheduledFinishAtText),
			StartPendingTimeoutSeconds: row.StartPendingTimeoutSecond,
			DurationMinutes:            row.DurationMinutes,
			AutoIdleAfterSeconds:       row.AutoIdleAfterSeconds,
			CreatedAt:                  liveServiceParseNullableTime(row.CreatedAtText),
			UpdatedAt:                  liveServiceParseNullableTime(row.UpdatedAtText),
		}
		if row.ProjectID != nil {
			if project, ok := projectMap[*row.ProjectID]; ok {
				p := project
				session.Project = &p
			}
		}
		sessions = append(sessions, session)
	}
	return sessions, nil
}

func liveServiceLoadEligibleTechnicians(merchantID uint) ([]models.Technician, error) {
	var techs []models.Technician
	err := config.DB.
		Model(&models.Technician{}).
		Joins("JOIN service_roles sr ON sr.id = technicians.service_role_id").
		Where("technicians.merchant_id = ? AND technicians.is_active = ?", merchantID, true).
		Where("sr.role_type = ? AND sr.`key` NOT IN ('store_manager','front_desk')", "professional").
		Order("technicians.id asc").
		Find(&techs).Error
	return techs, err
}

func liveServiceTechnicianIDs(techs []models.Technician) []uint {
	out := make([]uint, 0, len(techs))
	for _, tech := range techs {
		if tech.ID == 0 {
			continue
		}
		out = append(out, tech.ID)
	}
	return out
}

func liveServiceLoadOpenAttendances(merchantID uint, techIDs []uint, now time.Time) (map[uint]models.TechnicianAttendance, error) {
	out := make(map[uint]models.TechnicianAttendance, len(techIDs))
	if merchantID == 0 || len(techIDs) == 0 {
		return out, nil
	}

	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	type attendanceRow struct {
		ID               uint           `gorm:"column:id"`
		MerchantID       uint           `gorm:"column:merchant_id"`
		TechnicianID     uint           `gorm:"column:technician_id"`
		CheckedInAtText  sql.NullString `gorm:"column:checked_in_at"`
		CheckedOutAtText sql.NullString `gorm:"column:checked_out_at"`
		Status           string         `gorm:"column:status"`
		NextStatusText   sql.NullString `gorm:"column:next_status"`
		UpdatedAtText    sql.NullString `gorm:"column:updated_at"`
	}

	var rows []attendanceRow
	if err := config.DB.
		Table("technician_attendances").
		Select([]string{"id", "merchant_id", "technician_id", "checked_in_at", "checked_out_at", "status", "next_status", "updated_at"}).
		Where("merchant_id = ? AND technician_id IN ? AND checked_in_at >= ? AND checked_out_at IS NULL", merchantID, techIDs, start).
		Order("updated_at desc, id desc").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		if row.TechnicianID == 0 {
			continue
		}
		if _, ok := out[row.TechnicianID]; ok {
			continue
		}
		att := models.TechnicianAttendance{
			ID:           row.ID,
			MerchantID:   row.MerchantID,
			TechnicianID: row.TechnicianID,
			CheckedInAt:  liveServiceParseNullableTime(row.CheckedInAtText),
			CheckedOutAt: liveServiceParseNullableTime(row.CheckedOutAtText),
			Status:       row.Status,
			UpdatedAt:    liveServiceParseNullableTime(row.UpdatedAtText),
		}
		if row.NextStatusText.Valid {
			nextStatus := strings.TrimSpace(row.NextStatusText.String)
			if nextStatus != "" {
				att.NextStatus = &nextStatus
			}
		}
		out[row.TechnicianID] = att
	}
	return out, nil
}

func liveServiceBuildSessionCounts(projected []liveServiceProjectedSession) liveServiceStatusCounts {
	out := liveServiceStatusCounts{}
	for _, item := range projected {
		switch item.BaseStatus {
		case "serving", "auto_finishing":
			out.ServingCount++
			out.ArrivedCount++
		case "room_selecting", "room_locked", "staff_selecting", "start_pending", "delay_pending", "timeout_waiting":
			out.WaitingCount++
			out.ArrivedCount++
		}
		if item.BaseStatus == "start_pending" || item.BaseStatus == "delay_pending" {
			out.CalledCount++
		}
	}
	return out
}

func liveServiceBuildTechSnapshot(merchant *models.Merchant, mode string, techs []models.Technician, attMap map[uint]models.TechnicianAttendance, projected []liveServiceProjectedSession, now time.Time) liveServiceTechSnapshot {
	out := liveServiceTechSnapshot{
		SlotAvailableAt: make([]time.Time, 0, len(techs)),
	}
	if merchant == nil {
		return out
	}

	busySessionsByTech := make(map[uint][]liveServiceProjectedSession)
	busyStaffIDs := make(map[uint]struct{})
	for _, item := range projected {
		if !liveServiceOccupiesServiceSlot(item.BaseStatus) || item.Session == nil || item.Session.TechnicianID == nil || *item.Session.TechnicianID == 0 {
			continue
		}
		techID := *item.Session.TechnicianID
		busySessionsByTech[techID] = append(busySessionsByTech[techID], item)
		busyStaffIDs[techID] = struct{}{}
	}
	out.BusyCount = len(busyStaffIDs)

	for _, tech := range techs {
		att, hasAttendance := attMap[tech.ID]
		checkedIn := true
		if merchant.SupportTechnicianCheckin {
			checkedIn = hasAttendance && att.CheckedOutAt == nil
		}
		if checkedIn {
			out.CheckedInCount++
		}

		attendanceStatus := ""
		if hasAttendance {
			attendanceStatus = strings.TrimSpace(att.Status)
		}
		nextPaused := hasAttendance && att.NextStatus != nil && strings.TrimSpace(*att.NextStatus) == "paused"
		paused := false
		if merchant.SupportTechnicianCheckin {
			paused = attendanceStatus == "paused" || attendanceStatus == "rest"
		}
		if nextPaused {
			paused = true
		}
		if models.IsQueueMode(mode) && tech.QueuePaused {
			paused = true
		}
		if paused {
			out.PausedCount++
		}

		serviceable := checkedIn && !paused
		if !serviceable {
			continue
		}

		out.ServiceableCount++
		busySessions := busySessionsByTech[tech.ID]
		if len(busySessions) == 0 {
			if merchant.SupportTechnicianCheckin && attendanceStatus == "busy" {
				out.AnomalyCount++
			}
			out.SlotAvailableAt = append(out.SlotAvailableAt, now)
			continue
		}

		if len(busySessions) > 1 {
			out.AnomalyCount += len(busySessions) - 1
		}
		slotUntil := now
		for _, item := range busySessions {
			t := liveServiceProjectedBusyUntil(&item, now)
			if t.After(slotUntil) {
				slotUntil = t
			}
		}
		out.SlotAvailableAt = append(out.SlotAvailableAt, slotUntil)
	}

	out.IdleCount = out.ServiceableCount - minInt(out.ServiceableCount, liveServiceServiceableBusyCount(out.SlotAvailableAt, now))
	if out.IdleCount < 0 {
		out.IdleCount = 0
	}
	return out
}

func liveServiceServiceableBusyCount(slotAvailableAt []time.Time, now time.Time) int {
	count := 0
	for _, t := range slotAvailableAt {
		if t.After(now) {
			count++
		}
	}
	return count
}

func liveServiceBuildRoomAvailability(merchantID uint, projected []liveServiceProjectedSession, now time.Time, anomalyBase int) ([]time.Time, int, int, int, error) {
	var rooms []models.Room
	if err := config.DB.Where("merchant_id = ? AND is_active = ?", merchantID, true).Order("id asc").Find(&rooms).Error; err != nil {
		return nil, 0, 0, anomalyBase, err
	}
	if len(rooms) == 0 {
		return []time.Time{}, 0, 0, anomalyBase, nil
	}

	byRoom := make(map[uint][]liveServiceProjectedSession)
	occupiedRoomIDs := make(map[uint]struct{})
	for _, item := range projected {
		if item.Session == nil || item.Session.RoomID == nil || *item.Session.RoomID == 0 {
			continue
		}
		if !liveServiceOccupiesRoom(item.BaseStatus) {
			continue
		}
		roomID := *item.Session.RoomID
		byRoom[roomID] = append(byRoom[roomID], item)
		occupiedRoomIDs[roomID] = struct{}{}
	}

	out := make([]time.Time, 0, len(rooms))
	anomalyCount := anomalyBase
	for _, room := range rooms {
		items := byRoom[room.ID]
		if len(items) == 0 {
			out = append(out, now)
			continue
		}
		if len(items) > 1 {
			anomalyCount += len(items) - 1
		}
		roomUntil := now
		for _, item := range items {
			t := liveServiceProjectedBusyUntil(&item, now)
			if t.After(roomUntil) {
				roomUntil = t
			}
		}
		out = append(out, roomUntil)
	}
	return out, len(rooms), len(occupiedRoomIDs), anomalyCount, nil
}

func liveServiceBuildSlotAvailability(merchant *models.Merchant, mode string, projected []liveServiceProjectedSession, staffAvailableAt []time.Time, roomAvailableAt []time.Time, now time.Time) []time.Time {
	if merchant == nil {
		return nil
	}

	if models.IsQueueMode(mode) && !merchant.SupportMultiCustomerService {
		if !isMerchantInBusinessHours(merchant, now) || merchant.QueuePaused {
			return nil
		}
		laneBusyUntil := now
		for _, item := range projected {
			if !liveServiceOccupiesServiceSlot(item.BaseStatus) {
				continue
			}
			t := liveServiceProjectedBusyUntil(&item, now)
			if t.After(laneBusyUntil) {
				laneBusyUntil = t
			}
		}
		return []time.Time{laneBusyUntil}
	}

	if len(staffAvailableAt) == 0 {
		return nil
	}
	sort.Slice(staffAvailableAt, func(i, j int) bool { return staffAvailableAt[i].Before(staffAvailableAt[j]) })

	if models.IsCSMode(mode) && merchant.SupportRoom {
		if len(roomAvailableAt) == 0 {
			return nil
		}
		sort.Slice(roomAvailableAt, func(i, j int) bool { return roomAvailableAt[i].Before(roomAvailableAt[j]) })
		size := minInt(len(staffAvailableAt), len(roomAvailableAt))
		out := make([]time.Time, 0, size)
		for i := 0; i < size; i++ {
			out = append(out, maxTime(staffAvailableAt[i], roomAvailableAt[i]))
		}
		return out
	}

	return staffAvailableAt
}

func liveServiceEstimateWaitMinutes(merchant *models.Merchant, mode string, projected []liveServiceProjectedSession, slotAvailableAt []time.Time, now time.Time) (*int, int) {
	if merchant == nil || len(slotAvailableAt) == 0 {
		return nil, 0
	}
	if !isMerchantInBusinessHours(merchant, now) {
		return nil, 0
	}
	if models.IsQueueMode(mode) && merchant.QueuePaused {
		return nil, 0
	}

	waitTasks := liveServiceBuildWaitTasks(mode, projected)
	if len(waitTasks) == 0 {
		sort.Slice(slotAvailableAt, func(i, j int) bool { return slotAvailableAt[i].Before(slotAvailableAt[j]) })
		minutes := liveServiceRoundUpMinutes(slotAvailableAt[0].Sub(now))
		if minutes < 0 {
			minutes = 0
		}
		return &minutes, 0
	}

	slots := append([]time.Time(nil), slotAvailableAt...)
	sort.Slice(slots, func(i, j int) bool { return slots[i].Before(slots[j]) })
	for _, taskMinutes := range waitTasks {
		sort.Slice(slots, func(i, j int) bool { return slots[i].Before(slots[j]) })
		startAt := slots[0]
		if startAt.Before(now) {
			startAt = now
		}
		slots[0] = startAt.Add(time.Duration(taskMinutes) * time.Minute)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i].Before(slots[j]) })
	minutes := liveServiceRoundUpMinutes(slots[0].Sub(now))
	if minutes < 0 {
		minutes = 0
	}
	return &minutes, len(waitTasks)
}

func liveServiceBuildWaitTasks(mode string, projected []liveServiceProjectedSession) []int {
	if models.IsQueueMode(mode) {
		items := make([]liveServiceProjectedSession, 0)
		for _, item := range projected {
			if item.BaseStatus != "staff_selecting" && item.BaseStatus != "timeout_waiting" {
				continue
			}
			items = append(items, item)
		}
		sort.Slice(items, func(i, j int) bool {
			if items[i].QueueNo > 0 && items[j].QueueNo > 0 && items[i].QueueNo != items[j].QueueNo {
				return items[i].QueueNo < items[j].QueueNo
			}
			return liveServiceSessionSortTime(items[i].Session).Before(liveServiceSessionSortTime(items[j].Session))
		})

		out := make([]int, 0, len(items))
		for _, item := range items {
			out = append(out, liveServiceWeightedWaitMinutes(mode, item.BaseStatus, item.DurationMinutes))
		}
		return out
	}

	items := make([]liveServiceProjectedSession, 0)
	for _, item := range projected {
		if item.BaseStatus != "room_selecting" && item.BaseStatus != "room_locked" && item.BaseStatus != "staff_selecting" && item.BaseStatus != "timeout_waiting" {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		leftPriority := liveServiceCSPriority(items[i].BaseStatus)
		rightPriority := liveServiceCSPriority(items[j].BaseStatus)
		if leftPriority != rightPriority {
			return leftPriority > rightPriority
		}
		return liveServiceSessionSortTime(items[i].Session).Before(liveServiceSessionSortTime(items[j].Session))
	})

	out := make([]int, 0, len(items))
	for _, item := range items {
		out = append(out, liveServiceWeightedWaitMinutes(mode, item.BaseStatus, item.DurationMinutes))
	}
	return out
}

func liveServiceCurrentCalledNo(projected []liveServiceProjectedSession, snap queue.Snapshot) int {
	currentCalledNo := snap.MaxCalledNo
	for _, item := range projected {
		if item.QueueNo <= 0 {
			continue
		}
		if item.BaseStatus == "start_pending" || item.BaseStatus == "delay_pending" || item.BaseStatus == "serving" || item.BaseStatus == "auto_finishing" {
			if item.QueueNo > currentCalledNo {
				currentCalledNo = item.QueueNo
			}
		}
	}
	return currentCalledNo
}

func liveServiceQueueWaitingCount(projected []liveServiceProjectedSession) int {
	count := 0
	for _, item := range projected {
		if item.BaseStatus == "staff_selecting" || item.BaseStatus == "timeout_waiting" {
			count++
		}
	}
	return count
}

func liveServiceProjectedStatus(merchant *models.Merchant, session *models.ServiceSession, now time.Time) string {
	if session == nil {
		return ""
	}

	baseStatus := models.NormalizeSessionStatus(session.Status)
	if baseStatus != "start_pending" {
		return baseStatus
	}
	if session.StartConfirmedAt != nil {
		return baseStatus
	}
	if merchant != nil && !merchant.SupportCustomerServiceMode && merchant.SupportQueue && isManualQueueMode(merchant.QueueMode) {
		return baseStatus
	}
	if computeStartPendingRemainingSecondsForSession(session, now) > 0 {
		return baseStatus
	}
	return "staff_selecting"
}

func liveServiceProjectedBusyUntil(item *liveServiceProjectedSession, now time.Time) time.Time {
	if item == nil || item.Session == nil {
		return now
	}

	durationMinutes := item.DurationMinutes
	if durationMinutes <= 0 {
		durationMinutes = 30
	}
	duration := time.Duration(durationMinutes) * time.Minute
	autoIdle := time.Duration(liveServiceNormalizeAutoIdleSeconds(item.Session.AutoIdleAfterSeconds)) * time.Second

	switch item.BaseStatus {
	case "serving", "auto_finishing":
		if item.Session.ScheduledFinishAt != nil {
			return maxTime(*item.Session.ScheduledFinishAt, now).Add(autoIdle)
		}
		if item.Session.StartedAt != nil {
			return maxTime(item.Session.StartedAt.Add(duration), now).Add(autoIdle)
		}
		if item.Session.ScheduledStartAt != nil {
			return maxTime(item.Session.ScheduledStartAt.Add(duration), now).Add(autoIdle)
		}
	case "start_pending", "delay_pending", "room_locked", "staff_selecting":
		startAt := now
		if item.Session.ScheduledStartAt != nil && item.Session.ScheduledStartAt.After(startAt) {
			startAt = *item.Session.ScheduledStartAt
		}
		return startAt.Add(duration).Add(autoIdle)
	}
	return now
}

func liveServiceOccupiesServiceSlot(baseStatus string) bool {
	switch baseStatus {
	case "start_pending", "delay_pending", "serving", "auto_finishing":
		return true
	default:
		return false
	}
}

func liveServiceOccupiesRoom(baseStatus string) bool {
	switch baseStatus {
	case "room_locked", "staff_selecting", "start_pending", "delay_pending", "serving", "auto_finishing":
		return true
	default:
		return false
	}
}

func liveServiceSessionDurationMinutes(session *models.ServiceSession) int {
	if session == nil {
		return 30
	}
	if session.DurationMinutes > 0 {
		return session.DurationMinutes
	}
	if session.Project != nil && session.Project.Duration > 0 {
		return session.Project.Duration
	}
	return 30
}

func liveServiceLookupQueueNo(merchantID uint, now time.Time, usageID uint, snap queue.Snapshot) int {
	if usageID == 0 {
		return 0
	}
	if tk, ok := snap.ByID[usageID]; ok {
		return tk.No
	}
	if queue.Default == nil {
		return 0
	}
	no, ok := queue.Default.GetNo(merchantID, now.Format("2006-01-02"), queue.QueueTypeOnsite, usageID)
	if !ok {
		return 0
	}
	return no
}

func liveServiceWeightedWaitMinutes(mode string, baseStatus string, durationMinutes int) int {
	if durationMinutes <= 0 {
		durationMinutes = 30
	}

	weight := 1.0
	if models.IsCSMode(mode) {
		switch baseStatus {
		case "room_selecting":
			weight = 0.25
		case "room_locked":
			weight = 0.35
		case "staff_selecting":
			weight = 0.45
		case "timeout_waiting":
			weight = 0.6
		}
	} else {
		switch baseStatus {
		case "timeout_waiting":
			weight = 0.85
		default:
			weight = 1
		}
	}

	minutes := int(math.Ceil(float64(durationMinutes) * weight))
	if minutes < 5 {
		minutes = 5
	}
	return minutes
}

func liveServiceModeLabel(mode string) string {
	switch {
	case models.IsCSMode(mode):
		return "客服模式"
	case models.IsQueueMode(mode):
		return "叫号模式"
	default:
		return "普通模式"
	}
}

func liveServiceModeVariantLabel(mode string) string {
	switch strings.TrimSpace(mode) {
	case models.SessionModeCustomerService:
		return "客服模式"
	case models.SessionModeQueueAutoSingle:
		return "自动叫号 · 单窗口"
	case models.SessionModeQueueAutoMulti:
		return "自动叫号 · 多窗口"
	case models.SessionModeQueueManualSingle:
		return "人工叫号 · 单窗口"
	case models.SessionModeQueueManualMulti:
		return "人工叫号 · 多窗口"
	default:
		return "普通模式"
	}
}

func liveServiceAddStageCount(stages *liveServiceStatusStages, baseStatus string) {
	if stages == nil {
		return
	}
	switch baseStatus {
	case "room_selecting":
		stages.RoomSelecting++
	case "room_locked":
		stages.RoomLocked++
	case "staff_selecting":
		stages.StaffSelecting++
	case "start_pending":
		stages.StartPending++
	case "delay_pending":
		stages.DelayPending++
	case "timeout_waiting":
		stages.TimeoutWaiting++
	case "serving":
		stages.Serving++
	case "auto_finishing":
		stages.AutoFinishing++
	}
}

func liveServiceSessionSortTime(session *models.ServiceSession) time.Time {
	if session == nil {
		return time.Time{}
	}
	if session.CreatedAt != nil {
		return *session.CreatedAt
	}
	if session.UpdatedAt != nil {
		return *session.UpdatedAt
	}
	return time.Time{}
}

func liveServiceCSPriority(baseStatus string) int {
	switch baseStatus {
	case "timeout_waiting":
		return 4
	case "staff_selecting":
		return 3
	case "room_locked":
		return 2
	case "room_selecting":
		return 1
	default:
		return 0
	}
}

func liveServiceFinalizePresentation(out *liveServiceStatusResponse, merchant *models.Merchant, now time.Time, slotCount int) {
	if out == nil || merchant == nil {
		return
	}

	out.Estimate.ShouldVisitNow = false
	out.Estimate.SuggestAppointment = false
	out.Estimate.RecommendedArrivalText = "请以门店现场情况为准"

	if !out.BusinessOpen {
		out.StatusLevel = "closed"
		out.StatusText = "当前非营业时间"
		out.SummaryText = liveServiceBuildSummary(out)
		out.RecommendationText = "建议营业时间内再到店"
		out.Estimate.WaitText = "当前未营业"
		out.Estimate.SuggestAppointment = true
		out.Estimate.RecommendedArrivalText = "建议营业时间内到店"
		out.Confidence = "high"
		out.ConfidenceText = "门店当前处于非营业状态"
		return
	}

	if models.IsQueueMode(out.Mode) && out.QueuePaused {
		out.StatusLevel = "paused"
		out.StatusText = "门店当前暂停叫号"
		out.SummaryText = liveServiceBuildSummary(out)
		out.RecommendationText = "建议稍后再看或先预约"
		out.Estimate.WaitText = "暂停叫号中"
		out.Estimate.SuggestAppointment = true
		out.Estimate.RecommendedArrivalText = "建议叫号恢复后再到店"
		out.Confidence = "high"
		out.ConfidenceText = "基于当前叫号暂停状态"
		return
	}

	if slotCount <= 0 {
		out.StatusLevel = "unavailable"
		out.StatusText = "当前暂无可快速接待能力"
		out.SummaryText = liveServiceBuildSummary(out)
		out.RecommendationText = "建议稍后再看或先预约"
		out.Estimate.WaitText = "暂无法估算"
		out.Estimate.SuggestAppointment = true
		out.Estimate.RecommendedArrivalText = "建议先电话确认或预约"
		if out.AnomalyCount > 0 {
			out.Confidence = "low"
			out.ConfidenceText = "存在未完全收口的服务状态，仅供参考"
		}
		return
	}

	waitMinutes := 0
	if out.Estimate.WaitMinutes != nil {
		waitMinutes = *out.Estimate.WaitMinutes
	}
	out.SummaryText = liveServiceBuildSummary(out)

	switch {
	case waitMinutes <= 5 && out.Counts.IdleStaffCount > 0:
		out.StatusLevel = "smooth"
		out.StatusText = "现在到店较快"
		out.RecommendationText = "当前等待较少，可直接到店"
		out.Estimate.ShouldVisitNow = true
	case waitMinutes <= 15:
		out.StatusLevel = "normal"
		out.StatusText = "现在到店需稍等"
		out.RecommendationText = "可现在到店，预计会有少量等待"
		out.Estimate.ShouldVisitNow = true
	case waitMinutes <= 30:
		out.StatusLevel = "busy"
		out.StatusText = "当前较忙"
		out.RecommendationText = "建议稍晚一点到店更合适"
	default:
		out.StatusLevel = "busy"
		out.StatusText = "当前较忙，建议错峰"
		out.RecommendationText = "建议稍后到店或优先预约"
		out.Estimate.SuggestAppointment = true
	}

	if waitMinutes > 0 {
		recommendedAt := now.Add(time.Duration(waitMinutes) * time.Minute)
		out.Estimate.RecommendedArrivalAt = &recommendedAt
		out.Estimate.RecommendedArrivalText = "建议 " + recommendedAt.Format("15:04") + " 后到店"
	} else if out.Estimate.ShouldVisitNow {
		out.Estimate.RecommendedArrivalText = "建议现在到店"
	}

	if out.AnomalyCount > 0 {
		out.Confidence = "low"
		out.ConfidenceText = "存在未完全收口的服务状态，预计时间仅供参考"
		return
	}
	if models.IsQueueMode(out.Mode) && queue.Default == nil {
		out.Confidence = "medium"
		out.ConfidenceText = "未读取到叫号快照，当前仅按服务状态估算"
		return
	}
	if models.IsQueueMode(out.Mode) {
		out.Confidence = "high"
		out.ConfidenceText = "基于叫号进度与当前服务状态估算"
		return
	}
	out.Confidence = "medium"
	out.ConfidenceText = "基于客服服务状态估算"
}

func liveServiceBuildSummary(out *liveServiceStatusResponse) string {
	if out == nil {
		return ""
	}
	parts := []string{}
	if out.Counts.ServingCount > 0 {
		parts = append(parts, "服务中"+strconv.Itoa(out.Counts.ServingCount)+"人")
	}
	if out.Counts.WaitingCount > 0 {
		parts = append(parts, "待服务"+strconv.Itoa(out.Counts.WaitingCount)+"人")
	}
	if out.Counts.IdleStaffCount > 0 {
		parts = append(parts, "空闲客服"+strconv.Itoa(out.Counts.IdleStaffCount)+"人")
	}
	if len(parts) == 0 {
		return "当前门店等待较少"
	}
	return strings.Join(parts, "，")
}

func liveServiceFormatWaitText(waitMinutes int) string {
	if waitMinutes <= 0 {
		return "基本无需等待"
	}
	return "约 " + strconv.Itoa(waitMinutes) + " 分钟"
}

func liveServiceNormalizeAutoIdleSeconds(v int) int {
	if v <= 0 {
		return 180
	}
	return v
}

func liveServiceParseNullableTime(raw sql.NullString) *time.Time {
	if !raw.Valid {
		return nil
	}
	text := strings.TrimSpace(raw.String)
	if text == "" {
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
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, text, time.Local); err == nil {
			return &t
		}
	}
	return nil
}

func liveServiceRoundUpMinutes(d time.Duration) int {
	if d <= 0 {
		return 0
	}
	return int(math.Ceil(d.Minutes()))
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
