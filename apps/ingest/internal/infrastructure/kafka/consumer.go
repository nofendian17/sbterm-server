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

// NewConsumer builds a consumer-group client for the pipeline topics.
func NewConsumer(brokers []string, group string, topics []string) (*Consumer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(group),
		kgo.ConsumeTopics(topics...),
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

// AllowRebalance marks the current batch processed before the group rebalances.
func (c *Consumer) AllowRebalance() {
	c.client.AllowRebalance()
}

// Close leaves the group and closes the client.
func (c *Consumer) Close() {
	c.client.Close()
}
