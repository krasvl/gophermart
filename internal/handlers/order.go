package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/krasvl/market/internal/storage"
	"github.com/krasvl/market/internal/utils"
	"go.uber.org/zap"
)

// OrderResponse represents an order submitted by a user.
type OrderResponse struct {
	UploadedAt time.Time           `json:"uploaded_at"`
	Number     string              `json:"number"`
	Status     storage.OrderStatus `json:"status"`
	Accrual    float64             `json:"accrual,omitempty"`
}

type OrderHandler struct {
	logger  *zap.Logger
	storage storage.OrderStorage
	secret  string
}

func NewOrderHandler(logger *zap.Logger, storage storage.OrderStorage, secret string) *OrderHandler {
	return &OrderHandler{
		logger:  logger,
		storage: storage,
		secret:  secret,
	}
}

// AddOrder godoc.
// @Summary Submit an order number.
// @Description Submit an order number for loyalty points calculation.
// @Tags order
// @Accept plain
// @Produce json
// @Param order body string true "Order Number".
// @Success 200 {string} string "Order already uploaded by this user.".
// @Success 202 {string} string "Order accepted for processing.".
// @Failure 400 {string} string "Invalid request.".
// @Failure 401 {string} string "Unauthorized.".
// @Failure 409 {string} string "Order number already exists.".
// @Failure 422 {string} string "Invalid order number.".
// @Failure 500 {string} string "Internal server error.".
// @Security BearerAuth
// @Router /api/user/orders [post].
func (h *OrderHandler) AddOrder(c *gin.Context) {
	userID := c.GetInt("userID")

	orderNumber, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request."})
		return
	}

	if len(orderNumber) == 0 || !utils.IsValidLuhn(string(orderNumber)) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Invalid order number."})
		return
	}

	h.logger.Info("id", zap.Int("userID", userID))
	order := storage.Order{
		UserID: userID,
		Number: string(orderNumber),
		Status: "NEW",
	}

	holderID, ok, err := h.storage.GetOrderHolder(order.Number)
	if err != nil {
		h.logger.Error("failed to get order holder", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error."})
		return
	}
	if ok && holderID == userID {
		c.JSON(http.StatusOK, gin.H{"message": "Order already uploaded by this user."})
		return
	}
	if ok && holderID != userID {
		c.JSON(http.StatusConflict, gin.H{"error": "Order number already exists."})
		return
	}
	err = h.storage.AddOrder(&order)
	if err != nil {
		h.logger.Error("failed to add order", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error."})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "Order accepted for processing."})
}

// GetOrders godoc.
// @Summary Get list of orders.
// @Description Get list of orders submitted by the user.
// @Tags order
// @Produce json
// @Success 200 {array} OrderResponse
// @Failure 204 {string} string "No content.".
// @Failure 401 {string} string "Unauthorized.".
// @Failure 500 {string} string "Internal server error.".
// @Security BearerAuth
// @Router /api/user/orders [get].
func (h *OrderHandler) GetOrders(c *gin.Context) {
	userID := c.GetInt("userID")

	orders, err := h.storage.GetOrders(userID)
	if err != nil {
		h.logger.Error("failed to get orders", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error."})
		return
	}

	if len(orders) == 0 {
		c.JSON(http.StatusNoContent, nil)
		return
	}

	var response = make([]OrderResponse, 0)
	for _, order := range orders {
		response = append(response, OrderResponse{
			Number:     order.Number,
			Status:     order.Status,
			Accrual:    order.Accrual,
			UploadedAt: order.UploadedAt,
		})
	}

	c.JSON(http.StatusOK, response)
}
