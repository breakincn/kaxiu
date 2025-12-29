package main

import (
	"kaxiu/config"
	"kaxiu/routes"
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化数据库
	config.InitDB()

	// 初始化 Gin
	r := gin.Default()

	// CORS 配置
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	// 注册路由
	routes.SetupRoutes(r)

	// 启动服务
	log.Println("卡秀后端服务启动于 :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("服务启动失败:", err)
	}
}
