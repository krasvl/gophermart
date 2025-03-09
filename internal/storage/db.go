package storage

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

func NewDB(database string) (*sql.DB, error) {
	if err := runMigrations(database); err != nil {
		return nil, fmt.Errorf("failed to run DB migrations: %w", err)
	}
	db, err := sql.Open("postgres", database)
	if err != nil {
		return nil, fmt.Errorf("failed to create db: %w", err)
	}
	return db, nil
}

//go:embed migrations/*.sql
var migrationsDir embed.FS

func runMigrations(database string) error {
	d, err := iofs.New(migrationsDir, "migrations")
	if err != nil {
		return fmt.Errorf("failed to return an iofs driver: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, database)
	if err != nil {
		return fmt.Errorf("failed to get a new migrate instance: %w", err)
	}
	if err := m.Up(); err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("failed to apply migrations to the DB: %w", err)
		}
	}
	return nil
}
