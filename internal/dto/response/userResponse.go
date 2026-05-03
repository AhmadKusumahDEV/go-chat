package response

import (
	"time"

	"github.com/AhmadKusumahDEV/go-chat/internal/models"
	"github.com/google/uuid"
)

// UserResponse adalah data user yang aman untuk ditampilkan
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// LoginResponse menyertakan Token (JWT)
type LoginResponse struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"` // Biasanya "Bearer"
	ExpiresIn   int          `json:"expires_in"` // Detik
	User        UserResponse `json:"user"`
}

type JwtReponse struct {
	AccessToken  string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	Userinfo     *models.JwtUsersInfo
}
