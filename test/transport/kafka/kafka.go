package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/segmentio/kafka-go"
)

type Kafka struct {
	reader   *kafka.Reader
	messages chan kafka.Message
	timeout  time.Duration
	stopChan chan struct{}
	started  bool
}

func NewConsumer(brokers []string, topic string, groupID string, timeout time.Duration) *Kafka {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     groupID,
		StartOffset: kafka.LastOffset,
	})

	return &Kafka{
		reader:   reader,
		messages: make(chan kafka.Message, 100),
		timeout:  timeout,
		stopChan: make(chan struct{}),
	}
}

func (k *Kafka) StartReading(t provider.T) {
	if k.started {
		return
	}
	go k.readMessages()
	k.started = true
}

func (k *Kafka) readMessages() {
	for {
		select {
		case <-k.stopChan:
			return
		default:
			msg, err := k.reader.ReadMessage(context.Background())
			if err != nil {
				continue
			}
			k.messages <- msg
		}
	}
}

func (k *Kafka) FindMessage(t provider.T, filter func(Brand) bool) kafka.Message {
	ctx, cancel := context.WithTimeout(context.Background(), k.timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("Не удалось найти нужное сообщение за %v: %v", k.timeout, ctx.Err())
			return kafka.Message{}
		case msg := <-k.messages:
			var brand Brand
			if err := json.Unmarshal(msg.Value, &brand); err != nil {
				continue
			}

			if filter(brand) {
				return msg
			}
		}
	}
}

func (k *Kafka) Close(t provider.T) {
	close(k.stopChan)
	close(k.messages)
	if err := k.reader.Close(); err != nil {
		t.Fatalf("Ошибка при закрытии Kafka reader: %v", err)
	}
}

func ParseMessage[T any](t provider.T, message kafka.Message) T {
	var data T
	if err := json.Unmarshal(message.Value, &data); err != nil {
		t.Fatalf("Ошибка при парсинге сообщения Kafka: %v", err)
	}
	return data
}
