package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	consumerv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/consumer/entity/v1"
	datafeedv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/v1"
)

type fakePublisher struct {
	topics []record
	closed bool
}

type record struct {
	topic string
	key   string
	value []byte
}

func (f *fakePublisher) Publish(_ context.Context, topic string, key string, value []byte) error {
	f.topics = append(f.topics, record{topic: topic, key: key, value: append([]byte(nil), value...)})
	return nil
}
func (f *fakePublisher) Close() { f.closed = true }

func TestFrameRouter(t *testing.T) {
	topics := Topics{RunningTradeBatch: "datafeed.running_trade_batch"}

	t.Run("running trade batch publishes to its topic keyed by first symbol", func(t *testing.T) {
		pub := &fakePublisher{}
		router := NewFrameRouter(pub, topics)

		batch := &datafeedv1.RunningTradeBatch{Batch: []*datafeedv1.RunningTrade{{Stock: "BBRI", Price: 1000}}}
		msg := &datafeedv1.WebsocketWrapMessageChannel{MessageChannel: &datafeedv1.WebsocketWrapMessageChannel_RunningTradeBatch{RunningTradeBatch: batch}}

		require.NoError(t, router.Route(context.Background(), msg))
		require.Len(t, pub.topics, 1)
		assert.Equal(t, "datafeed.running_trade_batch", pub.topics[0].topic)
		assert.Equal(t, "BBRI", pub.topics[0].key)

		got := &datafeedv1.RunningTradeBatch{}
		require.NoError(t, proto.Unmarshal(pub.topics[0].value, got))
		assert.Equal(t, "BBRI", got.GetBatch()[0].GetStock())
	})

	t.Run("frames without an ingested channel publish nothing", func(t *testing.T) {
		pub := &fakePublisher{}
		router := NewFrameRouter(pub, topics)
		msg := &datafeedv1.WebsocketWrapMessageChannel{MessageChannel: &datafeedv1.WebsocketWrapMessageChannel_Liveprice{Liveprice: &consumerv1.LivePrice{}}}
		require.NoError(t, router.Route(context.Background(), msg))
		assert.Empty(t, pub.topics)
	})

	t.Run("publisher error propagates", func(t *testing.T) {
		pub := &errorPublisher{}
		router := NewFrameRouter(pub, topics)
		msg := &datafeedv1.WebsocketWrapMessageChannel{MessageChannel: &datafeedv1.WebsocketWrapMessageChannel_RunningTradeBatch{RunningTradeBatch: &datafeedv1.RunningTradeBatch{}}}
		require.Error(t, router.Route(context.Background(), msg))
	})
}

type errorPublisher struct{}

func (e *errorPublisher) Publish(_ context.Context, _ string, _ string, _ []byte) error {
	return errPublish
}
func (e *errorPublisher) Close() {}

var errPublish = assert.AnError
