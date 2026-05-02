package handlers

import (
	"crypto/rand"
	"fmt"
	"kabao/config"
	"kabao/models"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	merchantReferralDefaultCommissionRateBP = 5000
	withdrawalStatusClaimable               = "claimable"
	withdrawalStatusApplied                 = "applied"
	withdrawalStatusApproved                = "approved"
	withdrawalStatusPaid                    = "paid"
	withdrawalStatusExpired                 = "expired"
	withdrawalStatusRejected                = "rejected"

	referralWithdrawalPending  = "pending"
	referralWithdrawalApproved = "approved"
	referralWithdrawalPaid     = "paid"
	referralWithdrawalRejected = "rejected"

	merchantReferralLandingTitle    = "卡包商户入驻推广"
	merchantReferralLandingSubtitle = "让商户更方便地卖卡、核销、预约和沉淀客户"
)

var merchantReferralHighlights = []gin.H{
	{"title": "线上售卡与推广", "description": "支持售卡、推广短链、推广海报，方便老客转介绍新商户"},
	{"title": "到店核销与卡片管理", "description": "支持用户购卡、到店核销、卡片记录查询，减少手工登记"},
	{"title": "预约与服务流程", "description": "支持预约、到店、服务履约与记录沉淀，日常经营更顺畅"},
}

func randomAlphaDigits(length int) (string, error) {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	if length <= 0 {
		return "", fmt.Errorf("invalid length")
	}
	buf := make([]byte, length)
	raw := make([]byte, length)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	for i := range buf {
		buf[i] = chars[int(raw[i])%len(chars)]
	}
	return string(buf), nil
}

func getOrCreateMerchantReferralProfile(tx *gorm.DB, userID uint) (*models.MerchantReferralProfile, error) {
	if tx == nil || userID == 0 {
		return nil, fmt.Errorf("invalid referral profile input")
	}
	var profile models.MerchantReferralProfile
	if err := tx.Where("user_id = ?", userID).First(&profile).Error; err == nil {
		return &profile, nil
	} else if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	for i := 0; i < 8; i++ {
		code, err := randomAlphaDigits(8)
		if err != nil {
			return nil, err
		}
		profile = models.MerchantReferralProfile{
			UserID:                  userID,
			PromotionCode:           code,
			DefaultCommissionRateBP: merchantReferralDefaultCommissionRateBP,
		}
		if err := tx.Create(&profile).Error; err == nil {
			return &profile, nil
		}
	}
	return nil, fmt.Errorf("生成推广码失败")
}

func loadMerchantReferralByCode(tx *gorm.DB, code string) (*models.MerchantReferralProfile, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return nil, nil
	}
	var profile models.MerchantReferralProfile
	if err := tx.Where("promotion_code = ?", code).First(&profile).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apiErr{status: http.StatusBadRequest, msg: "推广码无效"}
		}
		return nil, err
	}
	return &profile, nil
}

func bindMerchantReferralByCode(tx *gorm.DB, referralCode string, merchantID uint, registeredAt time.Time) error {
	profile, err := loadMerchantReferralByCode(tx, referralCode)
	if err != nil || profile == nil {
		return err
	}

	var existing models.MerchantReferral
	if err := tx.Where("merchant_id = ?", merchantID).First(&existing).Error; err == nil {
		return nil
	} else if err != gorm.ErrRecordNotFound {
		return err
	}

	row := models.MerchantReferral{
		ReferrerUserID:          profile.UserID,
		MerchantID:              merchantID,
		PromotionCode:           profile.PromotionCode,
		RegisteredAt:            &registeredAt,
		DefaultCommissionRateBP: profile.DefaultCommissionRateBP,
	}
	return tx.Create(&row).Error
}

func buildMerchantReferralSharePath(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	return "/newmerchant/" + code
}

func RedirectLegacyMerchantReferralLanding(c *gin.Context) {
	code := strings.ToUpper(strings.TrimSpace(c.Param("refCode")))
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "推广码不能为空"})
		return
	}
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
	c.Redirect(http.StatusFound, buildMerchantReferralSharePath(code)+"?from=legacy")
}

