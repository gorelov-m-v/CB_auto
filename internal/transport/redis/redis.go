package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"CB_auto/internal/config"
	"CB_auto/pkg/utils"

	"github.com/ozontech/allure-go/pkg/allure"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	redis "github.com/redis/go-redis/v9"
)

var ErrKeyNotFound = errors.New("redis: key not found")

type RedisClientType string

const (
	PlayerClient RedisClientType = "player"
	WalletClient RedisClientType = "wallet"
)

type RedisClient struct {
	client        *redis.Client
	retryAttempts int
	retryDelay    time.Duration
}

func NewRedisClient(t provider.T, cfg *config.RedisConfig, clientType RedisClientType) *RedisClient {
	var addr string
	var db int

	switch clientType {
	case PlayerClient:
		addr = cfg.PlayerAddr
		db = cfg.PlayerDB
	case WalletClient:
		addr = cfg.WalletAddr
		db = cfg.WalletDB
	default:
		t.Fatalf("Неизвестный тип Redis-клиента: %s", clientType)
	}

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     cfg.Password,
		DB:           db,
		DialTimeout:  time.Duration(cfg.DialTimeout) * time.Second,
		ReadTimeout:  time.Duration(cfg.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.WriteTimeout) * time.Second,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("Ошибка подключения к Redis (%s): %v", clientType, err)
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

func (r *RedisClient) GetWithRetry(sCtx provider.StepCtx, key string, result interface{}) error {
	var lastErr error
	delay := r.retryDelay

	for i := 0; i < r.retryAttempts; i++ {
		value, err := r.Get(key)
		if err == nil && value != "" {
			if err := json.Unmarshal([]byte(value), result); err != nil {
				log.Printf("Failed to unmarshal Redis value: %v", err)
				lastErr = err
				continue
			}

			sCtx.WithAttachments(allure.NewAttachment("Redis Value", allure.JSON, utils.CreatePrettyJSON(result)))
			return nil
		}
		lastErr = err
		log.Printf("Attempt %d: Redis key %s not found or empty, retrying in %v...", i+1, key, delay)
		time.Sleep(delay)
		delay *= 2
	}

	log.Printf("Не удалось получить значение из Redis после %d попыток: %v", r.retryAttempts, lastErr)
	return lastErr
}

func (r *RedisClient) GetWithSeqCheck(sCtx provider.StepCtx, key string, result interface{}, expectedSeq int) error {
	var lastErr error
	delay := r.retryDelay
	maxAttempts := r.retryAttempts

	for i := 0; i < maxAttempts; i++ {
		value, err := r.Get(key)
		if err == nil && value != "" {
			if err := json.Unmarshal([]byte(value), result); err != nil {
				log.Printf("Не удалось десериализовать значение из Redis: %v", err)
				lastErr = err
				continue
			}

			var seqNumber int
			switch v := result.(type) {
			case *WalletFullData:
				seqNumber = v.LastSeqNumber
			default:
				if resultMap, ok := result.(map[string]interface{}); ok {
					if seqVal, exists := resultMap["LastSeqNumber"]; exists {
						if seqInt, ok := seqVal.(int); ok {
							seqNumber = seqInt
						}
					}
				}
			}

			if seqNumber == expectedSeq {
				sCtx.WithAttachments(allure.NewAttachment("Redis Value", allure.JSON, utils.CreatePrettyJSON(result)))
				return nil
			}

			log.Printf("Attempt %d: Redis key %s found, but LastSeqNumber %d != %d (expected), retrying in %v...",
				i+1, key, seqNumber, expectedSeq, delay)
		} else {
			lastErr = err
			log.Printf("Attempt %d: Redis key %s not found or empty, retrying in %v...", i+1, key, delay)
		}

		time.Sleep(delay)
		delay *= 2
	}

	log.Printf("Не удалось получить значение из Redis с ожидаемым LastSeqNumber после %d попыток: %v", maxAttempts, lastErr)
	return fmt.Errorf("redis get with sequence check failed after %d attempts: %w", maxAttempts, lastErr)
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}
