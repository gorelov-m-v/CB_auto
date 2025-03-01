package kafka

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"CB_auto/internal/config"

	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/segmentio/kafka-go"
)

type TopicType string

const (
	BrandTopic      TopicType = "beta-09_core.gambling.v1.Brand"
	PlayerTopic     TopicType = "beta-09_player.v1.account"
	LimitTopic      TopicType = "beta-09_limits.v2"
	ProjectionTopic TopicType = "beta-09_wallet.v8.projectionSource"
)

var allTopics = []TopicType{BrandTopic, PlayerTopic, LimitTopic, ProjectionTopic}

type subscriber struct {
	ch     chan kafka.Message
	done   chan struct{}
	closed atomic.Bool
	once   sync.Once
}

func newSubscriber() *subscriber {
	return &subscriber{
		ch:   make(chan kafka.Message, 100),
		done: make(chan struct{}),
	}
}

func (sub *subscriber) Close() {
	sub.once.Do(func() {
		sub.closed.Store(true)
		close(sub.done)
		close(sub.ch)
	})
}

type Kafka struct {
	readers          []*kafka.Reader
	subscribers      map[TopicType][]*subscriber
	bufferedMessages map[TopicType][]kafka.Message
	Timeout          time.Duration
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	started          bool
	mu               sync.RWMutex
}

var (
	instance   *Kafka
	once       sync.Once
	instanceMu sync.Mutex
	refCount   int
)

func GetInstance(t provider.T, cfg *config.Config) *Kafka {
	instanceMu.Lock()
	defer instanceMu.Unlock()

	once.Do(func() {
		t.Logf("Creating singleton Kafka consumer with all topics and buffering")
		instance = newConsumer(cfg)
		instance.startReading()
	})
	refCount++
	t.Logf("Kafka instance refCount increased to %d", refCount)
	return instance
}

func CloseInstance(t provider.T) {
	instanceMu.Lock()
	defer instanceMu.Unlock()

	if refCount > 0 {
		refCount--
		t.Logf("Kafka instance refCount decreased to %d", refCount)
	}
	if refCount == 0 && instance != nil {
		instance.close(t)
		instance = nil
		once = sync.Once{}
		t.Logf("Kafka singleton closed")
	}
}

func newConsumer(cfg *config.Config) *Kafka {
	var readers []*kafka.Reader
	subscribers := make(map[TopicType][]*subscriber)
	bufferedMessages := make(map[TopicType][]kafka.Message)

	for _, topic := range allTopics {
		bufferedMessages[topic] = make([]kafka.Message, 0)
	}

	for _, topicType := range allTopics {
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
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Kafka{
		readers:          readers,
		subscribers:      subscribers,
		bufferedMessages: bufferedMessages,
		Timeout:          30 * time.Second,
		ctx:              ctx,
		cancel:           cancel,
	}
}

func (k *Kafka) startReading() {
	k.mu.Lock()
	if k.started {
		k.mu.Unlock()
		return
	}
	k.started = true
	k.mu.Unlock()

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

			topic := TopicType(msg.Topic)
			k.mu.Lock()
			k.bufferedMessages[topic] = append(k.bufferedMessages[topic], msg)
			subs := make([]*subscriber, len(k.subscribers[topic]))
			copy(subs, k.subscribers[topic])
			k.mu.Unlock()

			for _, sub := range subs {
				if sub.closed.Load() {
					continue
				}
				select {
				case sub.ch <- msg:
				case <-sub.done:
				case <-k.ctx.Done():
					return
				}
			}
		}
	}
}

func FindMessageByFilter[T KafkaMessage](sCtx provider.StepCtx, k *Kafka, filter func(T) bool) T {
	var empty T
	var tmp T
	topic := tmp.GetTopic()

	ch := k.SubscribeToTopic(topic)
	defer k.Unsubscribe(ch)

	ctx, cancel := context.WithTimeout(k.ctx, k.Timeout)
	defer cancel()

	sCtx.Logf("Начало поиска сообщения в топике %s с таймаутом %v", topic, k.Timeout)
	for {
		select {
		case <-ctx.Done():
			sCtx.Logf("Таймаут при ожидании сообщения в топике %s", topic)
			return empty
		case msg, ok := <-ch:
			if !ok {
				sCtx.Logf("Канал подписки для топика %s закрыт", topic)
				return empty
			}
			sCtx.Logf("Получено сообщение: %s", string(msg.Value))
			var data T
			if err := json.Unmarshal(msg.Value, &data); err != nil {
				sCtx.Logf("Не удалось распарсить сообщение в тип %T: %v", data, err)
				continue
			}
			if filter(data) {
				sCtx.Logf("Найдено подходящее сообщение в топике %s", topic)
				return data
			}
		}
	}
}

func (k *Kafka) SubscribeToTopic(topic TopicType) chan kafka.Message {
	k.mu.Lock()
	defer k.mu.Unlock()

	buffered := make([]kafka.Message, len(k.bufferedMessages[topic]))
	copy(buffered, k.bufferedMessages[topic])

	sub := newSubscriber()
	k.subscribers[topic] = append(k.subscribers[topic], sub)

	go func() {
		for _, msg := range buffered {
			select {
			case sub.ch <- msg:
			case <-sub.done:
				return
			}
		}
	}()

	return sub.ch
}

func (k *Kafka) Unsubscribe(ch chan kafka.Message) {
	k.mu.Lock()
	defer k.mu.Unlock()

	for topic, subs := range k.subscribers {
		for i, sub := range subs {
			if sub.ch == ch {
				sub.Close()
				k.subscribers[topic] = append(subs[:i], subs[i+1:]...)
				return
			}
		}
	}
}

func (k *Kafka) close(t provider.T) {
	k.cancel()
	k.wg.Wait()

	k.mu.Lock()
	for _, subs := range k.subscribers {
		for _, sub := range subs {
			sub.Close()
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

type KafkaMessage interface {
	GetTopic() TopicType
}

func (m PlayerMessage) GetTopic() TopicType {
	return PlayerTopic
}

func (m LimitMessage) GetTopic() TopicType {
	return LimitTopic
}

func (m ProjectionSourceMessage) GetTopic() TopicType {
	return ProjectionTopic
}

func GetTopicForType[T KafkaMessage]() TopicType {
	var msg T
	return msg.GetTopic()
}
