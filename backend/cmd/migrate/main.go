package main

import (
	"kabao/config"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	config.LoadEnv()

	dsn := config.GetDatabaseConfig().GetDSN()
	if v := config.EnvString("KABAO_DSN"); v != "" {
		dsn = v
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}

	if err := config.RunMigrations(db); err != nil {
		log.Fatal("迁移失败:", err)
	}

	log.Println("迁移完成")
}
