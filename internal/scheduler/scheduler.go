package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/krasvl/market/internal/storage"
	"go.uber.org/zap"
)

type Scheduler struct {
	logger         *zap.Logger
	orderStorage   *storage.OrderStoragePostgres
	accrualTimeout atomic.Value
	accrualAddr    string
}

func NewScheduler(logger *zap.Logger, orderStorage *storage.OrderStoragePostgres, accrualAddr string) *Scheduler {
	return &Scheduler{
		logger:       logger,
		orderStorage: orderStorage,
		accrualAddr:  accrualAddr,
	}
}

func (s *Scheduler) Start() {
	s.accrualTimeout.Store(time.Duration(10) * time.Second)
	for range time.Tick(s.accrualTimeout.Load().(time.Duration)) {
		s.accrualTimeout.Store(time.Duration(10) * time.Second)
		s.checkOrders()
	}
}

func (s *Scheduler) checkOrders() {
	orders, err := s.orderStorage.GetPendingOrders()
	if err != nil {
		s.logger.Error("failed to get pending orders", zap.Error(err))
		return
	}

	jobs := make(chan *storage.Order, len(orders))
	results := make(chan *storage.Order, len(orders))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for w := 1; w <= 5; w++ {
		go s.worker(ctx, cancel, jobs, results)
	}
	for _, order := range orders {
		jobs <- &order
	}

	close(jobs)

	for order := range results {
		if err := s.orderStorage.ProcessOrder(order); err != nil {
			s.logger.Error("failed to update order status", zap.Error(err))
		}
		s.logger.Info("order was processed with status",
			zap.String("order", order.Number),
			zap.String("status", string(order.Status)),
		)
	}
}

func (s *Scheduler) worker(
	ctx context.Context,
	cancel context.CancelFunc,
	jobs <-chan *storage.Order,
	results chan<- *storage.Order,
) {
	for order := range jobs {
		select {
		case <-ctx.Done():
			return
		default:
			res, processedOrder := s.checkOrder(ctx, order)
			if res.status == Success {
				results <- processedOrder
			}
			if res.status == Busy {
				s.logger.Warn("accrual system is busy, retry after", zap.Int("timeout", res.timeout))
				s.accrualTimeout.Store(time.Duration(res.timeout) * time.Second)
				cancel()
				return
			}
			if res.status == Fail {
				s.logger.Error("failed to check order status")
				continue
			}
		}
	}
}

type checkResult struct {
	status  checkStatus
	timeout int
}

type checkStatus string

const (
	Success checkStatus = "SUCCESS"
	Fail    checkStatus = "FAIL"
	Busy    checkStatus = "BUSY"
)

func (s *Scheduler) checkOrder(
	ctx context.Context,
	order *storage.Order,
) (checkResult, *storage.Order) {
	url := fmt.Sprintf("%s/api/orders/%s", s.accrualAddr, order.Number)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		s.logger.Error("failed to create request", zap.Error(err))
		return checkResult{status: Fail}, nil
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s.logger.Error("failed to get order status", zap.Error(err))
		return checkResult{status: Fail}, nil
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Println("Failed to close response body:", err)
		}
	}()

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := 60
		if retryAfterStr := resp.Header.Get("Retry-After"); retryAfterStr != "" {
			if retryAfterSec, err := strconv.Atoi(retryAfterStr); err == nil {
				retryAfter = retryAfterSec
			}
		}
		return checkResult{status: Busy, timeout: retryAfter}, nil
	}

	if resp.StatusCode == http.StatusNoContent {
		order.Status = storage.StatusInvalid
		return checkResult{status: Success}, order
	}

	if resp.StatusCode != http.StatusOK {
		s.logger.Error("unexpected response from accrual system", zap.Int("status", resp.StatusCode))
		return checkResult{status: Fail}, nil
	}

	var result struct {
		Order   string              `json:"order"`
		Status  storage.OrderStatus `json:"status"`
		Accrual float64             `json:"accrual,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		s.logger.Error("failed to decode response", zap.Error(err))
		return checkResult{status: Fail}, nil
	}

	order.Status = result.Status
	order.Accrual = result.Accrual

	return checkResult{status: Success}, order
}
