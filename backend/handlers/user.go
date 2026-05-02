package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"kabao/config"
	"kabao/models"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const maxUserNicknameRunes = 7

func normalizeUserNickname(value string) string {
	return strings.TrimSpace(value)
}

func validateUserNickname(value string) (string, error) {
	nickname := normalizeUserNickname(value)
	if nickname == "" {
		return "", fmt.Errorf("昵称不能为空")
	}
	if strings.ContainsAny(nickname, " \t\r\n") {
		return "", fmt.Errorf("昵称不能包含空格")
	}
	if len([]rune(nickname)) > maxUserNicknameRunes {
		return "", fmt.Errorf("昵称最多 7 个字")
	}
	return nickname, nil
}

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
	merchantID, ok := getMerchantID(c)
	if !ok {
		return
	}
	canIssue, err := hasMerchantPermissionInHandler(c, "merchant.card.issue")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "权限校验失败"})
		return
	}
	canVerify, err := hasMerchantPermissionInHandler(c, "merchant.card.verify")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "权限校验失败"})
		return
	}
	if !canIssue && !canVerify {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限"})
		return
	}

	phone := strings.TrimSpace(c.Query("phone"))
	if phone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供手机号"})
		return
	}

	type merchantUserResult struct {
		ID       uint    `json:"id"`
		Phone    *string `json:"phone"`
		Nickname string  `json:"nickname"`
	}

	cardUserSubQuery := config.DB.Model(&models.Card{}).
		Select("DISTINCT user_id").
		Where("merchant_id = ?", merchantID)

	appointmentUserSubQuery := config.DB.Model(&models.Appointment{}).
		Select("DISTINCT user_id").
		Where("merchant_id = ?", merchantID)

	var users []merchantUserResult
	query := config.DB.
		Model(&models.User{}).
		Select("users.id, users.phone, users.nickname").
		Where("phone LIKE ?", "%"+phone+"%")
	if !canIssue {
		query = query.Where("(users.id IN (?) OR users.id IN (?))", cardUserSubQuery, appointmentUserSubQuery)
	}
	query.
		Order("users.id DESC").
		Limit(20).
		Find(&users)

	c.JSON(http.StatusOK, gin.H{"data": users})
}

func GetUserCode(c *gin.Context) {
	userIDAny, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	userID, ok := userIDAny.(uint)
	if !ok || userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	exp := time.Now().Add(5 * time.Minute).Unix()
	uidStr := strconv.FormatUint(uint64(userID), 10)
	expStr := strconv.FormatInt(exp, 10)
	msg := uidStr + ":" + expStr

	secret := config.UserCodeSecret()
	if secret == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "用户码服务未配置"})
		return
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(msg))
	sig := hex.EncodeToString(mac.Sum(nil))

	code := fmt.Sprintf("kabao-user:%s:%s:%s", uidStr, expStr, sig)
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"code":       code,
			"expires_at": exp,
		},
	})
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
		Username              string `json:"username" binding:"required"`
		Phone                 string `json:"phone"`
		Password              string `json:"password" binding:"required,min=6"`
		Code                  string `json:"code"`
		Nickname              string `json:"nickname"`
		PromotionCampaignSlug string `json:"promotion_campaign_slug"`
		ReferralCode          string `json:"referral_code"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input.Username = strings.TrimSpace(input.Username)
	input.Phone = strings.TrimSpace(input.Phone)
	input.Code = strings.TrimSpace(input.Code)
	input.Nickname = strings.TrimSpace(input.Nickname)
	input.PromotionCampaignSlug = strings.TrimSpace(input.PromotionCampaignSlug)
	input.ReferralCode = strings.TrimSpace(input.ReferralCode)

	if input.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供用户名"})
		return
	}

	if input.Phone != "" && input.Code == "" && !config.UserRegisterSMSVerificationDisabled() {
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
		if input.Phone != "" && !config.UserRegisterSMSVerificationDisabled() {
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
		if err := maybeCountPromotionRegistration(tx, input.PromotionCampaignSlug, input.ReferralCode, user.ID, time.Now()); err != nil {
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
	token, err := generateToken(registeredUser.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "签发token失败"})
		return
	}

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
	rlKey, ok := enforceLoginRateLimit(c, "user", loginReq.Username)
	if !ok {
		return
	}

	var user models.User
	// 兼容旧用户：允许用手机号登录
	if err := config.DB.Where("username = ? OR phone = ?", loginReq.Username, loginReq.Username).First(&user).Error; err != nil {
		recordLoginFailure(rlKey)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginReq.Password)); err != nil {
		recordLoginFailure(rlKey)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	token, err := generateToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "签发token失败"})
		return
	}
	recordLoginSuccess(rlKey)

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

func generateToken(userID uint) (string, error) {
	secret := config.UserJWTSecret()
	if secret == "" {
		return "", fmt.Errorf("missing KABAO_USER_JWT_SECRET")
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"type":    "user",
		"user_id": userID,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	return token.SignedString([]byte(secret))
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

// 更新用户昵称
func UpdateUserNickname(c *gin.Context) {
	userIDAny, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var input struct {
		Nickname string `json:"nickname" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供昵称"})
		return
	}

	normalizedNickname, err := validateUserNickname(input.Nickname)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.Nickname = normalizedNickname

	userID, _ := userIDAny.(uint)
	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	if err := config.DB.Model(&models.User{}).Where("id = ?", userID).Update("nickname", input.Nickname).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新昵称失败"})
		return
	}

	var updated models.User
	if err := config.DB.First(&updated, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": updated})
}
