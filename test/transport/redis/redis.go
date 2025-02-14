package redis

import (
	"context"
	"fmt"
	"time"

	"CB_auto/test/config"

	"github.com/ozontech/allure-go/pkg/framework/provider"
	redis "github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client        *redis.Client
	retryAttempts int
	retryDelay    time.Duration
}

func NewRedisClient(cfg *config.RedisConfig) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  cfg.DialTimeout * time.Second,
		ReadTimeout:  cfg.ReadTimeout * time.Second,
		WriteTimeout: cfg.WriteTimeout * time.Second,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %v", err)
	}

	return &RedisClient{
		client:        client,
		retryAttempts: cfg.RetryAttempts,
		retryDelay:    cfg.RetryDelay * time.Second,
	}, nil
}

func (r *RedisClient) Get(key string) (string, error) {
	ctx := context.Background()
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			fmt.Printf("Key %s not found in Redis\n", key)
			return "", nil
		}
		return "", fmt.Errorf("redis get failed: %v", err)
	}

	fmt.Printf("Redis value for key %s: %s\n", key, val)
	return val, nil
}

func (r *RedisClient) GetWithRetry(t provider.T, key string) string {
	var lastErr error
	for i := 0; i < r.retryAttempts; i++ {
		value, err := r.Get(key)
		if err == nil && value != "" {
			return value
		}
		lastErr = err
		fmt.Printf("Attempt %d: Redis key %s not found, retrying in %v...\n", i+1, key, r.retryDelay)
		time.Sleep(r.retryDelay)
	}
	t.Fatalf("Не удалось получить значение из Redis после %d попыток: %v", r.retryAttempts, lastErr)
	return ""
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}
