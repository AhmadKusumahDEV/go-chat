package handlers

import (
	"errors"
	"net/http"

	"github.com/AhmadKusumahDEV/go-chat/internal/dto/request"
	"github.com/AhmadKusumahDEV/go-chat/internal/dto/response"
	"github.com/AhmadKusumahDEV/go-chat/internal/models"
	"github.com/AhmadKusumahDEV/go-chat/internal/services"
	"github.com/gin-gonic/gin"
)

type HandlerOrder interface {
	HandlerAddOrder(c *gin.Context)
	HandlerGetTransaction(c *gin.Context)
	HandlerMidtransNotification(c *gin.Context)
}

type HandlerOrderImpl struct {
	Order services.OrderServices
}

func NewOrderHandler(order services.OrderServices) HandlerOrder {
	return &HandlerOrderImpl{Order: order}
}

// HandlerAddOrder implements [HandlerOrder].
func (h *HandlerOrderImpl) HandlerAddOrder(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.ApiResponse{
			Status:  http.StatusUnauthorized,
			Message: "unauthorized: user_id not found",
		})
		return
	}

	var req request.PlanRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "data yang dikirim tidak valid",
		})
		return
	}

	res, err := h.Order.AddOrder(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ApiResponse{
			Status:  http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response.ApiResponse{
		Status:  http.StatusOK,
		Data:    res,
		Message: "success add order",
	})
}

// HandlerGetSnapToken implements [HandlerOrder].
func (h *HandlerOrderImpl) HandlerGetTransaction(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.ApiResponse{
			Status:  http.StatusUnauthorized,
			Message: "unauthorized: user_id not found",
		})
		return
	}

	res, err := h.Order.GetTransactionList(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ApiResponse{
			Status:  http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response.ApiResponse{
		Status:  http.StatusOK,
		Data:    res,
		Message: "success get transaction",
	})
}

// HandlerMidtransNotification implements [HandlerOrder].
func (h *HandlerOrderImpl) HandlerMidtransNotification(c *gin.Context) {
	var req response.MidtransNotification
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "data yang dikirim tidak valid",
		})
		return
	}

	err = h.Order.MidtransNotification(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, models.ErrSiganature) {
			c.JSON(http.StatusForbidden, response.ApiResponse{
				Status:  http.StatusForbidden,
				Message: "Forbidden: Invalid Signature",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, response.ApiResponse{
			Status:  http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response.ApiResponse{
		Status:  http.StatusOK,
		Message: "success midtrans notification",
	})
}
