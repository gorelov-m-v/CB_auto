package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/ozontech/allure-go/pkg/framework/provider"
)

type MySQLCommonConfig struct {
	DriverName     string `json:"driver_name"`
	User           string `json:"user"`
	Password       string `json:"password"`
	Host           string `json:"host"`
	Port           int    `json:"port"`
	DatabasePrefix string `json:"database_prefix"`
}

type MySQLConfig struct {
	Common          MySQLCommonConfig `json:"common"`
	DatabaseCore    string            `json:"database_core"`
	DatabaseWallet  string            `json:"database_wallet"`
	DatabaseBonus   string            `json:"database_bonus"`
	PingTimeout     time.Duration     `json:"ping_timeout"`
	ConnMaxLifetime time.Duration     `json:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration     `json:"conn_max_idle_time"`
	MaxOpenConns    int               `json:"max_open_conns"`
	MaxIdleConns    int               `json:"max_idle_conns"`
	RetryAttempts   int               `json:"retry_attempts"`
	RetryDelay      time.Duration     `json:"retry_delay"`
}

type KafkaConfig struct {
	Brokers     string        `json:"brokers"`
	Timeout     time.Duration `json:"timeout"`
	TopicPrefix string        `json:"topic_prefix"`
	BufferSize  int           `json:"buffer_size"`
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
	PlayerAddr    string        `json:"player_addr"`
	WalletAddr    string        `json:"wallet_addr"`
	Password      string        `json:"password"`
	PlayerDB      int           `json:"player_db"`
	WalletDB      int           `json:"wallet_db"`
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
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("Ошибка определения пути к исходному файлу конфигурации")
	}
	projectRoot := filepath.Join(filepath.Dir(currentFile), "..", "..")

	allureOutputPath := filepath.Join(projectRoot, "allure-results")
	if err := os.Setenv("ALLURE_OUTPUT_PATH", allureOutputPath); err != nil {
		t.Fatalf("Ошибка установки пути для отчетов Allure: %v", err)
	}

	configPath := filepath.Join(projectRoot, "config.json")
	configFile, err := os.Open(configPath)
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
