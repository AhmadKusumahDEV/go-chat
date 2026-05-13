package router

import (
	"github.com/AhmadKusumahDEV/go-chat/internal/config"
	"github.com/AhmadKusumahDEV/go-chat/internal/handlers"
	"github.com/AhmadKusumahDEV/go-chat/internal/middelware"
	"github.com/gin-gonic/gin"
)

type MemberRoutes struct {
	handle handlers.HandlerMember
}

// RegisterRoutes implements config.RouteRegistrar.
func (r *MemberRoutes) RegisterRoutes(router *gin.Engine, srv *config.Server) {
	membergroup := router.Group("/api/members")
	membergroup.Use(middelware.JwtAuthMiddleware(srv.JwtConfig.SecretKeyAccess))

	membergroup.GET("/:room_id", r.handle.HandleGetMembers)
	membergroup.POST("/add", r.handle.HandleAddMember)
	membergroup.DELETE("/:room_id/leave", r.handle.HandleLeaveRoom)
	membergroup.DELETE("/:room_id/remove", r.handle.HandleRemoveMember)
}

func NewMemberRouter(handler handlers.HandlerMember) config.RouteRegistrar {
	return &MemberRoutes{
		handle: handler,
	}
}
