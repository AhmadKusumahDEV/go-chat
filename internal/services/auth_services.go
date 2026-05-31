package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/AhmadKusumahDEV/go-chat/internal/cahce"
	"github.com/AhmadKusumahDEV/go-chat/internal/config"
	"github.com/AhmadKusumahDEV/go-chat/internal/dto/response"
	"github.com/AhmadKusumahDEV/go-chat/internal/helpers"
	"github.com/AhmadKusumahDEV/go-chat/internal/models"
	"github.com/AhmadKusumahDEV/go-chat/internal/repository"
)

type OauthServices interface {
	GitHubCallback(ctx context.Context, code string, state string, provider string) (*models.OauthResult, error)
	GoogleCallBack(ctx context.Context, code string, state string, provider string) (*models.OauthResult, error)
	BuildGithubAuthURL(ctx context.Context) string
	BuildGoogleAuthURL(ctx context.Context) string
}

type OauthServicesImpl struct {
	cfg       config.Cfg
	rds       cahce.CahceRedis
	oauthRepo repository.RepositoryOauthStates
	userRepo  repository.RepositoryUser
}

func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// BuildGithubAuthURL implements [OauthServices].
func (o *OauthServicesImpl) BuildGithubAuthURL(ctx context.Context) string {
	state, err := generateState()
	if err != nil {
		return "err on generate state"
	}

	err = o.rds.Set(ctx, o.cfg.OAuth.GithubClientID+state, state, 5*time.Minute)
	if err != nil {
		fmt.Println("Redis set failed, falling back to DB for oauth state:", err)
		errDB := o.oauthRepo.Create(ctx, &models.OauthStates{
			State:     state,
			Provider:  "github",
			ExpiresAt: time.Now().Add(5 * time.Minute),
		})
		if errDB != nil {
			fmt.Println("Failed to save oauth state to DB:", errDB)
		}
	}

	params := url.Values{}
	params.Set("client_id", o.cfg.OAuth.GithubClientID)
	params.Set("redirect_uri", o.cfg.OAuth.GithubRedirectURL)
	params.Set("scope", "read:user,user:email")
	params.Set("state", state)

	return "https://github.com/login/oauth/authorize?" + params.Encode()
}

