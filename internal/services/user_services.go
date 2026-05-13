package services

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/AhmadKusumahDEV/go-chat/internal/config"
	"github.com/AhmadKusumahDEV/go-chat/internal/dto/request"
	"github.com/AhmadKusumahDEV/go-chat/internal/dto/response"
	"github.com/AhmadKusumahDEV/go-chat/internal/helpers"
	"github.com/AhmadKusumahDEV/go-chat/internal/models"
	"github.com/AhmadKusumahDEV/go-chat/internal/repository"
	"github.com/gofrs/uuid"
)

type UsersServices interface {
	LoginUser(ctx context.Context, req *request.LoginRequest) (*response.JwtReponse, error)
	RegisterUser(ctx context.Context, req *request.RegisterRequest) error
	RefreshUser(ctx context.Context, req *request.RefreshRequest) (*response.JwtReponse, error)
	GetAllUser(ctx context.Context) ([]*response.UserResponse, error)
	StoreFirebaseToken(ctx context.Context, fcm *request.FcmRequest, userId uuid.UUID) error
}

type UsersServivesImpl struct {
	userRepository     repository.RepositoryUser
	firebaseRepository repository.RepositoryFirebase

	jwtConfig config.JwtConfig
}

// LoginUser implements UsersServices.
func (u *UsersServivesImpl) LoginUser(ctx context.Context, req *request.LoginRequest) (*response.JwtReponse, error) {
	var mu sync.Mutex

	mu.Lock()
	defer mu.Unlock()

	users, err := u.userRepository.FindByEmail(ctx, req.Email)

	if err != nil {
		return nil, errors.New("user not found")
	}

	userinfo := models.JwtUsersInfo{
		UserID:   users.ID.String(),
		Email:    users.Email,
		Username: users.Username,
	}

	if !helpers.ValidatePassword(req.Password, users.Password) {
		return nil, errors.New("password not valid")
	}

	accessToken, refreshToken, err := helpers.GenerateAuthTokens(
		userinfo,
		u.jwtConfig.SecretKeyAccess,
		u.jwtConfig.SecretKeyrefresh,
		u.jwtConfig.AccessTokenExpiration,
		u.jwtConfig.RefreshTokenExpiration,
	)

	if err != nil {
		return nil, err
	}

	return &response.JwtReponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Userinfo:     &userinfo,
	}, nil
}

// RefreshUser implements UsersServices.
func (u *UsersServivesImpl) RefreshUser(ctx context.Context, req *request.RefreshRequest) (*response.JwtReponse, error) {
	claims, err := helpers.GetUserJWT(req.RefreshToken, u.jwtConfig.SecretKeyrefresh)
	if err != nil {
		return nil, errors.New("invalid or expired refresh token")
	}

	userInfoVal, ok := claims["user_info"].(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid claims format")
	}

	userID := fmt.Sprintf("%v", userInfoVal["user_id"])
	email := fmt.Sprintf("%v", userInfoVal["email"])
	username := fmt.Sprintf("%v", userInfoVal["username"])

	userinfo := models.JwtUsersInfo{
		UserID:   userID,
		Username: username,
		Email:    email,
	}

	accessToken, refreshToken, err := helpers.GenerateAuthTokens(
		userinfo,
		u.jwtConfig.SecretKeyAccess,
		u.jwtConfig.SecretKeyrefresh,
		u.jwtConfig.AccessTokenExpiration,
		u.jwtConfig.RefreshTokenExpiration,
	)

	if err != nil {
		return nil, err
	}

	return &response.JwtReponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Userinfo:     &userinfo,
	}, nil
}

// RegisterUser implements UsersServices.
func (u *UsersServivesImpl) RegisterUser(ctx context.Context, req *request.RegisterRequest) error {
	password := helpers.HashPassword(req.Password)

	dtoToModels := models.Users{
		Username: req.Username,
		Email:    req.Email,
		Password: password,
	}

	err := u.userRepository.Create(ctx, &dtoToModels)

	if err != nil {
		return errors.New("failed to create user")
	}

	return nil
}

func (u *UsersServivesImpl) GetAllUser(ctx context.Context) ([]*response.UserResponse, error) {
	users, err := u.userRepository.FindAll(ctx)

	userConvert := helpers.UserResponses(users)

	if err != nil {
		return nil, errors.New("failed to get users")
	}

	return userConvert, nil
}

func (u *UsersServivesImpl) StoreFirebaseToken(ctx context.Context, fcm *request.FcmRequest, userId uuid.UUID) error {
	dtoToModels := models.Firebase{
		UserID:         userId,
		InstallationID: fcm.Installation,
		FcmToken:       fcm.FcmToken,
		Platform:       fcm.Platform,
	}

	err := u.firebaseRepository.CreateFcmToken(ctx, &dtoToModels)
	if err != nil {
		return err
	}

	return nil
}

func NewUsersServices(userRepository repository.RepositoryUser, jwtConfig config.JwtConfig) UsersServices {
	return &UsersServivesImpl{
		userRepository: userRepository,
		jwtConfig:      jwtConfig,
	}
}
