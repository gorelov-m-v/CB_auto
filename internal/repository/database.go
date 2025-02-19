package repository

import (
	"CB_auto/internal/config"
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
)

func ExecuteWithRetry(ctx context.Context, cfg *config.MySQLConfig, operation func(context.Context) error) error {
	delayInSeconds := time.Duration(cfg.RetryDelay) * time.Second
	log.Printf("Starting database operation with %d attempts and %v delay", cfg.RetryAttempts, delayInSeconds)
	var lastErr error
	for attempt := 0; attempt < cfg.RetryAttempts; attempt++ {
		if err := operation(ctx); err == nil {
			log.Printf("Database operation succeeded on attempt %d", attempt+1)
			return nil
		} else if err == sql.ErrNoRows {
			lastErr = err
			log.Printf("No rows found, attempt %d/%d", attempt+1, cfg.RetryAttempts)
		} else {
			lastErr = err
			log.Printf("Database operation failed, attempt %d/%d: %v", attempt+1, cfg.RetryAttempts, err)
		}

		if attempt < cfg.RetryAttempts-1 {
			log.Printf("Waiting %v before next attempt", delayInSeconds)
			time.Sleep(delayInSeconds)
		}
	}
	return fmt.Errorf("operation failed after %d attempts: %w", cfg.RetryAttempts, lastErr)
}

type Connector struct {
	db *sqlx.DB
}

func NewConnector(db *sqlx.DB) Connector {
	return Connector{db: db}
}

type DSNType string

const (
	Core   DSNType = "core"
	Wallet DSNType = "wallet"
)

func OpenConnector(config *config.MySQLConfig, dsnType DSNType) (Connector, error) {
	dsn := config.DSNCore
	if dsnType == Wallet {
		dsn = config.DSNWallet
	}

	db, err := sqlx.Open(config.DriverName, dsn)
	if err != nil {
		return Connector{}, fmt.Errorf("open database: %w", err)
	}

	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	db.SetMaxOpenConns(config.MaxOpenConns)
	if config.MaxIdleConns != -1 {
		db.SetMaxIdleConns(config.MaxIdleConns)
	}

	if err := pingDB(context.Background(), db, config.PingTimeout); err != nil {
		if err := db.Close(); err != nil {
			log.Printf("failed to close db connection: %v", err)
		}
		return Connector{}, fmt.Errorf("ping db: %w", err)
	}

	return Connector{
		db: db,
	}, nil
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

func (c Connector) DB() *sql.DB {
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
