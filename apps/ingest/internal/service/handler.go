package service

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/nofendian17/sbterm/libs/pkg/log"
	datafeedv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/v1"
)

// Topics names the Kafka topics the handler understands. Values come from
// config and must match what the ws service publishes.
type Topics struct {
	RunningTradeBatch string
}

// RunningTradeBatchSink persists running trade batches on one writer goroutine.
// Implementations are not safe for concurrent use; Close flushes buffered rows
// and releases the underlying connection.
type RunningTradeBatchSink interface {
	Store(ctx context.Context, batch *datafeedv1.RunningTradeBatch) error
	Close(ctx context.Context) error
}

// FrameHandler decodes one Kafka record and persists it to the matching
// QuestDB sink. The sink is single-writer; Handler is not safe for
// concurrent use.
type FrameHandler struct {
	runningSink RunningTradeBatchSink
	topics      Topics
	logger      log.Logger
}

// NewFrameHandler builds a handler bound to the running trade sink.
func NewFrameHandler(runningSink RunningTradeBatchSink, topics Topics, logger log.Logger) *FrameHandler {
	return &FrameHandler{runningSink: runningSink, topics: topics, logger: logger}
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
	default:
		return fmt.Errorf("ingest: unexpected topic %q", topic)
	}
}

// Close flushes and releases the sink.
func (h *FrameHandler) Close(ctx context.Context) error {
	if err := h.runningSink.Close(ctx); err != nil {
		return fmt.Errorf("ingest: close running trade sink: %w", err)
	}
	return nil
}
