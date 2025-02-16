package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db            *sql.DB
	retryAttempts int
	retryDelay    time.Duration
}

func NewRepository(db *sql.DB) Repository {
	return Repository{
		db:            db,
		retryAttempts: 3,           // значение по умолчанию
		retryDelay:    time.Second, // значение по умолчанию
	}
}

func (r *Repository) ExecuteWithRetry(ctx context.Context, operation func(context.Context) error) error {
	var lastErr error
	for attempt := 0; attempt < r.retryAttempts; attempt++ {
		if err := operation(ctx); err == nil {
			return nil
		} else if err == sql.ErrNoRows {
			lastErr = err
			log.Printf("No rows found, attempt %d/%d", attempt+1, r.retryAttempts)
		} else {
			lastErr = err
			log.Printf("Database operation failed, attempt %d/%d: %v", attempt+1, r.retryAttempts, err)
		}

		if attempt < r.retryAttempts-1 {
			time.Sleep(r.retryDelay)
		}
	}
	return fmt.Errorf("operation failed after %d attempts: %w", r.retryAttempts, lastErr)
}

func (r *Repository) DB() *sql.DB {
	return r.db
}

type Config struct {
	DriverName string
	DSN        string

	PingTimeout     time.Duration
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	MaxOpenConns    int
	MaxIdleConns    int
	RetryAttempts   int
	RetryDelay      time.Duration
}

type Connector struct {
	db *sqlx.DB
	DB *sql.DB
}

func NewConnector(db *sqlx.DB) Connector {
	return Connector{db: db}
}

func OpenConnector(ctx context.Context, config Config) (Connector, error) {
	db, err := sqlx.Open(config.DriverName, config.DSN)
	if err != nil {
		return Connector{}, fmt.Errorf("open database: %w", err)
	}

	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	db.SetMaxOpenConns(config.MaxOpenConns)
	if config.MaxIdleConns != -1 {
		db.SetMaxIdleConns(config.MaxIdleConns)
	}

	if err := pingDB(ctx, db, config.PingTimeout); err != nil {
		if err := db.Close(); err != nil {
			log.Printf("failed to close db connection: %v", err)
		}
		return Connector{}, fmt.Errorf("ping db: %w", err)
	}

	return Connector{db: db, DB: db.DB}, nil
}

func (c Connector) QueryContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error) {
	return c.db.QueryxContext(ctx, query, args...)
}

func (c Connector) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row {
	return c.db.QueryRowxContext(ctx, query, args...)
}

func (c Connector) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return c.db.ExecContext(ctx, query, args...)
}

func (c Connector) Close() error {
	return c.db.Close()
}

func (c Connector) PingContext(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

func (c Connector) SqlxDB() *sqlx.DB {
	return c.db
}

func (c Connector) SqlDB() *sql.DB {
	return c.db.DB
}

func (c Connector) Stats() sql.DBStats {
	return c.db.DB.Stats()
}

func pingDB(ctx context.Context, db *sqlx.DB, timeout time.Duration) error {
	if timeout > 0 {
		var ctxCancel context.CancelFunc
		ctx, ctxCancel = context.WithTimeout(ctx, timeout)
		defer ctxCancel()
	}
	return db.PingContext(ctx)
}
