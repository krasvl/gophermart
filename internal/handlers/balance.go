package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/krasvl/market/internal/storage"
	"go.uber.org/zap"
)

// BalanceResponse represents the response body for a user's balance.
type BalanceResponse struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

// WithdrawRequest represents the request body for withdrawing points.
type WithdrawRequest struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

// WithdrawalResponse represents the response body for a withdrawal.
type WithdrawalResponse struct {
	ProcessedAt time.Time `json:"processed_at"`
	Order       string    `json:"order"`
	Sum         float64   `json:"sum"`
}

type BalanceHandler struct {
	logger  *zap.Logger
	storage storage.BalanceStorage
	secret  string
}

func NewBalanceHandler(logger *zap.Logger, storage storage.BalanceStorage, secret string) *BalanceHandler {
	return &BalanceHandler{
		logger:  logger,
		storage: storage,
		secret:  secret,
	}
}

// GetBalance godoc.
// @Summary Get user balance.
// @Description Get current balance and total withdrawn points.
// @Tags balance
// @Produce json
// @Success 200 {object} BalanceResponse
// @Failure 401 {string} string "Unauthorized".
// @Failure 500 {string} string "Internal server error".
// @Security BearerAuth
// @Router /api/user/balance [get].
func (h *BalanceHandler) GetBalance(c *gin.Context) {
	userID := c.GetInt("userID")

	balance, err := h.storage.GetBalance(userID)
	if err != nil {
		h.logger.Error("failed to get balance", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	response := BalanceResponse{
		Current:   balance.Current,
		Withdrawn: balance.Withdrawn,
	}

	c.JSON(http.StatusOK, response)
}

// Withdraw godoc.
// @Summary Withdraw points from balance.
// @Description Withdraw points from balance for a new order.
// @Tags balance
// @Accept json
// @Produce json
// @Param withdrawal body WithdrawRequest true "Withdrawal".
// @Success 200 {string} string "Withdrawal successful".
// @Failure 401 {string} string "Unauthorized".
// @Failure 402 {string} string "Insufficient funds".
// @Failure 422 {string} string "Invalid order number".
// @Failure 500 {string} string "Internal server error".
// @Security BearerAuth
// @Router /api/user/balance/withdraw [post].
func (h *BalanceHandler) Withdraw(c *gin.Context) {
	userID := c.GetInt("userID")

	var req WithdrawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	withdrawal := storage.Withdrawal{
		UserID:      userID,
		OrderNumber: req.Order,
		Sum:         req.Sum,
		ProcessedAt: time.Now(),
	}

	err := h.storage.Withdraw(userID, withdrawal)
	if err != nil {
		h.logger.Error("failed to withdraw balance", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Withdrawal successful"})
}

// GetWithdrawals godoc.
// @Summary Get list of withdrawals.
// @Description Get list of withdrawals made by the user.
// @Tags withdrawal
// @Produce json
// @Success 200 {array} WithdrawalResponse
// @Failure 204 {string} string "No content".
// @Failure 401 {string} string "Unauthorized".
// @Failure 500 {string} string "Internal server error".
// @Security BearerAuth
// @Router /api/user/withdrawals [get].
func (h *BalanceHandler) GetWithdrawals(c *gin.Context) {
	userID := c.GetInt("userID")

	withdrawals, err := h.storage.GetWithdrawals(userID)
	if err != nil {
		h.logger.Error("failed to get withdrawals", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if len(withdrawals) == 0 {
		c.JSON(http.StatusNoContent, nil)
		return
	}

	var response = make([]WithdrawalResponse, 0, len(withdrawals))
	for _, withdrawal := range withdrawals {
		response = append(response, WithdrawalResponse{
			Order:       withdrawal.OrderNumber,
			Sum:         withdrawal.Sum,
			ProcessedAt: withdrawal.ProcessedAt,
		})
	}

	c.JSON(http.StatusOK, response)
}
