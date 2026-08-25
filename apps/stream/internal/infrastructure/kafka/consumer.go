// Package kafka wraps the franz-go consumer-group client used by the stream
// fan-out.
package kafka

import (
	"context"
	"fmt"

	"github.com/twmb/franz-go/pkg/kgo"
)

// Consumer reads the datafeed pipeline topics inside one consumer group.
type Consumer struct {
	client *kgo.Client
}

// NewConsumer builds a consumer-group client for the pipeline topics. Offsets
// start at the end of every partition — the stream serves live data only, no
// history backfill. Offsets are never committed: this process is pure fan-out,
// durability belongs to apps/ingest.
func NewConsumer(brokers []string, group string, topics []string) (*Consumer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(group),
		kgo.ConsumeTopics(topics...),
		kgo.ConsumeStartOffset(kgo.NewOffset().AtEnd()),
		// Fan-out never commits: durability is apps/ingest's job. Without
		// this, franz-go auto-commits every 5s by default.
		kgo.DisableAutoCommit(),
	)
	if err != nil {
		return nil, fmt.Errorf("kafka: new consumer: %w", err)
	}
	return &Consumer{client: client}, nil
}

// PollFetches blocks until a fetch is available or ctx is cancelled.
func (c *Consumer) PollFetches(ctx context.Context) kgo.Fetches {
	return c.client.PollFetches(ctx)
}

// Close leaves the group and closes the client.
func (c *Consumer) Close() {
	c.client.Close()
}

// Shutdown implements the samber/do shutdown hook, closing the client and
// leaving the consumer group.
func (c *Consumer) Shutdown() error {
	c.Close()
	return nil
}
