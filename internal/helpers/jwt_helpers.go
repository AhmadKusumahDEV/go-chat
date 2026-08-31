package helpers

import (
	"time"

	"github.com/AhmadKusumahDEV/go-chat/internal/models"
	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"
)

var (
	JwtSigningMethod = jwt.SigningMethodHS256
	ErrGetUserJWT    = errors.New("fail get user jwt")
)

func GenerateUserJWT(param models.GenerateJwtParams) (*models.UserClaims, string, error) {
	userclaims := models.UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(param.ExpiresAt)),
		},
		UserID:   param.UserID,
		UserInfo: param.JwtUsersInfo,
	}

	token := jwt.NewWithClaims(
		JwtSigningMethod,
		userclaims,
	)

	signedToken, err := token.SignedString([]byte(param.Secretkey))
	if err != nil {
		return nil, "", errors.New("fail generate worker jwt")
	}

	return &userclaims, signedToken, nil
}

func GenerateAuthTokens(userinfo models.JwtUsersInfo, accessSecret, refreshSecret string, accessExp, refreshExp int) (string, string, error) {
	if accessExp == 0 {
		accessExp = 60
	}
	if refreshExp == 0 {
		refreshExp = 360
	}

	// Access Token
	jwtParams := models.GenerateJwtParams{
		UserID:       userinfo.UserID,
		Secretkey:    accessSecret,
		JwtUsersInfo: userinfo,
		ExpiresAt:    time.Duration(accessExp) * time.Minute,
	}
	_, accessToken, err := GenerateUserJWT(jwtParams)
	if err != nil {
		return "", "", errors.New("failed generate access token")
	}

	// Refresh Token
	refreshjwtParams := models.GenerateJwtParams{
		UserID:       userinfo.UserID,
		Secretkey:    refreshSecret,
		JwtUsersInfo: userinfo,
		ExpiresAt:    time.Duration(refreshExp) * time.Minute,
	}
	_, refreshToken, err := GenerateUserJWT(refreshjwtParams)
	if err != nil {
		return "", "", errors.New("failed generate access token")
	}

	return accessToken, refreshToken, nil
}

func GetUserJWT(authToken string, secretKey string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(authToken, func(t *jwt.Token) (interface{}, error) {
		if method, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.Wrap(ErrGetUserJWT, "convert to HMAC signing method")
		} else if method != JwtSigningMethod {
			return nil, errors.Wrap(ErrGetUserJWT, "signing method is not HMAC")
		}

		return []byte(secretKey), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, errors.New("token expired")
		}

		return nil, errors.Wrap(ErrGetUserJWT, "parse token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.Wrap(ErrGetUserJWT, "convert map claims")
	}

	return claims, nil
}
