package kafka

import (
	"context"
	"fmt"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
)

// Consumer reads the datafeed pipeline topics inside one consumer group.
type Consumer struct {
	client *kgo.Client
}

// NewConsumer builds a consumer-group client for the pipeline topics. Auto-commit
// is disabled: the service commits offsets explicitly, only after a record has
// been durably persisted (see service.Service). This keeps the pipeline
// at-least-once on a dead downstream instead of silently dropping records.
func NewConsumer(brokers []string, group string, topics []string) (*Consumer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(group),
		kgo.ConsumeTopics(topics...),
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

// AllowRebalance marks the current batch processed before the group rebalances.
func (c *Consumer) AllowRebalance() {
	c.client.AllowRebalance()
}

// MarkCommit commits the offsets of the processed records so the group resumes
// after them on the next assign/restart. Offset is the next-to-consume (last
// processed + 1); the leader epoch is carried for broker-side truncation
// detection. Records with a negative epoch (message-set legacy) commit with
// epoch -1, which the broker treats as "no truncation check".
//
// Passing an empty slice is a no-op (the broker rejects empty commits, and
// there is nothing to advance anyway).
func (c *Consumer) MarkCommit(ctx context.Context, processed []*kgo.Record) error {
	if len(processed) == 0 {
		return nil
	}
	offsets := map[string]map[int32]kgo.EpochOffset{}
	for _, r := range processed {
		po := offsets[r.Topic]
		if po == nil {
			po = map[int32]kgo.EpochOffset{}
			offsets[r.Topic] = po
		}
		next := r.Offset + 1
		cur, ok := po[r.Partition]
		// Keep the highest next-offset for the partition. A record may be
		// re-fetched out of order across batches, but within a single
		// PollFetches the service processes each partition sequentially, so
		// the max offset is the authoritative resume point.
		if !ok || next > cur.Offset {
			epoch := r.LeaderEpoch
			if epoch < 0 {
				epoch = -1
			}
			po[r.Partition] = kgo.EpochOffset{Epoch: epoch, Offset: next}
		}
	}
	return c.commitSync(ctx, offsets)
}

// commitSync wraps franz-go's CommitOffsetsSync, returning the callback error.
func (c *Consumer) commitSync(ctx context.Context, offsets map[string]map[int32]kgo.EpochOffset) error {
	done := make(chan error, 1)
	c.client.CommitOffsetsSync(ctx, offsets, func(_ *kgo.Client, _ *kmsg.OffsetCommitRequest, _ *kmsg.OffsetCommitResponse, err error) {
		done <- err
	})
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
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
