package midtrans

import (
	"errors"
)

type OrderStatus string

const (
	OrderStatusPending OrderStatus = "pending"
	OrderStatusSettled OrderStatus = "settled"
	OrderStatusExpired OrderStatus = "expired"
	OrderStatusCancel  OrderStatus = "cancel"
	OrderStatusDeny    OrderStatus = "deny"
	OrderStatusRefund  OrderStatus = "refund"
)

const Gateway string = "midtrans"

var (
	ErrStatusAlreadySettled = errors.New("Order is already settled")
	ErrSameStatus           = errors.New("Status have same value")
	ErrSiganature           = errors.New("signature invalid")
)

func MapMidtransStatus(statusTransaction string, fraud string) OrderStatus {
	switch statusTransaction {
	case "capture":
		if fraud == "accept" {
			return OrderStatusSettled
		}
		if fraud == "challenge" {
			return OrderStatusPending
		}
		return OrderStatusDeny
	case "pending":
		return OrderStatusPending
	case "settlement", "settled":
		return OrderStatusSettled
	case "expire", "expired":
		return OrderStatusExpired
	case "cancel":
		return OrderStatusCancel
	case "deny":
		return OrderStatusDeny
	case "refund", "chargeback", "partial_chargeback", "partial_refund":
		return OrderStatusRefund
	default:
		return OrderStatusDeny
	}
}

type SnapRequest struct {
	TransactionInfo   TransactionDetails `json:"transaction_details"`
	ItemInfo          ItemDetail         `json:"item_details"`
	CustomerInfo      CustomerDetail     `json:"customer_details"`
	ListPaymentEnable EnablePayment      `json:"enabled_payments"`
	Expired           Expiry             `json:"expiry"`
	CustomField1      string             `json:"custom_field1"`
}

type MidtransWebhooks struct {
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

type SnapResponse struct {
	Token       string `json:"token"`
	RedirectURL string `json:"redirect_url"`
}

type CustomerDetail struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
}

type ItemDetail []MerchantItemDetail

type EnablePayment []string

type Expiry struct {
	StartTime string `json:"start_time"`
	Unit      string `json:"unit"`
	Duration  int    `json:"duration"`
}

type MerchantItemDetail struct {
	ID           string `json:"id"`
	Price        int64  `json:"price"`
	Quantity     int    `json:"quantity"`
	Name         string `json:"name"`
	Brand        string `json:"brand,omitempty"`
	Category     string `json:"category,omitempty"`
	MerchantName string `json:"merchant_name,omitempty"`
}

type ChargeRequest struct {
	PaymentType        string             `json:"payment_type"`
	TransactionDetails TransactionDetails `json:"transaction_details"`
	ItemDetails        []ItemDetail       `json:"item_details,omitempty"`
	CustomerDetails    *CustomerDetails   `json:"customer_details,omitempty"`

	// E-Channel for payments like PLN
	EChannel *EChannelDetail `json:"echannel,omitempty"`

	// Populate exactly one according to payment_type.
	GoPay        *GoPayRequest        `json:"gopay,omitempty"`
	ShopeePay    *ShopeePayRequest    `json:"shopeepay,omitempty"`
	CreditCard   *CreditCardRequest   `json:"credit_card,omitempty"`
	BankTransfer *BankTransferRequest `json:"bank_transfer,omitempty"`
	CStore       *CStoreRequest       `json:"cstore,omitempty"`
	QRIS         *QRISRequest         `json:"qris,omitempty"`
}

type ShopeePayRequest struct {
	CallbackUrl string `json:"callback_url"`
}

// EMoneyRequest for electronic money payments (ShopeePay, etc.)
type EMoneyRequest struct {
	// Optional: specify e-money provider
	Provider string `json:"provider,omitempty"`
}

type EChannelDetail struct {
	BillInfo1 string `json:"bill_info1"`
	BillInfo2 string `json:"bill_info2"`

	BillInfo3 string `json:"bill_info3,omitempty"`
	BillInfo4 string `json:"bill_info4,omitempty"`
	BillInfo5 string `json:"bill_info5,omitempty"`
	BillInfo6 string `json:"bill_info6,omitempty"`
	BillInfo7 string `json:"bill_info7,omitempty"`
	BillInfo8 string `json:"bill_info8,omitempty"`

	// Optional custom bill key; accepts 6–12 digits.
	BillKey string `json:"bill_key,omitempty"`
}