func refreshReferralLedgerStatus(tx *gorm.DB, userID uint, now time.Time) error {
	if tx == nil || userID == 0 {
		return nil
	}
	return tx.Model(&models.MerchantReferralPaymentLedger{}).
		Where("referrer_user_id = ? AND withdrawal_status = ? AND claim_deadline IS NOT NULL AND claim_deadline <= ?", userID, withdrawalStatusClaimable, now).
		Update("withdrawal_status", withdrawalStatusExpired).Error
}

func buildReferralShareLink(c *gin.Context, code string) string {
	scheme := "https"
	if c.Request != nil && c.Request.TLS == nil {
		if proto := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")); proto != "" {
			scheme = proto
		} else if c.Request.URL != nil && strings.EqualFold(c.Request.URL.Scheme, "http") {
			scheme = "http"
		}
	}
	host := strings.TrimSpace(c.Request.Host)
	if host == "" {
		host = "kabao.app"
	}
	return fmt.Sprintf("%s://%s%s", scheme, host, buildMerchantReferralSharePath(code))
}

func buildMerchantRegisterLink(referralCode string) string {
	referralCode = strings.ToUpper(strings.TrimSpace(referralCode))
	if referralCode == "" {
		return "https://kabao.shop/merchant/login"
	}
	return "https://kabao.shop/merchant/" + referralCode + "/login"
}

func GetMerchantReferralLanding(c *gin.Context) {
	code := strings.ToUpper(strings.TrimSpace(c.Param("refCode")))
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "推广码不能为空"})
		return
	}
	var profile models.MerchantReferralProfile
	if err := config.DB.Where("promotion_code = ?", code).First(&profile).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "推广码不存在"})
		return
	}
	var user models.User
	if err := config.DB.First(&user, profile.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "推广用户不存在"})
		return
	}
	displayName := strings.TrimSpace(user.Nickname)
	if displayName == "" {
		displayName = strings.TrimSpace(user.Username)
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"promotion_code":       profile.PromotionCode,
		"referrer_user_id":     profile.UserID,
		"referrer_name":        displayName,
		"title":                merchantReferralLandingTitle,
		"subtitle":             merchantReferralLandingSubtitle,
		"highlights":           merchantReferralHighlights,
		"register_url":         buildMerchantRegisterLink(profile.PromotionCode),
		"share_path":           buildMerchantReferralSharePath(profile.PromotionCode),
		"default_rate_bp":      profile.DefaultCommissionRateBP,
		"default_rate_percent": float64(profile.DefaultCommissionRateBP) / 100,
	}})
}

func GetReferralCommissionOverview(c *gin.Context) {
	userID, ok := mustUserID(c)
	if !ok {
		return
	}
	now := time.Now()
	if err := refreshReferralLedgerStatus(config.DB, userID, now); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "刷新分成状态失败"})
		return
	}
	profile, err := getOrCreateMerchantReferralProfile(config.DB, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取推广信息失败"})
		return
	}

	type summaryRow struct {
		MerchantCount         int64 `json:"merchant_count"`
		TotalPaidAmount       int64 `json:"total_paid_amount"`
		TotalCommissionAmount int64 `json:"total_commission_amount"`
		ClaimableAmount       int64 `json:"claimable_amount"`
		AppliedAmount         int64 `json:"applied_amount"`
		PaidAmount            int64 `json:"paid_amount"`
	}
	var summary summaryRow
	config.DB.Table("merchant_referrals mr").
		Select(strings.Join([]string{
			"COUNT(DISTINCT mr.merchant_id) AS merchant_count",
			"COALESCE(SUM(mrpl.paid_amount), 0) AS total_paid_amount",
			"COALESCE(SUM(mrpl.commission_amount), 0) AS total_commission_amount",
			"COALESCE(SUM(CASE WHEN mrpl.withdrawal_status = 'claimable' THEN mrpl.commission_amount ELSE 0 END), 0) AS claimable_amount",
			"COALESCE(SUM(CASE WHEN mrpl.withdrawal_status = 'applied' THEN mrpl.commission_amount ELSE 0 END), 0) AS applied_amount",
			"COALESCE(SUM(CASE WHEN mrpl.withdrawal_status = 'paid' THEN mrpl.commission_amount ELSE 0 END), 0) AS paid_amount",
		}, ", ")).
		Joins("LEFT JOIN merchant_referral_payment_ledgers mrpl ON mrpl.merchant_referral_id = mr.id AND mrpl.is_within_commission_window = 1").
		Where("mr.referrer_user_id = ?", userID).
		Scan(&summary)

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"profile": gin.H{
			"promotion_code":             profile.PromotionCode,
			"default_commission_rate_bp": profile.DefaultCommissionRateBP,
			"share_link":                 buildReferralShareLink(c, profile.PromotionCode),
			"register_url":               buildMerchantRegisterLink(profile.PromotionCode),
			"share_path":                 buildMerchantReferralSharePath(profile.PromotionCode),
		},
		"summary": summary,
	}})
}

