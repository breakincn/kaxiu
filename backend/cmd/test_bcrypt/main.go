package main

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	password := "123456"
	
	// 生成新的哈希密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("加密失败:", err)
		return
	}
	
	fmt.Println("新生成的密码哈希:")
	fmt.Println(string(hashedPassword))
	
	// 测试 SQL 中的密码哈希
	sqlHash := "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
	err = bcrypt.CompareHashAndPassword([]byte(sqlHash), []byte(password))
	if err != nil {
		fmt.Println("\nSQL中的密码哈希验证失败:", err)
	} else {
		fmt.Println("\nSQL中的密码哈希验证成功!")
	}
}