type CStoreRequest struct {
	// Required: "indomaret" or "alfamart"
	Store string `json:"store"`

	// Optional; label displayed at the store POS.
	Message string `json:"message,omitempty"`

	// Alfamart only; optional receipt text.
	AlfamartFreeText1 string `json:"alfamart_free_text_1,omitempty"`
	AlfamartFreeText2 string `json:"alfamart_free_text_2,omitempty"`
	AlfamartFreeText3 string `json:"alfamart_free_text_3,omitempty"`
}

type QRISRequest struct {
	// Optional: specify acquirer (e.g., "gopay", "shopeepay", "dana", "ovo", "linkaja")
	Acquirer string `json:"acquirer,omitempty"`

	// Optional: specify merchant order ID for QRIS
	MerchantOrderID string `json:"merchant_order_id,omitempty"`
}

// AvailablePayments contains the list of available payment methods
type AvailablePayments struct {
	PaymentMethods []string `json:"payment_methods"`
}

// PaymentMethodConfig holds configuration for specific payment methods
type PaymentMethodConfig struct {
	// For bank transfer VA
	Bank string `json:"bank,omitempty"`

	// For e-channel (e.g., PLN)
	EChannel *EChannelDetail `json:"echannel,omitempty"`

	// For credit card
	SaveToken bool `json:"save_token,omitempty"`
}

type BankTransferRequest struct {
	Bank string `json:"bank,omitempty"` // e.g. "bca", "bni", "bri", "cimb"

	// Optional, depending on bank/configuration
	VANumber       string `json:"va_number,omitempty"`
	SubCompanyCode string `json:"sub_company_code,omitempty"`
	RecipientName  string `json:"recipient_name,omitempty"`

	FreeText *VAFreeText `json:"free_text,omitempty"`
}

type VAFreeText struct {
	Inquiry []LocalizedText `json:"inquiry,omitempty"`
	Payment []LocalizedText `json:"payment,omitempty"`
}

type LocalizedText struct {
	EN string `json:"en,omitempty"`
	ID string `json:"id,omitempty"`
}

type TransactionDetails struct {
	OrderID     string `json:"order_id"`
	GrossAmount int64  `json:"gross_amount"`
}

type CustomerDetails struct {
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Email     string `json:"email,omitempty"`
	Phone     string `json:"phone,omitempty"`
}

type GoPayRequest struct {
	EnableCallback     bool     `json:"enable_callback,omitempty"`
	CallbackURL        string   `json:"callback_url,omitempty"`
	AccountID          string   `json:"account_id,omitempty"`
	PaymentOptionToken string   `json:"payment_option_token,omitempty"`
	Recurring          bool     `json:"recurring,omitempty"`
	PromotionIDs       []string `json:"promotion_ids,omitempty"`
}

type CreditCardRequest struct {
	TokenID        string `json:"token_id"`
	Authentication bool   `json:"authentication,omitempty"`
}

type ChargeResponse struct {
	StatusCode             string `json:"status_code"`
	StatusMessage          string `json:"status_message"`
	TransactionID          string `json:"transaction_id"`
	OrderID                string `json:"order_id"`
	GrossAmount            string `json:"gross_amount"`
	Currency               string `json:"currency"`
	PaymentType            string `json:"payment_type"`
	TransactionTime        string `json:"transaction_time"`
	TransactionStatus      string `json:"transaction_status"`
	FraudStatus            string `json:"fraud_status,omitempty"`
	SignatureKey           string `json:"signature_key,omitempty"`
	ChannelResponseCode    string `json:"channel_response_code,omitempty"`
	ChannelResponseMessage string `json:"channel_response_message,omitempty"`

	Actions []Action `json:"actions,omitempty"`

	VANumbers   []VANumber `json:"va_numbers,omitempty"`
	BillerCode  string     `json:"biller_code,omitempty"`
	BillKey     string     `json:"bill_key,omitempty"`
	PaymentCode string     `json:"payment_code,omitempty"`

	// QRIS responses documented here provide QR URLs in Actions,
	// such as generate-qr-code / generate-qr-code-v2.
	QRString string `json:"qr_string,omitempty"`
}

type VaNumber struct {
	Bank     string `json:"bank"`
	VaNumber string `json:"va_number"`
}
