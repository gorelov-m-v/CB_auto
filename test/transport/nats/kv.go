package nats

import (
	"fmt"
	"strings"
	"time"

	"CB_auto/test/config"

	"github.com/nats-io/nats.go"
)

type KVStore struct {
	js   nats.JetStreamContext
	kv   nats.KeyValue
	conn *nats.Conn
}

func NewKVStore(cfg *config.NatsConfig) (*KVStore, error) {
	opts := []nats.Option{
		nats.Timeout(cfg.Timeout * time.Second),
		nats.ReconnectWait(cfg.ReconnectWait * time.Second),
		nats.MaxReconnects(cfg.MaxReconnects),
	}

	nc, err := nats.Connect(cfg.Hosts, opts...)
	if err != nil {
		return nil, fmt.Errorf("nats connect failed: %v", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		return nil, fmt.Errorf("jetstream context failed: %v", err)
	}

	kv, err := js.KeyValue(cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("get kv bucket failed: %v", err)
	}

	return &KVStore{
		js:   js,
		kv:   kv,
		conn: nc,
	}, nil
}

func (k *KVStore) Get(key string) ([]byte, error) {
	entry, err := k.kv.Get(key)
	if err != nil {
		if strings.Contains(err.Error(), "no key found") {
			return nil, nil
		}
		return nil, fmt.Errorf("get key failed: %v", err)
	}
	return entry.Value(), nil
}

func (k *KVStore) Close() {
	if k.conn != nil {
		k.conn.Close()
	}
}
