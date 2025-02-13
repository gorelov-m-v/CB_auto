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

type KafkaConfig struct {
	Brokers string        `json:"brokers"`
	Topic   string        `json:"topic"`
	Timeout time.Duration `json:"timeout"`
}

type Config struct {
	BaseURL        string      `json:"baseURL"`
	RequestTimeout int         `json:"requestTimeout"`
	UserName       string      `json:"username"`
	Password       string      `json:"password"`
	GroupID        string      `json:"groupId"`
	ProjectID      string      `json:"projectId"`
	MySQL          MySQLConfig `json:"mysql"`
	Kafka          KafkaConfig `json:"kafka"`
}

func (k *KafkaConfig) GetTimeout() time.Duration {
	return k.Timeout
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

	config.MySQL.PingTimeout *= time.Nanosecond
	config.MySQL.ConnMaxLifetime *= time.Nanosecond
	config.MySQL.ConnMaxIdleTime *= time.Nanosecond
	config.Kafka.Timeout *= time.Second

	return config, nil
}
