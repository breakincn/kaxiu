package handlers

import (
	"fmt"
	"kabao/config"
	"kabao/models"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func GetUsers(c *gin.Context) {
	var users []models.User
	config.DB.Find(&users)
	c.JSON(http.StatusOK, gin.H{"data": users})
}

func GetUser(c *gin.Context) {
	id := c.Param("id")
	var user models.User
	if err := config.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": user})
}

func BindUserPhone(c *gin.Context) {
	userIDAny, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var input struct {
		Phone string `json:"phone" binding:"required"`
		Code  string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.Phone = strings.TrimSpace(input.Phone)
	input.Code = strings.TrimSpace(input.Code)
	if input.Phone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供手机号"})
		return
	}
	if input.Code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入验证码"})
		return
	}

	userID, _ := userIDAny.(uint)
	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	if user.Phone != nil && strings.TrimSpace(*user.Phone) != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "已绑定手机号"})
		return
	}

	var existingByPhone models.User
	if err := config.DB.Where("phone = ?", input.Phone).First(&existingByPhone).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该手机号已被绑定"})
		return
	}

	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := consumeSMSCode(tx, input.Phone, "user_bind_phone", input.Code); err != nil {
			return err
		}
		return tx.Model(&models.User{}).Where("id = ?", userID).Update("phone", input.Phone).Error
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var updated models.User
	if err := config.DB.First(&updated, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "绑定失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": updated})
}

func MerchantSearchUsers(c *gin.Context) {
	if _, ok := c.Get("merchant_id"); !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	phone := strings.TrimSpace(c.Query("phone"))
	if phone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供手机号"})
		return
	}

	var users []models.User
	config.DB.
		Where("phone LIKE ?", "%"+phone+"%").
		Limit(20).
		Find(&users)

	c.JSON(http.StatusOK, gin.H{"data": users})
}

func CreateUser(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 如果提供了密码，进行加密
	if user.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
			return
		}
		user.Password = string(hashedPassword)
	}

	config.DB.Create(&user)
	c.JSON(http.StatusOK, gin.H{"data": user})
}

func UserRegister(c *gin.Context) {
	var input struct {
		Username string `json:"username" binding:"required"`
		Phone    string `json:"phone"`
		Password string `json:"password" binding:"required,min=6"`
		Code     string `json:"code"`
		Nickname string `json:"nickname"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input.Username = strings.TrimSpace(input.Username)
	input.Phone = strings.TrimSpace(input.Phone)
	input.Code = strings.TrimSpace(input.Code)
	input.Nickname = strings.TrimSpace(input.Nickname)

	if input.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供用户名"})
		return
	}

	if input.Phone != "" && input.Code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入验证码"})
		return
	}

	// 检查用户名是否已注册
	var existingByUsername models.User
	if err := config.DB.Where("username = ?", input.Username).First(&existingByUsername).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该用户名已被使用"})
		return
	}

	// 如果提供手机号，检查手机号是否已注册
	if input.Phone != "" {
		var existingByPhone models.User
		if err := config.DB.Where("phone = ?", input.Phone).First(&existingByPhone).Error; err == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "该手机号已注册"})
			return
		}
	}

	// 校验并消耗验证码（仅当用户提供手机号时）
	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		if input.Phone != "" {
			if err := consumeSMSCode(tx, input.Phone, "user_register", input.Code); err != nil {
				return err
			}
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		nickname := input.Nickname
		user := models.User{
			Username: input.Username,
			Password: string(hashedPassword),
			Nickname: nickname,
		}
		if input.Phone != "" {
			phone := input.Phone
			user.Phone = &phone
		}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		c.Set("_registered_user", user)
		return nil
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	registeredUserAny, _ := c.Get("_registered_user")
	registeredUser, _ := registeredUserAny.(models.User)
	token := generateToken(registeredUser.ID)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"token":    token,
			"user_id":  registeredUser.ID,
			"username": registeredUser.Username,
			"phone":    registeredUser.Phone,
			"nickname": registeredUser.Nickname,
		},
	})
}

// 用户登录
func UserLogin(c *gin.Context) {
	var loginReq struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&loginReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供用户名和密码"})
		return
	}
	loginReq.Username = strings.TrimSpace(loginReq.Username)

	var user models.User
	// 兼容旧用户：允许用手机号登录
	if err := config.DB.Where("username = ? OR phone = ?", loginReq.Username, loginReq.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginReq.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// 生成简单的 token（实际项目中应使用 JWT）
	token := generateToken(user.ID)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"token":    token,
			"user_id":  user.ID,
			"username": user.Username,
			"phone":    user.Phone,
			"nickname": user.Nickname,
		},
	})
}

// 生成简单的 token
func generateToken(userID uint) string {
	// 简化版本，实际应该使用 JWT
	// 这里暂时返回格式: "user_{userID}_{timestamp}"
	return fmt.Sprintf("user_%d_%d", userID, time.Now().Unix())
}

// 获取当前登录用户信息
func GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": user})
}
