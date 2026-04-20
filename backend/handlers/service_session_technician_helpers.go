package handlers

import (
	"fmt"
	"kabao/models"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

func uniqueOrderedUintIDs(ids []uint) []uint {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[uint]struct{}, len(ids))
	out := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func jsonArrayContainsLikePatterns(id uint) []interface{} {
	s := strconv.FormatUint(uint64(id), 10)
	return []interface{}{
		"%[" + s + "]%",
		"%[" + s + ",%",
		"%," + s + ",%",
		"%," + s + "]%",
	}
}

func serviceSessionAssignedTechnicianIDs(session *models.ServiceSession) []uint {
	if session == nil {
		return nil
	}
	ids := make([]uint, 0, len(session.ServiceTechnicianIDs)+2)
	ids = append(ids, []uint(session.ServiceTechnicianIDs)...)
	if len(ids) == 0 && session.LastTechnicianID != nil && *session.LastTechnicianID > 0 {
		ids = append(ids, *session.LastTechnicianID)
	}
	return uniqueOrderedUintIDs(ids)
}

func serviceSessionConfirmedTechnicianIDs(session *models.ServiceSession) []uint {
	if session == nil {
		return nil
	}
	confirmed := uniqueOrderedUintIDs([]uint(session.StartConfirmedTechnicianIDs))
	if len(confirmed) > 0 {
		return confirmed
	}
	if session.StartConfirmedAt != nil || session.StartedAt != nil || session.FinishedAt != nil {
		ids := make([]uint, 0, 1)
		if session.LastTechnicianID != nil && *session.LastTechnicianID > 0 {
			ids = append(ids, *session.LastTechnicianID)
		}
		if len(ids) > 0 {
			return uniqueOrderedUintIDs(ids)
		}
	}
	return nil
}

func serviceSessionPrimaryTechnicianIDs(session *models.ServiceSession) []uint {
	if ids := serviceSessionConfirmedTechnicianIDs(session); len(ids) > 0 {
		return ids
	}
	return serviceSessionAssignedTechnicianIDs(session)
}

func serviceSessionHasAssignedTechnician(session *models.ServiceSession, technicianID uint) bool {
	if session == nil || technicianID == 0 {
		return false
	}
	for _, id := range serviceSessionAssignedTechnicianIDs(session) {
		if id == technicianID {
			return true
		}
	}
	return false
}

func serviceSessionHasConfirmedTechnician(session *models.ServiceSession, technicianID uint) bool {
	if session == nil || technicianID == 0 {
		return false
	}
	for _, id := range serviceSessionConfirmedTechnicianIDs(session) {
		if id == technicianID {
			return true
		}
	}
	return false
}

func serviceSessionHasAnyAssignedTechnician(session *models.ServiceSession) bool {
	return len(serviceSessionAssignedTechnicianIDs(session)) > 0
}

func applyServiceSessionTechnicianFilter(query *gorm.DB, alias string, technicianID uint, includeLast bool) *gorm.DB {
	if query == nil || technicianID == 0 {
		return query
	}
	alias = strings.TrimSpace(alias)
	if alias == "" {
		alias = "service_sessions"
	}
	if query.Dialector != nil && query.Dialector.Name() == "mysql" {
		expr := fmt.Sprintf("(%s.last_technician_id = ? OR JSON_CONTAINS(%s.service_technician_ids, JSON_ARRAY(?)) OR JSON_CONTAINS(%s.start_confirmed_technician_ids, JSON_ARRAY(?)))", alias, alias, alias)
		return query.Where(expr, technicianID, technicianID, technicianID)
	}
	expr := fmt.Sprintf("(%s.last_technician_id = ? OR %s.service_technician_ids LIKE ? OR %s.service_technician_ids LIKE ? OR %s.service_technician_ids LIKE ? OR %s.service_technician_ids LIKE ? OR %s.start_confirmed_technician_ids LIKE ? OR %s.start_confirmed_technician_ids LIKE ? OR %s.start_confirmed_technician_ids LIKE ? OR %s.start_confirmed_technician_ids LIKE ?)", alias, alias, alias, alias, alias, alias, alias, alias, alias)
	patterns := jsonArrayContainsLikePatterns(technicianID)
	args := []interface{}{technicianID}
	for _, pattern := range patterns {
		args = append(args, pattern)
	}
	for _, pattern := range patterns {
		args = append(args, pattern)
	}
	return query.Where(expr, args...)
}

func applyServiceSessionUnassignedFilter(query *gorm.DB, alias string) *gorm.DB {
	if query == nil {
		return query
	}
	alias = strings.TrimSpace(alias)
	if alias == "" {
		alias = "service_sessions"
	}
	if query.Dialector != nil && query.Dialector.Name() == "mysql" {
		expr := fmt.Sprintf("(JSON_LENGTH(COALESCE(%s.service_technician_ids, JSON_ARRAY())) = 0 AND JSON_LENGTH(COALESCE(%s.start_confirmed_technician_ids, JSON_ARRAY())) = 0)", alias, alias)
		return query.Where(expr)
	}
	expr := fmt.Sprintf("(COALESCE(%s.service_technician_ids, '[]') IN ('[]', '', 'null') AND COALESCE(%s.start_confirmed_technician_ids, '[]') IN ('[]', '', 'null'))", alias, alias)
	return query.Where(expr)
}

func loadActiveServiceSessionsForTechnician(tx *gorm.DB, merchantID uint, technicianID uint, excludeSessionID uint, statuses []string) ([]models.ServiceSession, error) {
	if tx == nil || merchantID == 0 || technicianID == 0 || len(statuses) == 0 {
		return nil, nil
	}
	query := tx.Table("service_sessions").
		Select("id, merchant_id, last_technician_id, service_technician_ids, start_confirmed_technician_ids, status, predicted_ready_at, scheduled_finish_at, updated_at, start_confirmed_at, started_at, finished_at").
		Where("merchant_id = ? AND status IN ?", merchantID, models.ExpandStatusesWithKnownPrefixes(statuses))
	if excludeSessionID > 0 {
		query = query.Where("id <> ?", excludeSessionID)
	}
	query = applyServiceSessionTechnicianFilter(query, "service_sessions", technicianID, true)
	query = query.Order("COALESCE(predicted_ready_at, scheduled_finish_at, updated_at) asc, id asc")
	rows, err := query.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	sessions := make([]models.ServiceSession, 0)
	parseIDList := func(raw interface{}) models.MerchantProjectDefaultServiceTechnicianIDs {
		var ids models.MerchantProjectDefaultServiceTechnicianIDs
		switch v := raw.(type) {
		case []byte:
			if len(v) == 0 {
				return nil
			}
			_ = ids.Scan(v)
		case string:
			if len(v) == 0 {
				return nil
			}
			_ = ids.Scan([]byte(v))
		}
		return ids
	}
	for rows.Next() {
		var id uint
		var sessionMerchantID uint
		var lastTechnicianIDRaw *uint
		var serviceTechIDsRaw interface{}
		var confirmedTechIDsRaw interface{}
		var status string
		var predictedReadyAtRaw interface{}
		var scheduledFinishAtRaw interface{}
		var updatedAtRaw interface{}
		var startConfirmedAtRaw interface{}
		var startedAtRaw interface{}
		var finishedAtRaw interface{}
		if err := rows.Scan(&id, &sessionMerchantID, &lastTechnicianIDRaw, &serviceTechIDsRaw, &confirmedTechIDsRaw, &status, &predictedReadyAtRaw, &scheduledFinishAtRaw, &updatedAtRaw, &startConfirmedAtRaw, &startedAtRaw, &finishedAtRaw); err != nil {
			return nil, err
		}
		var predictedReadyAt *time.Time
		if v, ok := parseDBTimeValue(predictedReadyAtRaw); ok {
			predictedReadyAt = &v
		}
		var scheduledFinishAt *time.Time
		if v, ok := parseDBTimeValue(scheduledFinishAtRaw); ok {
			scheduledFinishAt = &v
		}
		var updatedAt *time.Time
		if v, ok := parseDBTimeValue(updatedAtRaw); ok {
			updatedAt = &v
		}
		var startConfirmedAt *time.Time
		if v, ok := parseDBTimeValue(startConfirmedAtRaw); ok {
			startConfirmedAt = &v
		}
		var startedAt *time.Time
		if v, ok := parseDBTimeValue(startedAtRaw); ok {
			startedAt = &v
		}
		var finishedAt *time.Time
		if v, ok := parseDBTimeValue(finishedAtRaw); ok {
			finishedAt = &v
		}
		sessions = append(sessions, models.ServiceSession{
			ID:                         id,
			MerchantID:                 sessionMerchantID,
			LastTechnicianID:           lastTechnicianIDRaw,
			ServiceTechnicianIDs:       parseIDList(serviceTechIDsRaw),
			StartConfirmedTechnicianIDs: parseIDList(confirmedTechIDsRaw),
			Status:                     status,
			PredictedReadyAt:           predictedReadyAt,
			ScheduledFinishAt:          scheduledFinishAt,
			UpdatedAt:                  updatedAt,
			StartConfirmedAt:           startConfirmedAt,
			StartedAt:                  startedAt,
			FinishedAt:                 finishedAt,
		})
	}
	return sessions, nil
}

func serviceSessionEarliestReadyAt(session *models.ServiceSession) time.Time {
	if session == nil {
		return time.Time{}
	}
	if session.PredictedReadyAt != nil {
		return *session.PredictedReadyAt
	}
	if session.ScheduledFinishAt != nil {
		return *session.ScheduledFinishAt
	}
	if session.UpdatedAt != nil {
		return *session.UpdatedAt
	}
	return time.Time{}
}
