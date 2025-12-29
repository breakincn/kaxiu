package routes

import (
	"kabao/handlers"
	"kabao/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	api := r.Group("/api")

	// 公开接口（不需要登录）
	api.POST("/login", handlers.UserLogin)
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

	// 商户相关
	auth.GET("/merchants", handlers.GetMerchants)
	auth.GET("/merchants/:id", handlers.GetMerchant)
	auth.POST("/merchants", handlers.CreateMerchant)
	auth.PUT("/merchants/:id", handlers.UpdateMerchant)

	// 卡片相关
	auth.GET("/cards", handlers.GetCards)
	auth.GET("/cards/:id", handlers.GetCard)
	auth.GET("/users/:id/cards", handlers.GetUserCards)
	auth.GET("/merchants/:id/cards", handlers.GetMerchantCards)
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
	auth.GET("/users/:id/appointments", handlers.GetUserAppointments)
	auth.GET("/cards/:id/appointment", handlers.GetCardAppointment)
	auth.POST("/appointments", handlers.CreateAppointment)
	auth.PUT("/appointments/:id/confirm", handlers.ConfirmAppointment)
	auth.PUT("/appointments/:id/finish", handlers.FinishAppointment)
	auth.PUT("/appointments/:id/cancel", handlers.CancelAppointment)
	auth.GET("/merchants/:id/queue", handlers.GetQueueStatus)
	auth.GET("/merchants/:id/available-slots", handlers.GetAvailableTimeSlots)
}
