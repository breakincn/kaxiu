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
	config.InitDB()

	// 启动服务会话调度器（自动分房/延迟开计时/自动结单/回收空闲）
	scheduler.StartServiceSessionScheduler()

	r := gin.Default()
	r.Use(cors.New(middleware.CORSMiddleware()))

	routes.SetupStaticRoutes(r)
	routes.SetupMerchantRoutes(r)

	log.Println("Kabao Merchant API service listening on :8082")
	if err := r.Run(":8082"); err != nil {
		log.Fatal("服务启动失败:", err)
	}
}