func (o *OauthServicesImpl) GitHubCallback(ctx context.Context, code string, state string, provider string) (*models.OauthResult, error) {
	var savedState string
	err := o.rds.Get(ctx, o.cfg.OAuth.GithubClientID+state, &savedState)
	if err == nil {
		if savedState != state {
			return nil, errors.New("state mismatch from redis")
		}
	} else {
		stateDB, errDB := o.oauthRepo.FindByState(ctx, state)
		if errDB != nil {
			return nil, fmt.Errorf("invalid or expired state (redis & db): %w", errDB)
		}
		if stateDB.Provider != provider {
			return nil, fmt.Errorf("invalid oauth provider: expected %s, got %s", provider, stateDB.Provider)
		}
		if time.Now().After(stateDB.ExpiresAt) {
			return nil, errors.New("state token has expired")
		}
	}

	accessToken, err := o.GitHubExchangeCodeForToken(code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	userInfo, err := o.fetchGitHubUserInfo(accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}

	user, err := o.findOrCreateGitHubUser(ctx, userInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to find or create user: %w", err)
	}

	userInfoJwt := models.JwtUsersInfo{
		UserID:   user.ID.String(),
		Email:    user.Email,
		Username: user.Username,
	}

	jwtAccessToken, jwtRefreshToken, err := helpers.GenerateAuthTokens(
		userInfoJwt,
		o.cfg.Jwt.SecretKeyAccess,
		o.cfg.Jwt.SecretKeyrefresh,
		o.cfg.Jwt.AccessTokenExpiration,
		o.cfg.Jwt.RefreshTokenExpiration,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	result := &models.OauthResult{
		AppScheme:        o.cfg.OAuth.AppScheme,
		Accesstoken:      jwtAccessToken,
		RefreshToken:     jwtRefreshToken,
		RedirectDeepLink: o.cfg.OAuth.MobileRedirect,
		Userid:           user.ID.String(),
	}

	return result, nil
}

func (o *OauthServicesImpl) GitHubExchangeCodeForToken(code string) (string, error) {
	data := url.Values{}
	data.Set("client_id", o.cfg.OAuth.GithubClientID)
	data.Set("client_secret", o.cfg.OAuth.GithubClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", o.cfg.OAuth.GithubRedirectURL)

	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if accessToken, ok := result["access_token"].(string); ok {
		return accessToken, nil
	}

	return "", errors.New("no access_token in response")
}

func (s *OauthServicesImpl) fetchGitHubUserInfo(accessToken string) (*models.GithubUserInfo, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var userInfo models.GithubUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	if userInfo.Email == "" {
		email, err := s.fetchGitHubEmail(accessToken)
		if err == nil {
			userInfo.Email = email
		}
	}

	return &userInfo, nil
}

func (o *OauthServicesImpl) fetchGitHubEmail(accessToken string) (string, error) {
	req, _ := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var emails []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}

	// Find primary email
	for _, email := range emails {
		if primary, ok := email["primary"].(bool); ok && primary {
			if emailStr, ok := email["email"].(string); ok {
				return emailStr, nil
			}
		}
	}

	return "", errors.New("no primary email found")
}

// findOrCreateGitHubUser looks up a user by provider_id=github, or by email (if already registered),
// or creates one. Returns error if email exists with different provider.
func (o *OauthServicesImpl) findOrCreateGitHubUser(ctx context.Context, userInfo *models.GithubUserInfo) (*models.Users, error) {
	oauthID := fmt.Sprintf("%d", userInfo.ID)

	user, err := o.userRepo.FindByProviderID(ctx, "github", oauthID)
	if err == nil {
		user.AvatarUrl = sql.NullString{String: userInfo.AvatarURL, Valid: true}
		if user.Username == "" {
			user.Username = userInfo.Login
		}

		errUpdate := o.userRepo.Update(ctx, user)
		if errUpdate != nil {
			return nil, errUpdate
		}

		return user, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	existingUser, errEmail := o.userRepo.FindByEmail(ctx, userInfo.Email)
	if errEmail == nil {
		if existingUser.ProviderName.Valid && existingUser.ProviderName.String != "" {
			return nil, fmt.Errorf("email %s is already registered with %s. Please login with %s instead.",
				userInfo.Email, existingUser.ProviderName.String, existingUser.ProviderName.String)
		}

		existingUser.ProviderName = sql.NullString{String: "github", Valid: true}
		existingUser.ProviderID = sql.NullString{String: oauthID, Valid: true}
		existingUser.AvatarUrl = sql.NullString{String: userInfo.AvatarURL, Valid: true}
		if existingUser.Username == "" {
			existingUser.Username = userInfo.Login
		}

		errUpdate := o.userRepo.Update(ctx, &existingUser)
		if errUpdate != nil {
			return nil, errUpdate
		}

		return &existingUser, nil
	}

	// 3. Email not found - create new user
	newUser := &models.Users{
		Username:     userInfo.Login,
		Email:        userInfo.Email,
		ProviderName: sql.NullString{String: "github", Valid: true},
		ProviderID:   sql.NullString{String: oauthID, Valid: true},
		AvatarUrl:    sql.NullString{String: userInfo.AvatarURL, Valid: true},
	}

	if err := o.userRepo.Create(ctx, newUser); err != nil {
		return nil, err
	}

	createdUser, _ := o.userRepo.FindByProviderID(ctx, "github", oauthID)
	if createdUser != nil {
		return createdUser, nil
	}

	return newUser, nil
}

func (o *OauthServicesImpl) GoogleCallBack(ctx context.Context, code string, state string, provider string) (*models.OauthResult, error) {
	var savedVerifier string

	err := o.rds.Get(ctx, o.cfg.OAuth.GoogleClientID+state, &savedVerifier)
	if err != nil {
		stateDB, errDB := o.oauthRepo.FindByState(ctx, state)
		if errDB != nil {
			return nil, fmt.Errorf("invalid or expired state (redis & db): %w", errDB)
		}
		if stateDB.Provider != provider {
			return nil, fmt.Errorf("invalid oauth provider: expected %s, got %s", provider, stateDB.Provider)
		}
		if time.Now().After(stateDB.ExpiresAt) {
			return nil, errors.New("state token has expired")
		}
		savedVerifier = stateDB.Verifier

		defer o.oauthRepo.DeleteByState(ctx, state)
	} else {
		defer o.rds.Del(ctx, o.cfg.OAuth.GoogleClientID+state)
	}

	accessToken, err := o.ExchangeCodeForToken(ctx, code, savedVerifier)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	userInfo, err := o.fetchGoogleUserInfo(accessToken.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}

	user, err := o.findOrCreateGoogleUser(ctx, userInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to find or create user: %w", err)
	}

	userInfoJwt := models.JwtUsersInfo{
		UserID:   user.ID.String(),
		Email:    user.Email,
		Username: user.Username,
	}

	jwtAccessToken, jwtRefreshToken, err := helpers.GenerateAuthTokens(
		userInfoJwt,
		o.cfg.Jwt.SecretKeyAccess,
		o.cfg.Jwt.SecretKeyrefresh,
		o.cfg.Jwt.AccessTokenExpiration,
		o.cfg.Jwt.RefreshTokenExpiration,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &models.OauthResult{
		AppScheme:        o.cfg.OAuth.AppScheme,
		Accesstoken:      jwtAccessToken,
		RefreshToken:     jwtRefreshToken,
		RedirectDeepLink: o.cfg.OAuth.MobileRedirect,
		Userid:           user.ID.String(),
	}, nil
}

// googleExchangeCodeForToken exchanges the Google authorization code for an access token.
func (o *OauthServicesImpl) ExchangeCodeForToken(ctx context.Context, code, verifier string) (*response.GoogleTokenResponse, error) {
	data := url.Values{}
	data.Set("client_id", o.cfg.OAuth.GoogleClientID)
	data.Set("client_secret", o.cfg.OAuth.GoogleClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", o.cfg.OAuth.GoogleRedirectURL)
	data.Set("grant_type", "authorization_code")
	data.Set("code_verifier", verifier)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://oauth2.googleapis.com/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error       string `json:"error"`
			Description string `json:"error_description"`
		}
		if json.Unmarshal(body, &errResp) == nil {
			return nil, fmt.Errorf("google oauth error: %s (%s)", errResp.Error, errResp.Description)
		}
		return nil, fmt.Errorf("token exchange failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	var tokenResp response.GoogleTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return nil, errors.New("no access_token in response")
	}

	return &tokenResp, nil
}

// fetchGoogleUserInfo fetches user profile from Google's userinfo endpoint.
func (o *OauthServicesImpl) fetchGoogleUserInfo(accessToken string) (*models.GoogleUserInfo, error) {
	req, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var userInfo models.GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	if userInfo.Email == "" {
		return nil, errors.New("google account has no email")
	}

	return &userInfo, nil
}

// findOrCreateGoogleUser looks up a user by provider_id=google, or by email, or creates one.
// Returns error if email exists with different provider.
func (o *OauthServicesImpl) findOrCreateGoogleUser(ctx context.Context, userInfo *models.GoogleUserInfo) (*models.Users, error) {
	oauthID := userInfo.ID

	user, err := o.userRepo.FindByProviderID(ctx, "google", oauthID)
	if err == nil {
		user.AvatarUrl = sql.NullString{String: userInfo.AvatarURL, Valid: true}
		if user.Username == "" {
			user.Username = userInfo.Name
		}

		errUpdate := o.userRepo.Update(ctx, user)
		if errUpdate != nil {
			return nil, errUpdate
		}

		return user, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	existingUser, errEmail := o.userRepo.FindByEmail(ctx, userInfo.Email)
	if errEmail == nil {
		if existingUser.ProviderName.Valid && existingUser.ProviderName.String != "" {
			return nil, fmt.Errorf("email %s is already registered with %s. Please login with %s instead.",
				userInfo.Email, existingUser.ProviderName.String, existingUser.ProviderName.String)
		}

		existingUser.ProviderName = sql.NullString{String: "google", Valid: true}
		existingUser.ProviderID = sql.NullString{String: oauthID, Valid: true}
		existingUser.AvatarUrl = sql.NullString{String: userInfo.AvatarURL, Valid: true}
		if existingUser.Username == "" {
			existingUser.Username = userInfo.Name
		}

		errUpdate := o.userRepo.Update(ctx, &existingUser)
		if errUpdate != nil {
			return nil, errUpdate
		}

		return &existingUser, nil
	}

	newUser := &models.Users{
		Username:     userInfo.Name,
		Email:        userInfo.Email,
		ProviderName: sql.NullString{String: "google", Valid: true},
		ProviderID:   sql.NullString{String: oauthID, Valid: true},
		AvatarUrl:    sql.NullString{String: userInfo.AvatarURL, Valid: true},
	}

	if err := o.userRepo.Create(ctx, newUser); err != nil {
		return nil, err
	}

	createdUser, _ := o.userRepo.FindByProviderID(ctx, "google", oauthID)
	if createdUser != nil {
		return createdUser, nil
	}

	return newUser, nil
}

// BuildGoogleAuthURL implements OauthServices.
func (o *OauthServicesImpl) BuildGoogleAuthURL(ctx context.Context) string {
	state, err := generateState()
	if err != nil {
		return "err on generate state"
	}

	verifier, err := generateCodeVerifier()
	if err != nil {
		return "err on generate code verifier"
	}
	challenge := generateCodeChallenge(verifier)

	err = o.rds.Set(ctx, o.cfg.OAuth.GoogleClientID+state, verifier, 5*time.Minute)
	if err != nil {
		fmt.Println("Redis set failed, falling back to DB for oauth state:", err)
		errDB := o.oauthRepo.Create(ctx, &models.OauthStates{
			State:     state,
			Provider:  "google",
			ExpiresAt: time.Now().Add(5 * time.Minute),
			Verifier:  verifier,
		})
		if errDB != nil {
			fmt.Println("Failed to save oauth state to DB:", errDB)
		}
	}

	params := url.Values{}
	params.Set("client_id", o.cfg.OAuth.GoogleClientID)
	params.Set("redirect_uri", o.cfg.OAuth.GoogleRedirectURL)
	params.Set("response_type", "code")
	params.Set("scope", "openid email profile")
	params.Set("state", state)
	params.Set("access_type", "offline")
	params.Set("prompt", "consent")
	params.Set("code_challenge", challenge)
	params.Set("code_challenge_method", "S256")

	return "https://accounts.google.com/o/oauth2/v2/auth?" + params.Encode()
}

// Generate PKCE verifier (random string 43-128 chars)
func generateCodeVerifier() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// Generate code challenge from verifier (SHA256 + base64url)
func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

func NewOauthServices(config config.Cfg, redis cahce.CahceRedis, oauthRepo repository.RepositoryOauthStates, userRepo repository.RepositoryUser) OauthServices {
	return &OauthServicesImpl{
		cfg:       config,
		rds:       redis,
		oauthRepo: oauthRepo,
		userRepo:  userRepo,
	}
}

func strPtr(s string) *string {
	return &s
}
