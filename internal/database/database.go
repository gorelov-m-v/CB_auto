package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"gitlab.b2bdev.pro/backend/go-packages/log"
)

type Config struct {
	DriverName string
	DSN        string

	PingTimeout     time.Duration
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	MaxOpenConns    int
	MaxIdleConns    int
}

type Connector struct {
	db *sqlx.DB
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
			log.Errorwe(ctx, "failed to close db connection", err)
		}
		return Connector{}, fmt.Errorf("ping db: %w", err)
	}

	return Connector{db: db}, nil
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
