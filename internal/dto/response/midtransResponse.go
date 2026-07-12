package response

import (
	"time"

	"github.com/AhmadKusumahDEV/go-chat/internal/models"
)

type SnapResponse struct {
	Token       string `json:"token"`
	RedirectURL string `json:"redirect_url"`
}

type OrderResponse struct {
	OrderId     string             `json:"order_id"`
	Plan        string             `json:"plan"`
	Amount      int64              `json:"amount"`
	Status      models.OrderStatus `json:"status"`
	Username    string             `json:"username"`
	PaymentType string             `json:"payment_type"`
	Email       string             `json:"email"`
	Redirecturl string             `json:"redirect_url,omitempty"`
	ExpiretAt   time.Time          `json:"created_at"`
}

type MidtransNotification struct {
	TransactionID            string                 `json:"transaction_id"`
	OrderID                  string                 `json:"order_id"`
	GrossAmount              string                 `json:"gross_amount"`
	Currency                 string                 `json:"currency"`
	PaymentType              string                 `json:"payment_type"`
	TransactionTime          string                 `json:"transaction_time"`
	TransactionStatus        string                 `json:"transaction_status"`
	StatusCode               string                 `json:"status_code"`
	StatusMessage            string                 `json:"status_message"`
	MerchantID               string                 `json:"merchant_id"`
	SignatureKey             string                 `json:"signature_key"`
	FraudStatus              string                 `json:"fraud_status,omitempty"`
	SettlementTime           string                 `json:"settlement_time,omitempty"`
	ExpiryTime               string                 `json:"expiry_time,omitempty"`
	MaskedCard               string                 `json:"masked_card,omitempty"`
	Bank                     string                 `json:"bank,omitempty"`
	ApprovalCode             string                 `json:"approval_code,omitempty"`
	CardType                 string                 `json:"card_type,omitempty"`
	ChannelResponseCode      string                 `json:"channel_response_code,omitempty"`
	ChannelResponseMessage   string                 `json:"channel_response_message,omitempty"`
	ECI                      string                 `json:"eci,omitempty"`
	PaymentOptionType        string                 `json:"payment_option_type,omitempty"`
	VANumbers                []VANumber             `json:"va_numbers,omitempty"`
	PaymentAmounts           []PaymentAmount        `json:"payment_amounts,omitempty"`
	ShopeePayReferenceNumber string                 `json:"shopeepay_reference_number,omitempty"`
	ReferenceID              string                 `json:"reference_id,omitempty"`
	Actions                  []Action               `json:"actions,omitempty"`
	RefundAmount             string                 `json:"refund_amount,omitempty"`
	Refunds                  []Refund               `json:"refunds,omitempty"`
	Metadata                 map[string]interface{} `json:"metadata,omitempty"`
	CustomField1             string                 `json:"custom_field1,omitempty"`
}

type VANumber struct {
	VANumber string `json:"va_number"`
	Bank     string `json:"bank"`
}

type PaymentAmount struct {
	PaidAt string `json:"paid_at"`
	Amount string `json:"amount"`
}

type Action struct {
	Name   string `json:"name"`
	Method string `json:"method"`
	URL    string `json:"url"`
}

type Refund struct {
	RefundChargebackID   int64  `json:"refund_chargeback_id"`
	RefundChargebackUUID string `json:"refund_chargeback_uuid"`
	RefundAmount         string `json:"refund_amount"`
	RefundKey            string `json:"refund_key"`
	RefundMethod         string `json:"refund_method"`
	CreatedAt            string `json:"created_at"`
	BankConfirmedAt      string `json:"bank_confirmed_at"`
}
