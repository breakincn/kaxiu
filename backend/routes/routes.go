package routes

import (
	"kabao/handlers"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	api := r.Group("/api")

	// 用户相关
	api.GET("/users", handlers.GetUsers)
	api.GET("/users/:id", handlers.GetUser)
	api.POST("/users", handlers.CreateUser)

	// 商户相关
	api.GET("/merchants", handlers.GetMerchants)
	api.GET("/merchants/:id", handlers.GetMerchant)
	api.POST("/merchants", handlers.CreateMerchant)
	api.PUT("/merchants/:id", handlers.UpdateMerchant)

	// 卡片相关
	api.GET("/cards", handlers.GetCards)
	api.GET("/cards/:id", handlers.GetCard)
	api.GET("/users/:id/cards", handlers.GetUserCards)
	api.GET("/merchants/:id/cards", handlers.GetMerchantCards)
	api.POST("/cards", handlers.CreateCard)
	api.PUT("/cards/:id", handlers.UpdateCard)

	// 核销相关
	api.POST("/cards/:id/verify-code", handlers.GenerateVerifyCode)
	api.POST("/verify", handlers.VerifyCard)
	api.GET("/merchants/:id/today-verify", handlers.GetTodayVerify)

	// 使用记录
	api.GET("/cards/:id/usages", handlers.GetCardUsages)
	api.GET("/merchants/:id/usages", handlers.GetMerchantUsages)

	// 通知相关
	api.GET("/merchants/:id/notices", handlers.GetMerchantNotices)
	api.POST("/notices", handlers.CreateNotice)

	// 预约相关
	api.GET("/merchants/:id/appointments", handlers.GetMerchantAppointments)
	api.GET("/users/:id/appointments", handlers.GetUserAppointments)
	api.GET("/cards/:id/appointment", handlers.GetCardAppointment)
	api.POST("/appointments", handlers.CreateAppointment)
	api.PUT("/appointments/:id/confirm", handlers.ConfirmAppointment)
	api.PUT("/appointments/:id/finish", handlers.FinishAppointment)
	api.PUT("/appointments/:id/cancel", handlers.CancelAppointment)
	api.GET("/merchants/:id/queue", handlers.GetQueueStatus)
}
