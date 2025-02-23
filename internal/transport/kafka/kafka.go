package kafka

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"CB_auto/internal/config"

	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/segmentio/kafka-go"
)

type Kafka struct {
	reader      *kafka.Reader
	readers     []*kafka.Reader
	subscribers map[TopicType][]chan kafka.Message
	Timeout     time.Duration
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	started     bool
	mu          sync.RWMutex
}

var (
	instance *Kafka
	once     sync.Once
	mu       sync.Mutex
)

func GetInstance(t provider.T, cfg *config.Config, topicTypes ...TopicType) *Kafka {
	once.Do(func() {
		t.Logf("Creating singleton Kafka consumer")
		instance = newConsumer(cfg, topicTypes...)
		instance.startReading()
	})
	return instance
}

func CloseInstance(t provider.T) {
	mu.Lock()
	defer mu.Unlock()

	if instance != nil {
		instance.close(t)
		instance = nil
	}
}

type TopicType string

const (
	BrandTopic  TopicType = "beta-09_core.gambling.v1.Brand"
	PlayerTopic TopicType = "beta-09_player.v1.account"
	LimitTopic  TopicType = "beta-09_limits.v2"
)

func newConsumer(cfg *config.Config, topicTypes ...TopicType) *Kafka {
	var readers []*kafka.Reader
	subscribers := make(map[TopicType][]chan kafka.Message)

	for _, topicType := range topicTypes {
		topic := string(topicType)
		log.Printf("Creating Kafka reader: brokers=%v, topic=%s, groupID=%s", cfg.Kafka.Brokers, topic, cfg.Node.GroupID)

		reader := kafka.NewReader(kafka.ReaderConfig{
			Brokers:        []string{cfg.Kafka.Brokers},
			Topic:          topic,
			GroupID:        cfg.Node.GroupID,
			StartOffset:    kafka.LastOffset,
			ReadBackoffMin: 100 * time.Millisecond,
			ReadBackoffMax: 500 * time.Millisecond,
		})
		readers = append(readers, reader)
		subscribers[topicType] = make([]chan kafka.Message, 0)
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Kafka{
		readers:     readers,
		subscribers: subscribers,
		Timeout:     30 * time.Second,
		ctx:         ctx,
		cancel:      cancel,
	}
}

func (k *Kafka) startReading() {
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
			k.mu.RLock()
			for _, ch := range k.subscribers[TopicType(msg.Topic)] {
				select {
				case ch <- msg:
				case <-k.ctx.Done():
					k.mu.RUnlock()
					return
				}
			}
			k.mu.RUnlock()
		}
	}
}

type KafkaMessage interface {
	GetTopic() TopicType
}

func (m PlayerMessage) GetTopic() TopicType {
	return PlayerTopic
}

func (m LimitMessage) GetTopic() TopicType {
	return LimitTopic
}

func GetTopicForType[T KafkaMessage]() TopicType {
	var msg T
	return msg.GetTopic()
}

func FindMessageByFilter[T KafkaMessage](sCtx provider.StepCtx, k *Kafka, filter func(T) bool) T {
	var msg T
	ch := k.SubscribeToTopic(msg.GetTopic())
	defer k.Unsubscribe(ch)

	ctx, cancel := context.WithTimeout(k.ctx, k.Timeout)
	defer cancel()

	log.Printf("Starting to look for message in Kafka with timeout %v", k.Timeout)

	var empty T
	for {
		select {
		case <-ctx.Done():
			return empty

		case msg, ok := <-ch:
			if !ok {
				log.Printf("Kafka messages channel closed")
				return empty
			}

			log.Printf("Received raw message: %s", string(msg.Value))

			var data T
			if err := json.Unmarshal(msg.Value, &data); err != nil {
				log.Printf("Failed to unmarshal message into %T: %v\nMessage: %s", data, err, string(msg.Value))
				continue
			}

			log.Printf("Successfully unmarshaled message. Type: %T, Value: %+v", data, data)

			if filter(data) {
				log.Printf("Found matching message in Kafka")
				return data
			} else {
				log.Printf("Message did not match filter criteria")
			}
		}
	}
}

func (k *Kafka) SubscribeToTopic(topic TopicType) chan kafka.Message {
	k.mu.Lock()
	defer k.mu.Unlock()

	ch := make(chan kafka.Message, 100)
	k.subscribers[topic] = append(k.subscribers[topic], ch)
	return ch
}

func (k *Kafka) Unsubscribe(ch chan kafka.Message) {
	k.mu.Lock()
	defer k.mu.Unlock()

	for topic, subs := range k.subscribers {
		for i, sub := range subs {
			if sub == ch {
				k.subscribers[topic] = append(k.subscribers[topic][:i], k.subscribers[topic][i+1:]...)
				close(ch)
				break
			}
		}
	}
}

func (k *Kafka) close(t provider.T) {
	k.cancel()
	k.wg.Wait()

	k.mu.Lock()
	for _, subs := range k.subscribers {
		for _, ch := range subs {
			close(ch)
		}
	}
	k.subscribers = nil
	k.mu.Unlock()

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
