package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// HistoryEntry is a single price observation row returned from the DB.
type HistoryEntry struct {
	Value      float64
	RecordedAt time.Time
}

// Store is the interface for all database operations used by this service.
// Exposing it here (rather than in the handler package) lets main.go wire a
// concrete implementation without creating an import cycle.
type Store interface {
	// InsertRate persists a new price observation for the given currency.
	InsertRate(ctx context.Context, currency string, value float64, recordedAt time.Time) error

	// GetHistory returns all observations for currency in [from, to).
	GetHistory(ctx context.Context, currency string, from, to time.Time) ([]HistoryEntry, error)

	// Close releases the underlying connection pool.
	Close()
}

// DBStore is the live PostgreSQL implementation of Store.
type DBStore struct {
	pool *pgxpool.Pool
}

// New opens (and validates) a connection pool against the given connString.
// The caller must call Close when done.
func New(ctx context.Context, connString string) (*DBStore, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("db.New: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db.New ping: %w", err)
	}
	return &DBStore{pool: pool}, nil
}

// RunMigrations applies all pending goose migrations against connString.
// Safe to call on every startup — goose tracks applied versions in the
// schema_migrations table and is a no-op when already up-to-date.
func RunMigrations(connString string) error {
	db, err := sql.Open("pgx", connString)
	if err != nil {
		return fmt.Errorf("db.RunMigrations open: %w", err)
	}
	defer func() { _ = db.Close() }()

	goose.SetBaseFS(migrationFiles)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("db.RunMigrations dialect: %w", err)
	}
	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("db.RunMigrations: %w", err)
	}
	return nil
}

// InsertRate writes a single price observation to the DB.
func (d *DBStore) InsertRate(ctx context.Context, currency string, value float64, recordedAt time.Time) error {
	_, err := d.pool.Exec(ctx,
		`INSERT INTO prices_history (currency, value, recorded_at) VALUES ($1, $2, $3)`,
		currency, value, recordedAt,
	)
	if err != nil {
		return fmt.Errorf("db.InsertRate: %w", err)
	}
	return nil
}

// GetHistory returns all price observations for currency where
// recorded_at >= from AND recorded_at < to, ordered ascending.
func (d *DBStore) GetHistory(ctx context.Context, currency string, from, to time.Time) ([]HistoryEntry, error) {
	rows, err := d.pool.Query(ctx,
		`SELECT value, recorded_at
		   FROM prices_history
		  WHERE currency = $1
		    AND recorded_at >= $2
		    AND recorded_at <  $3
		  ORDER BY recorded_at ASC`,
		currency, from, to,
	)
	if err != nil {
		return nil, fmt.Errorf("db.GetHistory query: %w", err)
	}
	defer rows.Close()

	var entries []HistoryEntry
	for rows.Next() {
		var e HistoryEntry
		if err := rows.Scan(&e.Value, &e.RecordedAt); err != nil {
			return nil, fmt.Errorf("db.GetHistory scan: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db.GetHistory rows: %w", err)
	}
	return entries, nil
}

// Close releases the connection pool.
func (d *DBStore) Close() {
	d.pool.Close()
}
