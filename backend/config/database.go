package config

import (
	"kabao/models"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	dsn := "root:root123@tcp(127.0.0.1:3306)/kabao?charset=utf8mb4&parseTime=True&loc=Local"
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}

	// 先删除旧的外键约束(忽略错误)
	DB.Exec("ALTER TABLE `cards` DROP FOREIGN KEY `fk_cards_user`")
	DB.Exec("ALTER TABLE `cards` DROP FOREIGN KEY `fk_cards_merchant`")
	DB.Exec("ALTER TABLE `usages` DROP FOREIGN KEY `fk_usages_card`")
	DB.Exec("ALTER TABLE `usages` DROP FOREIGN KEY `fk_usages_merchant`")
	DB.Exec("ALTER TABLE `notices` DROP FOREIGN KEY `fk_notices_merchant`")
	DB.Exec("ALTER TABLE `appointments` DROP FOREIGN KEY `fk_appointments_user`")
	DB.Exec("ALTER TABLE `appointments` DROP FOREIGN KEY `fk_appointments_merchant`")

	// 自动迁移
	err = DB.AutoMigrate(
		&models.User{},
		&models.Merchant{},
		&models.Card{},
		&models.Usage{},
		&models.Notice{},
		&models.Appointment{},
		&models.VerifyCode{},
	)
	if err != nil {
		log.Fatal("数据库迁移失败:", err)
	}

	log.Println("数据库初始化成功")

	// 初始化测试数据
	initTestData()
}

func initTestData() {
	// 检查是否已有数据
	var count int64
	DB.Model(&models.User{}).Count(&count)
	if count > 0 {
		return
	}

	// 创建测试用户
	users := []models.User{
		{Phone: "13800138001", Nickname: "张三"},
		{Phone: "13800138002", Nickname: "u1"},
		{Phone: "13800138003", Nickname: "u2"},
	}
	DB.Create(&users)

	// 创建测试商户
	merchants := []models.Merchant{
		{Name: "快剪理发店", Type: "理发", SupportAppointment: true, AvgServiceMinutes: 30},
		{Name: "顺风洗车", Type: "洗车", SupportAppointment: false, AvgServiceMinutes: 20},
	}
	DB.Create(&merchants)

	// 创建测试卡片
	cards := []models.Card{
		{
			UserID:         1,
			MerchantID:     1,
			CardNo:         "C1",
			CardType:       "10次专业剪发卡",
			TotalTimes:     10,
			RemainTimes:    8,
			UsedTimes:      2,
			RechargeAmount: 200,
			RechargeAt:     "2023-10-01",
			StartDate:      "2023-10-01",
			EndDate:        "2030-01-01",
		},
		{
			UserID:         1,
			MerchantID:     2,
			CardNo:         "C2",
			CardType:       "精洗美容次卡",
			TotalTimes:     10,
			RemainTimes:    4,
			UsedTimes:      6,
			RechargeAmount: 300,
			RechargeAt:     "2023-06-01",
			StartDate:      "2023-06-01",
			EndDate:        "2030-06-01",
		},
		{
			UserID:         1,
			MerchantID:     1,
			CardNo:         "C3",
			CardType:       "10次洗头卡",
			TotalTimes:     10,
			RemainTimes:    7,
			UsedTimes:      3,
			RechargeAmount: 200,
			RechargeAt:     "2023-10-01",
			LastUsedAt:     "2023-11-20 22:30:00",
			StartDate:      "2023-10-01",
			EndDate:        "2024-10-01",
		},
	}
	DB.Create(&cards)

	// 创建测试使用记录
	usages := []models.Usage{
		{CardID: 3, MerchantID: 1, UsedTimes: 1, UsedAt: "2023-11-20 22:30:00", Status: "success"},
		{CardID: 3, MerchantID: 1, UsedTimes: 1, UsedAt: "2023-11-15 17:30:00", Status: "success"},
	}
	DB.Create(&usages)

	// 创建测试通知
	notices := []models.Notice{
		{MerchantID: 1, Title: "元旦休息通知", Content: "本店将于1月1日至1月3日放假休息，敬请谅解...", CreatedAt: "2023-12-28"},
		{MerchantID: 1, Title: "会员升级活动", Content: "新老客户充值享8折优惠，仅限本周。", CreatedAt: "2023-12-20"},
	}
	DB.Create(&notices)

	// 创建测试预约
	appointments := []models.Appointment{
		{MerchantID: 1, UserID: 2, AppointmentTime: "2024-01-05 18:00:00", Status: "pending"},
		{MerchantID: 1, UserID: 3, AppointmentTime: "2024-01-05 17:00:00", Status: "confirmed"},
		{MerchantID: 1, UserID: 1, AppointmentTime: "2024-01-05 18:00:00", Status: "pending"},
	}
	DB.Create(&appointments)

	log.Println("测试数据初始化完成")
}
