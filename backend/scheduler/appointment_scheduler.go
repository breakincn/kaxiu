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
	appointmentSchedulerTickInterval = 15 * time.Second
	appointmentAssignDelay           = 10 * time.Minute
	appointmentSchedulerBatchLimit   = 200
)

func StartAppointmentScheduler() {
	go func() {
		ticker := time.NewTicker(appointmentSchedulerTickInterval)
		defer ticker.Stop()

		for range ticker.C {
			if config.DB == nil {
				continue
			}
			if err := runAppointmentAssignOnce(config.DB); err != nil {
				log.Printf("appointment scheduler error: %v", err)
			}
		}
	}()
}

func runAppointmentAssignOnce(db *gorm.DB) error {
	now := time.Now()
	cutoff := now.Add(-appointmentAssignDelay)

	var appts []models.Appointment
	err := db.
		Where("technician_id IS NULL AND status = ? AND appointment_time IS NOT NULL AND created_at IS NOT NULL AND created_at <= ?", "pending", cutoff).
		Order("created_at asc").
		Limit(appointmentSchedulerBatchLimit).
		Find(&appts).Error
	if err != nil {
		return err
	}

	for i := range appts {
		a := appts[i]
		if err := tryAssignOneAppointment(db, a.ID, now); err != nil {
			log.Printf("assign appointment %d error: %v", a.ID, err)
		}
	}
	return nil
}

func tryAssignOneAppointment(db *gorm.DB, appointmentID uint, now time.Time) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var a models.Appointment
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&a, appointmentID).Error; err != nil {
			return err
		}
		if a.Status != "pending" {
			return nil
		}
		if a.TechnicianID != nil {
			return nil
		}
		if a.AppointmentTime == nil {
			return nil
		}

		var m models.Merchant
		if err := tx.First(&m, a.MerchantID).Error; err != nil {
			return err
		}
		// 未开启客服则不做自动分配
		if !m.SupportCustomerService {
			return nil
		}

		// 查找候选专业客服（排除运营客服；兼容历史 role_type 为空）
		var techs []models.Technician
		err := tx.
			Model(&models.Technician{}).
			Joins("JOIN service_roles sr ON sr.id = technicians.service_role_id").
			Where("technicians.merchant_id = ? AND technicians.is_active = ? AND (sr.role_type IS NULL OR sr.role_type = '' OR sr.role_type <> ?)", a.MerchantID, true, "operational").
			Order("technicians.id asc").
			Find(&techs).Error
		if err != nil {
			return err
		}
		if len(techs) == 0 {
			failAt := now
			return tx.Model(&models.Appointment{}).
				Where("id = ? AND technician_id IS NULL AND status = ?", a.ID, "pending").
				Updates(map[string]interface{}{"status": "failed", "failed_at": &failAt, "failed_reason": "该时间段无可分配客服"}).Error
		}

		projectDurationCache := map[uint]int{}
		getDurationMinutes := func(ap models.Appointment) int {
			if ap.ProjectID == nil || *ap.ProjectID == 0 {
				return 30
			}
			pid := *ap.ProjectID
			if v, ok := projectDurationCache[pid]; ok {
				if v > 0 {
					return v
				}
				return 30
			}
			var p models.MerchantProject
			err := tx.Where("id = ? AND merchant_id = ?", pid, ap.MerchantID).First(&p).Error
			if err != nil {
				projectDurationCache[pid] = 0
				return 30
			}
			if p.Duration <= 0 {
				projectDurationCache[pid] = 0
				return 30
			}
			projectDurationCache[pid] = p.Duration
			return p.Duration
		}

		start := *a.AppointmentTime
		serviceMinutes := getDurationMinutes(a)
		end := start.Add(time.Duration(serviceMinutes) * time.Minute)

		overlaps := func(s1, e1, s2, e2 time.Time) bool {
			return s1.Before(e2) && s2.Before(e1)
		}

		// 从候选客服中选择一个在该时间段不冲突的
		for _, t := range techs {
			var existing []models.Appointment
			err := tx.
				Where("merchant_id = ? AND technician_id = ? AND status IN ('pending','confirmed') AND appointment_time IS NOT NULL", a.MerchantID, t.ID).
				Order("appointment_time asc").
				Find(&existing).Error
			if err != nil {
				return err
			}

			conflict := false
			for _, e := range existing {
				if e.AppointmentTime == nil {
					continue
				}
				es := *e.AppointmentTime
				eMin := getDurationMinutes(e)
				ee := es.Add(time.Duration(eMin) * time.Minute)
				if overlaps(start, end, es, ee) {
					conflict = true
					break
				}
			}
			if conflict {
				continue
			}

			// 绑定到该客服
			if err := tx.Model(&models.Appointment{}).
				Where("id = ? AND technician_id IS NULL AND status = ?", a.ID, "pending").
				Update("technician_id", t.ID).Error; err != nil {
				return err
			}
			return nil
		}

		// 全部冲突，标记失败
		failAt := now
		if err := tx.Model(&models.Appointment{}).
			Where("id = ? AND technician_id IS NULL AND status = ?", a.ID, "pending").
			Updates(map[string]interface{}{"status": "failed", "failed_at": &failAt, "failed_reason": "该时间段无可分配客服"}).Error; err != nil {
			return err
		}
		return nil
	})
}
