package service

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/nofendian17/sbterm/libs/pkg/log"
	consumerv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/consumer/entity/v1"
	datafeedv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/v1"
)

type fakeRunningSink struct {
	stored   []*datafeedv1.RunningTradeBatch
	closeErr error
}

func (f *fakeRunningSink) Store(_ context.Context, b *datafeedv1.RunningTradeBatch) error {
	f.stored = append(f.stored, b)
	return nil
}

func (f *fakeRunningSink) Close(context.Context) error { return f.closeErr }

type fakeObSink struct {
	stored   []*consumerv1.Orderbook
	closeErr error
}

func (f *fakeObSink) Store(_ context.Context, ob *consumerv1.Orderbook) error {
	f.stored = append(f.stored, ob)
	return nil
}

func (f *fakeObSink) Close(context.Context) error { return f.closeErr }

func TestFrameHandler(t *testing.T) {
	topics := Topics{RunningTradeBatch: "datafeed.running_trade_batch", OrderBook: "datafeed.order_book"}
	logger := log.New(log.WithWriter(io.Discard))

	t.Run("running trade batch topic decodes and stores", func(t *testing.T) {
		rs, os := &fakeRunningSink{}, &fakeObSink{}
		h := NewFrameHandler(rs, os, topics, logger)

		bytes, err := proto.Marshal(&datafeedv1.RunningTradeBatch{Batch: []*datafeedv1.RunningTrade{{Stock: "BBRI"}}})
		require.NoError(t, err)

		require.NoError(t, h.Handle(context.Background(), "datafeed.running_trade_batch", bytes))
		require.Len(t, rs.stored, 1)
		assert.Equal(t, "BBRI", rs.stored[0].GetBatch()[0].GetStock())
		require.NoError(t, h.Close(context.Background()))
	})

	t.Run("order book topic decodes and stores", func(t *testing.T) {
		rs, os := &fakeRunningSink{}, &fakeObSink{}
		h := NewFrameHandler(rs, os, topics, logger)

		bytes, err := proto.Marshal(&consumerv1.Orderbook{StockCode: "BBCA"})
		require.NoError(t, err)

		require.NoError(t, h.Handle(context.Background(), "datafeed.order_book", bytes))
		require.Len(t, os.stored, 1)
		assert.Equal(t, "BBCA", os.stored[0].GetStockCode())
		require.NoError(t, h.Close(context.Background()))
	})

	t.Run("undecodable record errors", func(t *testing.T) {
		h := NewFrameHandler(&fakeRunningSink{}, &fakeObSink{}, topics, logger)
		require.Error(t, h.Handle(context.Background(), "datafeed.running_trade_batch", []byte("not a proto")))
	})

	t.Run("unknown topic errors", func(t *testing.T) {
		h := NewFrameHandler(&fakeRunningSink{}, &fakeObSink{}, topics, logger)
		require.Error(t, h.Handle(context.Background(), "datafeed.who", []byte{1}))
	})

	t.Run("sink store error propagates", func(t *testing.T) {
		rs := &errRunningSink{err: errBoom}
		h := NewFrameHandler(rs, &fakeObSink{}, topics, logger)
		bytes, err := proto.Marshal(&datafeedv1.RunningTradeBatch{Batch: []*datafeedv1.RunningTrade{}})
		require.NoError(t, err)
		require.ErrorIs(t, h.Handle(context.Background(), "datafeed.running_trade_batch", bytes), errBoom)
	})
}

var errBoom = errors.New("boom")

type errRunningSink struct {
	fakeRunningSink
	err error
}

func (e *errRunningSink) Store(_ context.Context, _ *datafeedv1.RunningTradeBatch) error {
	return e.err
}
