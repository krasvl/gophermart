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

type MockOrderStorage struct {
	orders []storage.Order
}

func NewMockOrderStorage() *MockOrderStorage {
	return &MockOrderStorage{orders: []storage.Order{}}
}

func (m *MockOrderStorage) GetOrderHolder(order string) (int, bool, error) {
	for _, o := range m.orders {
		if o.Number == order {
			return o.UserID, true, nil
		}
	}
	return -1, false, nil
}

func (m *MockOrderStorage) AddOrder(order *storage.Order) error {
	for _, o := range m.orders {
		if o.Number == order.Number {
			return errors.New("order already exists")
		}
	}
	m.orders = append(m.orders, *order)
	return nil
}

func (m *MockOrderStorage) GetOrders(userID int) ([]storage.Order, error) {
	var userOrders []storage.Order
	for _, order := range m.orders {
		if order.UserID == userID {
			userOrders = append(userOrders, order)
		}
	}
	return userOrders, nil
}

func (m *MockOrderStorage) GetPendingOrders() ([]storage.Order, error) {
	var pendingOrders []storage.Order
	for _, order := range m.orders {
		if order.Status == storage.StatusNew || order.Status == storage.StatusProcessing {
			pendingOrders = append(pendingOrders, order)
		}
	}
	return pendingOrders, nil
}

func (m *MockOrderStorage) ProcessOrder(order *storage.Order) error {
	for i, o := range m.orders {
		if o.ID == order.ID {
			m.orders[i] = *order
			return nil
		}
	}
	return errors.New("order not found")
}

func TestAddOrder(t *testing.T) {
	logger := zap.NewNop()
	storage := NewMockOrderStorage()
	handler := NewOrderHandler(logger, storage, "testsecret")

	router := gin.New()
	router.POST("/api/user/orders", handler.AddOrder)

	t.Run("Invalid Request", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBufferString(""))
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("Invalid Request", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBufferString("123"))
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("Valid Request", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBufferString("4111111111111111"))
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusAccepted, w.Code)
	})

	t.Run("Valid Request same order", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBufferString("4627100101654724"))
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusAccepted, w.Code)

		w = httptest.NewRecorder()
		req, _ = http.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBufferString("4627100101654724"))
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestGetOrders(t *testing.T) {
	logger := zap.NewNop()
	mockStorage := NewMockOrderStorage()
	handler := NewOrderHandler(logger, mockStorage, "testsecret")

	router := gin.New()
	router.POST("/api/user/orders", handler.AddOrder)
	router.GET("/api/user/orders", handler.GetOrders)

	t.Run("No Orders", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/user/orders", http.NoBody)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("With Orders", func(t *testing.T) {
		// Add an order first
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBufferString("4111111111111111"))
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusAccepted, w.Code)

		w = httptest.NewRecorder()
		req, _ = http.NewRequest(http.MethodGet, "/api/user/orders", http.NoBody)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
