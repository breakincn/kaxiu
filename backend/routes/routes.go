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
	auth.GET("/cards/:id/projects", handlers.GetCardProjects)
	auth.GET("/users/:id/cards", handlers.GetUserCards)
	auth.POST("/cards/:id/verify-code", handlers.GenerateVerifyCode)
	auth.GET("/cards/:id/usages", handlers.GetCardUsages)
	auth.PUT("/usages/:id/revoke", handlers.UserRevokeUsage)
	auth.GET("/verify-codes/:code/status", handlers.UserGetVerifyCodeStatus)

	// 服务会话（用户端：选房/选工作人员/查询）
	auth.GET("/service-sessions/:id", handlers.UserGetServiceSession)
	auth.GET("/service-sessions/:id/rooms", handlers.UserListAvailableRooms)
	auth.POST("/service-sessions/:id/resume", handlers.UserResumeServiceSession)
	auth.POST("/service-sessions/:id/room", handlers.UserChooseServiceSessionRoom)
	auth.GET("/service-sessions/:id/technicians", handlers.UserListAvailableTechnicians)
	auth.POST("/service-sessions/:id/technician", handlers.UserChooseServiceSessionTechnician)
	auth.POST("/service-sessions/:id/extend", handlers.ExtendServiceSessionUser)

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
	auth.PUT("/services", middleware.RequirePermission("merchant.service.manage"), handlers.UpdateCurrentMerchantServices)
	auth.PUT("/info", middleware.RequirePermission("merchant.info.manage"), handlers.UpdateMerchantInfo)
	auth.PUT("/technician-alias", handlers.UpdateTechnicianAlias)
	auth.PUT("/business-status", middleware.RequirePermission("merchant.business_status.manage"), handlers.ToggleMerchantBusinessStatus)
	auth.GET("/permissions", handlers.GetMyPermissions)
	auth.GET("/config", handlers.GetConfig)
	// 搜索用户
	auth.GET("/users/search", handlers.MerchantSearchUsers)

	// 商户资源
	auth.GET("/merchants", handlers.GetMerchants)
	auth.GET("/merchants/:id", handlers.GetMerchant)
	auth.PUT("/merchants/:id", middleware.RequirePermission("merchant.info.manage"), handlers.UpdateMerchant)
	auth.GET("/merchants/:id/queue", handlers.GetQueueStatus)

	// 商户项目（项目设置）
	auth.GET("/projects", middleware.RequirePermission("merchant.service.manage"), handlers.ListMerchantProjects)
	auth.POST("/projects", middleware.RequirePermission("merchant.service.manage"), handlers.CreateMerchantProject)
	auth.PUT("/projects/:id", middleware.RequirePermission("merchant.service.manage"), handlers.UpdateMerchantProject)
	auth.DELETE("/projects/:id", middleware.RequirePermission("merchant.service.manage"), handlers.DeleteMerchantProject)

	// 卡片（商户视角）
	auth.GET("/merchants/:id/cards", handlers.GetMerchantCards)
	auth.GET("/cards/:id", handlers.GetMerchantCard)
	auth.GET("/next-card-no", handlers.GetNextMerchantCardNo)
	auth.POST("/cards", middleware.RequirePermission("merchant.card.issue"), handlers.CreateCard)
	auth.PUT("/cards/:id", middleware.RequirePermission("merchant.card.issue"), handlers.UpdateCard)

	// 核销（商户/技师）
	auth.POST("/verify", middleware.RequirePermission("merchant.card.verify"), handlers.VerifyCard)
	auth.POST("/verify/scan", middleware.RequireAnyPermission("merchant.card.verify", "merchant.card.finish"), handlers.ScanVerifyCard)
	auth.GET("/merchants/:id/today-verify", handlers.GetTodayVerify)

	// 使用记录
	auth.GET("/merchants/:id/usages", handlers.GetMerchantUsages)

	// 通知
	auth.GET("/merchants/:id/notices", handlers.GetMerchantNotices)
	auth.POST("/notices", middleware.RequirePermission("merchant.notice.manage"), handlers.CreateNotice)
	auth.DELETE("/notices/:id", middleware.RequirePermission("merchant.notice.manage"), handlers.DeleteNotice)
	auth.PUT("/notices/:id/pin", middleware.RequirePermission("merchant.notice.manage"), handlers.TogglePinNotice)

	// 预约（商户侧）
	auth.GET("/merchants/:id/appointments", handlers.GetMerchantAppointments)
	auth.GET("/merchants/:id/technicians", handlers.GetTechniciansByMerchantID)
	auth.GET("/merchants/:id/available-slots", handlers.GetAvailableTimeSlots)
	auth.PUT("/appointments/:id/confirm", middleware.RequirePermission("merchant.appointment.manage"), handlers.ConfirmAppointment)
	auth.PUT("/appointments/:id/finish", middleware.RequirePermission("merchant.appointment.manage"), handlers.FinishAppointment)

	// 技师自身
	auth.GET("/technician/me", handlers.GetCurrentTechnician)
	auth.POST("/technician/bind-phone", handlers.BindTechnicianPhone)

	// 技师账号管理
	auth.GET("/technicians", middleware.RequirePermission("merchant.cs.manage"), handlers.GetMerchantTechnicians)
	auth.POST("/technicians", middleware.RequirePermission("merchant.cs.manage"), handlers.CreateMerchantTechnician)
	auth.PUT("/technicians/:id", middleware.RequirePermission("merchant.cs.manage"), handlers.UpdateMerchantTechnician)
	auth.DELETE("/technicians/:id", middleware.RequirePermission("merchant.cs.manage"), handlers.DeleteMerchantTechnician)

	// 商户端：专业客服岗位（称谓+前缀）
	auth.GET("/professional-roles", middleware.RequirePermission("merchant.cs.manage"), handlers.GetMerchantProfessionalRoles)
	auth.POST("/professional-roles", middleware.RequirePermission("merchant.cs.manage"), handlers.CreateMerchantProfessionalRole)

	// 房间管理
	auth.GET("/rooms", middleware.RequireAnyPermission("merchant.room.view", "merchant.service.manage"), handlers.ListRooms)
	auth.POST("/rooms", middleware.RequirePermission("merchant.service.manage"), handlers.CreateRoom)
	auth.PUT("/rooms/:id", middleware.RequirePermission("merchant.service.manage"), handlers.UpdateRoom)
	auth.DELETE("/rooms/:id", middleware.RequirePermission("merchant.service.manage"), handlers.DeleteRoom)

	// 看板（Table）：房间/客服状态
	auth.GET("/table/rooms", middleware.RequirePermission("merchant.room.view"), handlers.TableRooms)
	auth.GET("/table/staff", middleware.RequirePermission("merchant.staff.view"), handlers.TableStaff)

	// 工作人员签到/状态
	auth.POST("/technician/checkin", handlers.TechnicianCheckIn)
	auth.POST("/technician/checkout", handlers.TechnicianCheckOut)
	auth.PUT("/technician/status", handlers.UpdateTechnicianServiceStatus)
	auth.GET("/technicians/available", handlers.ListAvailableTechnicians)
	auth.GET("/technician/attendance", handlers.GetCurrentTechnicianAttendance)

	// 服务会话（先提供查询，后续补齐核销创建/选房/选人/预结单/加钟）
	auth.GET("/service-sessions", handlers.ListServiceSessions)
	auth.GET("/service-sessions/:id", handlers.GetServiceSession)
	auth.POST("/service-sessions/:id/room", handlers.ChooseServiceSessionRoom)
	auth.POST("/service-sessions/:id/technician", handlers.ChooseServiceSessionTechnician)
	auth.POST("/service-sessions/:id/extend", handlers.ExtendServiceSession)
	auth.POST("/service-sessions/:id/extend-duration", handlers.ExtendServiceSessionDuration)

	// 商户端：角色权限微调
	auth.GET("/role-permissions/:roleKey", handlers.GetMerchantRolePermissionOverrides)
	auth.POST("/role-permissions/:roleKey", middleware.RequirePermission("merchant.permission.adjust"), handlers.SetMerchantRolePermissionOverrides)

	// ==================== Shop 模块（商户收款二维码 + 卡包直购） ====================
	// 商户端：收款配置
	auth.GET("/payment-config", middleware.RequirePermission("merchant.direct_sale.manage"), handlers.GetPaymentConfig)
	auth.POST("/payment-config", middleware.RequirePermission("merchant.direct_sale.manage"), handlers.SavePaymentConfig)
	auth.POST("/payment-qrcode/upload", middleware.RequirePermission("merchant.direct_sale.manage"), handlers.UploadPaymentQRCode)

	// 商户端：卡片模板管理
	auth.GET("/card-templates", middleware.RequirePermission("merchant.card.sell"), handlers.GetCardTemplates)
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

	// 平台管理商户
	admin.POST("/merchants", handlers.CreateMerchant)
	admin.PUT("/merchants/:id", handlers.UpdateMerchant)
}
