package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type MySQLConfig struct {
	User            string `json:"user"`
	Password        string `json:"password"`
	DBName          string `json:"dbName"`
	Host            string `json:"host"`
	Port            string `json:"port"`
	ConnMaxLifetime string `json:"connMaxLifetime"`
	MaxOpenConns    int    `json:"maxOpenConns"`
	MaxIdleConns    int    `json:"maxIdleConns"`
}

type Config struct {
	BaseURL        string      `json:"baseURL"`
	RequestTimeout int         `json:"requestTimeout"`
	UserName       string      `json:"username"`
	Password       string      `json:"password"`
	GroupID        string      `json:"groupID"`
	ProjectID      string      `json:"projectID"`
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
