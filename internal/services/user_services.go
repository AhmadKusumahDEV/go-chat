package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/AhmadKusumahDEV/go-chat/internal/config"
	"github.com/AhmadKusumahDEV/go-chat/internal/dto/request"
	"github.com/AhmadKusumahDEV/go-chat/internal/dto/response"
	"github.com/AhmadKusumahDEV/go-chat/internal/helpers"
	"github.com/AhmadKusumahDEV/go-chat/internal/models"
	"github.com/AhmadKusumahDEV/go-chat/internal/repository"
	"github.com/AhmadKusumahDEV/go-chat/pkg/storage"
	"github.com/gofrs/uuid"
)

type UsersServices interface {
	LoginUser(ctx context.Context, req *request.LoginRequest) (*response.JwtReponse, error)
	RegisterUser(ctx context.Context, req *request.RegisterRequest) error
	RefreshUser(ctx context.Context, refreshToken string) (*response.JwtReponse, error)
	GetAllUser(ctx context.Context) ([]*response.UserResponse, error)
	GetDetailUser(ctx context.Context, userId uuid.UUID) (*response.UserResponse, error)
	GetUserByID(ctx context.Context, userId uuid.UUID) (*response.UserResponse, error)
	StoreFirebaseToken(ctx context.Context, fcm *request.FcmRequest, userId uuid.UUID) error
	LogoutUser(ctx context.Context, userId uuid.UUID, installationID string) (int, error)
	UpdatedUserInfo(ctx context.Context, userId uuid.UUID, req *request.UpdateProfileRequest) error
	UpdateAvatar(ctx context.Context, userID string, reader io.Reader, size int64, contentType, objectName string) (string, error)
}

type UsersServivesImpl struct {
	userRepository     repository.RepositoryUser
	firebaseRepository repository.RepositoryFirebase
	minioS3            storage.ObjectStorage

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
// Accepts refresh token as string parameter
func (u *UsersServivesImpl) RefreshUser(ctx context.Context, refreshToken string) (*response.JwtReponse, error) {
	claims, err := helpers.GetUserJWT(refreshToken, u.jwtConfig.SecretKeyrefresh)
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
	// 1. Check if email already exists (might be registered via OAuth)
	existingUser, err := u.userRepository.FindByEmail(ctx, req.Email)
	if err == nil {
		// Email exists! Check if user has OAuth provider linked
		if existingUser.ProviderName.Valid && existingUser.ProviderName.String != "" {
			return fmt.Errorf("email %s is already registered with %s. Please use %s login or use a different email.",
				req.Email, existingUser.ProviderName.String, existingUser.ProviderName.String)
		}
		// User exists but no provider - this is a duplicate registration
		return fmt.Errorf("email %s is already registered. Please login instead.", req.Email)
	}

	// 2. Hash password and create user
	password := helpers.HashPassword(req.Password)

	avatarDefault := fmt.Sprintf("https://api.dicebear.com/10.x/initials/png?seed=%s", req.Username)

	dtoToModels := models.Users{
		Username: req.Username,
		Email:    req.Email,
		Password: password,
		AvatarUrl: sql.NullString{
			String: avatarDefault,
			Valid:  true,
		},
	}

	err = u.userRepository.Create(ctx, &dtoToModels)
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

func (u *UsersServivesImpl) GetDetailUser(ctx context.Context, userId uuid.UUID) (*response.UserResponse, error) {
	user, err := u.userRepository.FindByID(ctx, userId)
	if err != nil {
		return nil, errors.New("user not found")
	}

	return helpers.UserResponse(user), nil
}

func (u *UsersServivesImpl) GetUserByID(ctx context.Context, userId uuid.UUID) (*response.UserResponse, error) {
	user, err := u.userRepository.FindByID(ctx, userId)
	if err != nil {
		return nil, errors.New("user not found")
	}

	return helpers.UserResponse(user), nil
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

func (u *UsersServivesImpl) LogoutUser(ctx context.Context, userId uuid.UUID, fcmToken string) (int, error) {
	var count int
	var err error

	if fcmToken != "" {
		count, err = u.firebaseRepository.DeactivateTokenByUserID(ctx, userId.String(), fcmToken)
		if err != nil {
			return 0, fmt.Errorf("failed to deactivate token on logout: %w", err)
		}
	} else {
		count, err = u.firebaseRepository.DeactivateAllTokensByUserID(ctx, userId.String())
		if err != nil {
			return 0, fmt.Errorf("failed to deactivate tokens on logout: %w", err)
		}
	}
	return count, nil
}

func (u *UsersServivesImpl) UpdatedUserInfo(ctx context.Context, userId uuid.UUID, req *request.UpdateProfileRequest) error {
	userinfo, err := u.userRepository.FindByID(ctx, userId)
	if err != nil {
		return errors.New("user not found")
	}

	if req.Username != "" {
		userinfo.Username = req.Username
	}

	if req.About != "" {
		userinfo.About.String = req.About
		userinfo.About.Valid = true
	}

	err = u.userRepository.Update(ctx, userinfo)
	if err != nil {
		return errors.New(err.Error())
	}

	return nil
}

func (u *UsersServivesImpl) UpdateAvatar(ctx context.Context, userID string, reader io.Reader, size int64, contentType, objectName string) (string, error) {
	// 1. Upload ke MinIO
	err := u.minioS3.UploadFile(ctx, "chat-app", objectName, reader, size, contentType)
	if err != nil {
		return "", fmt.Errorf("failed to upload to storage: %w", err)
	}

	// 2. Build URL
	avatarURL := fmt.Sprintf("/chat-app/%s", objectName)

	// 3. Update DB (rollback MinIO kalau gagal)
	err = u.userRepository.UpdateAvatar(ctx, userID, avatarURL)
	if err != nil {
		rollbackCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if rmErr := u.minioS3.DeleteObject(rollbackCtx, "chat-app", objectName); rmErr != nil {
			log.Printf("[ERR] Rollback MinIO failed for user %s: %v", userID, rmErr)
		}
		return "", fmt.Errorf("failed to update avatar in db: %w", err)
	}

	return avatarURL, nil
}

func NewUsersServices(userRepository repository.RepositoryUser, firebase repository.RepositoryFirebase, jwtConfig config.JwtConfig, minio storage.ObjectStorage) UsersServices {
	return &UsersServivesImpl{
		userRepository:     userRepository,
		firebaseRepository: firebase,
		minioS3:            minio,
		jwtConfig:          jwtConfig,
	}
}
