package scheduler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/krasvl/market/internal/storage"
	"go.uber.org/zap"
)

type checkStatus string

const (
	Success checkStatus = "SUCCESS"
	Fail    checkStatus = "FAIL"
	Busy    checkStatus = "BUSY"
)

type Scheduler struct {
	logger       *zap.Logger
	orderStorage *storage.OrderStoragePostgres
	accrualAddr  string
}

func NewScheduler(logger *zap.Logger, orderStorage *storage.OrderStoragePostgres, accrualAddr string) *Scheduler {
	return &Scheduler{
		logger:       logger,
		orderStorage: orderStorage,
		accrualAddr:  accrualAddr,
	}
}

func (s *Scheduler) Start() {
	for range time.Tick(10 * time.Second) {
		s.checkOrders()
	}
}

func (s *Scheduler) checkOrders() {
	orders, err := s.orderStorage.GetPendingOrders()
	if err != nil {
		s.logger.Error("failed to get pending orders", zap.Error(err))
		return
	}
	s.logger.Debug("got pending orders", zap.Int("count", len(orders)))

	for _, order := range orders {
		switch status, order := s.checkOrder(&order); status {
		case Fail:
			continue
		case Busy:
			s.logger.Warn("too many requests to accrual system")
			return
		case Success:
			if err := s.orderStorage.ProcessOrder(order); err != nil {
				s.logger.Error("failed to update order status", zap.Error(err))
			}
			s.logger.Info("order was processed with status",
				zap.String("order", order.Number),
				zap.String("status", string(order.Status)),
			)
		}
	}
}

func (s *Scheduler) checkOrder(order *storage.Order) (checkStatus, *storage.Order) {
	url := fmt.Sprintf("%s/api/orders/%s", s.accrualAddr, order.Number)
	resp, err := http.Get(url)
	if err != nil {
		s.logger.Error("failed to get order status", zap.Error(err))
		return Fail, nil
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Println("Failed to close response body:", err)
		}
	}()

	if resp.StatusCode == http.StatusTooManyRequests {
		return Busy, nil
	}

	if resp.StatusCode == http.StatusNoContent {
		order.Status = storage.StatusInvalid
		return Success, order
	}

	if resp.StatusCode != http.StatusOK {
		s.logger.Error("unexpected response from accrual system", zap.Int("status", resp.StatusCode))
		return Fail, nil
	}

	var result struct {
		Order   string              `json:"order"`
		Status  storage.OrderStatus `json:"status"`
		Accrual float64             `json:"accrual,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		s.logger.Error("failed to decode response", zap.Error(err))
		return Fail, nil
	}

	order.Status = result.Status
	order.Accrual = result.Accrual

	return Success, order
}
