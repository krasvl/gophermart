package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/krasvl/market/internal/storage"
	"go.uber.org/zap"
)

type Scheduler struct {
	logger          *zap.Logger
	orderStorage    *storage.OrderStoragePostgres
	accrualAddr     string
	accrualInterval time.Duration
	intervalMu      sync.RWMutex
	workerPoolSize  int
}

func NewScheduler(
	logger *zap.Logger,
	orderStorage *storage.OrderStoragePostgres,
	accrualAddr string,
) *Scheduler {
	return &Scheduler{
		logger:          logger,
		orderStorage:    orderStorage,
		accrualAddr:     accrualAddr,
		accrualInterval: 10 * time.Second,
		workerPoolSize:  5,
	}
}

func (s *Scheduler) Start() {
	ticker := time.NewTicker(s.getAccrualInterval())
	defer ticker.Stop()

	for range ticker.C {
		s.checkOrders()
		ticker.Reset(s.getAccrualInterval())
	}
}

func (s *Scheduler) checkOrders() {
	orders, err := s.orderStorage.GetPendingOrders()
	if err != nil {
		s.logger.Error("failed to get pending orders", zap.Error(err))
		return
	}

	if len(orders) == 0 {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobs := make(chan storage.Order, len(orders))
	results := make(chan storage.Order, len(orders))

	for range s.workerPoolSize {
		go s.worker(ctx, cancel, jobs, results)
	}

	for _, order := range orders {
		jobs <- order
	}
	close(jobs)

	for order := range results {
		if err := s.orderStorage.ProcessOrder(&order); err != nil {
			s.logger.Error("failed to update order status",
				zap.String("order", order.Number),
				zap.Error(err),
			)
			continue
		}
		s.logger.Info("order processed",
			zap.String("order", order.Number),
			zap.String("status", string(order.Status)),
		)
	}
}

func (s *Scheduler) worker(
	ctx context.Context,
	cancel context.CancelFunc,
	jobs <-chan storage.Order,
	results chan<- storage.Order,
) {
	for order := range jobs {
		select {
		case <-ctx.Done():
			return
		default:
			result, processedOrder := s.checkOrder(ctx, &order)
			switch result.status {
			case Success:
				results <- *processedOrder
			case Busy:
				s.logger.Warn("accrual system busy, retrying after timeout",
					zap.Int("timeout_sec", result.timeout),
				)
				s.setAccrualInterval(time.Duration(result.timeout) * time.Second)
				cancel()
				return
			case Fail:
				s.logger.Error("failed to check order status",
					zap.String("order", order.Number),
				)
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
		s.logger.Error("failed to create request",
			zap.String("order", order.Number),
			zap.Error(err),
		)
		return checkResult{status: Fail}, nil
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s.logger.Error("failed to get order status",
			zap.String("order", order.Number),
			zap.Error(err),
		)
		return checkResult{status: Fail}, nil
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			s.logger.Error("failed to close response body")
		}
	}()

	switch resp.StatusCode {
	case http.StatusOK:
		var result struct {
			Order   string              `json:"order"`
			Status  storage.OrderStatus `json:"status"`
			Accrual float64             `json:"accrual,omitempty"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			s.logger.Error("failed to decode response",
				zap.String("order", order.Number),
				zap.Error(err),
			)
			return checkResult{status: Fail}, nil
		}

		order.Status = result.Status
		order.Accrual = result.Accrual
		return checkResult{status: Success}, order

	case http.StatusNoContent:
		order.Status = storage.StatusInvalid
		return checkResult{status: Success}, order

	case http.StatusTooManyRequests:
		timeout := 60
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			if sec, err := strconv.Atoi(retryAfter); err == nil {
				timeout = sec
			}
		}
		return checkResult{status: Busy, timeout: timeout}, nil

	default:
		s.logger.Error("unexpected status from accrual system",
			zap.String("order", order.Number),
			zap.Int("status_code", resp.StatusCode),
		)
		return checkResult{status: Fail}, nil
	}
}

func (s *Scheduler) getAccrualInterval() time.Duration {
	s.intervalMu.RLock()
	defer s.intervalMu.RUnlock()
	return s.accrualInterval
}

func (s *Scheduler) setAccrualInterval(newInterval time.Duration) {
	s.intervalMu.Lock()
	defer s.intervalMu.Unlock()
	s.accrualInterval = newInterval
}
