package models

type paymentStatus string

const (
	PaymentStatusPending paymentStatus = "pending"
	PaymentStatusSuccess paymentStatus = "success"
	PaymentStatusFailed  paymentStatus = "failed"
)