func ListReferralCommissionMerchants(c *gin.Context) {
	userID, ok := mustUserID(c)
	if !ok {
		return
	}
	now := time.Now()
	if err := refreshReferralLedgerStatus(config.DB, userID, now); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "刷新分成状态失败"})
		return
	}

	type merchantRow struct {
		ReferralID            uint       `json:"referral_id"`
		MerchantID            uint       `json:"merchant_id"`
		MerchantName          string     `json:"merchant_name"`
		PromotionCode         string     `json:"promotion_code"`
		RegisteredAt          *time.Time `json:"registered_at"`
		FirstPaidAt           *time.Time `json:"first_paid_at"`
		CommissionExpiresAt   *time.Time `json:"commission_expires_at"`
		TotalPaidAmount       int64      `json:"total_paid_amount"`
		TotalCommissionAmount int64      `json:"total_commission_amount"`
		ClaimableAmount       int64      `json:"claimable_amount"`
		AppliedAmount         int64      `json:"applied_amount"`
		PaidAmount            int64      `json:"paid_amount"`
	}
	rows := make([]merchantRow, 0)
	if err := config.DB.Table("merchant_referrals mr").
		Select(strings.Join([]string{
			"mr.id AS referral_id",
			"mr.merchant_id",
			"m.name AS merchant_name",
			"mr.promotion_code",
			"mr.registered_at",
			"mr.first_paid_at",
			"mr.commission_expires_at",
			"COALESCE(SUM(mrpl.paid_amount), 0) AS total_paid_amount",
			"COALESCE(SUM(mrpl.commission_amount), 0) AS total_commission_amount",
			"COALESCE(SUM(CASE WHEN mrpl.withdrawal_status = 'claimable' THEN mrpl.commission_amount ELSE 0 END), 0) AS claimable_amount",
			"COALESCE(SUM(CASE WHEN mrpl.withdrawal_status = 'applied' THEN mrpl.commission_amount ELSE 0 END), 0) AS applied_amount",
			"COALESCE(SUM(CASE WHEN mrpl.withdrawal_status = 'paid' THEN mrpl.commission_amount ELSE 0 END), 0) AS paid_amount",
		}, ", ")).
		Joins("JOIN merchants m ON m.id = mr.merchant_id").
		Joins("LEFT JOIN merchant_referral_payment_ledgers mrpl ON mrpl.merchant_referral_id = mr.id AND mrpl.is_within_commission_window = 1").
		Where("mr.referrer_user_id = ?", userID).
		Group("mr.id, mr.merchant_id, m.name, mr.promotion_code, mr.registered_at, mr.first_paid_at, mr.commission_expires_at").
		Order("mr.created_at DESC").
		Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询推广商户失败"})
		return
	}

	referralIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		referralIDs = append(referralIDs, row.ReferralID)
	}
	type ledgerRow struct {
		ID                 uint       `json:"id"`
		MerchantReferralID uint       `json:"merchant_referral_id"`
		PaidAmount         int        `json:"paid_amount"`
		CommissionRateBP   int        `json:"commission_rate_bp"`
		CommissionAmount   int        `json:"commission_amount"`
		PaidAt             *time.Time `json:"paid_at"`
		ClaimDeadline      *time.Time `json:"claim_deadline"`
		WithdrawalStatus   string     `json:"withdrawal_status"`
		WithdrawalID       *uint      `json:"withdrawal_id"`
		Note               string     `json:"note"`
	}
	ledgerRows := make([]ledgerRow, 0)
	if len(referralIDs) > 0 {
		if err := config.DB.Table("merchant_referral_payment_ledgers").
			Select("id, merchant_referral_id, paid_amount, commission_rate_bp, commission_amount, paid_at, claim_deadline, withdrawal_status, withdrawal_id, note").
			Where("merchant_referral_id IN ?", referralIDs).
			Order("paid_at DESC, id DESC").
			Scan(&ledgerRows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询分成明细失败"})
			return
		}
	}
	ledgerMap := make(map[uint][]ledgerRow)
	for _, row := range ledgerRows {
		ledgerMap[row.MerchantReferralID] = append(ledgerMap[row.MerchantReferralID], row)
	}
	resp := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		resp = append(resp, gin.H{
			"referral_id":             row.ReferralID,
			"merchant_id":             row.MerchantID,
			"merchant_name":           row.MerchantName,
			"promotion_code":          row.PromotionCode,
			"registered_at":           row.RegisteredAt,
			"first_paid_at":           row.FirstPaidAt,
			"commission_expires_at":   row.CommissionExpiresAt,
			"total_paid_amount":       row.TotalPaidAmount,
			"total_commission_amount": row.TotalCommissionAmount,
			"claimable_amount":        row.ClaimableAmount,
			"applied_amount":          row.AppliedAmount,
			"paid_amount":             row.PaidAmount,
			"payment_ledgers":         ledgerMap[row.ReferralID],
		})
	}
	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func CreateReferralCommissionShareLink(c *gin.Context) {
	userID, ok := mustUserID(c)
	if !ok {
		return
	}
	profile, err := getOrCreateMerchantReferralProfile(config.DB, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成推广链接失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"promotion_code": profile.PromotionCode,
		"share_path":     buildMerchantReferralSharePath(profile.PromotionCode),
		"share_link":     buildReferralShareLink(c, profile.PromotionCode),
		"register_url":   buildMerchantRegisterLink(profile.PromotionCode),
	}})
}

