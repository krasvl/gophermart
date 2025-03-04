package storage

import (
	"database/sql"
	"time"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

type User struct {
	CreatedAt time.Time
	Login     string
	Password  string
	ID        int
}

type UserStorage interface {
	AddUser(user User) (int, error)
	GetUser(login string) (User, error)
}

type UserStoragePostgres struct {
	logger *zap.Logger
	db     *sql.DB
}

func NewUserStorage(db *sql.DB, logger *zap.Logger) (*UserStoragePostgres, error) {
	return &UserStoragePostgres{
		logger: logger,
		db:     db,
	}, nil
}

func (s *UserStoragePostgres) AddUser(user User) (int, error) {
	var userID int
	err := s.db.QueryRow(
		"INSERT INTO users (login, password) VALUES ($1, $2) RETURNING id",
		user.Login, user.Password,
	).Scan(&userID)
	if err != nil {
		s.logger.Error("failed to add user", zap.Error(err))
		return 0, err
	}
	return userID, nil
}

func (s *UserStoragePostgres) GetUser(login string) (User, error) {
	var user User
	err := s.db.QueryRow("SELECT id, login, password, created_at FROM users WHERE login = $1", login).
		Scan(&user.ID, &user.Login, &user.Password, &user.CreatedAt)
	if err != nil {
		s.logger.Error("failed to get user", zap.Error(err))
		return User{}, err
	}
	return user, nil
}
