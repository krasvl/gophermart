package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/krasvl/market/internal/storage"
	"github.com/krasvl/market/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware(t *testing.T) {
	secret := "testsecret"
	token, _ := utils.GenerateToken(storage.User{ID: 1}, secret)

	router := gin.New()
	router.Use(AuthMiddleware(secret))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	t.Run("No Authorization Header", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test", http.NoBody)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Valid Token", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test", http.NoBody)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
