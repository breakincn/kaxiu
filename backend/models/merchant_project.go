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

type MerchantProjectDefaultServiceTechnicianIDs []uint

func (ids MerchantProjectDefaultServiceTechnicianIDs) Value() (driver.Value, error) {
	if ids == nil {
		return "[]", nil
	}
	b, err := json.Marshal(ids)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (ids *MerchantProjectDefaultServiceTechnicianIDs) Scan(value interface{}) error {
	if ids == nil {
		return fmt.Errorf("MerchantProjectDefaultServiceTechnicianIDs scan target is nil")
	}
	if value == nil {
		*ids = MerchantProjectDefaultServiceTechnicianIDs{}
		return nil
	}

	var raw []byte
	switch v := value.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("unsupported MerchantProjectDefaultServiceTechnicianIDs value type %T", value)
	}
	if len(raw) == 0 {
		*ids = MerchantProjectDefaultServiceTechnicianIDs{}
		return nil
	}
	if err := json.Unmarshal(raw, ids); err != nil {
		return err
	}
	if *ids == nil {
		*ids = MerchantProjectDefaultServiceTechnicianIDs{}
	}
	return nil
}

type MerchantProject struct {
	ID                                                  uint                                       `json:"id" gorm:"primaryKey;comment:项目ID"`
	MerchantID                                          uint                                       `json:"merchant_id" gorm:"index;index:idx_merchant_active,priority:1;comment:商户ID"`
	Name                                                string                                     `json:"name" gorm:"size:100;not null;comment:项目名称"`
	Duration                                            int                                        `json:"duration" gorm:"not null;comment:服务时长（分钟）"`
	BookableOnline                                      bool                                       `json:"bookable_online" gorm:"not null;default:true;comment:是否参与公开预约"`
	ServiceGapMinutes                                   int                                        `json:"service_gap_minutes" gorm:"not null;default:3;comment:服务间歇时间（分钟）"`
	StartDelaySeconds                                   int                                        `json:"start_delay_seconds" gorm:"not null;default:60;comment:服务开始延迟时间（秒）"`
	RoomSelectTimeoutSeconds                            int                                        `json:"room_select_timeout_seconds" gorm:"not null;default:90;comment:自动分配房间延迟时间（秒）"`
	StartPendingTimeoutSeconds                          int                                        `json:"start_pending_timeout_seconds" gorm:"not null;default:300;comment:待开始服务倒计时秒数"`
	ServiceCapacity                                     int                                        `json:"service_capacity" gorm:"not null;default:1;comment:服务人数"`
	MultiServiceBookingCancelDeadlineMinutesBeforeStart int                                        `json:"multi_service_booking_cancel_deadline_minutes_before_start" gorm:"not null;default:60;comment:多人项目课程预约取消截止时间（距开课前分钟数）"`
	ShowParticipants                                    bool                                       `json:"show_participants" gorm:"not null;default:true;comment:用户端是否展示参与服务用户"`
	ServiceTimeSlots                                    MerchantProjectServiceTimeSlots            `json:"service_time_slots" gorm:"type:json;not null;comment:服务时间槽"`
	DefaultServiceTechnicianIDs                         MerchantProjectDefaultServiceTechnicianIDs `json:"default_service_technician_ids" gorm:"type:json;not null;comment:多人项目默认服务人员ID列表（专业客服）"`
	AutoAssignTechnicianDelayMinutes                    int                                        `json:"auto_assign_technician_delay_minutes" gorm:"not null;default:5;comment:自动分配客服延迟时间（分钟）"`
	DelayToleranceMinutes                               int                                        `json:"delay_tolerance_minutes" gorm:"not null;default:1;comment:拖堂补偿容忍分钟数"`
	DelayCompensationMode                               string                                     `json:"delay_compensation_mode" gorm:"size:20;not null;default:'minutes_bucket';comment:拖堂补偿模式（minutes_bucket/amount_bucket/fixed_unit）"`
	DelayRedeemThresholdPercent                         int                                        `json:"delay_redeem_threshold_percent" gorm:"not null;default:100;comment:拖堂补偿兑现阈值百分比"`
	DelayFixedUnitValue                                 int                                        `json:"delay_fixed_unit_value" gorm:"not null;default:0;comment:固定单位补偿值"`
	IsDefault                                           bool                                       `json:"is_default" gorm:"not null;default:false;index:idx_merchant_default,priority:2;comment:是否为商户默认项目"`
	Price                                               float64                                    `json:"price" gorm:"type:decimal(10,2);default:0.00;comment:项目价格（元）"`
	Description                                         string                                     `json:"description" gorm:"type:text;comment:项目描述"`
	IsActive                                            bool                                       `json:"is_active" gorm:"default:true;index:idx_merchant_active,priority:2;comment:是否启用"`
	SortOrder                                           int                                        `json:"sort_order" gorm:"default:0;index;comment:排序顺序"`
	CreatedAt                                           *time.Time                                 `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt                                           *time.Time                                 `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`

	MultiServiceOverview *MerchantProjectMultiServiceOverview `json:"multi_service_overview,omitempty" gorm:"-"`
}

type MerchantProjectMultiServiceParticipant struct {
	UserID            uint   `json:"user_id"`
	Nickname          string `json:"nickname"`
	Booked            bool   `json:"booked"`
	BookingStatus     string `json:"booking_status"`
	CheckedIn         bool   `json:"checked_in"`
	RecentNoShowCount int    `json:"recent_no_show_count"`
	ShowNoShowCount   bool   `json:"show_no_show_count"`
}

type MerchantProjectDefaultServiceTechnicianStatus struct {
	ID              uint   `json:"id"`
	Name            string `json:"name"`
	Account         string `json:"account"`
	ServiceRoleName string `json:"service_role_name"`
	CheckedIn       bool   `json:"checked_in"`
}

type MerchantProjectMultiServiceOverview struct {
	Visible                              bool                                            `json:"visible"`
	InBookingWindow                      bool                                            `json:"in_booking_window"`
	InServiceTimeWindow                  bool                                            `json:"in_service_time_window"`
	CurrentWindowVerifyCodeGenerated     bool                                            `json:"current_window_verify_code_generated"`
	CurrentSlotStartAt                   *time.Time                                      `json:"current_slot_start_at"`
	CurrentSlotEndAt                     *time.Time                                      `json:"current_slot_end_at"`
	BookingOpenAt                        *time.Time                                      `json:"booking_open_at"`
	ServiceCapacity                      int                                             `json:"service_capacity"`
	BookedCount                          int                                             `json:"booked_count"`
	UsedCount                            int                                             `json:"used_count"`
	RemainingCount                       int                                             `json:"remaining_count"`
	Participants                         []MerchantProjectMultiServiceParticipant        `json:"participants"`
	DefaultServiceTechnicians            []MerchantProjectDefaultServiceTechnicianStatus `json:"default_service_technicians"`
	AnyDefaultServiceTechnicianCheckedIn bool                                            `json:"any_default_service_technician_checked_in"`
	DefaultServiceAttendanceWarning      string                                          `json:"default_service_attendance_warning"`
	DefaultServiceAttendanceRoleName     string                                          `json:"default_service_attendance_role_name"`
	CurrentUserBooked                    bool                                            `json:"current_user_booked"`
	CurrentUserBookingID                 *uint                                           `json:"current_user_booking_id"`
	CurrentUserBookingStatus             string                                          `json:"current_user_booking_status"`
	CurrentUserCanCancel                 bool                                            `json:"current_user_can_cancel"`
	CurrentUserCancelDeadlineAt          *time.Time                                      `json:"current_user_cancel_deadline_at"`
}

func (MerchantProject) TableName() string {
	return "merchant_projects"
}

func (MerchantProject) TableComment() string {
	return "商户项目表"
}
