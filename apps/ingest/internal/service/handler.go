package service

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/nofendian17/sbterm/libs/pkg/log"
	consumerv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/consumer/entity/v1"
	datafeedv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/v1"
)

// Topics names the Kafka topics the handler understands. Values come from
// config and must match what the ws service publishes.
type Topics struct {
	RunningTradeBatch string
	OrderBook         string
}

// RunningTradeBatchSink persists running trade batches on one writer goroutine.
// Implementations are not safe for concurrent use; Close flushes buffered rows
// and releases the underlying connection.
type RunningTradeBatchSink interface {
	Store(ctx context.Context, batch *datafeedv1.RunningTradeBatch) error
	Close(ctx context.Context) error
}

// OrderBookSink persists one order book side snapshot on one writer goroutine.
// Implementations are not safe for concurrent use; Close flushes buffered rows
// and releases the underlying connection.
type OrderBookSink interface {
	Store(ctx context.Context, ob *consumerv1.Orderbook) error
	Close(ctx context.Context) error
}

// FrameHandler decodes one Kafka record and persists it to the matching
// QuestDB sink. Both sinks are single-writer; Handler is not safe for
// concurrent use.
type FrameHandler struct {
	runningSink RunningTradeBatchSink
	obSink      OrderBookSink
	topics      Topics
	logger      log.Logger
}

// NewFrameHandler builds a handler bound to the two sinks.
func NewFrameHandler(runningSink RunningTradeBatchSink, obSink OrderBookSink, topics Topics, logger log.Logger) *FrameHandler {
	return &FrameHandler{runningSink: runningSink, obSink: obSink, topics: topics, logger: logger}
}

// Handle routes one record by topic, unmarshalling the protobuf payload and
// storing it.
func (h *FrameHandler) Handle(ctx context.Context, topic string, value []byte) error {
	switch topic {
	case h.topics.RunningTradeBatch:
		batch := &datafeedv1.RunningTradeBatch{}
		if err := proto.Unmarshal(value, batch); err != nil {
			return fmt.Errorf("ingest: decode running trade batch: %w", err)
		}
		return h.runningSink.Store(ctx, batch)
	case h.topics.OrderBook:
		ob := &consumerv1.Orderbook{}
		if err := proto.Unmarshal(value, ob); err != nil {
			return fmt.Errorf("ingest: decode order book: %w", err)
		}
		return h.obSink.Store(ctx, ob)
	default:
		return fmt.Errorf("ingest: unexpected topic %q", topic)
	}
}

// Close flushes and releases both sinks.
func (h *FrameHandler) Close(ctx context.Context) error {
	var errs []error
	if err := h.runningSink.Close(ctx); err != nil {
		errs = append(errs, fmt.Errorf("ingest: close running trade sink: %w", err))
	}
	if err := h.obSink.Close(ctx); err != nil {
		errs = append(errs, fmt.Errorf("ingest: close order book sink: %w", err))
	}
	return errors.Join(errs...)
}
