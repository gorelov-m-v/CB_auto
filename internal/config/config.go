package config

import (
	"encoding/json"
	"os"
	"time"

	"github.com/ozontech/allure-go/pkg/framework/provider"
)

type MySQLConfig struct {
	DriverName      string        `json:"driver_name"`
	DSNCore         string        `json:"dsn_core"`
	DSNWallet       string        `json:"dsn_wallet"`
	PingTimeout     time.Duration `json:"ping_timeout"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time"`
	MaxOpenConns    int           `json:"max_open_conns"`
	MaxIdleConns    int           `json:"max_idle_conns"`
	RetryAttempts   int           `json:"retry_attempts"`
	RetryDelay      time.Duration `json:"retry_delay"`
}

type KafkaConfig struct {
	Brokers string        `json:"brokers"`
	Timeout time.Duration `json:"timeout"`
}

type NatsConfig struct {
	Hosts         string        `json:"hosts"`
	StreamPrefix  string        `json:"stream_prefix"`
	ReconnectWait time.Duration `json:"reconnect_wait"`
	MaxReconnects int           `json:"max_reconnects"`
	Timeout       time.Duration `json:"timeout"`
	StreamTimeout time.Duration `json:"stream_timeout"`
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

func ReadConfig(t provider.T) *Config {
	configFile, err := os.Open("../../config.json")
	if err != nil {
		t.Fatalf("Ошибка открытия файла конфигурации: %v", err)
	}
	defer configFile.Close()

	var config Config
	decoder := json.NewDecoder(configFile)
	if err := decoder.Decode(&config); err != nil {
		t.Fatalf("Ошибка декодирования конфигурации: %v", err)
	}

	config.MySQL.PingTimeout *= time.Nanosecond
	config.MySQL.ConnMaxLifetime *= time.Nanosecond
	config.MySQL.ConnMaxIdleTime *= time.Nanosecond
	config.Kafka.Timeout *= time.Second

	return &config
}
