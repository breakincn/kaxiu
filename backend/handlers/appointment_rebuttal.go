package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"kabao/config"
	"kabao/models"

	"github.com/gin-gonic/gin"
)

func appointmentAllowsUserRebuttal(appt models.Appointment) bool {
	if appt.ID == 0 {
		return false
	}
	if normalizeAppointmentStatus(appt.Status) != "canceled" {
		return false
	}
	closedByType := strings.TrimSpace(appt.ClosedByType)
	if closedByType == "merchant" || closedByType == "staff" {
		return true
	}
	return strings.TrimSpace(appt.MerchantCancelReason) != ""
}

func UpdateAppointmentUserRebuttal(c *gin.Context) {
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
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此预约"})
		return
	}
	if !appointmentAllowsUserRebuttal(*appointment) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前预约状态不可填写抗辩意见"})
		return
	}
	var input struct {
		UserRebuttalNote string `json:"user_rebuttal_note" binding:"required,max=255"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	note := strings.TrimSpace(input.UserRebuttalNote)
	if note == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "抗辩意见不能为空"})
		return
	}
	resolutionNote := appendAppointmentResolutionNote(appointment.ResolutionNote, "用户抗辩："+note)
	if err := config.DB.Model(&models.Appointment{}).Where("id = ?", appointment.ID).Updates(map[string]interface{}{
		"user_rebuttal_note": note,
		"resolution_note":    resolutionNote,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存抗辩意见失败"})
		return
	}
	appointment.UserRebuttalNote = note
	appointment.ResolutionNote = resolutionNote
	c.JSON(http.StatusOK, gin.H{"data": appointment})
}
