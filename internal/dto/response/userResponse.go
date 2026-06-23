package response

import (
	"time"

	"github.com/AhmadKusumahDEV/go-chat/internal/models"
)

// UserResponse adalah data user yang aman untuk ditampilkan
type UserResponse struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	About     *string   `json:"about"`
	CreatedAt time.Time `json:"created_at"`
	AvatarUrl *string   `json:"avatar"`
}

type JwtReponse struct {
	AccessToken  string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	Userinfo     *models.JwtUsersInfo
}