func GetReferralCommissionPoster(c *gin.Context) {
	userID, ok := mustUserID(c)
	if !ok {
		return
	}
	profile, err := getOrCreateMerchantReferralProfile(config.DB, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成推广海报信息失败"})
		return
	}
	var user models.User
	config.DB.First(&user, userID)
	displayName := strings.TrimSpace(user.Nickname)
	if displayName == "" {
		displayName = strings.TrimSpace(user.Username)
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"promotion_code": profile.PromotionCode,
		"share_path":     buildMerchantReferralSharePath(profile.PromotionCode),
		"share_link":     buildReferralShareLink(c, profile.PromotionCode),
		"register_url":   buildMerchantRegisterLink(profile.PromotionCode),
		"title":          merchantReferralLandingTitle,
		"subtitle":       merchantReferralLandingSubtitle,
		"highlights":     merchantReferralHighlights,
		"referrer_name":  displayName,
	}})
}

func CreateReferralCommissionWithdrawal(c *gin.Context) {
	userID, ok := mustUserID(c)
	if !ok {
		return
	}
	var input struct {
		PaymentLedgerIDs []uint `json:"payment_ledger_ids"`
		PayeeName        string `json:"payee_name"`
		PayeeAccount     string `json:"payee_account"`
		PayeeChannel     string `json:"payee_channel"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.PayeeName = strings.TrimSpace(input.PayeeName)
	input.PayeeAccount = strings.TrimSpace(input.PayeeAccount)
	input.PayeeChannel = strings.TrimSpace(input.PayeeChannel)
	if len(input.PaymentLedgerIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择可提现分成明细"})
		return
	}
	if input.PayeeName == "" || input.PayeeAccount == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请完整填写收款信息"})
		return
	}
	if input.PayeeChannel == "" {
		input.PayeeChannel = "wechat"
	}
	now := time.Now()
	var created models.MerchantReferralWithdrawal
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := refreshReferralLedgerStatus(tx, userID, now); err != nil {
			return err
		}
		var ledgers []models.MerchantReferralPaymentLedger
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id IN ? AND referrer_user_id = ?", input.PaymentLedgerIDs, userID).
			Find(&ledgers).Error; err != nil {
			return err
		}
		if len(ledgers) != len(input.PaymentLedgerIDs) {
			return apiErr{status: http.StatusBadRequest, msg: "包含无效分成明细"}
		}
		total := 0
		for _, ledger := range ledgers {
			if ledger.WithdrawalStatus != withdrawalStatusClaimable {
				return apiErr{status: http.StatusBadRequest, msg: "存在不可申请的分成明细"}
			}
			if ledger.ClaimDeadline == nil || !ledger.ClaimDeadline.After(now) {
				return apiErr{status: http.StatusBadRequest, msg: "存在已过期的分成明细"}
			}
			total += ledger.CommissionAmount
		}
		created = models.MerchantReferralWithdrawal{
			ReferrerUserID: userID,
			AppliedAmount:  total,
			PayeeName:      input.PayeeName,
			PayeeAccount:   input.PayeeAccount,
			PayeeChannel:   input.PayeeChannel,
			Status:         referralWithdrawalPending,
		}
		if err := tx.Create(&created).Error; err != nil {
			return err
		}
		for _, ledger := range ledgers {
			item := models.MerchantReferralWithdrawalItem{
				WithdrawalID:     created.ID,
				PaymentLedgerID:  ledger.ID,
				CommissionAmount: ledger.CommissionAmount,
			}
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
		}
		return tx.Model(&models.MerchantReferralPaymentLedger{}).
			Where("id IN ?", input.PaymentLedgerIDs).
			Updates(map[string]interface{}{
				"withdrawal_status": withdrawalStatusApplied,
				"withdrawal_id":     created.ID,
			}).Error
	})
	if err != nil {
		if ae, ok := err.(apiErr); ok {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交提现申请失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": created})
}

func ListReferralCommissionWithdrawals(c *gin.Context) {
	userID, ok := mustUserID(c)
	if !ok {
		return
	}
	var list []models.MerchantReferralWithdrawal
	if err := config.DB.Where("referrer_user_id = ?", userID).Order("created_at DESC, id DESC").Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询提现申请失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

func CreateMerchantReferralPaymentLedger(c *gin.Context) {
	var input struct {
		MerchantID uint   `json:"merchant_id"`
		PaidAmount int    `json:"paid_amount"`
		PaidAt     string `json:"paid_at"`
		Note       string `json:"note"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if input.MerchantID == 0 || input.PaidAmount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供有效的商户ID和付费金额"})
		return
	}
	paidAt := time.Now()
	if strings.TrimSpace(input.PaidAt) != "" {
		if parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(input.PaidAt)); err == nil {
			paidAt = parsed
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "paid_at 必须为 RFC3339 时间"})
			return
		}
	}
	input.Note = strings.TrimSpace(input.Note)

	var created models.MerchantReferralPaymentLedger
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var referral models.MerchantReferral
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("merchant_id = ?", input.MerchantID).
			First(&referral).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return apiErr{status: http.StatusNotFound, msg: "商户未绑定推广关系"}
			}
			return err
		}
		if referral.FirstPaidAt == nil {
			referral.FirstPaidAt = &paidAt
			expiresAt := paidAt.AddDate(2, 0, 0)
			referral.CommissionExpiresAt = &expiresAt
			if err := tx.Save(&referral).Error; err != nil {
				return err
			}
		}
		withinWindow := referral.CommissionExpiresAt != nil && !paidAt.After(*referral.CommissionExpiresAt)
		claimDeadline := paidAt.AddDate(0, 1, 0)
		commissionAmount := 0
		status := withdrawalStatusExpired
		if withinWindow {
			commissionAmount = input.PaidAmount * referral.DefaultCommissionRateBP / 10000
			if claimDeadline.After(time.Now()) {
				status = withdrawalStatusClaimable
			}
		}
		created = models.MerchantReferralPaymentLedger{
			MerchantReferralID:       referral.ID,
			ReferrerUserID:           referral.ReferrerUserID,
			MerchantID:               referral.MerchantID,
			PaidAmount:               input.PaidAmount,
			CommissionRateBP:         referral.DefaultCommissionRateBP,
			CommissionAmount:         commissionAmount,
			PaidAt:                   &paidAt,
			ClaimDeadline:            &claimDeadline,
			IsWithinCommissionWindow: withinWindow,
			WithdrawalStatus:         status,
			Note:                     input.Note,
		}
		return tx.Create(&created).Error
	})
	if err != nil {
		if ae, ok := err.(apiErr); ok {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "记录商户付费失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": created})
}

