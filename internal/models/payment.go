package models

type PaymentStatus string

const (
	PaymentStatusPending PaymentStatus = "pending"
	PaymentStatusSettled PaymentStatus = "settled"
	PaymentStatusExpired PaymentStatus = "expired"
	PaymentStatusCancel  PaymentStatus = "cancel"
	PaymentStatusDeny    PaymentStatus = "deny"
	PaymentStatusRefund  PaymentStatus = "refund"
)

func MapMidtransStatus(status string) PaymentStatus {
	switch status {
	case "pending":
		return PaymentStatusPending
	case "settlement":
		return PaymentStatusSettled
	case "expire":
		return PaymentStatusExpired
	case "cancel":
		return PaymentStatusCancel
	case "deny":
		return PaymentStatusDeny
	case "refund":
		return PaymentStatusRefund
	default:
		return PaymentStatusDeny
	}
}
