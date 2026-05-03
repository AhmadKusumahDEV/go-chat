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
	roomgroup.GET("/:id/members", r.handle.HandlerGetRoomMembers)
	roomgroup.GET("/search", r.handle.HandleGetRoomByName)

	roomgroup.POST("/", r.handle.HandleCreateRoom)
	roomgroup.POST("/join", r.handle.HandleJoinRoom)
	roomgroup.POST("/:id/leave", r.handle.HandleLeaveRoom)

	roomgroup.PUT("/:id", r.handle.HandlerUpdatedRoom)

	roomgroup.DELETE("/:id", r.handle.HandlerDeleteRoom)
	roomgroup.DELETE("/:id/members", r.handle.HandleRemoveMember)
}

func NewRoomRouter(handler handlers.HandlerRoom) config.RouteRegistrar {
	return &RoomRoutes{
		handle: handler,
	}
}
