package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func WithLogging(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)

		logger.Info("Request processed",
			zap.String("uri", c.Request.RequestURI),
			zap.String("method", c.Request.Method),
			zap.Duration("duration", duration),
			zap.Int("status", c.Writer.Status()),
			zap.Int("content_length", c.Writer.Size()),
		)
	}
}
