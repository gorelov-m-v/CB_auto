package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/segmentio/kafka-go"
)

type Kafka struct {
	reader   *kafka.Reader
	Messages chan kafka.Message
	Timeout  time.Duration
	stopChan chan struct{}
	started  bool
}

func NewConsumer(brokers []string, topic string, groupID string, timeout time.Duration) *Kafka {
	fmt.Printf("Creating new Kafka consumer: brokers=%v, topic=%s, groupID=%s\n", brokers, topic, groupID)

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		StartOffset:    kafka.LastOffset,
		ReadBackoffMin: time.Millisecond * 100,
		ReadBackoffMax: time.Second * 1,
	})

	return &Kafka{
		reader:   reader,
		Messages: make(chan kafka.Message, 100),
		Timeout:  timeout,
		stopChan: make(chan struct{}),
	}
}

func (k *Kafka) StartReading(t provider.T) {
	if k.started {
		return
	}
	time.Sleep(1 * time.Second)
	go k.readMessages()
	k.started = true
}

func (k *Kafka) readMessages() {
	fmt.Printf("Started reading messages\n")
	for {
		select {
		case <-k.stopChan:
			fmt.Printf("Stopping message reader\n")
			return
		default:
			msg, err := k.reader.ReadMessage(context.Background())
			if err != nil {
				fmt.Printf("Error reading message: %v\n", err)
				continue
			}
			fmt.Printf("Received message from topic %s, partition %d, offset %d: %s\n",
				msg.Topic, msg.Partition, msg.Offset, string(msg.Value))
			k.Messages <- msg
		}
	}
}

type Message interface {
	Brand | PlayerMessage
}

func FindMessageByFilter[T any](k *Kafka, t provider.T, filter func(T) bool) kafka.Message {
	ctx, cancel := context.WithTimeout(context.Background(), k.Timeout)
	defer cancel()

	msgBuffer := make([]kafka.Message, 0)

	fmt.Printf("Starting to look for message with timeout %v\n", k.Timeout)
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("Не удалось найти нужное сообщение за %v: %v", k.Timeout, ctx.Err())
		case msg, ok := <-k.Messages:
			if !ok {
				t.Fatal("Канал сообщений закрыт")
			}

			fmt.Printf("Checking new message: %s\n", string(msg.Value))
			var data T
			if err := json.Unmarshal(msg.Value, &data); err != nil {
				fmt.Printf("Failed to unmarshal message: %v\n", err)
				msgBuffer = append(msgBuffer, msg)
				continue
			}
			if filter(data) {
				fmt.Printf("Found matching message\n")
				return msg
			}
			fmt.Printf("Message didn't match filter\n")
			msgBuffer = append(msgBuffer, msg)

		default:
			for i, bufferedMsg := range msgBuffer {
				var data T
				if err := json.Unmarshal(bufferedMsg.Value, &data); err != nil {
					continue
				}
				if filter(data) {
					msgBuffer = append(msgBuffer[:i], msgBuffer[i+1:]...)
					return bufferedMsg
				}
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (k *Kafka) Close(t provider.T) {
	close(k.stopChan)
	time.Sleep(500 * time.Millisecond)

	close(k.Messages)

	done := make(chan struct{})
	go func() {
		if err := k.reader.Close(); err != nil {
			t.Errorf("Ошибка при закрытии Kafka reader: %v", err)
		}
		close(done)
	}()

	select {
	case <-time.After(30 * time.Second):
		t.Errorf("Таймаут при закрытии Kafka reader")
	case <-done:
	}
}

func ParseMessage[T any](t provider.T, message kafka.Message) T {
	var data T
	if err := json.Unmarshal(message.Value, &data); err != nil {
		t.Fatalf("Ошибка при парсинге сообщения Kafka: %v", err)
	}
	return data
}

func (k *Kafka) GetMessages() <-chan kafka.Message {
	return k.Messages
}

func (k *Kafka) GetTimeout() time.Duration {
	return k.Timeout
}
