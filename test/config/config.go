package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type MySQLConfig struct {
	DriverName      string        `json:"driverName"`
	DSN             string        `json:"dsn"`
	PingTimeout     time.Duration `json:"pingTimeout"`
	ConnMaxLifetime time.Duration `json:"connMaxLifetime"`
	ConnMaxIdleTime time.Duration `json:"connMaxIdleTime"`
	MaxOpenConns    int           `json:"maxOpenConns"`
	MaxIdleConns    int           `json:"maxIdleConns"`
}

type Config struct {
	BaseURL        string      `json:"baseURL"`
	RequestTimeout int         `json:"requestTimeout"`
	UserName       string      `json:"username"`
	Password       string      `json:"password"`
	GroupID        string      `json:"groupId"`   // Исправлено на "groupId"
	ProjectID      string      `json:"projectId"` // Исправлено на "projectId"
	MySQL          MySQLConfig `json:"mysql"`
}

func ReadConfig() (*Config, error) {
	file, err := os.Open("config.json")
	if err != nil {
		return nil, fmt.Errorf("open config file failed: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	config := &Config{}
	err = decoder.Decode(config)
	if err != nil {
		return nil, fmt.Errorf("config decode failed: %w", err)
	}

	return config, nil
}
