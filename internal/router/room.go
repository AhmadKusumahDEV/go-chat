package router

import (
	"github.com/AhmadKusumahDEV/go-chat/internal/config"
	"github.com/AhmadKusumahDEV/go-chat/internal/handlers"
	"github.com/AhmadKusumahDEV/go-chat/internal/middelware"
	"github.com/gin-gonic/gin"
)

type RoomRoutes struct {
	handle handlers.HandlerRoom
}

// RegisterRoutes implements config.RouteRegistrar.
func (r *RoomRoutes) RegisterRoutes(router *gin.Engine, srv *config.Server) {
	roomgroup := router.Group("/api/room")
	roomgroup.Use(middelware.JwtAuthMiddleware(srv.JwtConfig.SecretKeyAccess))

	// roomgroup.GET("/", r.handle.HandleGetAllRoom)
	roomgroup.GET("/", r.handle.HandleGetRoomByUserID)
	roomgroup.GET("/search", r.handle.HandleGetRoomByName)
	roomgroup.GET("/:id", r.handle.HandleGetRoomDetail)
	roomgroup.GET("/direct/:id", r.handle.HandleCheckDirectRoom)

	roomgroup.POST("/", r.handle.HandleCreateRoom)
	roomgroup.POST("/direct", r.handle.HandleCreateDirectRoom)
	roomgroup.POST("/presigned-url", r.handle.GeneratePresignedUrl)

	roomgroup.PATCH("/avatar/:id", r.handle.HandlerUpdateAvatarRoom)
	roomgroup.PATCH("/:id", r.handle.HandlerUpdatedRoom)

	roomgroup.DELETE("/:id", r.handle.HandlerDeleteRoom)
}

func NewRoomRouter(handler handlers.HandlerRoom) config.RouteRegistrar {
	return &RoomRoutes{
		handle: handler,
	}
}
