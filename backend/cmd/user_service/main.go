package main

import (
	"kabao/config"
	"kabao/middleware"
	"kabao/queue"
	"kabao/routes"
	"kabao/scheduler"
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	config.LoadEnv()
	config.InitDB()
	if err := queue.InitDefaultStoreFromEnv(); err != nil {
		log.Fatal("初始化Redis队列失败:", err)
	}

	// 启动服务会话调度器（自动分房/延迟开计时/自动结单/回收空闲）
	scheduler.StartServiceSessionScheduler()
	// 启动预约调度器（未选客服预约：10分钟后自动分配客服/失败原因写入）
	scheduler.StartAppointmentScheduler()

	r := gin.Default()
	r.Use(cors.New(middleware.CORSMiddleware()))

	routes.SetupStaticRoutes(r)
	routes.SetupUserRoutes(r)

	log.Println("Kabao User API service listening on :8081")
	if err := r.Run(":8081"); err != nil {
		log.Fatal("服务启动失败:", err)
	}
}
