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
	Brokers     string        `json:"brokers"`
	BrandTopic  string        `json:"brand_topic"`
	PlayerTopic string        `json:"player_topic"`
	Timeout     time.Duration `json:"timeout"`
}

type NatsConfig struct {
	Hosts         string        `json:"hosts"`
	ReconnectWait time.Duration `json:"reconnect_wait"`
	MaxReconnects int           `json:"max_reconnects"`
	Timeout       time.Duration `json:"timeout"`
	StreamTimeout time.Duration `json:"stream_timeout"`
	StreamPrefix  string        `json:"stream_prefix"`
}

type NodeConfig struct {
	GroupID         string `json:"group_id"`
	ProjectID       string `json:"project_id"`
	DefaultCountry  string `json:"default_country"`
	DefaultCurrency string `json:"default_currency"`
}

type HTTPConfig struct {
	CapURL      string `json:"cap_url"`
	PublicURL   string `json:"public_url"`
	Timeout     int    `json:"timeout"`
	CapUsername string `json:"cap_username"`
	CapPassword string `json:"cap_password"`
}

type RedisConfig struct {
	Addr          string        `json:"addr"`
	Password      string        `json:"password"`
	DB            int           `json:"db"`
	DialTimeout   time.Duration `json:"dial_timeout"`
	ReadTimeout   time.Duration `json:"read_timeout"`
	WriteTimeout  time.Duration `json:"write_timeout"`
	RetryAttempts int           `json:"retryAttempts"`
	RetryDelay    time.Duration `json:"retryDelay"`
}

type Config struct {
	HTTP  HTTPConfig  `json:"http"`
	Node  NodeConfig  `json:"node"`
	MySQL MySQLConfig `json:"mysql"`
	Kafka KafkaConfig `json:"kafka"`
	Nats  NatsConfig  `json:"nats"`
	Redis RedisConfig `json:"redis"`
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
