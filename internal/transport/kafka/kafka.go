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
type TopicPart string

const (
	BrandTopicPart      TopicPart = "core.gambling.v1.Brand"
	PlayerTopicPart     TopicPart = "player.v1.account"
	LimitTopicPart      TopicPart = "limits.v2"
	ProjectionTopicPart TopicPart = "wallet.v8.projectionSource"
	GameTopicPart       TopicPart = "core.gambling.v2.Game"
)

type Topics struct {
	Brand      TopicType
	Player     TopicType
	Limit      TopicType
	Projection TopicType
	Game       TopicType
}

func NewTopics(prefix string) Topics {
	return Topics{
		Brand:      TopicType(prefix + string(BrandTopicPart)),
		Player:     TopicType(prefix + string(PlayerTopicPart)),
		Limit:      TopicType(prefix + string(LimitTopicPart)),
		Projection: TopicType(prefix + string(ProjectionTopicPart)),
		Game:       TopicType(prefix + string(GameTopicPart)),
	}
}

var TopicsConfig Topics

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

type RingBuffer struct {
	mu     sync.Mutex
	buffer []kafka.Message
	size   int
	head   int
	count  int
}

func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		buffer: make([]kafka.Message, size),
		size:   size,
	}
}

func (rb *RingBuffer) Add(msg kafka.Message) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.buffer[rb.head] = msg
	rb.head = (rb.head + 1) % rb.size
	if rb.count < rb.size {
		rb.count++
	}
}

func (rb *RingBuffer) GetAll() []kafka.Message {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	messages := make([]kafka.Message, rb.count)
	for i := 0; i < rb.count; i++ {
		index := (rb.head - rb.count + i + rb.size) % rb.size
		messages[i] = rb.buffer[index]
	}
	return messages
}

type Kafka struct {
	readers          []*kafka.Reader
	subscribers      map[TopicType][]*subscriber
	bufferedMessages map[TopicType]*RingBuffer
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
		t.Logf("Создание синглтона Kafka consumer с динамическими топиками")
		TopicsConfig = NewTopics(cfg.Kafka.TopicPrefix)
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
	bufferedMessages := make(map[TopicType]*RingBuffer)

	topicList := []TopicType{
		TopicsConfig.Brand,
		TopicsConfig.Player,
		TopicsConfig.Limit,
		TopicsConfig.Projection,
		TopicsConfig.Game,
	}

	bufferSize := cfg.Kafka.BufferSize
	if bufferSize < 1 {
		bufferSize = 500
	}

	for _, topic := range topicList {
		bufferedMessages[topic] = NewRingBuffer(bufferSize)
	}

	for _, topicType := range topicList {
		topic := string(topicType)
		log.Printf("Создание Kafka reader: brokers=%v, topic=%s, groupID=%s", cfg.Kafka.Brokers, topic, cfg.Node.GroupID)
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
	log.Printf("Начало чтения сообщений")
	for {
		select {
		case <-k.ctx.Done():
			log.Printf("Остановка чтения сообщений")
			return
		default:
			msg, err := reader.ReadMessage(k.ctx)
			if err != nil {
				if k.ctx.Err() != nil {
					log.Printf("Чтение сообщений остановлено: %v", err)
					return
				}
				log.Printf("Ошибка чтения сообщения: %v", err)
				time.Sleep(100 * time.Millisecond)
				continue
			}
			log.Printf("Получено сообщение из топика %s, partition %d, offset %d: %s",
				msg.Topic, msg.Partition, msg.Offset, string(msg.Value))

			topic := TopicType(msg.Topic)
			k.mu.Lock()
			k.bufferedMessages[topic].Add(msg)
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
			sCtx.Logf("Таймаут ожидания сообщения в топике %s", topic)
			return empty
		case msg, ok := <-ch:
			if !ok {
				sCtx.Logf("Канал подписки для топика %s закрыт", topic)
				return empty
			}
			sCtx.Logf("Получено сообщение: %s", string(msg.Value))
			var data T
			if err := json.Unmarshal(msg.Value, &data); err != nil {
				sCtx.Logf("Ошибка парсинга сообщения в тип %T: %v", data, err)
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

	buffered := k.bufferedMessages[topic].GetAll()
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

// Метод close с форсированным закрытием reader-ов:
// После отмены контекста сразу закрываются подписчики и Kafka reader-ы параллельно.
// Если reader не закроется в течение 5 секунд, просто логируем предупреждение.
func (k *Kafka) close(t provider.T) {
	// Отменяем контекст
	k.cancel()

	// Закрываем подписчиков немедленно
	k.mu.Lock()
	for _, subs := range k.subscribers {
		for _, sub := range subs {
			sub.Close()
		}
	}
	k.subscribers = nil
	k.mu.Unlock()

	// Параллельное закрытие всех Kafka reader-ов
	var closeWg sync.WaitGroup
	closeWg.Add(len(k.readers))
	for _, reader := range k.readers {
		go func(r *kafka.Reader) {
			defer closeWg.Done()
			_ = r.Close() // Ошибку игнорируем, просто форсируем закрытие
		}(reader)
	}
	doneCh := make(chan struct{})
	go func() {
		closeWg.Wait()
		close(doneCh)
	}()
	select {
	case <-doneCh:
		// Все reader-ы закрыты
	case <-time.After(5 * time.Second):
		t.Logf("Таймаут при закрытии Kafka reader. Форсированное закрытие")
	}
}

type KafkaMessage interface {
	GetTopic() TopicType
}

func (m PlayerMessage) GetTopic() TopicType {
	return TopicsConfig.Player
}

func (m LimitMessage) GetTopic() TopicType {
	return TopicsConfig.Limit
}

func (m ProjectionSourceMessage) GetTopic() TopicType {
	return TopicsConfig.Projection
}

func (m GameMessage) GetTopic() TopicType {
	return TopicsConfig.Game
}

func GetTopicForType[T KafkaMessage]() TopicType {
	var msg T
	return msg.GetTopic()
}
