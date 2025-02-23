package nats

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"CB_auto/internal/config"
	"CB_auto/pkg/utils"

	"github.com/nats-io/nats.go"
	"github.com/ozontech/allure-go/pkg/allure"
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

func NewClient(cfg *config.NatsConfig) *NatsClient {
	log.Printf("Creating new NATS client: hosts=%s", cfg.Hosts)

	nc, err := nats.Connect(cfg.Hosts,
		nats.ReconnectWait(time.Duration(cfg.ReconnectWait)*time.Second),
		nats.MaxReconnects(cfg.MaxReconnects))
	if err != nil {
		log.Printf("Ошибка подключения к NATS: %v", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		log.Printf("Ошибка создания JetStream контекста: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &NatsClient{
		conn:     nc,
		js:       js,
		Messages: make(chan *nats.Msg, 100),
		ctx:      ctx,
		cancel:   cancel,
		timeout:  cfg.StreamTimeout * time.Second,
	}
}

func FindMessageByFilter[T any](sCtx provider.StepCtx, n *NatsClient, filter func(T, string) bool) *NatsMessage[T] {
	ctx, cancel := context.WithTimeout(n.ctx, 30*time.Second)
	defer cancel()

	msgBuffer := make([]*nats.Msg, 0)
	log.Printf("Starting to look for message with timeout %v", n.timeout)

	maxAttempts := 20
	attempt := 1
	for {
		select {
		case <-ctx.Done():
			log.Printf("Не удалось найти нужное сообщение за %v: %v", n.timeout, ctx.Err())
		case msg, ok := <-n.Messages:
			if !ok {
				log.Printf("Канал сообщений закрыт")
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
				result := &NatsMessage[T]{
					Payload:   data,
					Metadata:  meta,
					Subject:   msg.Subject,
					Sequence:  meta.Sequence.Stream,
					Seq:       meta.Sequence.Stream,
					Timestamp: meta.Timestamp,
					Type:      msg.Header.Get("type"),
				}

				sCtx.WithAttachments(allure.NewAttachment("NATS Message", allure.JSON, utils.CreatePrettyJSON(result)))

				return result
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
						Seq:       meta.Sequence.Stream,
						Timestamp: meta.Timestamp,
						Type:      bufferedMsg.Header.Get("type"),
					}
				}
			}
			if attempt < maxAttempts {
				log.Printf("No matching message found, attempt %d/%d", attempt, maxAttempts)
				attempt++
				time.Sleep(1 * time.Second)
			} else {
				time.Sleep(1 * time.Second)
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

func (c *NatsClient) SubscribeWithDeliverAll(subject string) {
	opts := []nats.SubOpt{
		nats.DeliverAll(),
		nats.AckExplicit(),
		nats.ReplayInstant(),
		nats.StartSequence(1),
		nats.BindStream("beta-09_wallet"),
	}

	sub, err := c.js.Subscribe(subject, func(msg *nats.Msg) {
		c.Messages <- msg
	}, opts...)

	if err != nil {
		log.Printf("Ошибка при подписке на NATS: %v", err)
	}
	c.subs = append(c.subs, sub)
}

func FindMessageInStream[T any](sCtx provider.StepCtx, n *NatsClient, subject string, filter func(T, string) bool) *NatsMessage[T] {
	maxAttempts := 10
	attempt := 1

	log.Printf("Starting to look for message in stream with subject %s", subject)

	for attempt <= maxAttempts {
		n.SubscribeWithDeliverAll(subject)

		ctx, cancel := context.WithTimeout(n.ctx, 1*time.Second)

		msgBuffer := make([]*nats.Msg, 0)

	searchLoop:
		for {
			select {
			case <-ctx.Done():
				log.Printf("Timeout reached for attempt %d/%d", attempt, maxAttempts)
				break searchLoop
			case msg, ok := <-n.Messages:
				if !ok {
					log.Printf("Messages channel closed")
					break searchLoop
				}

				var data T
				if err := json.Unmarshal(msg.Data, &data); err != nil {
					log.Printf("Failed to unmarshal message: %v", err)
					msgBuffer = append(msgBuffer, msg)
					continue
				}

				if filter(data, msg.Header.Get("type")) {
					log.Printf("Found matching message on attempt %d", attempt)
					meta, _ := msg.Metadata()
					result := &NatsMessage[T]{
						Payload:   data,
						Metadata:  meta,
						Subject:   msg.Subject,
						Sequence:  meta.Sequence.Stream,
						Seq:       meta.Sequence.Stream,
						Timestamp: meta.Timestamp,
						Type:      msg.Header.Get("type"),
					}

					sCtx.WithAttachments(allure.NewAttachment("NATS Message", allure.JSON, utils.CreatePrettyJSON(result)))
					cancel()
					return result
				}

				msgBuffer = append(msgBuffer, msg)
			}
		}

		cancel()

		n.subsMutex.Lock()
		for _, sub := range n.subs {
			if err := sub.Unsubscribe(); err != nil {
				log.Printf("Error unsubscribing: %v", err)
			}
		}
		n.subs = nil
		n.subsMutex.Unlock()

		log.Printf("Message not found on attempt %d/%d, retrying...", attempt, maxAttempts)
		attempt++

		if attempt <= maxAttempts {
			time.Sleep(1 * time.Second)
		}
	}

	log.Printf("Message not found after %d attempts", maxAttempts)
	sCtx.Errorf("Message not found in stream after %d attempts", maxAttempts)
	return nil
}
