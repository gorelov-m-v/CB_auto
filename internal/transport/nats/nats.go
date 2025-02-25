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

// NatsClient управляет подключением к NATS и получает сообщения в канал Messages.
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

// NatsMessage оборачивает полученные данные с метаданными.
type NatsMessage[T any] struct {
	Payload   T
	Metadata  *nats.MsgMetadata
	Subject   string
	Sequence  uint64
	Seq       uint64
	Timestamp time.Time
	Type      string
}

// NewClient создаёт нового NATS клиента согласно конфигурации.
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

// SubscribeWithDeliverAll подписывается на указанный subject с опциями DeliverAll и другими.
func (c *NatsClient) subscribeWithDeliverAll(subject string) {
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
		return
	}
	c.subsMutex.Lock()
	c.subs = append(c.subs, sub)
	c.subsMutex.Unlock()
}

// FindMessageInStream подписывается на заданный subject и ждёт появления нужного сообщения,
// используя внутренний таймаут (c.timeout). Если сообщение, удовлетворяющее filter, приходит – возвращается оно.
// Если таймаут истекает, функция возвращает nil.
func FindMessageInStream[T any](sCtx provider.StepCtx, n *NatsClient, subject string, filter func(data T, msgType string) bool) *NatsMessage[T] {
	// Подписываемся на заданный subject.
	n.subscribeWithDeliverAll(subject)

	// Создаем контекст с таймаутом, равным внутреннему значению n.timeout.
	ctx, cancel := context.WithTimeout(n.ctx, n.timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			sCtx.Errorf("Timeout waiting for message on subject %s", subject)
			return nil
		case msg, ok := <-n.Messages:
			if !ok {
				sCtx.Errorf("Messages channel closed")
				return nil
			}
			log.Printf("Received NATS message on subject %s: %s", msg.Subject, string(msg.Data))
			var data T
			if err := json.Unmarshal(msg.Data, &data); err != nil {
				sCtx.Logf("Error unmarshaling message: %v", err)
				continue
			}
			if filter(data, msg.Header.Get("type")) {
				meta, _ := msg.Metadata()
				sCtx.WithAttachments(allure.NewAttachment("NATS Message", allure.JSON, utils.CreatePrettyJSON(data)))
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
		}
	}
}

// Close корректно завершает работу NATS клиента.
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
