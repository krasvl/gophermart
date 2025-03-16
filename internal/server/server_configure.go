package server

import (
	"flag"
	"fmt"
	"os"

	"github.com/krasvl/market/internal/storage"
	"go.uber.org/zap"
)

func GetConfiguredServer(databaseDefault, addrDefault, secretDefault string) (*Server, error) {
	database := flag.String("d", databaseDefault, "database-dsn")
	addr := flag.String("a", addrDefault, "address")
	sec := flag.String("s", secretDefault, "secret")

	flag.Parse()

	if value, ok := os.LookupEnv("DATABASE_URI"); ok && value != "" {
		database = &value
	}
	if value, ok := os.LookupEnv("RUN_ADDRESS"); ok && value != "" {
		addr = &value
	}
	if value, ok := os.LookupEnv("SECRET"); ok && value != "" {
		sec = &value
	}

	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("cant create logger: %w", err)
	}

	db, err := storage.NewDB(*database)
	if err != nil {
		return nil, fmt.Errorf("cant open database: %w", err)
	}

	userStorage, err := storage.NewUserStorage(db, logger)
	if err != nil {
		return nil, fmt.Errorf("cant create user storage: %w", err)
	}

	orderStorage, err := storage.NewOrderStorage(db, logger)
	if err != nil {
		return nil, fmt.Errorf("cant create order storage: %w", err)
	}

	balanceStorage, err := storage.NewBalanceStorage(db, logger)
	if err != nil {
		return nil, fmt.Errorf("cant create balance storage: %w", err)
	}

	logger.Info("server created:",
		zap.String("address", *addr),
		zap.String("database", *database),
	)

	return NewServer(*addr, userStorage, orderStorage, balanceStorage, logger, *sec), nil
}
