package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver

	"github.com/dimasim/integrative-prog/internal/config"
)

// NewPostgresDB membuat koneksi ke PostgreSQL dengan connection pool yang sudah dikonfigurasi.
func NewPostgresDB(cfg config.DatabaseConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("database: open connection failed: %w", err)
	}

	// Konfigurasi connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)

	// Verifikasi koneksi benar-benar berhasil
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("database: ping failed (apakah PostgreSQL di VM menyala?): %w", err)
	}

	return db, nil
}