func ListMerchantReferralWithdrawals(c *gin.Context) {
	status := strings.TrimSpace(c.Query("status"))
	query := config.DB.Table("merchant_referral_withdrawals mrw").
		Select("mrw.id, mrw.referrer_user_id, mrw.applied_amount, mrw.payee_name, mrw.payee_account, mrw.payee_channel, mrw.status, mrw.reviewed_by, mrw.review_note, mrw.reviewed_at, mrw.paid_at, mrw.created_at, mrw.updated_at, users.nickname AS referrer_nickname, users.username AS referrer_username").
		Joins("LEFT JOIN users ON users.id = mrw.referrer_user_id")
	if status != "" {
		query = query.Where("mrw.status = ?", status)
	}
	var rows []gin.H
	if err := query.Order("mrw.created_at DESC, mrw.id DESC").Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询提现申请失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rows})
}

func ReviewMerchantReferralWithdrawal(c *gin.Context) {
	id64, err := strconv.ParseUint(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的提现申请ID"})
		return
	}
	var input struct {
		Status     string `json:"status"`
		ReviewedBy string `json:"reviewed_by"`
		ReviewNote string `json:"review_note"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.Status = strings.TrimSpace(input.Status)
	input.ReviewedBy = strings.TrimSpace(input.ReviewedBy)
	input.ReviewNote = strings.TrimSpace(input.ReviewNote)
	if input.Status != referralWithdrawalApproved && input.Status != referralWithdrawalPaid && input.Status != referralWithdrawalRejected {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status 仅支持 approved、paid、rejected"})
		return
	}

	now := time.Now()
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		var withdrawal models.MerchantReferralWithdrawal
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&withdrawal, uint(id64)).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return apiErr{status: http.StatusNotFound, msg: "提现申请不存在"}
			}
			return err
		}
		if withdrawal.Status == referralWithdrawalPaid {
			return apiErr{status: http.StatusBadRequest, msg: "提现申请已打款"}
		}
		updates := map[string]interface{}{
			"status":      input.Status,
			"reviewed_by": input.ReviewedBy,
			"review_note": input.ReviewNote,
			"reviewed_at": &now,
		}
		if input.Status == referralWithdrawalPaid {
			updates["paid_at"] = &now
		}
		if err := tx.Model(&withdrawal).Updates(updates).Error; err != nil {
			return err
		}
		ledgerStatus := withdrawalStatusApplied
		switch input.Status {
		case referralWithdrawalApproved:
			ledgerStatus = withdrawalStatusApproved
		case referralWithdrawalPaid:
			ledgerStatus = withdrawalStatusPaid
		case referralWithdrawalRejected:
			ledgerStatus = withdrawalStatusClaimable
		}
		var itemRows []models.MerchantReferralWithdrawalItem
		if err := tx.Where("withdrawal_id = ?", withdrawal.ID).Find(&itemRows).Error; err != nil {
			return err
		}
		ledgerIDs := make([]uint, 0, len(itemRows))
		for _, item := range itemRows {
			ledgerIDs = append(ledgerIDs, item.PaymentLedgerID)
		}
		ledgerUpdates := map[string]interface{}{"withdrawal_status": ledgerStatus}
		if input.Status == referralWithdrawalRejected {
			ledgerUpdates["withdrawal_id"] = nil
		}
		if len(ledgerIDs) > 0 {
			if err := tx.Model(&models.MerchantReferralPaymentLedger{}).
				Where("id IN ?", ledgerIDs).
				Updates(ledgerUpdates).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if ae, ok := err.(apiErr); ok {
			c.JSON(ae.status, gin.H{"error": ae.msg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "审核提现申请失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "处理成功"})
}
