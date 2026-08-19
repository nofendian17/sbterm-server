package kafka

import (
	"context"
	"fmt"

	"github.com/twmb/franz-go/pkg/kgo"

	service "github.com/nofendian17/sbterm/apps/ws/internal/service"
	"github.com/nofendian17/sbterm/libs/pkg/log"
)

// Producer publishes datafeed frames to Redpanda/Kafka via franz-go.
type Producer struct {
	client *kgo.Client
	logger log.Logger
}

// NewProducer builds a franz-go producer seeded with the given brokers.
func NewProducer(brokers []string, logger log.Logger) (*Producer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.AllowAutoTopicCreation(),
		kgo.RequiredAcks(kgo.AllISRAcks()),
	)
	if err != nil {
		return nil, fmt.Errorf("kafka: new producer: %w", err)
	}
	return &Producer{client: client, logger: logger}, nil
}

// Publish sends one record synchronously. A non-nil error means the record was
// not acknowledged; the ws reconnect loop provides the backpressure.
func (p *Producer) Publish(ctx context.Context, topic string, key string, value []byte) error {
	res := p.client.ProduceSync(ctx, &kgo.Record{Topic: topic, Key: []byte(key), Value: value})
	if err := res.FirstErr(); err != nil {
		return fmt.Errorf("kafka: produce %s: %w", topic, err)
	}
	return nil
}

// Close shuts down the producer and flushes buffered records.
func (p *Producer) Close() {
	p.client.Close()
}

// ponytail: Producer sits alone behind its port in service while the app
// stays flat. Split the port into its own package when a second publisher
// implementation or a second consuming app appears.
var _ service.Publisher = (*Producer)(nil)
