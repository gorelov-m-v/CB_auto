package kafka

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"CB_auto/internal/config"
	"CB_auto/pkg/utils"

	"github.com/ozontech/allure-go/pkg/allure"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/segmentio/kafka-go"
)

type Kafka struct {
	reader   *kafka.Reader
	readers  []*kafka.Reader
	Messages chan kafka.Message
	Timeout  time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	started  bool
}

type TopicType string

const (
	BrandTopic  TopicType = "beta-09_core.gambling.v1.Brand"
	PlayerTopic TopicType = "beta-09_player.v1.account"
	LimitTopic  TopicType = "beta-09_limits.v2"
)

func NewConsumer(t provider.T, cfg *config.Config, topicTypes ...TopicType) *Kafka {
	var readers []*kafka.Reader
	for _, topicType := range topicTypes {
		topic := string(topicType)
		log.Printf("Creating Kafka reader: brokers=%v, topic=%s, groupID=%s", cfg.Kafka.Brokers, topic, cfg.Node.GroupID)

		reader := kafka.NewReader(kafka.ReaderConfig{
			Brokers:        []string{cfg.Kafka.Brokers},
			Topic:          topic,
			GroupID:        cfg.Node.GroupID,
			StartOffset:    kafka.LastOffset,
			ReadBackoffMin: 100 * time.Millisecond,
			ReadBackoffMax: 1 * time.Second,
		})
		readers = append(readers, reader)
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Kafka{
		readers:  readers,
		Messages: make(chan kafka.Message, 100),
		Timeout:  cfg.Kafka.GetTimeout(),
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (k *Kafka) StartReading(t provider.T) {
	if k.started {
		return
	}
	k.started = true
	k.wg.Add(len(k.readers))
	for _, reader := range k.readers {
		go k.readMessages(reader)
	}
}

func (k *Kafka) readMessages(reader *kafka.Reader) {
	defer k.wg.Done()
	log.Printf("Started reading messages")
	for {
		select {
		case <-k.ctx.Done():
			log.Printf("Stopping message reader")
			return
		default:
			msg, err := reader.ReadMessage(k.ctx)
			if err != nil {
				if k.ctx.Err() != nil {
					log.Printf("Message reading stopped: %v", err)
					return
				}
				log.Printf("Error reading message: %v", err)
				time.Sleep(100 * time.Millisecond)
				continue
			}
			log.Printf("Received message from topic %s, partition %d, offset %d: %s",
				msg.Topic, msg.Partition, msg.Offset, string(msg.Value))
			select {
			case k.Messages <- msg:
			case <-k.ctx.Done():
				return
			}
		}
	}
}

func FindMessageByFilter[T any](k *Kafka, t provider.T, filter func(T) bool) kafka.Message {
	ctx, cancel := context.WithTimeout(k.ctx, k.Timeout)
	defer cancel()

	msgBuffer := make([]kafka.Message, 0)
	log.Printf("Starting to look for message with timeout %v", k.Timeout)

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("Не удалось найти нужное сообщение за %v: %v", k.Timeout, ctx.Err())
		case msg, ok := <-k.Messages:
			if !ok {
				t.Fatal("Канал сообщений закрыт")
			}
			log.Printf("Checking new message: %s", string(msg.Value))
			var data T
			if err := json.Unmarshal(msg.Value, &data); err != nil {
				log.Printf("Failed to unmarshal message: %v", err)
				msgBuffer = append(msgBuffer, msg)
				continue
			}
			if filter(data) {
				log.Printf("Found matching message")
				return msg
			}
			log.Printf("Message didn't match filter")
			msgBuffer = append(msgBuffer, msg)
		default:
			for _, bufferedMsg := range msgBuffer {
				var data T
				if err := json.Unmarshal(bufferedMsg.Value, &data); err != nil {
					continue
				}
				if filter(data) {
					return bufferedMsg
				}
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (k *Kafka) Close(t provider.T) {
	k.cancel()
	k.wg.Wait()

	close(k.Messages)

	done := make(chan struct{})
	go func() {
		for _, reader := range k.readers {
			if err := reader.Close(); err != nil {
				t.Errorf("Ошибка при закрытии Kafka reader: %v", err)
			}
		}
		close(done)
	}()

	select {
	case <-time.After(30 * time.Second):
		t.Errorf("Таймаут при закрытии Kafka reader")
	case <-done:
	}
}

func ParseMessage[T any](sCtx provider.StepCtx, message kafka.Message) T {
	var data T
	if err := json.Unmarshal(message.Value, &data); err != nil {
		log.Printf("Ошибка при парсинге сообщения Kafka: %v", err)
	}

	sCtx.WithAttachments(allure.NewAttachment("Kafka Message", allure.JSON, utils.CreatePrettyJSON(data)))

	return data
}

func (k *Kafka) GetMessages() <-chan kafka.Message {
	return k.Messages
}

func (k *Kafka) GetTimeout() time.Duration {
	return k.Timeout
}
