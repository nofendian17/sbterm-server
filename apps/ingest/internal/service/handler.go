package service

import (
	"time"

	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/nofendian17/sbterm/libs/pkg/log"
	datapricefeedv1 "github.com/nofendian17/sbterm/libs/proto/financial/company_price_feed/entity/v1"
	datapricefeedv2 "github.com/nofendian17/sbterm/libs/proto/financial/company_price_feed/entity/v2"
	consumerv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/consumer/entity/v1"
	datafeedv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/v1"
)

// Topics names the Kafka topics the handler understands. Values come from
// config and must match what the ws service publishes.
type Topics struct {
	RunningTradeBatch string
	OrderBook         string
	BestBidOffer      string
	IepIev            string
	LivePrice         string
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
// concurrent use. Optional pipelines (order book, trade observer, liveness)
// attach through HandlerOption and stay nil when the feature is disabled.
type FrameHandler struct {
	runningSink   RunningTradeBatchSink
	topics        Topics
	logger        log.Logger
	bookPipe      BookPipeline
	tradeObserver TradeObserver
	liveness      LivenessToucher
}

// NewFrameHandler builds a handler bound to the running trade sink.
func NewFrameHandler(runningSink RunningTradeBatchSink, topics Topics, logger log.Logger,
	opts ...HandlerOption,
) *FrameHandler {
	h := &FrameHandler{runningSink: runningSink, topics: topics, logger: logger}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// handleTimeout bounds one record's full processing. A hung downstream write
// (e.g. a QuestDB sender whose connection died while idle) must surface as an
// error — which triggers redelivery — instead of freezing the poll loop.
const handleTimeout = 10 * time.Second

// Handle routes one record by topic, unmarshalling the protobuf payload and
// storing it.
func (h *FrameHandler) Handle(ctx context.Context, topic string, value []byte) error {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, handleTimeout)
	defer cancel()
	switch topic {
	case h.topics.RunningTradeBatch:
		batch := &datafeedv1.RunningTradeBatch{}
		if err := proto.Unmarshal(value, batch); err != nil {
			return fmt.Errorf("ingest: decode running trade batch: %w", err)
		}
		if h.tradeObserver != nil {
			now := time.Now()
			for _, tr := range batch.GetBatch() {
				if err := h.tradeObserver.ObserveTrade(ctx, protoTradeToDetectorTrade(tr, now)); err != nil {
					h.logger.Warn("detection: observe trade failed", "symbol", tr.GetStock(), "error", err)
				}
			}
		}
		return h.runningSink.Store(ctx, batch)
	case h.topics.OrderBook:
		if h.bookPipe == nil {
			return nil // feature disabled; frame intentionally dropped
		}
		ob := &consumerv1.Orderbook{}
		if err := proto.Unmarshal(value, ob); err != nil {
			return fmt.Errorf("ingest: decode order book: %w", err)
		}
		return h.bookPipe.Process(ctx, ob)
	case h.topics.BestBidOffer:
		if h.liveness == nil {
			return nil
		}
		msg := &datapricefeedv1.BestBidOfferWS{}
		if err := proto.Unmarshal(value, msg); err != nil {
			return fmt.Errorf("ingest: decode best bid offer: %w", err)
		}
		return h.liveness.TouchBook(ctx, msg.GetStockCode(), time.Now())
	case h.topics.IepIev:
		if h.liveness == nil {
			return nil
		}
		msg := &datapricefeedv2.IEPIEV{}
		if err := proto.Unmarshal(value, msg); err != nil {
			return fmt.Errorf("ingest: decode iep iev: %w", err)
		}
		return h.liveness.TouchBook(ctx, msg.GetStockCode(), time.Now())
	case h.topics.LivePrice:
		if h.liveness == nil {
			return nil
		}
		msg := &consumerv1.LivePrice{}
		if err := proto.Unmarshal(value, msg); err != nil {
			return fmt.Errorf("ingest: decode live price: %w", err)
		}
		return h.liveness.TouchBook(ctx, msg.GetStockCode(), time.Now())
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
