package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"CB_auto/internal/config"

	"github.com/ozontech/allure-go/pkg/framework/provider"
	redis "github.com/redis/go-redis/v9"
)

var ErrKeyNotFound = errors.New("redis: key not found")

type RedisClient struct {
	client        *redis.Client
	retryAttempts int
	retryDelay    time.Duration
}

func NewRedisClient(t provider.T, cfg *config.RedisConfig) *RedisClient {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  time.Duration(cfg.DialTimeout) * time.Second,
		ReadTimeout:  time.Duration(cfg.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.WriteTimeout) * time.Second,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("Ошибка подключения к Redis: %v", err)
	}

	return &RedisClient{
		client:        client,
		retryAttempts: cfg.RetryAttempts,
		retryDelay:    time.Duration(cfg.RetryDelay) * time.Second,
	}
}

func (r *RedisClient) Get(key string) (string, error) {
	ctx := context.Background()
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			log.Printf("Key %s not found in Redis", key)
			return "", ErrKeyNotFound
		}
		return "", fmt.Errorf("redis get failed: %v", err)
	}

	log.Printf("Redis value for key %s: %s", key, val)
	return val, nil
}

func (r *RedisClient) GetWithRetry(t provider.T, key string) WalletsMap {
	var lastErr error
	delay := r.retryDelay
	var result WalletsMap

	for i := 0; i < r.retryAttempts; i++ {
		value, err := r.Get(key)
		if err == nil && value != "" {
			if err := json.Unmarshal([]byte(value), &result); err != nil {
				t.Fatalf("Failed to unmarshal Redis value: %v", err)
			}
			return result
		}
		lastErr = err
		log.Printf("Attempt %d: Redis key %s not found or empty, retrying in %v...", i+1, key, delay)
		time.Sleep(delay)
		delay *= 2
	}
	t.Fatalf("Не удалось получить значение из Redis после %d попыток: %v", r.retryAttempts, lastErr)
	return nil
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}
