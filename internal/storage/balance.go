package storage

import (
	"database/sql"
	"time"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

type Balance struct {
	UserID    int
	Current   float64
	Withdrawn float64
}

type Withdrawal struct {
	ProcessedAt time.Time
	OrderNumber string
	ID          int
	UserID      int
	Sum         float64
}

type BalanceStorage interface {
	GetBalance(userID int) (Balance, error)
	Withdraw(userID int, withdrawal Withdrawal) error
	GetWithdrawals(userID int) ([]Withdrawal, error)
}

type BalanceStoragePostgres struct {
	logger *zap.Logger
	db     *sql.DB
}

func NewBalanceStorage(db *sql.DB, logger *zap.Logger) (*BalanceStoragePostgres, error) {
	return &BalanceStoragePostgres{
		logger: logger,
		db:     db,
	}, nil
}

func (s *BalanceStoragePostgres) GetBalance(userID int) (Balance, error) {
	var balance Balance
	err := s.db.QueryRow(
		"SELECT user_id, current, withdrawn FROM balances WHERE user_id = $1",
		userID,
	).Scan(&balance.UserID, &balance.Current, &balance.Withdrawn)
	if err != nil {
		s.logger.Error("failed to get balance", zap.Error(err))
		return Balance{}, err
	}
	return balance, nil
}

func (s *BalanceStoragePostgres) Withdraw(userID int, withdrawal Withdrawal) error {
	tx, err := s.db.Begin()
	if err != nil {
		s.logger.Error("failed to begin transaction", zap.Error(err))
		return err
	}

	_, err = tx.Exec(
		"UPDATE balances SET current = current - $1, withdrawn = withdrawn + $1 WHERE user_id = $2",
		withdrawal.Sum, userID,
	)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			s.logger.Error("failed to rollback transaction", zap.Error(err))
		}
		s.logger.Error("failed to update balance", zap.Error(err))
		return err
	}

	_, err = tx.Exec(
		"INSERT INTO withdrawals (user_id, order_number, sum, processed_at) VALUES ($1, $2, $3, $4)",
		withdrawal.UserID, withdrawal.OrderNumber, withdrawal.Sum, withdrawal.ProcessedAt,
	)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			s.logger.Error("failed to rollback transaction", zap.Error(err))
		}
		s.logger.Error("failed to insert withdrawal", zap.Error(err))
		return err
	}

	if err := tx.Commit(); err != nil {
		s.logger.Error("failed to commit transaction", zap.Error(err))
		return err
	}

	return nil
}

func (s *BalanceStoragePostgres) GetWithdrawals(userID int) ([]Withdrawal, error) {
	rows, err := s.db.Query(
		"SELECT id, user_id, order_number, sum, processed_at FROM withdrawals WHERE user_id = $1 ORDER BY processed_at DESC",
		userID,
	)
	if err != nil {
		s.logger.Error("failed to get withdrawals", zap.Error(err))
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Error("failed to close rows", zap.Error(err))
		}
	}()

	var withdrawals []Withdrawal
	for rows.Next() {
		var withdrawal Withdrawal
		if err := rows.Scan(
			&withdrawal.ID, &withdrawal.UserID, &withdrawal.OrderNumber, &withdrawal.Sum, &withdrawal.ProcessedAt,
		); err != nil {
			s.logger.Error("failed to scan withdrawal", zap.Error(err))
			return nil, err
		}
		withdrawals = append(withdrawals, withdrawal)
	}
	if err := rows.Err(); err != nil {
		s.logger.Error("failed to iterate over rows", zap.Error(err))
		return nil, err
	}
	return withdrawals, nil
}
