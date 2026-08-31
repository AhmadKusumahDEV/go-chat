package response

import (
	"time"

	"github.com/AhmadKusumahDEV/go-chat/internal/models"
)

type OrderResponse struct {
	OrderId       string             `json:"order_id"`
	Plan          string             `json:"plan"`
	Amount        int64              `json:"amount"`
	Status        models.OrderStatus `json:"status"`
	Username      string             `json:"username"`
	PaymentMethod string             `json:"payment_method,omitempty"`
	Email         string             `json:"email"`
	Redirecturl   string             `json:"redirect_url,omitempty"`
	ExpiretAt     time.Time          `json:"created_at"`
}
