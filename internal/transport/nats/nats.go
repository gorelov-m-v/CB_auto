package nats

import (
	"context"
	"encoding/json"
	"log"
	"strings"
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
		conn:    nc,
		js:      js,
		ctx:     ctx,
		cancel:  cancel,
		timeout: cfg.StreamTimeout * time.Second,
	}
}

func (c *NatsClient) subscribeWithDeliverAll(subject string) (chan *nats.Msg, *nats.Subscription, error) {
	msgCh := make(chan *nats.Msg, 100)

	opts := []nats.SubOpt{
		nats.DeliverAll(),
		nats.AckExplicit(),
		nats.ReplayInstant(),
		nats.StartSequence(1),
		nats.BindStream("beta-09_wallet"),
	}

	sub, err := c.js.Subscribe(subject, func(msg *nats.Msg) {
		msgCh <- msg
	}, opts...)
	if err != nil {
		return nil, nil, err
	}

	c.subsMutex.Lock()
	c.subs = append(c.subs, sub)
	c.subsMutex.Unlock()
	return msgCh, sub, nil
}

func FindMessageInStream[T any](sCtx provider.StepCtx, n *NatsClient, subject string, filter func(data T, msgType string) bool) *NatsMessage[T] {
	msgCh, sub, err := n.subscribeWithDeliverAll(subject)
	if err != nil {
		sCtx.Errorf("Ошибка при подписке на NATS: %v", err)
		return nil
	}
	sCtx.Logf("NATS ПОИСК: Подписались на шаблон: %s", subject)

	n.wg.Add(1)
	defer n.wg.Done()
	defer sub.Unsubscribe()

	ctx, cancel := context.WithTimeout(n.ctx, n.timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			sCtx.Errorf("Timeout waiting for message on subject %s", subject)
			return nil
		case msg, ok := <-msgCh:
			if !ok {
				sCtx.Errorf("Канал сообщений закрыт для темы %s", subject)
				return nil
			}

			sCtx.Logf("NATS ПОИСК [%s]: Получено сообщение с темой: %s", subject, msg.Subject)

			subjectParts := strings.Split(msg.Subject, ".")
			templateParts := strings.Split(subject, ".")
			if len(subjectParts) != len(templateParts) {
				sCtx.Logf("NATS ПОИСК [%s]: Пропуск - разное количество частей в теме", subject)
				continue
			}

			if len(subjectParts) > 3 && len(templateParts) > 3 {
				subjectUUID := subjectParts[3]
				templateUUID := templateParts[3]
				if templateUUID != "*" && subjectUUID != templateUUID {
					sCtx.Logf("NATS ПОИСК [%s]: Пропуск - UUID игрока не совпадает: %s != %s",
						subject, templateUUID, subjectUUID)
					continue
				}
			}

			sCtx.Logf("NATS ПОИСК [%s]: Сообщение прошло проверку темы: %s", subject, msg.Subject)

			var data T
			if err := json.Unmarshal(msg.Data, &data); err != nil {
				sCtx.Logf("NATS ПОИСК [%s]: Ошибка распаковки JSON: %v", subject, err)
				continue
			}

			if filter(data, msg.Header.Get("type")) {
				sCtx.Logf("NATS ПОИСК [%s]: Сообщение прошло фильтр данных", subject)
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
			} else {
				sCtx.Logf("NATS ПОИСК [%s]: Сообщение НЕ прошло фильтр данных", subject)
			}
		}
	}
}

func (n *NatsClient) Close() {
	n.cancel()

	n.wg.Wait()

	n.subsMutex.Lock()
	for _, sub := range n.subs {
		if sub != nil {
			if err := sub.Unsubscribe(); err != nil {
				log.Printf("Ошибка при отписке от NATS: %v", err)
			}
		}
	}
	n.subsMutex.Unlock()

	if err := n.conn.Drain(); err != nil {
		log.Printf("Ошибка при закрытии NATS connection: %v", err)
	}
	n.conn.Close()
}
