package router

import (
	"github.com/AhmadKusumahDEV/go-chat/internal/config"
	"github.com/AhmadKusumahDEV/go-chat/internal/handlers"
	"github.com/AhmadKusumahDEV/go-chat/internal/middelware"
	"github.com/gin-gonic/gin"
)

type OrderRoutes struct {
	handle handlers.HandlerOrder
}

func (r *OrderRoutes) RegisterRoutes(router *gin.Engine, srv *config.Server) {
	ordergroup := router.Group("/api/order")

	ordergroup.POST("", middelware.JwtAuthMiddleware(srv.JwtConfig.SecretKeyAccess), r.handle.HandlerAddOrder)
	ordergroup.GET("", middelware.JwtAuthMiddleware(srv.JwtConfig.SecretKeyAccess), r.handle.HandlerGetTransaction)
	ordergroup.POST("/midtrans/webhooks", r.handle.HandlerMidtransNotification)
}

func NewOrderRouter(handler handlers.HandlerOrder) config.RouteRegistrar {
	return &OrderRoutes{
		handle: handler,
	}
}
