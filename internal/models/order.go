package models

import (
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	"github.com/gofrs/uuid"
	"github.com/jackc/pgx/v5/pgtype"
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

func MapMidtransStatus(status string) OrderStatus {
	switch status {
	case "pending":
		return OrderStatusPending
	case "settlement":
		return OrderStatusSettled
	case "expire":
		return OrderStatusExpired
	case "cancel":
		return OrderStatusCancel
	case "deny":
		return OrderStatusDeny
	case "refund":
		return OrderStatusRefund
	default:
		return OrderStatusDeny
	}
}

type Order struct {
	ID             string
	OrderID        string
	Plan           string
	UserID         uuid.UUID
	Amount         int64
	Status         OrderStatus
	Gateway        string
	GatewayTxID    sql.NullString
	SnapToken      sql.NullString
	WebHookPayload []byte
	ExpiretAt      time.Time
	PaidAt         pgtype.Timestamptz
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type TransactionDetail struct {
	OrderID     string `json:"order_id"`
	GrossAmount int64  `json:"gross_amount"`
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

func GenerateOrderID(now time.Time) string {
	date := now.Format("20060102")
	random := randomString(8)
	return fmt.Sprintf("INV-%s-%s", date, random)
}

func randomString(n int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
