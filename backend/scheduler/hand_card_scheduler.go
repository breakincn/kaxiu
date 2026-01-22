package scheduler

import (
	"kabao/config"
	"kabao/models"
	"log"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	handCardSchedulerTickInterval = 5 * time.Second
	handCardLockWindow           = 8 * time.Minute
	handCardSchedulerBatchLimit  = 200
)

// StartHandCardScheduler 手牌锁卡调度器：
// - 仅对开启 support_hand_card 的商户生效
// - 以卡片“首次手牌分配时间”为起点
// - 超过 8 分钟仍存在“已分配未归还”的手牌，则锁定该卡片并记录原因
func StartHandCardScheduler() {
	go func() {
		ticker := time.NewTicker(handCardSchedulerTickInterval)
		defer ticker.Stop()
		for range ticker.C {
			if config.DB == nil {
				continue
			}
			if err := runHandCardLockOnce(config.DB); err != nil {
				log.Printf("hand card scheduler error: %v", err)
			}
		}
	}()
}

type handCardLockCandidate struct {
	CardID         uint      `gorm:"column:card_id"`
	MerchantID     uint      `gorm:"column:merchant_id"`
	FirstAssigned  time.Time `gorm:"column:first_assigned"`
	UnreturnedCount int64    `gorm:"column:unreturned_count"`
}

func runHandCardLockOnce(db *gorm.DB) error {
	now := time.Now()
	cutoff := now.Add(-handCardLockWindow)

	var cands []handCardLockCandidate
	err := db.
		Table("cards c").
		Select("c.id AS card_id, c.merchant_id AS merchant_id, MIN(u.hand_card_assigned_at) AS first_assigned, COUNT(1) AS unreturned_count").
		Joins("JOIN merchants m ON m.id = c.merchant_id").
		Joins("JOIN usages u ON u.card_id = c.id").
		Where("m.support_hand_card = ?", true).
		Where("c.locked = ?", false).
		Where("u.hand_card_assigned_at IS NOT NULL AND u.hand_card_returned_at IS NULL").
		Group("c.id").
		Having("MIN(u.hand_card_assigned_at) <= ?", cutoff).
		Order("first_assigned asc").
		Limit(handCardSchedulerBatchLimit).
		Find(&cands).Error
	if err != nil {
		return err
	}
	if len(cands) == 0 {
		return nil
	}

	for i := range cands {
		cand := cands[i]
		if err := lockOneCardIfNeeded(db, cand.CardID, cand.MerchantID, now); err != nil {
			log.Printf("hand card lock card %d error: %v", cand.CardID, err)
		}
	}
	return nil
}

func lockOneCardIfNeeded(db *gorm.DB, cardID uint, merchantID uint, now time.Time) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var card models.Card
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND merchant_id = ?", cardID, merchantID).First(&card).Error; err != nil {
			return err
		}
		if card.Locked {
			return nil
		}

		// 重新计算该卡首次分配时间 & 未归还数量，避免并发误锁
		var firstAssigned time.Time
		row := tx.Table("usages").
			Select("MIN(hand_card_assigned_at) AS first_assigned").
			Where("card_id = ? AND merchant_id = ? AND hand_card_assigned_at IS NOT NULL AND hand_card_returned_at IS NULL", cardID, merchantID).
			Row()
		_ = row.Scan(&firstAssigned)
		if firstAssigned.IsZero() {
			return nil
		}
		if now.Sub(firstAssigned) < handCardLockWindow {
			return nil
		}

		var cnt int64
		if err := tx.Model(&models.Usage{}).
			Where("card_id = ? AND merchant_id = ? AND hand_card_assigned_at IS NOT NULL AND hand_card_returned_at IS NULL", cardID, merchantID).
			Count(&cnt).Error; err != nil {
			return err
		}
		if cnt <= 0 {
			return nil
		}

		// 取一个样例手牌号用于展示
		var sample string
		row2 := tx.Table("usages").
			Select("hand_card_no").
			Where("card_id = ? AND merchant_id = ? AND hand_card_assigned_at IS NOT NULL AND hand_card_returned_at IS NULL AND hand_card_no IS NOT NULL", cardID, merchantID).
			Order("hand_card_assigned_at asc").
			Limit(1).
			Row()
		_ = row2.Scan(&sample)

		reason := "未归还手牌"
		if sample != "" {
			reason = "未归还手牌：" + sample
			if cnt > 1 {
				reason = reason + " 等" + fmtInt64(cnt) + "个"
			}
		} else {
			if cnt > 1 {
				reason = reason + "（" + fmtInt64(cnt) + "个）"
			}
		}

		updates := map[string]interface{}{
			"locked":        true,
			"locked_reason": reason,
			"locked_at":     &now,
			"locked_by":     nil,
		}
		return tx.Model(&models.Card{}).Where("id = ? AND merchant_id = ? AND locked = ?", cardID, merchantID, false).Updates(updates).Error
	})
}

func fmtInt64(v int64) string {
	// 避免引入 strconv 造成更多导入变化；手牌原因字符串不需要极致性能
	if v == 0 {
		return "0"
	}
	neg := false
	if v < 0 {
		neg = true
		v = -v
	}
	buf := make([]byte, 0, 20)
	for v > 0 {
		d := byte(v % 10)
		buf = append(buf, '0'+d)
		v /= 10
	}
	// reverse
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}
