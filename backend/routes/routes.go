package routes

import (
	"kabao/handlers"
	"kabao/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	// 线上常见仅反代 /api，因此静态资源挂到 /api/uploads
	// 同时保留 /uploads 兼容旧数据
	r.Static("/api/uploads", "./uploads")
	r.Static("/uploads", "./uploads")

	api := r.Group("/api")

	// 公开接口（不需要登录）
	api.POST("/sms/send", handlers.SendSMSCode)
	api.POST("/login", handlers.UserLogin)
	api.POST("/user/register", handlers.UserRegister)
	api.POST("/users", handlers.CreateUser)
	api.POST("/merchant/register", handlers.MerchantRegister)
	api.POST("/merchant/login", handlers.MerchantLogin)

	// 需要认证的接口
	auth := api.Group("")
	auth.Use(middleware.AuthMiddleware())

	// 用户相关
	auth.GET("/users", handlers.GetUsers)
	auth.GET("/users/:id", handlers.GetUser)
	auth.GET("/me", handlers.GetCurrentUser)
	auth.POST("/user/bind-phone", handlers.BindUserPhone)
	auth.PUT("/user/nickname", handlers.UpdateUserNickname)
	auth.GET("/user/code", handlers.GetUserCode)
	auth.GET("/merchant/users/search", handlers.MerchantSearchUsers)

	// 商户相关
	auth.GET("/merchants", handlers.GetMerchants)
	auth.GET("/merchants/:id", handlers.GetMerchant)
	auth.POST("/merchants", handlers.CreateMerchant)
	auth.PUT("/merchants/:id", handlers.UpdateMerchant)
	auth.GET("/merchant/me", handlers.GetCurrentUserMerchant)
	auth.POST("/merchant/bind-phone", handlers.BindMerchantPhone)
	auth.PUT("/merchant/services", handlers.UpdateCurrentMerchantServices)
	auth.PUT("/merchant/info", handlers.UpdateMerchantInfo)
	auth.PUT("/merchant/business-status", handlers.ToggleMerchantBusinessStatus)

	// 技师账号管理
	auth.GET("/merchant/technicians", handlers.GetMerchantTechnicians)
	auth.POST("/merchant/technicians", handlers.CreateMerchantTechnician)

	// 卡片相关
	auth.GET("/cards", handlers.GetCards)
	auth.GET("/cards/:id", handlers.GetCard)
	auth.GET("/users/:id/cards", handlers.GetUserCards)
	auth.GET("/merchants/:id/cards", handlers.GetMerchantCards)
	auth.GET("/merchant/cards/:id", handlers.GetMerchantCard)
	auth.GET("/merchant/next-card-no", handlers.GetNextMerchantCardNo)
	auth.POST("/cards", handlers.CreateCard)
	auth.PUT("/cards/:id", handlers.UpdateCard)

	// 核销相关
	auth.POST("/cards/:id/verify-code", handlers.GenerateVerifyCode)
	auth.POST("/verify", handlers.VerifyCard)
	auth.GET("/merchants/:id/today-verify", handlers.GetTodayVerify)

	// 使用记录
	auth.GET("/cards/:id/usages", handlers.GetCardUsages)
	auth.GET("/merchants/:id/usages", handlers.GetMerchantUsages)

	// 通知相关
	auth.GET("/merchants/:id/notices", handlers.GetMerchantNotices)
	auth.POST("/notices", handlers.CreateNotice)
	auth.DELETE("/notices/:id", handlers.DeleteNotice)
	auth.PUT("/notices/:id/pin", handlers.TogglePinNotice)

	// 预约相关
	auth.GET("/merchants/:id/appointments", handlers.GetMerchantAppointments)
	auth.GET("/merchants/:id/technicians", handlers.GetTechniciansByMerchantID)
	auth.GET("/users/:id/appointments", handlers.GetUserAppointments)
	auth.GET("/cards/:id/appointment", handlers.GetCardAppointment)
	auth.POST("/appointments", handlers.CreateAppointment)
	auth.PUT("/appointments/:id/confirm", handlers.ConfirmAppointment)
	auth.PUT("/appointments/:id/finish", handlers.FinishAppointment)
	auth.PUT("/appointments/:id/cancel", handlers.CancelAppointment)
	auth.GET("/merchants/:id/queue", handlers.GetQueueStatus)
	auth.GET("/merchants/:id/available-slots", handlers.GetAvailableTimeSlots)

	// ==================== Shop 模块（商户收款二维码 + 卡包直购） ====================
	// 公开接口（无需登录）
	api.GET("/shop/:slug", handlers.GetShopInfo)
	api.GET("/shop/id/:id", handlers.GetShopInfoByID)

	// 商户端：收款配置
	auth.GET("/merchant/payment-config", handlers.GetPaymentConfig)
	auth.POST("/merchant/payment-config", handlers.SavePaymentConfig)
	auth.POST("/merchant/payment-qrcode/upload", handlers.UploadPaymentQRCode)

	// 商户端：卡片模板管理
	auth.GET("/merchant/card-templates", handlers.GetCardTemplates)
	auth.POST("/merchant/card-templates", handlers.CreateCardTemplate)
	auth.PUT("/merchant/card-templates/:id", handlers.UpdateCardTemplate)
	auth.DELETE("/merchant/card-templates/:id", handlers.DeleteCardTemplate)

	// 商户端：店铺短链接
	auth.GET("/merchant/shop-slug", handlers.GetShopSlug)
	auth.POST("/merchant/shop-slug", handlers.SaveShopSlug)

	// 商户端：直购订单
	auth.GET("/merchant/direct-purchases", handlers.GetMerchantDirectPurchases)
	auth.POST("/merchant/direct-purchases/:order_no/confirm", handlers.MerchantConfirmDirectPurchase)

	// 用户端：直购流程
	auth.POST("/direct-purchase", handlers.CreateDirectPurchase)
	auth.POST("/direct-purchase/:order_no/confirm", handlers.ConfirmDirectPurchase)
	auth.GET("/direct-purchases", handlers.GetDirectPurchases)
}
