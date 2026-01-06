package main

import (
	"kabao/config"
	"kabao/middleware"
	"kabao/routes"
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	config.InitDB()

	r := gin.Default()
	r.Use(cors.New(middleware.CORSMiddleware()))

	routes.SetupStaticRoutes(r)
	routes.SetupUserRoutes(r)

	log.Println("Kabao User API service listening on :8081")
	if err := r.Run(":8081"); err != nil {
		log.Fatal("服务启动失败:", err)
	}
}
