package storage

import (
	"database/sql"
	"time"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

type OrderStatus string

const (
	StatusNew        OrderStatus = "NEW"
	StatusProcessing OrderStatus = "PROCESSING"
	StatusInvalid    OrderStatus = "INVALID"
	StatusProcessed  OrderStatus = "PROCESSED"
)

type Order struct {
	UploadedAt time.Time
	Number     string
	Status     OrderStatus
	ID         int
	UserID     int
	Accrual    float64
}

type OrderStorage interface {
	AddOrder(order *Order) error
	GetOrders(userID int) ([]Order, error)
	GetPendingOrders() ([]Order, error)
	ProcessOrder(order *Order) error
}

type OrderStoragePostgres struct {
	logger *zap.Logger
	db     *sql.DB
}

func NewOrderStorage(db *sql.DB, logger *zap.Logger) (*OrderStoragePostgres, error) {
	return &OrderStoragePostgres{
		logger: logger,
		db:     db,
	}, nil
}

func (s *OrderStoragePostgres) AddOrder(order *Order) error {
	_, err := s.db.Exec(
		"INSERT INTO orders (user_id, number, status, accrual) VALUES ($1, $2, $3, $4)",
		order.UserID, order.Number, order.Status, order.Accrual,
	)
	if err != nil {
		s.logger.Error("failed to add order", zap.Error(err))
		return err
	}
	return nil
}

func (s *OrderStoragePostgres) GetOrders(userID int) ([]Order, error) {
	rows, err := s.db.Query(
		"SELECT id, user_id, number, status, accrual, uploaded_at FROM orders WHERE user_id = $1 ORDER BY uploaded_at DESC",
		userID,
	)
	if err != nil {
		s.logger.Error("failed to get orders", zap.Error(err))
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Error("failed to close rows", zap.Error(err))
		}
	}()

	var orders []Order
	for rows.Next() {
		var order Order
		if err := rows.Scan(&order.ID, &order.UserID, &order.Number, &order.Status, &order.Accrual,
			&order.UploadedAt); err != nil {
			s.logger.Error("failed to scan order", zap.Error(err))
			return nil, err
		}
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		s.logger.Error("rows error", zap.Error(err))
		return nil, err
	}
	return orders, nil
}

func (s *OrderStoragePostgres) GetPendingOrders() ([]Order, error) {
	rows, err := s.db.Query(
		"SELECT id, user_id, number, status, uploaded_at FROM orders WHERE status IN ('NEW', 'PROCESSING')",
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Error("failed to close rows", zap.Error(err))
		}
	}()

	var orders []Order
	for rows.Next() {
		var order Order
		if err := rows.Scan(&order.ID, &order.UserID, &order.Number, &order.Status, &order.UploadedAt); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return orders, nil
}

func (s *OrderStoragePostgres) ProcessOrder(order *Order) error {
	tx, err := s.db.Begin()
	if err != nil {
		s.logger.Error("failed to begin transaction", zap.Error(err))
		return err
	}

	_, err = tx.Exec("UPDATE orders SET status = $1, accrual = $2 WHERE id = $3", order.Status, order.Accrual, order.ID)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			s.logger.Error("failed to rollback transaction", zap.Error(err))
		}
		s.logger.Error("failed to update status", zap.Error(err))
		return err
	}

	if order.Status == StatusProcessed {
		_, err = tx.Exec("UPDATE balances SET current = current + $1 WHERE user_id = $2", order.Accrual, order.UserID)
		if err != nil {
			if err := tx.Rollback(); err != nil {
				s.logger.Error("failed to rollback transaction", zap.Error(err))
			}
			s.logger.Error("failed to update balance", zap.Error(err))
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		s.logger.Error("failed to commit transaction", zap.Error(err))
		return err
	}

	return nil
}
