package scheduler

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/krasvl/market/internal/storage"
	"go.uber.org/zap"
)

func GetConfiguredScheduler(databaseDefault, accrualAddrDefault string) (*Scheduler, error) {
	database := flag.String("d", databaseDefault, "database-dsn")
	accrualAddr := flag.String("r", accrualAddrDefault, "acccural-address")

	flag.Parse()

	if value, ok := os.LookupEnv("DATABASE_URI"); ok && value != "" {
		database = &value
	}
	if value, ok := os.LookupEnv("ACCRUAL_SYSTEM_ADDRESS"); ok && value != "" {
		accrualAddr = &value
	}

	if !strings.HasPrefix(*accrualAddr, "http://") && !strings.HasPrefix(*accrualAddr, "https://") {
		*accrualAddr = "http://" + *accrualAddr
	}

	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("cant create logger: %w", err)
	}

	db, err := storage.NewDB(*database)
	if err != nil {
		return nil, fmt.Errorf("cant open database: %w", err)
	}

	orderStorage, err := storage.NewOrderStorage(db, logger)
	if err != nil {
		return nil, fmt.Errorf("cant create order storage: %w", err)
	}

	logger.Info("scheduler created:",
		zap.String("accural", *accrualAddr),
		zap.String("database", *database),
	)

	return NewScheduler(logger, orderStorage, *accrualAddr), nil
}
