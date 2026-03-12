package handlers

import (
	"strings"

	"kabao/models"
)

func isQueueModeMerchantTerm(merchant *models.Merchant) bool {
	if merchant == nil {
		return false
	}
	queueMode := strings.TrimSpace(merchant.QueueMode)
	return !merchant.SupportCustomerServiceMode && merchant.SupportQueue && (queueMode == "auto" || queueMode == "manual")
}

func resolveStartTerm(merchant *models.Merchant) string {
	if isQueueModeMerchantTerm(merchant) {
		return "叫号"
	}
	if merchant != nil {
		if term := strings.TrimSpace(merchant.StartTerm); term != "" {
			return term
		}
	}
	return "起单"
}

func resolveFinishTerm(merchant *models.Merchant) string {
	if isQueueModeMerchantTerm(merchant) {
		return "结号"
	}
	if merchant != nil {
		if term := strings.TrimSpace(merchant.FinishTerm); term != "" {
			return term
		}
	}
	return "结单"
}
