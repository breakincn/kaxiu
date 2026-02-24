package main

import (
	"kabao/config"
	"kabao/middleware"
	"kabao/queue"
	"kabao/routes"
	"kabao/scheduler"
	"log"
	"os"
	"path/filepath"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	config.LoadEnv()
	// 初始化数据库
	config.InitDB()
	if err := queue.InitDefaultStoreFromEnv(); err != nil {
		log.Fatal("初始化Redis队列失败:", err)
	}

	// 启动服务会话调度器（自动分房/延迟开计时/自动结单/回收空闲）
	scheduler.StartServiceSessionScheduler()
	// 启动预约调度器（未选客服预约：10分钟后自动分配客服/失败原因写入）
	scheduler.StartAppointmentScheduler()
	// 启动手牌调度器（手牌未归还超时锁卡）
	scheduler.StartHandCardScheduler()

	// 初始化 Gin
	r := gin.Default()

	// CORS 配置
	r.Use(cors.New(middleware.CORSMiddleware()))

	// 注册路由
	routes.SetupRoutes(r)

	// 获取SSL证书的绝对路径
	workDir, _ := os.Getwd()
	certPath := filepath.Join(workDir, "..", "frontend", "ssl", "cert.pem")
	keyPath := filepath.Join(workDir, "..", "frontend", "ssl", "key.pem")
	if _, err := os.Stat(certPath); err != nil {
		altCertPath := filepath.Join(workDir, "frontend", "ssl", "cert.pem")
		altKeyPath := filepath.Join(workDir, "frontend", "ssl", "key.pem")
		if _, altErr := os.Stat(altCertPath); altErr == nil {
			certPath = altCertPath
			keyPath = altKeyPath
		}
	}

	// 启动服务
	log.Println("卡包后端服务启动于 https://10.0.0.20:8080")
	log.Println("SSL证书路径:", certPath)
	if err := r.RunTLS(":8080", certPath, keyPath); err != nil {
		log.Fatal("HTTPS服务启动失败:", err)
	}
}
