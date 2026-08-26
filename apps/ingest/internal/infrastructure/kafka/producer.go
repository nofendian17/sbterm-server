package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/nofendian17/sbterm/libs/pkg/log"
)

// Publisher publishes detection alerts (and future outbound events) to
// Kafka. It is safe for concurrent use.
type Publisher struct {
	cl     *kgo.Client
	logger log.Logger
}

// NewPublisher builds a producer over the given brokers. Delivery is
// synchronous with a short timeout: an alert is small and rare enough that
// waiting is cheap, and the caller learns about failures immediately.
func NewPublisher(brokers []string, logger log.Logger) (*Publisher, error) {
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ProducerBatchCompression(kgo.ZstdCompression()),
		kgo.RecordDeliveryTimeout(5*time.Second),
		kgo.WithLogger(klogBridge{logger}),
	)
	if err != nil {
		return nil, fmt.Errorf("kafka: new publisher: %w", err)
	}
	return &Publisher{cl: cl, logger: logger}, nil
}

// Publish writes one record synchronously.
func (p *Publisher) Publish(ctx context.Context, topic, key string, value []byte) error {
	rec := &kgo.Record{Topic: topic, Key: []byte(key), Value: value}
	if err := p.cl.ProduceSync(ctx, rec).FirstErr(); err != nil {
		return fmt.Errorf("kafka: publish to %s: %w", topic, err)
	}
	return nil
}

// Close releases the producer, flushing pending records.
func (p *Publisher) Close() { p.cl.Close() }

// klogBridge adapts libs/pkg/log onto franz-go's logging interface so broker
// diagnostics land in the same pipeline as everything else.
type klogBridge struct{ l log.Logger }

func (k klogBridge) Level() kgo.LogLevel { return kgo.LogLevelWarn }
func (k klogBridge) Log(level kgo.LogLevel, msg string, keyvals ...any) {
	switch level {
	case kgo.LogLevelError:
		k.l.Error(msg, keyvals...)
	case kgo.LogLevelWarn:
		k.l.Warn(msg, keyvals...)
	default:
		k.l.Info(msg, keyvals...)
	}
}
