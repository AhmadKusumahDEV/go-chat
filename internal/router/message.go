package router

import (
	"github.com/AhmadKusumahDEV/go-chat/internal/config"
	"github.com/AhmadKusumahDEV/go-chat/internal/handlers"
	"github.com/AhmadKusumahDEV/go-chat/internal/middelware"
	"github.com/gin-gonic/gin"
)

type MessageRoutes struct {
	handle handlers.HandlerMessage
}

// RegisterRoutes implements config.RouteRegistrar.
func (r *MessageRoutes) RegisterRoutes(router *gin.Engine, srv *config.Server) {
	msgGroup := router.Group("/api/message")
	msgGroup.Use(middelware.JwtAuthMiddleware(srv.JwtConfig.SecretKeyAccess))

	msgGroup.GET("/:room_id", r.handle.HandleGetRoomMessages)
	msgGroup.POST("/", r.handle.HandleSendMessage)
	msgGroup.PUT("/:id", r.handle.HandleEditMessage)
	msgGroup.POST("/upload", r.handle.UploadMultipleImages)
}

func NewMessageRouter(handler handlers.HandlerMessage) config.RouteRegistrar {
	return &MessageRoutes{
		handle: handler,
	}
}
