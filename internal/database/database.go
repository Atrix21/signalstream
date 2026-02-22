package database

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

type DB struct {
	*sql.DB
}

func Connect(url string) (*DB, error) {
	db, err := sql.Open("pgx", url)
	if err != nil {
		return nil, fmt.Errorf("open database connection: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	slog.Info("database connected")
	return &DB{db}, nil
}

func (db *DB) Migrate() error {
	migrations, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations directory: %w", err)
	}

	for _, entry := range migrations {
		if entry.IsDir() {
			continue
		}
		content, err := migrationFS.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}

		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("execute migration %s: %w", entry.Name(), err)
		}
		slog.Info("migration applied", "file", entry.Name())
	}

	return nil
}
