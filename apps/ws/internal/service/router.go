package service

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	datafeedv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/v1"
)

// FrameRouter maps decoded datafeed frames onto Kafka by channel type. It is
// the single boundary between the websocket transport and the ingestion
// pipeline.
type FrameRouter struct {
	publisher Publisher
	topics    Topics
}

// NewFrameRouter builds a router that publishes ingested datafeed frames to
// the configured topics.
func NewFrameRouter(publisher Publisher, topics Topics) *FrameRouter {
	return &FrameRouter{publisher: publisher, topics: topics}
}

// Route publishes every ingested channel present in the frame. Non-ingested
// channels are ignored. A topic keyed by symbol keeps one symbol's frames in a
// single Kafka partition, preserving per-symbol ordering.
func (r *FrameRouter) Route(ctx context.Context, m *datafeedv1.WebsocketWrapMessageChannel) error {
	if batch := m.GetRunningTradeBatch(); batch != nil {
		value, err := proto.Marshal(batch)
		if err != nil {
			return fmt.Errorf("ws: marshal running trade batch: %w", err)
		}
		trades := batch.GetBatch()
		symbol := ""
		if len(trades) > 0 {
			symbol = trades[0].GetStock()
		}
		return r.publisher.Publish(ctx, r.topics.RunningTradeBatch, symbol, value)
	}
	if ob := m.GetOrderbook(); ob != nil {
		value, err := proto.Marshal(ob)
		if err != nil {
			return fmt.Errorf("ws: marshal order book: %w", err)
		}
		return r.publisher.Publish(ctx, r.topics.OrderBook, ob.GetStockCode(), value)
	}
	if bbo := m.GetBestBidOffer(); bbo != nil {
		value, err := proto.Marshal(bbo)
		if err != nil {
			return fmt.Errorf("ws: marshal best bid offer: %w", err)
		}
		return r.publisher.Publish(ctx, r.topics.BestBidOffer, bbo.GetStockCode(), value)
	}
	if iep := m.GetIepiev(); iep != nil {
		value, err := proto.Marshal(iep)
		if err != nil {
			return fmt.Errorf("ws: marshal iep iev: %w", err)
		}
		return r.publisher.Publish(ctx, r.topics.IepIev, iep.GetStockCode(), value)
	}
	if lp := m.GetLiveprice(); lp != nil {
		value, err := proto.Marshal(lp)
		if err != nil {
			return fmt.Errorf("ws: marshal live price: %w", err)
		}
		return r.publisher.Publish(ctx, r.topics.LivePrice, lp.GetStockCode(), value)
	}
	return nil
}
