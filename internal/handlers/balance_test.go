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

type MockBalanceStorage struct {
	balances    map[int]storage.Balance
	withdrawals map[int][]storage.Withdrawal
}

func NewMockBalanceStorage() *MockBalanceStorage {
	return &MockBalanceStorage{
		balances:    make(map[int]storage.Balance),
		withdrawals: make(map[int][]storage.Withdrawal),
	}
}

func (m *MockBalanceStorage) GetBalance(userID int) (storage.Balance, error) {
	balance, exists := m.balances[userID]
	if !exists {
		return storage.Balance{}, errors.New("balance not found")
	}
	return balance, nil
}

func (m *MockBalanceStorage) Withdraw(userID int, withdrawal storage.Withdrawal) error {
	balance, exists := m.balances[userID]
	if !exists || balance.Current < withdrawal.Sum {
		return errors.New("insufficient funds")
	}
	balance.Current -= withdrawal.Sum
	balance.Withdrawn += withdrawal.Sum
	m.balances[userID] = balance
	m.withdrawals[userID] = append(m.withdrawals[userID], withdrawal)
	return nil
}

func (m *MockBalanceStorage) GetWithdrawals(userID int) ([]storage.Withdrawal, error) {
	return m.withdrawals[userID], nil
}

func (m *MockBalanceStorage) AddBalance(balance storage.Balance) error {
	if _, exists := m.balances[balance.UserID]; exists {
		return errors.New("balance already exists")
	}
	m.balances[balance.UserID] = balance
	return nil
}

func (m *MockBalanceStorage) UpdateBalance(balance storage.Balance) error {
	if _, exists := m.balances[balance.UserID]; !exists {
		return errors.New("balance not found")
	}
	m.balances[balance.UserID] = balance
	return nil
}

func TestGetBalance(t *testing.T) {
	logger := zap.NewNop()
	mockStorage := NewMockBalanceStorage()
	handler := NewBalanceHandler(logger, mockStorage, "testsecret")

	router := gin.New()
	router.GET("/api/user/balance", handler.GetBalance)

	mockStorage.balances[0] = storage.Balance{UserID: 1, Current: 1000, Withdrawn: 0}

	t.Run("Valid Request", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/user/balance", http.NoBody)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestWithdraw(t *testing.T) {
	logger := zap.NewNop()
	mockStorage := NewMockBalanceStorage()
	handler := NewBalanceHandler(logger, mockStorage, "testsecret")

	router := gin.New()
	router.POST("/api/user/balance/withdraw", handler.Withdraw)

	mockStorage.balances[0] = storage.Balance{UserID: 1, Current: 1000, Withdrawn: 0}

	t.Run("Invalid Request", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/user/balance/withdraw", bytes.NewBufferString(""))
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Valid Request", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/user/balance/withdraw",
			bytes.NewBufferString(`{"order": "123456789012", "sum": 100}`))
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestGetWithdrawals(t *testing.T) {
	logger := zap.NewNop()
	mockStorage := NewMockBalanceStorage()
	handler := NewBalanceHandler(logger, mockStorage, "testsecret")

	router := gin.New()
	router.POST("/api/user/balance/withdraw", handler.Withdraw)
	router.GET("/api/user/withdrawals", handler.GetWithdrawals)

	mockStorage.balances[0] = storage.Balance{UserID: 1, Current: 1000, Withdrawn: 0}

	t.Run("No Withdrawals", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/user/withdrawals", http.NoBody)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("With Withdrawals", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/user/balance/withdraw",
			bytes.NewBufferString(`{"order": "123456789012", "sum": 100}`))
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		w = httptest.NewRecorder()
		req, _ = http.NewRequest(http.MethodGet, "/api/user/withdrawals", http.NoBody)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
