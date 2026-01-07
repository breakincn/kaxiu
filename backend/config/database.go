package config

import (
	"kabao/models"
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	dsn := os.Getenv("KABAO_DSN")
	if dsn == "" {
		dbConfig := GetDatabaseConfig()
		dsn = dbConfig.GetDSN()
	}
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
		&models.Technician{},
		&models.ServiceRole{},
		&models.Permission{},
		&models.RolePermission{},
		&models.MerchantRolePermissionOverride{},
		&models.SystemConfig{},
		&models.Card{},
		&models.Usage{},
		&models.Notice{},
		&models.Appointment{},
		&models.VerifyCode{},
		&models.SMSCode{},
		&models.InviteCode{},
		// Shop 模块（商户收款二维码 + 卡包直购）
		&models.PaymentConfig{},
		&models.CardTemplate{},
		&models.DirectPurchase{},
		&models.MerchantShopSlug{},
	)
	if err != nil {
		log.Fatal("数据库迁移失败:", err)
	}

	// 兼容历史数据：为旧用户补充默认 username，避免新增唯一索引导致异常
	DB.Exec("UPDATE users SET username = CONCAT('u', id) WHERE username IS NULL OR username = ''")

	// 技师账号唯一性调整：从 account 全局唯一改为 (merchant_id, account) 商户内唯一
	// 尝试删除旧的 account 唯一索引（不同环境下索引名可能不同，忽略错误即可）
	DB.Exec("ALTER TABLE `technicians` DROP INDEX `idx_technicians_account`")
	DB.Exec("ALTER TABLE `technicians` DROP INDEX `account`")

	// 添加表注释
	DB.Exec("ALTER TABLE `users` COMMENT = '用户表'")
	DB.Exec("ALTER TABLE `merchants` COMMENT = '商户表'")
	DB.Exec("ALTER TABLE `technicians` COMMENT = '商户技师账号表'")
	DB.Exec("ALTER TABLE `service_roles` COMMENT = '平台客服类型表'")
	DB.Exec("ALTER TABLE `permissions` COMMENT = '权限枚举表'")
	DB.Exec("ALTER TABLE `role_permissions` COMMENT = '平台角色默认权限表'")
	DB.Exec("ALTER TABLE `merchant_role_permission_overrides` COMMENT = '商户角色权限微调表'")
	DB.Exec("ALTER TABLE `cards` COMMENT = '用户会员卡表'")
	DB.Exec("ALTER TABLE `usages` COMMENT = '卡片使用记录表'")
	DB.Exec("ALTER TABLE `notices` COMMENT = '商户通知表'")
	DB.Exec("ALTER TABLE `appointments` COMMENT = '用户预约排队表'")
	DB.Exec("ALTER TABLE `verify_codes` COMMENT = '核销码表'")
	DB.Exec("ALTER TABLE `sms_codes` COMMENT = '短信验证码表'")
	DB.Exec("ALTER TABLE `invite_codes` COMMENT = '邀请码表'")
	// Shop 模块表注释
	DB.Exec("ALTER TABLE `payment_configs` COMMENT = '商户收款配置表'")
	DB.Exec("ALTER TABLE `card_templates` COMMENT = '卡片售卖模板表'")
	DB.Exec("ALTER TABLE `direct_purchases` COMMENT = '直购订单记录表'")
	DB.Exec("ALTER TABLE `merchant_shop_slugs` COMMENT = '商户店铺短链接表'")

	log.Println("数据库初始化成功")

	// 初始化商户注册邀请码（幂等）
	initInviteCodes()
	initServiceRoles()
	initPermissions()
	initRolePermissions()

	// 初始化测试数据
	initTestData()
}

