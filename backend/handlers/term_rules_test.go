package handlers

import (
	"kabao/models"
	"testing"
)

func TestResolveStartTerm(t *testing.T) {
	t.Run("queue_mode_uses_calling_term", func(t *testing.T) {
		merchant := &models.Merchant{
			SupportQueue:               true,
			SupportCustomerServiceMode: false,
			QueueMode:                  "manual",
			StartTerm:                  "上钟",
		}
		if got := resolveStartTerm(merchant); got != "叫号" {
			t.Fatalf("want 叫号, got %s", got)
		}
	})

	t.Run("customer_service_mode_uses_custom_start_term", func(t *testing.T) {
		merchant := &models.Merchant{
			SupportQueue:               false,
			SupportCustomerServiceMode: true,
			StartTerm:                  "上钟",
		}
		if got := resolveStartTerm(merchant); got != "上钟" {
			t.Fatalf("want 上钟, got %s", got)
		}
	})

	t.Run("customer_service_mode_falls_back_to_default", func(t *testing.T) {
		merchant := &models.Merchant{
			SupportCustomerServiceMode: true,
		}
		if got := resolveStartTerm(merchant); got != "起单" {
			t.Fatalf("want 起单, got %s", got)
		}
	})
}

func TestResolveFinishTerm(t *testing.T) {
	t.Run("queue_mode_uses_finish_queue_term", func(t *testing.T) {
		merchant := &models.Merchant{
			SupportQueue:               true,
			SupportCustomerServiceMode: false,
			QueueMode:                  "auto",
			FinishTerm:                 "下钟",
		}
		if got := resolveFinishTerm(merchant); got != "结号" {
			t.Fatalf("want 结号, got %s", got)
		}
	})

	t.Run("customer_service_mode_uses_custom_finish_term", func(t *testing.T) {
		merchant := &models.Merchant{
			SupportQueue:               false,
			SupportCustomerServiceMode: true,
			FinishTerm:                 "下钟",
		}
		if got := resolveFinishTerm(merchant); got != "下钟" {
			t.Fatalf("want 下钟, got %s", got)
		}
	})

	t.Run("customer_service_mode_falls_back_to_default", func(t *testing.T) {
		merchant := &models.Merchant{
			SupportCustomerServiceMode: true,
		}
		if got := resolveFinishTerm(merchant); got != "结单" {
			t.Fatalf("want 结单, got %s", got)
		}
	})
}

func TestResolveTermsByMode(t *testing.T) {
	t.Run("queue_mode_always_uses_calling_terms", func(t *testing.T) {
		merchant := &models.Merchant{
			SupportQueue:               true,
			SupportCustomerServiceMode: false,
			QueueMode:                  "manual",
			StartTerm:                  "上钟",
			FinishTerm:                 "下钟",
		}
		if got := resolveStartTerm(merchant); got != "叫号" {
			t.Fatalf("want queue start term 叫号, got %s", got)
		}
		if got := resolveFinishTerm(merchant); got != "结号" {
			t.Fatalf("want queue finish term 结号, got %s", got)
		}
	})

	t.Run("customer_service_mode_uses_merchant_terms", func(t *testing.T) {
		merchant := &models.Merchant{
			SupportQueue:               false,
			SupportCustomerServiceMode: true,
			StartTerm:                  "上钟",
			FinishTerm:                 "下钟",
		}
		if got := resolveStartTerm(merchant); got != "上钟" {
			t.Fatalf("want customer-service start term 上钟, got %s", got)
		}
		if got := resolveFinishTerm(merchant); got != "下钟" {
			t.Fatalf("want customer-service finish term 下钟, got %s", got)
		}
	})
}
