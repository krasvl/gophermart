package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/krasvl/market/internal/storage"
	"github.com/krasvl/market/internal/utils"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// RegisterRequest represents the request body for user registration.
type RegisterRequest struct {
	Login    string `json:"login" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginRequest represents the request body for user login.
type LoginRequest struct {
	Login    string `json:"login" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UserHandler struct {
	logger  *zap.Logger
	storage storage.UserStorage
	secret  string
}

func NewUserHandler(logger *zap.Logger, storage storage.UserStorage, secret string) *UserHandler {
	return &UserHandler{
		logger:  logger,
		storage: storage,
		secret:  secret,
	}
}

// RegisterUser godoc.
// @Summary Register a new user.
// @Description Register a new user with login and password.
// @Tags user
// @Accept json
// @Produce json
// @Param user body RegisterRequest true "User".
// @Success 200 {string} string "JWT token".
// @Failure 400 {string} string "Invalid request".
// @Failure 409 {string} string "Login already exists".
// @Failure 500 {string} string "Internal server error".
// @Router /api/user/register [post].
func (h *UserHandler) RegisterUser(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		h.logger.Error("failed to hash password", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	user := storage.User{
		Login:     req.Login,
		Password:  string(hashedPassword),
		CreatedAt: time.Now(),
	}

	userID, err := h.storage.AddUser(user)
	if err != nil {
		if err.Error() == "pq: duplicate key value violates unique constraint \"orders_user_id_number_key\"" {
			c.JSON(http.StatusConflict, gin.H{"error": "Login already exists"})
		} else {
			h.logger.Error("failed to add user", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}
		return
	}

	user.ID = userID
	token, err := utils.GenerateToken(user, h.secret)
	if err != nil {
		h.logger.Error("failed to generate token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.Header("Authorization", "Bearer "+token)
	c.JSON(http.StatusOK, gin.H{"message": "User registered successfully"})
}

// LoginUser godoc.
// @Summary Login a user.
// @Description Login a user with login and password.
// @Tags user
// @Accept json
// @Produce json
// @Param user body LoginRequest true "User".
// @Success 200 {string} string "JWT token".
// @Failure 400 {string} string "Invalid request".
// @Failure 401 {string} string "Invalid login or password".
// @Failure 500 {string} string "Internal server error".
// @Router /api/user/login [post].
func (h *UserHandler) LoginUser(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	user, err := h.storage.GetUser(req.Login)
	if err != nil {
		h.logger.Error("failed to get user", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid login or password"})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid login or password"})
		return
	}

	token, err := utils.GenerateToken(user, h.secret)
	if err != nil {
		h.logger.Error("failed to generate token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.Header("Authorization", "Bearer "+token)
	c.JSON(http.StatusOK, gin.H{"message": "User logged in successfully"})
}