func initServiceRoles() {
	defaults := []models.ServiceRole{
		{Key: "technician", Name: "技师", Description: "技师账号", IsActive: true, AllowPermissionAdjust: true, Sort: 10},
		{Key: "teacher", Name: "老师", Description: "老师", IsActive: true, AllowPermissionAdjust: false, Sort: 20},
		{Key: "coach", Name: "教练", Description: "教练", IsActive: true, AllowPermissionAdjust: false, Sort: 30},
		{Key: "pet_doctor", Name: "宠物医生", Description: "宠物医生", IsActive: true, AllowPermissionAdjust: false, Sort: 40},
	}

	for _, r := range defaults {
		var existing models.ServiceRole
		if err := DB.Where("`key` = ?", r.Key).First(&existing).Error; err == nil {
			updates := map[string]interface{}{}
			if existing.Name == "" {
				updates["name"] = r.Name
			}
			if existing.Description == "" {
				updates["description"] = r.Description
			}
			if existing.Sort == 0 {
				updates["sort"] = r.Sort
			}
			if len(updates) > 0 {
				DB.Model(&models.ServiceRole{}).Where("id = ?", existing.ID).Updates(updates)
			}
			continue
		}
		DB.Create(&r)
	}
}

func initInviteCodes() {
	defaultCodes := []string{
		"KABAO-20260101-001",
		"KABAO-20260101-002",
		"KABAO-20260101-003",
		"KABAO-20260101-004",
		"KABAO-20260101-005",
	}

	for _, code := range defaultCodes {
		var existing models.InviteCode
		if err := DB.Where("code = ?", code).First(&existing).Error; err == nil {
			continue
		}
		DB.Create(&models.InviteCode{Code: code, Used: false})
	}
}

func initTestData() {
	// 检查是否已有数据
	var count int64
	DB.Model(&models.User{}).Count(&count)
	if count > 0 {
		return
	}

	// 创建测试用户
	p1 := "13800138001"
	p2 := "13800138002"
	p3 := "13800138003"
	users := []models.User{
		{Username: "u1", Phone: &p1, Nickname: "张三"},
		{Username: "u2", Phone: &p2, Nickname: "u1"},
		{Username: "u3", Phone: &p3, Nickname: "u2"},
	}
	DB.Create(&users)

	// 创建测试商户
	merchants := []models.Merchant{
		{Name: "快剪理发店", Type: "理发", SupportAppointment: true, AvgServiceMinutes: 30},
		{Name: "顺风洗车", Type: "洗车", SupportAppointment: false, AvgServiceMinutes: 20},
	}
	DB.Create(&merchants)

	// 创建测试卡片
	parseDate := func(s string) *time.Time {
		t, err := time.ParseInLocation("2006-01-02", s, time.Local)
		if err != nil {
			return nil
		}
		return &t
	}
	parseDateTime := func(s string) *time.Time {
		t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local)
		if err != nil {
			return nil
		}
		return &t
	}
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
			RechargeAt:     parseDate("2023-10-01"),
			StartDate:      parseDate("2023-10-01"),
			EndDate:        parseDate("2030-01-01"),
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
			RechargeAt:     parseDate("2023-06-01"),
			StartDate:      parseDate("2023-06-01"),
			EndDate:        parseDate("2030-06-01"),
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
			RechargeAt:     parseDate("2023-10-01"),
			LastUsedAt:     parseDateTime("2023-11-20 22:30:00"),
			StartDate:      parseDate("2023-10-01"),
			EndDate:        parseDate("2024-10-01"),
		},
	}
	DB.Create(&cards)

	// 创建测试使用记录
	usages := []models.Usage{
		{CardID: 3, MerchantID: 1, UsedTimes: 1, UsedAt: parseDateTime("2023-11-20 22:30:00"), Status: "success"},
		{CardID: 3, MerchantID: 1, UsedTimes: 1, UsedAt: parseDateTime("2023-11-15 17:30:00"), Status: "success"},
	}
	DB.Create(&usages)

	// 创建测试通知
	notices := []models.Notice{
		{MerchantID: 1, Title: "元旦休息通知", Content: "本店将于1月1日至1月3日放假休息，敬请谅解..."},
		{MerchantID: 1, Title: "会员升级活动", Content: "新老客户充值享8折优惠，仅限本周。"},
	}
	DB.Create(&notices)

	// 创建测试预约
	appointments := []models.Appointment{
		{MerchantID: 1, UserID: 2, AppointmentTime: parseDateTime("2024-01-05 18:00:00"), Status: "pending"},
		{MerchantID: 1, UserID: 3, AppointmentTime: parseDateTime("2024-01-05 17:00:00"), Status: "confirmed"},
		{MerchantID: 1, UserID: 1, AppointmentTime: parseDateTime("2024-01-05 18:00:00"), Status: "pending"},
	}
	DB.Create(&appointments)

	log.Println("测试数据初始化完成")
}

