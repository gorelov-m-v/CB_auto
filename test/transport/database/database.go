package database

import (
	"CB_auto/test/config"
	"database/sql"
	"fmt"
	"time"
)

const (
	dialect = "mysql"
)

func InitDB(cfg *config.MySQLConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)
	db, err := sql.Open(dialect, dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql connection: %w", err)
	}

	connMaxLifetime, err := time.ParseDuration(cfg.ConnMaxLifetime)
	if err != nil {
		return nil, fmt.Errorf("parse connMaxLifetime: %w", err)
	}

	db.SetConnMaxLifetime(connMaxLifetime)
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)

	return db, nil
}

func CloseDB(db *sql.DB) error {
	if db != nil {
		return db.Close()
	}
	return nil
}
