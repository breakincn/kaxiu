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

func MerchantQueueTimeoutWaitingSeconds(merchant *models.Merchant) int {
	defaultSeconds := 15 * 60
	if merchant == nil {
		return defaultSeconds
	}
	if merchant.QueueTimeoutWaitingSeconds > 0 {
		return merchant.QueueTimeoutWaitingSeconds
	}
	return defaultSeconds
}