func initPermissions() {
	defaults := []models.Permission{
		{Key: "merchant.tech.manage", Name: "管理技师账号", Group: "客服管理", Description: "新增/编辑/禁用/删除技师账号", Sort: 10},
		{Key: "merchant.card.issue", Name: "发卡/开卡", Group: "卡片", Description: "创建卡片、发卡", Sort: 20},
		{Key: "merchant.card.verify", Name: "核销", Group: "卡片", Description: "核销会员卡", Sort: 30},
		{Key: "merchant.card.finish", Name: "结单", Group: "卡片", Description: "技师扫码结单，将进行中核销置为完成", Sort: 31},
		{Key: "merchant.card.sell", Name: "售卡", Group: "卡片", Description: "技师售卡：查询售卡模板、生成售卡二维码", Sort: 32},
		{Key: "merchant.direct_sale.manage", Name: "售卡管理", Group: "售卡", Description: "管理直购售卡配置、模板、订单等", Sort: 35},
		{Key: "merchant.merchant.update", Name: "修改商户信息", Group: "商户", Description: "更新商户基本信息", Sort: 40},
		{Key: "merchant.service.update", Name: "修改商户服务设置", Group: "商户", Description: "更新商户服务能力开关", Sort: 50},
		{Key: "merchant.business_status.update", Name: "切换营业状态", Group: "商户", Description: "操作营业中/打烊", Sort: 60},
	}

	for _, p := range defaults {
		var existing models.Permission
		if err := DB.Where("`key` = ?", p.Key).First(&existing).Error; err == nil {
			updates := map[string]interface{}{}
			if existing.Name == "" {
				updates["name"] = p.Name
			}
			if existing.Group == "" {
				updates["group"] = p.Group
			}
			if existing.Description == "" {
				updates["description"] = p.Description
			}
			if existing.Sort == 0 {
				updates["sort"] = p.Sort
			}
			if len(updates) > 0 {
				DB.Model(&models.Permission{}).Where("id = ?", existing.ID).Updates(updates)
			}
			continue
		}
		DB.Create(&p)
	}
}

func initRolePermissions() {
	// 默认策略：技师允许核销和售卡；其它角色默认无权限（后续可在平台后台配置）
	var technicianRole models.ServiceRole
	if err := DB.Where("`key` = ?", "technician").First(&technicianRole).Error; err != nil {
		return
	}

	var permVerify models.Permission
	if err := DB.Where("`key` = ?", "merchant.card.verify").First(&permVerify).Error; err != nil {
		return
	}
	var permFinish models.Permission
	if err := DB.Where("`key` = ?", "merchant.card.finish").First(&permFinish).Error; err != nil {
		return
	}
	var permSell models.Permission
	if err := DB.Where("`key` = ?", "merchant.card.sell").First(&permSell).Error; err != nil {
		return
	}

	var existing models.RolePermission
	if err := DB.Where("service_role_id = ? AND permission_id = ?", technicianRole.ID, permVerify.ID).First(&existing).Error; err != nil {
		DB.Create(&models.RolePermission{ServiceRoleID: technicianRole.ID, PermissionID: permVerify.ID, Allowed: true})
	}
	if err := DB.Where("service_role_id = ? AND permission_id = ?", technicianRole.ID, permFinish.ID).First(&existing).Error; err != nil {
		DB.Create(&models.RolePermission{ServiceRoleID: technicianRole.ID, PermissionID: permFinish.ID, Allowed: true})
	}
	if err := DB.Where("service_role_id = ? AND permission_id = ?", technicianRole.ID, permSell.ID).First(&existing).Error; err != nil {
		DB.Create(&models.RolePermission{ServiceRoleID: technicianRole.ID, PermissionID: permSell.ID, Allowed: true})
	}
}
