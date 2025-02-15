package redis

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"CB_auto/test/config"

	"github.com/ozontech/allure-go/pkg/framework/provider"
	redis "github.com/redis/go-redis/v9"
)

var ErrKeyNotFound = errors.New("redis: key not found")

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
		if errors.Is(err, redis.Nil) {
			log.Printf("Key %s not found in Redis", key)
			return "", ErrKeyNotFound
		}
		return "", fmt.Errorf("redis get failed: %v", err)
	}

	log.Printf("Redis value for key %s: %s", key, val)
	return val, nil
}

func (r *RedisClient) GetWithRetry(t provider.T, key string) string {
	var lastErr error
	delay := r.retryDelay
	for i := 0; i < r.retryAttempts; i++ {
		value, err := r.Get(key)
		if err == nil && value != "" {
			return value
		}
		lastErr = err
		log.Printf("Attempt %d: Redis key %s not found or empty, retrying in %v...", i+1, key, delay)
		time.Sleep(delay)
		delay *= 2
	}
	t.Fatalf("Не удалось получить значение из Redis после %d попыток: %v", r.retryAttempts, lastErr)
	return ""
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}
