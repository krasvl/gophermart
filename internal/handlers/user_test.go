package handlers

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/krasvl/market/internal/storage"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type MockUserStorage struct {
	users map[string]storage.User
}

func NewMockUserStorage() *MockUserStorage {
	return &MockUserStorage{users: make(map[string]storage.User)}
}

func (m *MockUserStorage) AddUser(user storage.User) (int, error) {
	if _, exists := m.users[user.Login]; exists {
		return 0, errors.New("user already exists")
	}
	user.ID = len(m.users) + 1
	m.users[user.Login] = user
	return user.ID, nil
}

func (m *MockUserStorage) GetUser(login string) (storage.User, error) {
	user, exists := m.users[login]
	if !exists {
		return storage.User{}, errors.New("user not found")
	}
	return user, nil
}

func TestRegisterUser(t *testing.T) {
	logger := zap.NewNop()
	storage := NewMockUserStorage()
	handler := NewUserHandler(logger, storage, "testsecret")

	router := gin.New()
	router.POST("/api/user/register", handler.RegisterUser)

	t.Run("Invalid Request", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/user/register",
			bytes.NewBufferString(`{"login": "test"}`))
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Valid Request", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/user/register",
			bytes.NewBufferString(`{"login": "test", "password": "password"}`))
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		authHeader := w.Header().Get("Authorization")
		assert.NotEmpty(t, authHeader, "Authorization header should not be empty")
	})
}

func TestLoginUser(t *testing.T) {
	logger := zap.NewNop()
	storage := NewMockUserStorage()
	handler := NewUserHandler(logger, storage, "testsecret")

	router := gin.New()
	router.POST("/api/user/login", handler.LoginUser)
	router.POST("/api/user/register", handler.RegisterUser)

	t.Run("Invalid Request", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/user/login",
			bytes.NewBufferString(`{"login": "test"}`))
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Invalid Login or Password", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/user/login",
			bytes.NewBufferString(`{"login": "test", "password": "wrongpassword"}`))
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Valid Request", func(t *testing.T) {
		// First, register the user
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/user/register",
			bytes.NewBufferString(`{"login": "test", "password": "password"}`))
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// Then, login with the same credentials
		w = httptest.NewRecorder()
		req, _ = http.NewRequest(http.MethodPost, "/api/user/login",
			bytes.NewBufferString(`{"login": "test", "password": "password"}`))
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		authHeader := w.Header().Get("Authorization")
		assert.NotEmpty(t, authHeader, "Authorization header should not be empty")
	})
}
