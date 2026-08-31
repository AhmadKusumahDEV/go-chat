package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid or malformed token")
	ErrTokenExpired = errors.New("token has expired")
)

type Pair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role,omitempty"`
	jwt.RegisteredClaims
}

type JWTManager struct {
	accessSecret  string
	refreshSecret string
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

func NewJWTManager(accessSecret, refreshSecret string, accessTTL, refreshTTL time.Duration) *JWTManager {
	return &JWTManager{
		accessSecret:  accessSecret,
		refreshSecret: refreshSecret,
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
	}
}

func (m *JWTManager) GenerateAuthTokens(userID string) (*Pair, error) {
	accessToken, err := m.generate(userID, m.accessSecret, m.accessTTL)
	if err != nil {
		return nil, fmt.Errorf("failed generating access token: %w", err)
	}

	refreshToken, err := m.generate(userID, m.refreshSecret, m.refreshTTL)
	if err != nil {
		return nil, fmt.Errorf("failed generating refresh token: %w", err)
	}

	return &Pair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (m *JWTManager) ValidateAccessToken(tokenString string) (*Claims, error) {
	return m.validate(tokenString, m.accessSecret)
}

func (m *JWTManager) ValidateRefreshToken(tokenString string) (*Claims, error) {
	return m.validate(tokenString, m.refreshSecret)
}

func (m *JWTManager) generate(userID, secret string, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "my-awesome-app",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func (m *JWTManager) validate(tokenString, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
