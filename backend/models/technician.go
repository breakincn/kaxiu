package models

import "time"

type Technician struct {
	ID            uint       `json:"id" gorm:"primaryKey;comment:客服(工作人员)ID"`
	MerchantID    uint       `json:"merchant_id" gorm:"index;uniqueIndex:uidx_merchant_code;uniqueIndex:uidx_merchant_account;comment:商户ID（外键关联merchants表）"`
	ServiceRoleID uint       `json:"service_role_id" gorm:"index;comment:客服类型ID（平台 service_roles）"`
	Phone         string     `json:"phone" gorm:"size:20;default:'';comment:绑定手机号（工作人员自身）"`
	Name          string     `json:"name" gorm:"size:50;comment:工作人员姓名"`
	Code          string     `json:"code" gorm:"size:20;uniqueIndex:uidx_merchant_code;comment:工作人员编号"`
	Account       string     `json:"account" gorm:"size:50;uniqueIndex:uidx_merchant_account;comment:登录账号（如js0001）"`
	Password      string     `json:"-" gorm:"size:255;comment:登录密码（bcrypt加密）"`
	IsActive      bool       `json:"is_active" gorm:"default:true;comment:是否启用"`
	CreatedAt     *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt     *time.Time `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`

	Merchant    Merchant    `json:"merchant" gorm:"foreignKey:MerchantID"`
	ServiceRole ServiceRole `json:"service_role" gorm:"foreignKey:ServiceRoleID"`
}

func (Technician) TableName() string {
	return "technicians"
}

func (Technician) TableComment() string {
	return "商户客服(工作人员)账号表"
}
