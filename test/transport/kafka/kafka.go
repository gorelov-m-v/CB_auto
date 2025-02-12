package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/segmentio/kafka-go"
)

type Kafka struct {
	reader *kafka.Reader
}

func NewTestUtils(brokers []string, topic string, groupID string) *Kafka {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     groupID,
		StartOffset: kafka.FirstOffset,
	})

	return &Kafka{
		reader: reader,
	}
}

func (k *Kafka) ReadMessageWithFilter(t provider.T, timeout time.Duration, filter func(Brand) bool) kafka.Message {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("Timeout: не удалось найти нужное сообщение за %v", timeout)
			return kafka.Message{}
		default:
			msg, err := k.reader.ReadMessage(ctx)
			if err != nil {
				t.Fatalf("Ошибка при чтении сообщения из Kafka: %v", err)
			}

			var brand Brand
			if err := json.Unmarshal(msg.Value, &brand); err != nil {
				t.Logf("Пропуск сообщения: ошибка десериализации: %v", err)
				continue
			}

			if filter(brand) {
				return msg
			}
			t.Logf("Пропуск сообщения: не соответствует фильтру")
		}
	}
}

func (k *Kafka) Close(t provider.T) {
	if err := k.reader.Close(); err != nil {
		t.Fatalf("Ошибка при закрытии Kafka reader: %v", err)
	}
}
