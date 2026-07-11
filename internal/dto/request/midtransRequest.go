package request

import "github.com/AhmadKusumahDEV/go-chat/internal/models"

type PlanRequest struct {
	Plan string `json:"plan" binding:"required,oneof=premium standard basic"`
}

type SnapRequest struct {
	TransactionInfo   models.TransactionDetail `json:"transaction_details"`
	ItemInfo          models.ItemDetail        `json:"item_details"`
	CustomerInfo      models.CustomerDetail    `json:"customer_details"`
	ListPaymentEnable models.EnablePayment     `json:"enabled_payments"`
	Expired           models.Expiry            `json:"expiry"`
	CustomField1      string                   `json:"custom_field1"`
}
