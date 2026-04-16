package handlers

import (
	"errors"
	"kabao/config"
	"kabao/models"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func CreateServiceSessionExtendRequest(c *gin.Context) {
	userIDAny, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	userID, _ := userIDAny.(uint)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	sessionID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if sessionID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "服务单不存在"})
		return
	}

	var input struct {
		ProjectID uint `json:"project_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择加钟项目"})
		return
	}

	var out models.ServiceSessionExtendRequest
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var s models.ServiceSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND user_id = ?", sessionID, userID).First(&s).Error; err != nil {
			return err
		}
		if models.NormalizeSessionStatus(s.Status) != "serving" {
			return apiErr{status: http.StatusBadRequest, msg: "仅服务中可申请加钟"}
		}
		if s.CardID == 0 || s.InitialUsageID == 0 {
			return apiErr{status: http.StatusBadRequest, msg: "当前服务单不可申请加钟"}
		}

		project, err := loadCardOwnedExtendProject(tx, s.CardID, s.MerchantID, input.ProjectID)
		if err != nil {
			return err
		}
		if project.Duration < 5 || project.Duration > 180 {
			return apiErr{status: http.StatusBadRequest, msg: "项目时长不在可加钟范围内"}
		}

		var pendingCount int64
		if err := tx.Model(&models.ServiceSessionExtendRequest{}).
			Where("service_session_id = ? AND status = ?", s.ID, "pending").
			Count(&pendingCount).Error; err != nil {
			return err
		}
		if pendingCount > 0 {
			return apiErr{status: http.StatusBadRequest, msg: "已有待处理加钟申请"}
		}

		req := models.ServiceSessionExtendRequest{
			MerchantID:       s.MerchantID,
			UserID:           s.UserID,
			CardID:           s.CardID,
			ServiceSessionID: s.ID,
			InitialUsageID:   s.InitialUsageID,
			ProjectID:        project.ID,
			Minutes:          project.Duration,
			Status:           "pending",
		}
		if err := tx.Create(&req).Error; err != nil {
			return err
		}
		return tx.Preload("Project").First(&out, req.ID).Error
	})
	if err != nil {
		writeExtendRequestError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

func ApproveServiceSessionExtendRequest(c *gin.Context) {
	handleServiceSessionExtendRequest(c, true)
}

func RejectServiceSessionExtendRequest(c *gin.Context) {
	handleServiceSessionExtendRequest(c, false)
}

func handleServiceSessionExtendRequest(c *gin.Context, approve bool) {
	merchantIDAny, ok := c.Get("merchant_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	merchantID, _ := merchantIDAny.(uint)
	if merchantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	requestID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if requestID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "加钟申请不存在"})
		return
	}

	var input struct {
		RejectReason string `json:"reject_reason"`
	}
	_ = c.ShouldBindJSON(&input)

	var out models.ServiceSessionExtendRequest
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var req models.ServiceSessionExtendRequest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND merchant_id = ?", requestID, merchantID).First(&req).Error; err != nil {
			return err
		}
		if req.Status != "pending" {
			return apiErr{status: http.StatusBadRequest, msg: "该加钟申请已处理"}
		}

		var s models.ServiceSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND merchant_id = ?", req.ServiceSessionID, merchantID).First(&s).Error; err != nil {
			return err
		}
		if models.NormalizeSessionStatus(s.Status) != "serving" {
			return apiErr{status: http.StatusBadRequest, msg: "当前服务已不在服务中"}
		}
		if s.TechnicianID != nil {
			authType, _ := c.Get("auth_type")
			technicianIDAny, _ := c.Get("technician_id")
			technicianID, _ := technicianIDAny.(uint)
			if authType == "staff" && technicianID > 0 && *s.TechnicianID != technicianID {
				return apiErr{status: http.StatusForbidden, msg: "只能处理自己的服务加钟申请"}
			}
		}

		now := time.Now()
		beforeRemainingSeconds := 0
		afterRemainingSeconds := 0
		updates := map[string]interface{}{
			"status":          "rejected",
			"reject_reason":   strings.TrimSpace(input.RejectReason),
			"handled_by_type": "merchant",
			"handled_at":      now,
		}
		if authType, _ := c.Get("auth_type"); authType == "staff" {
			updates["handled_by_type"] = "staff"
		}
		if technicianIDAny, exists := c.Get("technician_id"); exists {
			if technicianID, _ := technicianIDAny.(uint); technicianID > 0 {
				updates["handled_by_id"] = technicianID
			}
		}

		if approve {
			beforeRemainingSeconds = computeServiceSessionRemainingSeconds(&s, now)
			afterRemainingSeconds = beforeRemainingSeconds + req.Minutes*60
			if err := applyServiceSessionExtension(tx, &s, req.Minutes); err != nil {
				return err
			}
			updates["status"] = "approved"
			updates["reject_reason"] = ""
			updates["before_remaining_seconds"] = beforeRemainingSeconds
			updates["after_remaining_seconds"] = afterRemainingSeconds
		} else if strings.TrimSpace(input.RejectReason) == "" {
			updates["reject_reason"] = "客服拒绝加钟申请"
		}

		if err := tx.Model(&models.ServiceSessionExtendRequest{}).Where("id = ?", req.ID).Updates(updates).Error; err != nil {
			return err
		}

		req.Status = updates["status"].(string)
		req.RejectReason = updates["reject_reason"].(string)
		req.HandledByType = updates["handled_by_type"].(string)
		req.HandledAt = &now
		req.BeforeRemainingSeconds = beforeRemainingSeconds
		req.AfterRemainingSeconds = afterRemainingSeconds
		if handledByID, ok := updates["handled_by_id"].(uint); ok {
			req.HandledByID = &handledByID
		}
		var project models.MerchantProject
		if err := tx.First(&project, req.ProjectID).Error; err != nil {
			return err
		}
		req.Project = &project
		out = req
		return nil
	})
	if err != nil {
		writeExtendRequestError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

func computeServiceSessionRemainingSeconds(s *models.ServiceSession, now time.Time) int {
	if s == nil {
		return 0
	}
	var finishAt time.Time
	if s.ScheduledFinishAt != nil {
		finishAt = *s.ScheduledFinishAt
	} else if s.StartedAt != nil && s.DurationMinutes > 0 {
		finishAt = s.StartedAt.Add(time.Duration(s.DurationMinutes) * time.Minute)
	} else {
		return 0
	}
	remain := int(finishAt.Sub(now).Seconds())
	if remain < 0 {
		return 0
	}
	return remain
}

func loadCardOwnedExtendProject(tx *gorm.DB, cardID uint, merchantID uint, projectID uint) (*models.MerchantProject, error) {
	if tx == nil || cardID == 0 || merchantID == 0 || projectID == 0 {
		return nil, apiErr{status: http.StatusBadRequest, msg: "请选择加钟项目"}
	}

	var count int64
	if err := tx.Model(&models.CardProject{}).Where("card_id = ? AND project_id = ?", cardID, projectID).Count(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		var cpCount int64
		if err := tx.Model(&models.CardProject{}).Where("card_id = ?", cardID).Count(&cpCount).Error; err != nil {
			return nil, err
		}
		if cpCount > 0 {
			return nil, apiErr{status: http.StatusBadRequest, msg: "该项目不属于当前卡片"}
		}
		defaultProject, err := config.GetDefaultMerchantProject(tx, merchantID)
		if err != nil || defaultProject == nil || defaultProject.ID != projectID {
			return nil, apiErr{status: http.StatusBadRequest, msg: "该项目不属于当前卡片"}
		}
		return defaultProject, nil
	}

	var project models.MerchantProject
	if err := tx.Where("id = ? AND merchant_id = ? AND is_active = ?", projectID, merchantID, true).First(&project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apiErr{status: http.StatusBadRequest, msg: "加钟项目不可用"}
		}
		return nil, err
	}
	return &project, nil
}

func applyServiceSessionExtension(tx *gorm.DB, s *models.ServiceSession, minutes int) error {
	if tx == nil || s == nil || s.ID == 0 {
		return nil
	}
	if minutes < 5 || minutes > 180 {
		return apiErr{status: http.StatusBadRequest, msg: "minutes 范围应为 5-180"}
	}
	updates := map[string]interface{}{
		"duration_minutes":          gorm.Expr("duration_minutes + ?", minutes),
		"auto_finish_delay_seconds": gorm.Expr("auto_finish_delay_seconds + ?", minutes*60),
	}
	if s.ScheduledFinishAt != nil {
		updates["scheduled_finish_at"] = s.ScheduledFinishAt.Add(time.Duration(minutes) * time.Minute)
	}
	return tx.Model(&models.ServiceSession{}).Where("id = ?", s.ID).Updates(updates).Error
}

func writeExtendRequestError(c *gin.Context, err error) {
	var ae apiErr
	if errors.As(err, &ae) {
		c.JSON(ae.status, gin.H{"error": ae.msg})
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "记录不存在"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

func attachLatestExtendRequestsToSessions(sessions []models.ServiceSession) {
	if len(sessions) == 0 {
		return
	}
	ids := make([]uint, 0, len(sessions))
	for i := range sessions {
		if sessions[i].ID > 0 {
			ids = append(ids, sessions[i].ID)
		}
	}
	if len(ids) == 0 {
		return
	}

	var reqs []models.ServiceSessionExtendRequest
	if err := config.DB.Preload("Project").
		Where("service_session_id IN ?", ids).
		Order("id desc").
		Find(&reqs).Error; err != nil {
		return
	}
	bySession := make(map[uint]*models.ServiceSessionExtendRequest, len(reqs))
	for i := range reqs {
		req := reqs[i]
		if _, exists := bySession[req.ServiceSessionID]; exists {
			continue
		}
		bySession[req.ServiceSessionID] = &reqs[i]
	}
	for i := range sessions {
		if req, ok := bySession[sessions[i].ID]; ok {
			sessions[i].LatestExtendRequest = req
		}
	}
}

func attachLatestExtendRequestToSession(session *models.ServiceSession) {
	if session == nil || session.ID == 0 {
		return
	}
	var req models.ServiceSessionExtendRequest
	if err := config.DB.Preload("Project").
		Where("service_session_id = ?", session.ID).
		Order("id desc").
		First(&req).Error; err != nil {
		return
	}
	session.LatestExtendRequest = &req
}

func attachLatestExtendRequestsToUsages(usages []models.Usage) {
	if len(usages) == 0 {
		return
	}
	sessionIDs := make([]uint, 0, len(usages))
	for i := range usages {
		if usages[i].ServiceSessionID != nil && *usages[i].ServiceSessionID > 0 {
			sessionIDs = append(sessionIDs, *usages[i].ServiceSessionID)
		}
	}
	if len(sessionIDs) == 0 {
		return
	}

	var reqs []models.ServiceSessionExtendRequest
	if err := config.DB.Preload("Project").
		Where("service_session_id IN ?", sessionIDs).
		Order("id desc").
		Find(&reqs).Error; err != nil {
		return
	}
	bySession := make(map[uint]*models.ServiceSessionExtendRequest, len(reqs))
	for i := range reqs {
		req := reqs[i]
		if _, exists := bySession[req.ServiceSessionID]; exists {
			continue
		}
		bySession[req.ServiceSessionID] = &reqs[i]
	}
	for i := range usages {
		if usages[i].ServiceSessionID == nil {
			continue
		}
		if req, ok := bySession[*usages[i].ServiceSessionID]; ok {
			usages[i].LatestExtendRequest = req
		}
	}
}
