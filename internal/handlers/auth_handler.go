package handlers

import (
	"fmt"
	"net/http"

	"github.com/AhmadKusumahDEV/go-chat/internal/helpers"
	"github.com/AhmadKusumahDEV/go-chat/internal/services"
	"github.com/gin-gonic/gin"
)

type HandlerOauth interface {
	GoogleCallback(c *gin.Context)
	GithubCallback(c *gin.Context)
	GoogleSIgnIn(c *gin.Context)
	GithubSIgnIn(c *gin.Context)
}

type HandlerAuthImpl struct {
	srv services.OauthServices
}

func NewHandlerOauth(srv services.OauthServices) HandlerOauth {
	return &HandlerAuthImpl{srv: srv}
}

func (h *HandlerAuthImpl) GithubSIgnIn(c *gin.Context) {
	authURL := h.srv.BuildGithubAuthURL(c.Request.Context())
	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

func (h *HandlerAuthImpl) GoogleSIgnIn(c *gin.Context) {
	authURL := h.srv.BuildGoogleAuthURL(c.Request.Context())
	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

func (h *HandlerAuthImpl) GithubCallback(c *gin.Context) {
	h.handleGithubCallback(c, "github")
}

func (h *HandlerAuthImpl) GoogleCallback(c *gin.Context) {
	h.handleGoogleCallback(c, "google")
}

func (h *HandlerAuthImpl) handleGithubCallback(c *gin.Context, provider string) {
	ctx := c.Request.Context()

	code := c.Query("code")
	state := c.Query("state")
	errorParam := c.Query("error")
	errorDescription := c.Query("error_description")

	if errorParam != "" {
		fmt.Printf("[OAuth] %s callback error: %s - %s\n", provider, errorParam, errorDescription)

		redirectURL := helpers.BuildErrorRedirectURL(errorParam, errorDescription)
		c.Redirect(http.StatusTemporaryRedirect, redirectURL)
		return
	}

	if code == "" {
		redirectURL := helpers.BuildErrorRedirectURL("missing_code", "Authorization code not provided")
		c.Redirect(http.StatusTemporaryRedirect, redirectURL)
		return
	}

	result, err := h.srv.GitHubCallback(ctx, code, state, provider)
	if err != nil {
		fmt.Printf("[OAuth] %s callback process error: %v\n", provider, err)
		redirectURL := helpers.BuildErrorRedirectURL("oauth_failed", "Failed to process OAuth callback")
		c.Redirect(http.StatusTemporaryRedirect, redirectURL)
		return
	}

	redirectURL := helpers.BuildSuccessRedirectURL(result)
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

func (h *HandlerAuthImpl) handleGoogleCallback(c *gin.Context, provider string) {
	ctx := c.Request.Context()

	code := c.Query("code")
	state := c.Query("state")
	errorParam := c.Query("error")
	errorDescription := c.Query("error_description")

	if errorParam != "" {
		fmt.Printf("[OAuth] %s callback error: %s - %s\n", provider, errorParam, errorDescription)

		redirectURL := helpers.BuildErrorRedirectURL(errorParam, errorDescription)
		c.Redirect(http.StatusTemporaryRedirect, redirectURL)
		return
	}

	if code == "" {
		redirectURL := helpers.BuildErrorRedirectURL("missing_code", "Authorization code not provided")
		c.Redirect(http.StatusTemporaryRedirect, redirectURL)
		return
	}

	result, err := h.srv.GoogleCallBack(ctx, code, state, provider)
	if err != nil {
		fmt.Printf("[OAuth] %s callback process error: %v\n", provider, err)
		redirectURL := helpers.BuildErrorRedirectURL("oauth_failed", "Failed to process OAuth callback")
		c.Redirect(http.StatusTemporaryRedirect, redirectURL)
		return
	}

	redirectURL := helpers.BuildSuccessRedirectURL(result)
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}
