package router

import (
	"github.com/AhmadKusumahDEV/go-chat/internal/config"
	"github.com/AhmadKusumahDEV/go-chat/internal/handlers"
	"github.com/gin-gonic/gin"
)

type UsersRoutes struct {
	handle handlers.UserHandler
}

// RegisterRoutes implements config.RouteRegistrar.
func (r *UsersRoutes) RegisterRoutes(router *gin.Engine, srv *config.Server) {
	usersgroup := router.Group("/api/users")

	usersgroup.POST("/register", r.handle.HandlerRegister)
	usersgroup.POST("/login", r.handle.HandlerLogin)
	usersgroup.POST("/refresh", r.handle.HandlerRefresh)

}

func NewUsersRouter(handler handlers.UserHandler) config.RouteRegistrar {
	return &UsersRoutes{
		handle: handler,
	}
}
