package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
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

	// 生成密码哈希
	password := "123456"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("密码加密失败:", err)
	}

	fmt.Printf("生成的密码哈希: %s\n", string(hashedPassword))

	// 更新张三的密码
	result1, err := db.Exec("UPDATE users SET password = ? WHERE phone = ?", string(hashedPassword), "13800138001")
	if err != nil {
		log.Fatal("更新张三密码失败:", err)
	}
	rows1, _ := result1.RowsAffected()
	fmt.Printf("张三密码更新成功，影响行数: %d\n", rows1)

	// 更新李四的密码
	result2, err := db.Exec("UPDATE users SET password = ? WHERE phone = ?", string(hashedPassword), "13800138002")
	if err != nil {
		log.Fatal("更新李四密码失败:", err)
	}
	rows2, _ := result2.RowsAffected()
	fmt.Printf("李四密码更新成功，影响行数: %d\n", rows2)

	// 验证密码
	var storedPassword string
	err = db.QueryRow("SELECT password FROM users WHERE phone = ?", "13800138001").Scan(&storedPassword)
	if err != nil {
		log.Fatal("查询密码失败:", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(password))
	if err != nil {
		fmt.Println("❌ 密码验证失败:", err)
	} else {
		fmt.Println("✅ 密码验证成功!")
	}

	// 显示用户信息
	fmt.Println("\n用户信息:")
	rows, err := db.Query("SELECT id, phone, nickname, LENGTH(password) as pwd_len FROM users WHERE phone IN ('13800138001', '13800138002')")
	if err != nil {
		log.Fatal("查询用户失败:", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var phone, nickname string
		var pwdLen int
		rows.Scan(&id, &phone, &nickname, &pwdLen)
		fmt.Printf("ID: %d, Phone: %s, Nickname: %s, Password Length: %d\n", id, phone, nickname, pwdLen)
	}
}
