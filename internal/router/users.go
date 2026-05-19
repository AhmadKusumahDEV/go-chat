package router

import (
	"github.com/AhmadKusumahDEV/go-chat/internal/config"
	"github.com/AhmadKusumahDEV/go-chat/internal/handlers"
	"github.com/AhmadKusumahDEV/go-chat/internal/middelware"
	"github.com/gin-gonic/gin"
)

type UsersRoutes struct {
	handle handlers.UserHandler
}

// RegisterRoutes implements config.RouteRegistrar.
func (r *UsersRoutes) RegisterRoutes(router *gin.Engine, srv *config.Server) {
	usersgroup := router.Group("/api/users")

	usersgroup.GET("/", middelware.JwtAuthMiddleware(srv.JwtConfig.SecretKeyAccess), r.handle.HandlerGetAllUser)
	usersgroup.GET("/refresh", r.handle.HandlerRefresh)

	usersgroup.POST("/register", r.handle.HandlerRegister)
	usersgroup.POST("/login", r.handle.HandlerLogin)
	usersgroup.POST("/fcm-token", middelware.JwtAuthMiddleware(srv.JwtConfig.SecretKeyAccess), r.handle.HandlerFcmToken)
}

func NewUsersRouter(handler handlers.UserHandler) config.RouteRegistrar {
	return &UsersRoutes{
		handle: handler,
	}
}
