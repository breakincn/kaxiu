package routes

import (
	"kabao/handlers"
	"kabao/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	SetupStaticRoutes(r)
	SetupUserRoutes(r)
	SetupMerchantRoutes(r)
	SetupAdminRoutes(r)
}

func SetupStaticRoutes(r *gin.Engine) {
	// 静态资源挂载
	r.Static("/uploads", "./uploads")
}

func SetupUserRoutes(r *gin.Engine) {
	user := r.Group("/user")

	// 公开接口（用户端）
	user.POST("/sms/send", handlers.SendSMSCode)
	user.POST("/login", handlers.UserLogin)
	user.POST("/register", handlers.UserRegister)

	// 用户端：店铺信息/直购（不需要登录）
	user.GET("/s/:slug", handlers.GetShopInfo)
	user.GET("/s/id/:id", handlers.GetShopInfoByID)

	// 需要认证的接口（用户端）
	auth := user.Group("")
	auth.Use(middleware.AuthMiddleware())
	auth.GET("/me", handlers.GetCurrentUser)
	auth.POST("/bind-phone", handlers.BindUserPhone)
	auth.PUT("/nickname", handlers.UpdateUserNickname)
	auth.GET("/code", handlers.GetUserCode)

	// 用户卡片
	auth.GET("/cards", handlers.GetCards)
	auth.GET("/cards/:id", handlers.GetCard)
	auth.GET("/users/:id/cards", handlers.GetUserCards)
	auth.POST("/cards/:id/verify-code", handlers.GenerateVerifyCode)
	auth.GET("/cards/:id/usages", handlers.GetCardUsages)

	// 用户预约/排队
	auth.GET("/users/:id/appointments", handlers.GetUserAppointments)
	auth.GET("/cards/:id/appointment", handlers.GetCardAppointment)
	auth.POST("/appointments", handlers.CreateAppointment)
	auth.PUT("/appointments/:id/cancel", handlers.CancelAppointment)

	// 用户端：直购流程
	auth.POST("/direct-purchase", handlers.CreateDirectPurchase)
	auth.POST("/direct-purchase/:order_no/confirm", handlers.ConfirmDirectPurchase)
	auth.GET("/direct-purchases", handlers.GetDirectPurchases)

	// 兼容：用户端创建用户（历史接口）
	user.POST("/users", handlers.CreateUser)

	// 平台公开接口（不需要登录）
	platform := r.Group("/platform")
	platform.GET("/service-roles", handlers.GetPlatformServiceRoles)
}

