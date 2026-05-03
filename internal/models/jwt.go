package models

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type GenerateJwtParams struct {
	Secretkey string
	UserID    string
	JwtUsersInfo
	ExpiresAt time.Duration
}

type JwtUsersInfo struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

type UserClaims struct {
	jwt.RegisteredClaims
	UserID   string       `json:"user_id"`
	UserInfo JwtUsersInfo `json:"user_info"`
}
