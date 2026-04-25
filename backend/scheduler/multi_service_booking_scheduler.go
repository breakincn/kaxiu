package scheduler

import (
	"errors"
	"kabao/models"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func runMultiServiceBookingSettlementOnce(db *gorm.DB, now time.Time) error {
	if db == nil {
		return nil
	}
	var dueIDs []uint
	if err := db.Model(&models.MultiServiceBooking{}).
		Where("status = ? AND slot_start_at IS NOT NULL AND slot_start_at <= ?", "booked", now).
		Order("slot_start_at asc, id asc").
		Limit(appointmentSchedulerBatchLimit).
		Pluck("id", &dueIDs).Error; err != nil {
		return err
	}
	for _, bookingID := range dueIDs {
		if err := settleOneMultiServiceBookingNoShow(db, bookingID, now); err != nil {
			// 单条失败不中断批次
			continue
		}
	}
	return nil
}

func settleOneMultiServiceBookingNoShow(db *gorm.DB, bookingID uint, now time.Time) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var booking models.MultiServiceBooking
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&booking, bookingID).Error; err != nil {
			return err
		}
		if strings.TrimSpace(booking.Status) != "booked" || booking.SlotStartAt == nil || now.Before(*booking.SlotStartAt) {
			return nil
		}
		var card models.Card
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&card, booking.CardID).Error; err != nil {
			return err
		}
		var project models.MerchantProject
		if err := tx.First(&project, booking.ProjectID).Error; err != nil {
			return err
		}
		usage, chargedTimes, chargedAmount, err := schedulerCreateMultiServicePenaltyUsage(tx, &card, project, "预约未到", now)
		if err != nil {
			return err
		}
		updates := map[string]interface{}{
			"status":     "no_show",
			"no_show_at": &now,
		}
		if usage != nil {
			updates["usage_id"] = usage.ID
		}
		if err := tx.Model(&models.MultiServiceBooking{}).Where("id = ? AND status = ?", booking.ID, "booked").Updates(updates).Error; err != nil {
			return err
		}
		usageID := uint(0)
		if usage != nil {
			usageID = usage.ID
		}
		if err := schedulerCreateMultiServicePenaltyLedger(tx, card, project, &booking, "no_show", true, chargedTimes, chargedAmount, "预约未到", now, usageID); err != nil {
			return err
		}
		return schedulerApplyMultiServiceOverLimitPenaltyIfNeeded(tx, card, project, &booking, now)
	})
}

func schedulerCreateMultiServicePenaltyUsage(tx *gorm.DB, card *models.Card, project models.MerchantProject, note string, now time.Time) (*models.Usage, int, int, error) {
	if tx == nil || card == nil || card.ID == 0 {
		return nil, 0, 0, gorm.ErrRecordNotFound
	}
	chargedTimes := 0
	chargedAmount := 0
	if card.TotalTimes > 0 {
		res := tx.Model(&models.Card{}).
			Where("id = ? AND remain_times > 0", card.ID).
			Updates(map[string]interface{}{
				"remain_times": gorm.Expr("remain_times - 1"),
				"used_times":   gorm.Expr("used_times + 1"),
				"last_used_at": now,
			})
		if res.Error != nil {
			return nil, 0, 0, res.Error
		}
		if res.RowsAffected == 0 {
			return nil, 0, 0, errors.New("card remain_times insufficient")
		}
		card.RemainTimes--
		card.UsedTimes++
		chargedTimes = 1
	} else {
		chargedAmount = int(math.Ceil(project.Price))
		if chargedAmount <= 0 {
			return nil, 0, 0, errors.New("project price invalid")
		}
		res := tx.Model(&models.Card{}).
			Where("id = ? AND recharge_amount >= ?", card.ID, chargedAmount).
			Updates(map[string]interface{}{
				"recharge_amount": gorm.Expr("recharge_amount - ?", chargedAmount),
				"last_used_at":    now,
			})
		if res.Error != nil {
			return nil, 0, 0, res.Error
		}
		if res.RowsAffected == 0 {
			return nil, 0, 0, errors.New("card recharge_amount insufficient")
		}
		card.RechargeAmount -= chargedAmount
	}
	card.LastUsedAt = &now
	usage := &models.Usage{
		CardID:             card.ID,
		MerchantID:         card.MerchantID,
		ProjectID:          &project.ID,
		UsedTimes:          chargedTimes,
		UsedAt:             &now,
		VerifyCodeExpireAt: now.Unix(),
		Status:             "success",
		SourceType:         "multi_service_booking_penalty",
		SourceNote:         note,
		CardNoSnapshot:     card.CardNo,
		CardTypeSnapshot:   card.CardType,
	}
	totalTimes := card.TotalTimes
	usedTimes := card.UsedTimes
	remainTimes := card.RemainTimes
	usage.CardTotalTimesSnapshot = &totalTimes
	usage.CardUsedTimesSnapshot = &usedTimes
	usage.CardRemainTimesSnapshot = &remainTimes
	if err := tx.Create(usage).Error; err != nil {
		return nil, 0, 0, err
	}
	return usage, chargedTimes, chargedAmount, nil
}

func schedulerCreateMultiServicePenaltyLedger(tx *gorm.DB, card models.Card, project models.MerchantProject, booking *models.MultiServiceBooking, penaltyType string, countsToward bool, chargedTimes int, chargedAmount int, remark string, now time.Time, usageID uint) error {
	ledger := models.MultiServicePenaltyLedger{
		MerchantID:         card.MerchantID,
		ProjectID:          project.ID,
		CardID:             card.ID,
		UserID:             card.UserID,
		PenaltyType:        penaltyType,
		CountsTowardNoShow: countsToward,
		ChargedTimes:       chargedTimes,
		ChargedAmount:      chargedAmount,
		Remark:             remark,
		PenaltyAt:          &now,
	}
	if booking != nil && booking.ID > 0 {
		ledger.BookingID = &booking.ID
	}
	if usageID > 0 {
		ledger.UsageID = &usageID
	}
	return tx.Create(&ledger).Error
}

func schedulerApplyMultiServiceOverLimitPenaltyIfNeeded(tx *gorm.DB, card models.Card, project models.MerchantProject, booking *models.MultiServiceBooking, now time.Time) error {
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
	cardCopy := card
	usage, chargedTimes, chargedAmount, err := schedulerCreateMultiServicePenaltyUsage(tx, &cardCopy, project, "失约超限", now)
	if err != nil {
		return err
	}
	usageID := uint(0)
	if usage != nil {
		usageID = usage.ID
	}
	return schedulerCreateMultiServicePenaltyLedger(tx, cardCopy, project, booking, "over_limit", false, chargedTimes, chargedAmount, "失约超限", now, usageID)
}
