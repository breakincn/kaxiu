package main

import (
	"kabao/config"
	"kabao/middleware"
	"kabao/routes"
	"kabao/scheduler"
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化数据库
	config.InitDB()

	// 启动服务会话调度器（自动分房/延迟开计时/自动结单/回收空闲）
	scheduler.StartServiceSessionScheduler()

	// 初始化 Gin
	r := gin.Default()

	// CORS 配置
	r.Use(cors.New(middleware.CORSMiddleware()))

	// 注册路由
	routes.SetupRoutes(r)

	// 启动服务
	log.Println("卡包后端服务启动于 :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("服务启动失败:", err)
	}
}
