package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/nofendian17/sbterm/libs/pkg/log"
	datafeedv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/v1"
)

type nopPublisher struct{}

func (nopPublisher) Publish(context.Context, string, string, []byte) error { return nil }
func (nopPublisher) Close()                                                {}

func benchService(lvl log.Level) *Service {
	logger := log.New(log.WithLevel(lvl), log.WithWriter(discard{}))
	router := NewFrameRouter(nopPublisher{}, Topics{RunningTradeBatch: "bench.topic"})
	return New(nil, nil, router, logger)
}

func benchFrame(trades int) *datafeedv1.WebsocketWrapMessageChannel {
	batch := make([]*datafeedv1.RunningTrade, 0, trades)
	for i := 0; i < trades; i++ {
		batch = append(batch, &datafeedv1.RunningTrade{
			Stock:       fmt.Sprintf("SYM%04d", i),
			Price:       float64(1000 + i),
			Volume:      float64(100 + i),
			Value:       float64((1000 + i) * (100 + i)),
			TradeNumber: int32(i),
		})
	}
	return &datafeedv1.WebsocketWrapMessageChannel{
		MessageChannel: &datafeedv1.WebsocketWrapMessageChannel_RunningTradeBatch{
			RunningTradeBatch: &datafeedv1.RunningTradeBatch{Batch: batch},
		},
	}
}

// BenchmarkServiceHandleFrame measures the per-frame handler cost with the
// logger at info level (the production setting): debug formatting must not be
// paid when the level is disabled.
func BenchmarkServiceHandleFrame(b *testing.B) {
	svc := benchService(log.LevelInfo)
	sub := &Subscription{Name: "bench"}
	ctx := context.Background()

	for _, trades := range []int{1, 25, 100} {
		msg := benchFrame(trades)
		b.Run(fmt.Sprintf("trades_%03d", trades), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if err := svc.handleFrame(ctx, sub, msg); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkServiceHandleFrameDebug is the reference point: the same path with
// debug logging enabled, where the protojson cost is genuinely paid.
func BenchmarkServiceHandleFrameDebug(b *testing.B) {
	svc := benchService(log.LevelDebug)
	sub := &Subscription{Name: "bench"}
	ctx := context.Background()

	msg := benchFrame(25)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := svc.handleFrame(ctx, sub, msg); err != nil {
			b.Fatal(err)
		}
	}
}

type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }
