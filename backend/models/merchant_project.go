package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type MerchantProjectServiceTimeSlot struct {
	RecurrenceType string `json:"recurrence_type"`
	Weekday        int    `json:"weekday,omitempty"`
	MonthDay       int    `json:"month_day,omitempty"`
	StartTime      string `json:"start_time"`
}

type MerchantProjectServiceTimeSlots []MerchantProjectServiceTimeSlot

func (slots MerchantProjectServiceTimeSlots) Value() (driver.Value, error) {
	if slots == nil {
		return "[]", nil
	}
	b, err := json.Marshal(slots)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (slots *MerchantProjectServiceTimeSlots) Scan(value interface{}) error {
	if slots == nil {
		return fmt.Errorf("MerchantProjectServiceTimeSlots scan target is nil")
	}
	if value == nil {
		*slots = MerchantProjectServiceTimeSlots{}
		return nil
	}

	var raw []byte
	switch v := value.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("unsupported MerchantProjectServiceTimeSlots value type %T", value)
	}
	if len(raw) == 0 {
		*slots = MerchantProjectServiceTimeSlots{}
		return nil
	}
	if err := json.Unmarshal(raw, slots); err != nil {
		return err
	}
	if *slots == nil {
		*slots = MerchantProjectServiceTimeSlots{}
	}
	return nil
}

type MerchantProject struct {
	ID                               uint                            `json:"id" gorm:"primaryKey;comment:项目ID"`
	MerchantID                       uint                            `json:"merchant_id" gorm:"index;index:idx_merchant_active,priority:1;comment:商户ID"`
	Name                             string                          `json:"name" gorm:"size:100;not null;comment:项目名称"`
	Duration                         int                             `json:"duration" gorm:"not null;comment:服务时长（分钟）"`
	BookableOnline                   bool                            `json:"bookable_online" gorm:"not null;default:true;comment:是否参与公开预约"`
	ServiceGapMinutes                int                             `json:"service_gap_minutes" gorm:"not null;default:3;comment:服务间歇时间（分钟）"`
	StartDelaySeconds                int                             `json:"start_delay_seconds" gorm:"not null;default:60;comment:服务开始延迟时间（秒）"`
	RoomSelectTimeoutSeconds         int                             `json:"room_select_timeout_seconds" gorm:"not null;default:90;comment:自动分配房间延迟时间（秒）"`
	StartPendingTimeoutSeconds       int                             `json:"start_pending_timeout_seconds" gorm:"not null;default:300;comment:待开始服务倒计时秒数"`
	ServiceCapacity                  int                             `json:"service_capacity" gorm:"not null;default:1;comment:服务人数"`
	ServiceTimeSlots                 MerchantProjectServiceTimeSlots `json:"service_time_slots" gorm:"type:json;not null;comment:服务时间槽"`
	AutoAssignTechnicianDelayMinutes int                             `json:"auto_assign_technician_delay_minutes" gorm:"not null;default:5;comment:自动分配客服延迟时间（分钟）"`
	DelayToleranceMinutes            int                             `json:"delay_tolerance_minutes" gorm:"not null;default:1;comment:拖堂补偿容忍分钟数"`
	DelayCompensationMode            string                          `json:"delay_compensation_mode" gorm:"size:20;not null;default:'minutes_bucket';comment:拖堂补偿模式（minutes_bucket/amount_bucket/fixed_unit）"`
	DelayRedeemThresholdPercent      int                             `json:"delay_redeem_threshold_percent" gorm:"not null;default:100;comment:拖堂补偿兑现阈值百分比"`
	DelayFixedUnitValue              int                             `json:"delay_fixed_unit_value" gorm:"not null;default:0;comment:固定单位补偿值"`
	IsDefault                        bool                            `json:"is_default" gorm:"not null;default:false;index:idx_merchant_default,priority:2;comment:是否为商户默认项目"`
	Price                            float64                         `json:"price" gorm:"type:decimal(10,2);default:0.00;comment:项目价格（元）"`
	Description                      string                          `json:"description" gorm:"type:text;comment:项目描述"`
	IsActive                         bool                            `json:"is_active" gorm:"default:true;index:idx_merchant_active,priority:2;comment:是否启用"`
	SortOrder                        int                             `json:"sort_order" gorm:"default:0;index;comment:排序顺序"`
	CreatedAt                        *time.Time                      `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt                        *time.Time                      `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`
}

func (MerchantProject) TableName() string {
	return "merchant_projects"
}

func (MerchantProject) TableComment() string {
	return "商户项目表"
}
