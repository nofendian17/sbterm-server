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

func TestFrameHandler(t *testing.T) {
	topics := Topics{RunningTradeBatch: "datafeed.running_trade_batch"}
	logger := log.New(log.WithWriter(io.Discard))

	t.Run("running trade batch topic decodes and stores", func(t *testing.T) {
		rs := &fakeRunningSink{}
		h := NewFrameHandler(rs, topics, logger)

		bytes, err := proto.Marshal(&datafeedv1.RunningTradeBatch{Batch: []*datafeedv1.RunningTrade{{Stock: "BBRI"}}})
		require.NoError(t, err)

		require.NoError(t, h.Handle(context.Background(), "datafeed.running_trade_batch", bytes))
		require.Len(t, rs.stored, 1)
		assert.Equal(t, "BBRI", rs.stored[0].GetBatch()[0].GetStock())
		require.NoError(t, h.Close(context.Background()))
	})

	t.Run("undecodable record errors", func(t *testing.T) {
		h := NewFrameHandler(&fakeRunningSink{}, topics, logger)
		require.Error(t, h.Handle(context.Background(), "datafeed.running_trade_batch", []byte("not a proto")))
	})

	t.Run("unknown topic errors", func(t *testing.T) {
		h := NewFrameHandler(&fakeRunningSink{}, topics, logger)
		require.Error(t, h.Handle(context.Background(), "datafeed.who", []byte{1}))
	})

	t.Run("sink store error propagates", func(t *testing.T) {
		rs := &errRunningSink{err: errBoom}
		h := NewFrameHandler(rs, topics, logger)
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
