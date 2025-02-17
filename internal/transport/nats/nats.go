package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"CB_auto/internal/config"

	"github.com/nats-io/nats.go"
	"github.com/ozontech/allure-go/pkg/framework/provider"
)

type NatsClient struct {
	conn      *nats.Conn
	js        nats.JetStreamContext
	Messages  chan *nats.Msg
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	timeout   time.Duration
	subsMutex sync.Mutex
	subs      []*nats.Subscription
}

type NatsMessage[T any] struct {
	Payload   T
	Metadata  *nats.MsgMetadata
	Subject   string
	Sequence  uint64
	Seq       uint64
	Timestamp time.Time
	Type      string
}

func NewClient(cfg *config.NatsConfig) (*NatsClient, error) {
	log.Printf("Creating new NATS client: hosts=%s", cfg.Hosts)

	opts := []nats.Option{
		nats.ReconnectWait(cfg.ReconnectWait * time.Second),
		nats.MaxReconnects(cfg.MaxReconnects),
		nats.Timeout(cfg.Timeout * time.Second),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Printf("NATS disconnected: %v", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("NATS reconnected to %v", nc.ConnectedUrl())
		}),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			log.Printf("NATS error: %v", err)
		}),
	}

	nc, err := nats.Connect(cfg.Hosts, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &NatsClient{
		conn:     nc,
		js:       js,
		Messages: make(chan *nats.Msg, 100),
		ctx:      ctx,
		cancel:   cancel,
		timeout:  cfg.StreamTimeout * time.Second,
	}, nil
}

func (n *NatsClient) Subscribe(t provider.T, subjectPattern string) {
	n.subsMutex.Lock()
	defer n.subsMutex.Unlock()

	log.Printf("Subscribing to subject: %s", subjectPattern)
	sub, err := n.js.Subscribe(subjectPattern,
		n.messageHandler,
		nats.DeliverLast(),
		nats.AckExplicit(),
		nats.ManualAck(),
	)
	if err != nil {
		t.Fatalf("Ошибка при подписке на NATS: %v", err)
	}

	n.subs = append(n.subs, sub)
}

func (n *NatsClient) messageHandler(msg *nats.Msg) {
	meta, err := msg.Metadata()
	if err == nil {
		log.Printf("Received message: subject=%s, sequence=%d, timestamp=%v",
			msg.Subject, meta.Sequence.Stream, meta.Timestamp)
	}
	log.Printf("Received message: subject=%s, data=%s", msg.Subject, string(msg.Data))

	select {
	case <-n.ctx.Done():
		return
	case n.Messages <- msg:
		msg.Ack()
	}
}

func FindMessageByFilter[T any](n *NatsClient, t provider.T, filter func(T, string) bool) *NatsMessage[T] {
	ctx, cancel := context.WithTimeout(n.ctx, n.timeout)
	defer cancel()

	msgBuffer := make([]*nats.Msg, 0)
	log.Printf("Starting to look for message with timeout %v", n.timeout)

	maxAttempts := 5 // Увеличиваем количество попыток
	attempt := 1
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("Не удалось найти нужное сообщение за %v: %v", n.timeout, ctx.Err())
		case msg, ok := <-n.Messages:
			if !ok {
				t.Fatal("Канал сообщений закрыт")
			}
			log.Printf("Checking new message: %s", string(msg.Data))

			var data T
			if err := json.Unmarshal(msg.Data, &data); err != nil {
				log.Printf("Failed to unmarshal message data: %v", err)
				msgBuffer = append(msgBuffer, msg)
				continue
			}

			if filter(data, msg.Header.Get("type")) {
				log.Printf("Found matching message")
				meta, _ := msg.Metadata()
				return &NatsMessage[T]{
					Payload:   data,
					Metadata:  meta,
					Subject:   msg.Subject,
					Sequence:  meta.Sequence.Stream,
					Seq:       meta.Sequence.Stream,
					Timestamp: meta.Timestamp,
					Type:      msg.Header.Get("type"),
				}
			}

			log.Printf("Message didn't match filter")
			msgBuffer = append(msgBuffer, msg)
		default:
			for _, bufferedMsg := range msgBuffer {
				var data T
				if err := mapEventData(bufferedMsg.Data, &data); err != nil {
					continue
				}
				if filter(data, bufferedMsg.Header.Get("type")) {
					log.Printf("Found matching message in buffer")
					meta, _ := bufferedMsg.Metadata()
					return &NatsMessage[T]{
						Payload:   data,
						Metadata:  meta,
						Subject:   bufferedMsg.Subject,
						Sequence:  meta.Sequence.Stream,
						Seq:       meta.Sequence.Stream,
						Timestamp: meta.Timestamp,
						Type:      bufferedMsg.Header.Get("type"),
					}
				}
			}
			if attempt < maxAttempts {
				log.Printf("No matching message found, attempt %d/%d", attempt, maxAttempts)
				attempt++
				time.Sleep(500 * time.Millisecond) // Увеличиваем интервал между попытками
			} else {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
}

func ParseMessage[T any](t provider.T, message *nats.Msg) T {
	var data T
	if err := json.Unmarshal(message.Data, &data); err != nil {
		t.Fatalf("Ошибка при парсинге сообщения NATS: %v", err)
	}
	return data
}

func (n *NatsClient) Close() {
	n.cancel()

	n.subsMutex.Lock()
	for _, sub := range n.subs {
		if err := sub.Unsubscribe(); err != nil {
			log.Printf("Ошибка при отписке от NATS: %v", err)
		}
	}
	n.subsMutex.Unlock()

	close(n.Messages)

	if err := n.conn.Drain(); err != nil {
		log.Printf("Ошибка при закрытии NATS connection: %v", err)
	}
	n.conn.Close()
}

func mapEventData(data interface{}, target interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, target)
}