func SetupMerchantRoutes(r *gin.Engine) {
	merchant := r.Group("/merchant")

	// 公开接口（商户端）
	merchant.POST("/register", handlers.MerchantRegister)
	merchant.POST("/login", handlers.MerchantLogin)
	merchant.POST("/s/:slug/login", handlers.TechnicianLogin)

	// 需要认证的接口（商户端）
	auth := merchant.Group("")
	auth.Use(middleware.AuthMiddleware())

	// 商户自身
	auth.GET("/me", handlers.GetCurrentUserMerchant)
	auth.POST("/bind-phone", handlers.BindMerchantPhone)
	auth.PUT("/services", middleware.RequirePermission("merchant.service.update"), handlers.UpdateCurrentMerchantServices)
	auth.PUT("/info", middleware.RequirePermission("merchant.merchant.update"), handlers.UpdateMerchantInfo)
	auth.PUT("/business-status", middleware.RequirePermission("merchant.business_status.update"), handlers.ToggleMerchantBusinessStatus)
	auth.GET("/permissions", handlers.GetMyPermissions)
	// 搜索用户
	auth.GET("/users/search", handlers.MerchantSearchUsers)

	// 商户资源
	auth.GET("/merchants", handlers.GetMerchants)
	auth.GET("/merchants/:id", handlers.GetMerchant)
	auth.POST("/merchants", handlers.CreateMerchant)
	auth.PUT("/merchants/:id", handlers.UpdateMerchant)
	auth.GET("/merchants/:id/queue", handlers.GetQueueStatus)

	// 卡片（商户视角）
	auth.GET("/merchants/:id/cards", handlers.GetMerchantCards)
	auth.GET("/cards/:id", handlers.GetMerchantCard)
	auth.GET("/next-card-no", handlers.GetNextMerchantCardNo)
	auth.POST("/cards", middleware.RequirePermission("merchant.card.issue"), handlers.CreateCard)
	auth.PUT("/cards/:id", handlers.UpdateCard)

	// 核销（商户/技师）
	auth.POST("/verify", middleware.RequirePermission("merchant.card.verify"), handlers.VerifyCard)
	auth.POST("/verify/scan", middleware.RequireAnyPermission("merchant.card.verify", "merchant.card.finish"), handlers.ScanVerifyCard)
	auth.POST("/verify/finish", middleware.RequirePermission("merchant.card.finish"), handlers.FinishVerifyCard)
	auth.GET("/merchants/:id/today-verify", handlers.GetTodayVerify)

	// 使用记录
	auth.GET("/merchants/:id/usages", handlers.GetMerchantUsages)

	// 通知
	auth.GET("/merchants/:id/notices", handlers.GetMerchantNotices)
	auth.POST("/notices", handlers.CreateNotice)
	auth.DELETE("/notices/:id", handlers.DeleteNotice)
	auth.PUT("/notices/:id/pin", handlers.TogglePinNotice)

	// 预约（商户侧）
	auth.GET("/merchants/:id/appointments", handlers.GetMerchantAppointments)
	auth.GET("/merchants/:id/technicians", handlers.GetTechniciansByMerchantID)
	auth.GET("/merchants/:id/available-slots", handlers.GetAvailableTimeSlots)
	auth.PUT("/appointments/:id/confirm", handlers.ConfirmAppointment)
	auth.PUT("/appointments/:id/finish", handlers.FinishAppointment)

	// 技师自身
	auth.GET("/technician/me", handlers.GetCurrentTechnician)
	auth.POST("/technician/bind-phone", handlers.BindTechnicianPhone)

	// 技师账号管理
	auth.GET("/technicians", middleware.RequirePermission("merchant.tech.manage"), handlers.GetMerchantTechnicians)
	auth.POST("/technicians", middleware.RequirePermission("merchant.tech.manage"), handlers.CreateMerchantTechnician)
	auth.PUT("/technicians/:id", middleware.RequirePermission("merchant.tech.manage"), handlers.UpdateMerchantTechnician)
	auth.DELETE("/technicians/:id", middleware.RequirePermission("merchant.tech.manage"), handlers.DeleteMerchantTechnician)

	// 商户端：角色权限微调
	auth.GET("/role-permissions/:roleKey", handlers.GetMerchantRolePermissionOverrides)
	auth.POST("/role-permissions/:roleKey", handlers.SetMerchantRolePermissionOverrides)

	// ==================== Shop 模块（商户收款二维码 + 卡包直购） ====================
	// 商户端：收款配置
	auth.GET("/payment-config", middleware.RequirePermission("merchant.direct_sale.manage"), handlers.GetPaymentConfig)
	auth.POST("/payment-config", middleware.RequirePermission("merchant.direct_sale.manage"), handlers.SavePaymentConfig)
	auth.POST("/payment-qrcode/upload", middleware.RequirePermission("merchant.direct_sale.manage"), handlers.UploadPaymentQRCode)

	// 商户端：卡片模板管理
	auth.GET("/card-templates", middleware.RequireAnyPermission("merchant.direct_sale.manage", "merchant.card.sell"), handlers.GetCardTemplates)
	auth.POST("/card-templates", middleware.RequirePermission("merchant.direct_sale.manage"), handlers.CreateCardTemplate)
	auth.PUT("/card-templates/:id", middleware.RequirePermission("merchant.direct_sale.manage"), handlers.UpdateCardTemplate)
	auth.DELETE("/card-templates/:id", middleware.RequirePermission("merchant.direct_sale.manage"), handlers.DeleteCardTemplate)

	// 商户端：店铺短链接
	auth.GET("/shop-slug", middleware.RequirePermission("merchant.direct_sale.manage"), handlers.GetShopSlug)
	auth.POST("/shop-slug", middleware.RequirePermission("merchant.direct_sale.manage"), handlers.SaveShopSlug)

	// 商户端：直购订单
	auth.GET("/direct-purchases", middleware.RequirePermission("merchant.direct_sale.manage"), handlers.GetMerchantDirectPurchases)
	auth.POST("/direct-purchases/:order_no/confirm", middleware.RequirePermission("merchant.direct_sale.manage"), handlers.MerchantConfirmDirectPurchase)
}

func SetupAdminRoutes(r *gin.Engine) {
	admin := r.Group("/admin")

	admin.Use(middleware.PlatformAdminMiddleware())
	admin.GET("/service-roles", handlers.AdminListServiceRoles)
	admin.POST("/service-roles", handlers.AdminCreateServiceRole)
	admin.PUT("/service-roles/:id", handlers.AdminUpdateServiceRole)
	admin.DELETE("/service-roles/:id", handlers.AdminDeleteServiceRole)
	admin.GET("/system/config", handlers.AdminGetSystemConfig)
	admin.PUT("/system/config", handlers.AdminUpdateSystemConfig)
	admin.GET("/permissions", handlers.AdminListPermissions)
	admin.POST("/permissions", handlers.AdminCreatePermission)
	admin.PUT("/permissions/:id", handlers.AdminUpdatePermission)
	admin.DELETE("/permissions/:id", handlers.AdminDeletePermission)
	admin.GET("/service-roles/:roleId/permissions", handlers.AdminGetRolePermissions)
	admin.POST("/service-roles/:roleId/permissions", handlers.AdminSetRolePermissions)
}
