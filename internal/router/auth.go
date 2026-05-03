package router

import (
	"github.com/AhmadKusumahDEV/go-chat/internal/config"
	"github.com/AhmadKusumahDEV/go-chat/internal/handlers"
	"github.com/gin-gonic/gin"
)

type AuthRoutes struct {
	handle handlers.HandlerOauth
}

// RegisterRoutes implements config.RouteRegistrar.
func (r *AuthRoutes) RegisterRoutes(router *gin.Engine, srv *config.Server) {
	roomgroup := router.Group("/api/auth")

	roomgroup.GET("/github", r.handle.GithubSIgnIn)
	roomgroup.GET("/github/callback", r.handle.GithubCallback)
	roomgroup.GET("/google", r.handle.GoogleSIgnIn)
	roomgroup.GET("/google/callback", r.handle.GoogleCallback)
}

func NewAuthRouter(handler handlers.HandlerOauth) config.RouteRegistrar {
	return &AuthRoutes{
		handle: handler,
	}
}
