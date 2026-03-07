package config

import "kabao/models"

func MerchantQueueWaitingStartSeconds(merchant *models.Merchant) int {
	defaultSeconds := int(StartPendingTimeout().Seconds())
	if merchant == nil {
		return defaultSeconds
	}
	if merchant.QueueWaitingStartSeconds > 0 {
		return merchant.QueueWaitingStartSeconds
	}
	return defaultSeconds
}
