package models

import "time"

type Technician struct {
	ID         uint       `json:"id" gorm:"primaryKey;comment:技师ID"`
	MerchantID uint       `json:"merchant_id" gorm:"index;uniqueIndex:uidx_merchant_code;uniqueIndex:uidx_merchant_account;comment:商户ID（外键关联merchants表）"`
	Name       string     `json:"name" gorm:"size:50;comment:技师姓名"`
	Code       string     `json:"code" gorm:"size:20;uniqueIndex:uidx_merchant_code;comment:技师编号"`
	Account    string     `json:"account" gorm:"size:50;uniqueIndex:uidx_merchant_account;comment:登录账号（如js0001）"`
	Password   string     `json:"-" gorm:"size:255;comment:登录密码（bcrypt加密）"`
	CreatedAt  *time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`

	Merchant Merchant `json:"merchant" gorm:"foreignKey:MerchantID"`
}

func (Technician) TableName() string {
	return "technicians"
}

func (Technician) TableComment() string {
	return "商户技师账号表"
}
