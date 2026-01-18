package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// 数据库连接
	dsn := "root:root123@tcp(127.0.0.1:3306)/kabao?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		log.Fatal("数据库 ping 失败:", err)
	}
	fmt.Println("数据库连接成功!")

	// 检查 service_sessions 表结构
	fmt.Println("\n=== service_sessions 表结构 ===")
	rows, err := db.Query("DESCRIBE service_sessions")
	if err != nil {
		log.Fatal("查询 service_sessions 表结构失败:", err)
	}
	defer rows.Close()

	for rows.Next() {
		var field, typ, null, key, def, extra sql.NullString
		err := rows.Scan(&field, &typ, &null, &key, &def, &extra)
		if err != nil {
			log.Fatal("扫描表结构失败:", err)
		}
		fmt.Printf("字段: %-20s 类型: %-20s NULL: %-5s KEY: %-5s\n", field.String, typ.String, null.String, key.String)
	}

	// 检查是否已有 start_timeout_count 字段
	fmt.Println("\n=== 检查 start_timeout_count 字段 ===")
	var found bool
	err = db.QueryRow("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = 'kabao' AND table_name = 'service_sessions' AND column_name = 'start_timeout_count'").Scan(&found)
	if err != nil {
		log.Fatal("查询字段失败:", err)
	}
	
	if found {
		fmt.Println("✅ start_timeout_count 字段已存在")
	} else {
		fmt.Println("❌ start_timeout_count 字段不存在，正在添加...")
		_, err = db.Exec("ALTER TABLE service_sessions ADD COLUMN start_timeout_count INT DEFAULT 0 COMMENT '起单超时次数'")
		if err != nil {
			log.Fatal("添加 start_timeout_count 字段失败:", err)
		}
		fmt.Println("✅ start_timeout_count 字段添加成功")
	}

	// 检查是否已有 start_timeout_last_at 字段
	fmt.Println("\n=== 检查 start_timeout_last_at 字段 ===")
	err = db.QueryRow("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = 'kabao' AND table_name = 'service_sessions' AND column_name = 'start_timeout_last_at'").Scan(&found)
	if err != nil {
		log.Fatal("查询字段失败:", err)
	}
	
	if found {
		fmt.Println("✅ start_timeout_last_at 字段已存在")
	} else {
		fmt.Println("❌ start_timeout_last_at 字段不存在，正在添加...")
		_, err = db.Exec("ALTER TABLE service_sessions ADD COLUMN start_timeout_last_at TIMESTAMP NULL COMMENT '最近起单超时时间'")
		if err != nil {
			log.Fatal("添加 start_timeout_last_at 字段失败:", err)
		}
		fmt.Println("✅ start_timeout_last_at 字段添加成功")
	}

	fmt.Println("\n=== 数据库迁移完成 ===")
}
