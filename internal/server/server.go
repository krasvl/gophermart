package server

import (
	"github.com/gin-gonic/gin"
	_ "github.com/krasvl/market/docs"
	"github.com/krasvl/market/internal/handlers"
	"github.com/krasvl/market/internal/middleware"
	"github.com/krasvl/market/internal/storage"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

type Server struct {
	userHandler    *handlers.UserHandler
	orderHandler   *handlers.OrderHandler
	balanceHandler *handlers.BalanceHandler
	logger         *zap.Logger
	addr           string
	secret         string
}

func NewServer(
	addr string,
	userStorage *storage.UserStoragePostgres,
	orderStorage *storage.OrderStoragePostgres,
	balanceStorage *storage.BalanceStoragePostgres,
	logger *zap.Logger,
	secret string,
) *Server {
	userHandler := handlers.NewUserHandler(logger, userStorage, secret)
	orderHandler := handlers.NewOrderHandler(logger, orderStorage, secret)
	balanceHandler := handlers.NewBalanceHandler(logger, balanceStorage, secret)
	return &Server{
		addr:           addr,
		userHandler:    userHandler,
		orderHandler:   orderHandler,
		balanceHandler: balanceHandler,
		logger:         logger,
		secret:         secret,
	}
}

// Start Server
// @title Gophermart API.
// @version 1.0.
// @description This is a sample server for Gophermart.
// @host localhost:8081.
// @BasePath /.
// @securityDefinitions.apikey BearerAuth.
// @in header.
// @name Authorization.
func (s *Server) Start() {
	r := gin.Default()

	// Serve Swagger documentation.
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.Use(middleware.WithLogging(s.logger))

	r.POST("/api/user/register", s.userHandler.RegisterUser)
	r.POST("/api/user/login", s.userHandler.LoginUser)

	auth := r.Group("/")
	auth.Use(middleware.AuthMiddleware(s.secret))
	{
		auth.POST("/api/user/orders", s.orderHandler.AddOrder)
		auth.GET("/api/user/orders", s.orderHandler.GetOrders)
		auth.GET("/api/user/balance", s.balanceHandler.GetBalance)
		auth.POST("/api/user/balance/withdraw", s.balanceHandler.Withdraw)
		auth.GET("/api/user/withdrawals", s.balanceHandler.GetWithdrawals)
	}

	if err := r.Run(s.addr); err != nil {
		s.logger.Fatal("cant start server", zap.Error(err))
	}
}
