package router

import (
	"github.com/AhmadKusumahDEV/go-chat/internal/config"
	"github.com/AhmadKusumahDEV/go-chat/internal/handlers"
	"github.com/AhmadKusumahDEV/go-chat/internal/middelware"
	"github.com/gin-gonic/gin"
)

type WebsocketRouter struct {
	handler handlers.WebsocketHandler
}

// RegisterRoutes implements config.RouteRegistrar.
func (r *WebsocketRouter) RegisterRoutes(router *gin.Engine, srv *config.Server) {
	ws := router.Group("/ws")
	ws.Use(middelware.JwtAuthMiddleware(srv.JwtConfig.SecretKeyAccess))
	{
		ws.GET("", r.handler.HandleConnection)
		ws.POST("/notifiactions", r.handler.BroadcastToAll)
		ws.POST("/notifiaction", r.handler.SendToUser)

	}
}

func NewWebsocketRouter(handler handlers.WebsocketHandler) *WebsocketRouter {
	return &WebsocketRouter{handler: handler}
}
