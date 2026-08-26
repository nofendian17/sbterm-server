package service

import (
	"context"
	"testing"
	"time"

	"github.com/nofendian17/sbterm/apps/ingest/internal/detection"
	hotstate "github.com/nofendian17/sbterm/apps/ingest/internal/infrastructure/hotstate"
	"github.com/nofendian17/sbterm/apps/ingest/internal/infrastructure/questdb"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	datapricefeedv1 "github.com/nofendian17/sbterm/libs/proto/financial/company_price_feed/entity/v1"
	datapricefeedv2 "github.com/nofendian17/sbterm/libs/proto/financial/company_price_feed/entity/v2"
	consumerv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/consumer/entity/v1"
	datafeedv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/v1"
)

type fakeBookPipe struct {
	calls []*consumerv1.Orderbook
	err   error
}

func (f *fakeBookPipe) Process(_ context.Context, ob *consumerv1.Orderbook) error {
	if f.err != nil {
		return f.err
	}
	f.calls = append(f.calls, ob)
	return nil
}

func (f *fakeBookPipe) Pending() int { return 0 }

type fakeTradeObs struct {
	buys  int
	sells int
}

func (f *fakeTradeObs) ObserveTrade(_ context.Context, t TradeFeed) error {
	if t.Buy {
		f.buys++
	} else {
		f.sells++
	}
	return nil
}

type fakeStore struct {
	updates int
}

func (f *fakeStore) SetBook(context.Context, hotstate.BookUpdate) error { f.updates++; return nil }
func (f *fakeStore) TouchBook(context.Context, string, time.Time) error { return nil }
func (f *fakeStore) DedupBook(context.Context, string, string) (bool, error) {
	return false, nil
}

type nopAlertSink struct{}

func (nopAlertSink) Emit(context.Context, detection.Alert) error { return nil }

type fakeLiveness struct {
	symbols []string
}

func (f *fakeLiveness) TouchBook(_ context.Context, symbol string, _ time.Time) error {
	f.symbols = append(f.symbols, symbol)
	return nil
}

func TestHandlerRoutesOrderBookToPipeline(t *testing.T) {
	topics := Topics{RunningTradeBatch: "rt", OrderBook: "ob"}
	pipe := &fakeBookPipe{}
	h := NewFrameHandler(&fakeRunningSink{}, topics, nopLogger{}, WithBookPipeline(pipe))

	ob := &consumerv1.Orderbook{StockCode: "BBCA", Body: "#O|BBCA|BID|7750;1;100"}
	raw, err := proto.Marshal(ob)
	require.NoError(t, err)

	require.NoError(t, h.Handle(context.Background(), topics.OrderBook, raw))
	require.Len(t, pipe.calls, 1)
	assert.Equal(t, "BBCA", pipe.calls[0].GetStockCode())
}

func TestHandlerFeedsTradesToObserver(t *testing.T) {
	topics := Topics{RunningTradeBatch: "rt"}
	obs := &fakeTradeObs{}
	h := NewFrameHandler(&fakeRunningSink{}, topics, nopLogger{}, WithTradeObserver(obs))

	batch := &datafeedv1.RunningTradeBatch{Batch: []*datafeedv1.RunningTrade{
		{Stock: "BBCA", Price: 7750, Value: 1000, Action: datafeedv1.TradeType_TRADE_TYPE_BUY},
		{Stock: "BBCA", Price: 7745, Value: 500, Action: datafeedv1.TradeType_TRADE_TYPE_SELL},
	}}
	raw, err := proto.Marshal(batch)
	require.NoError(t, err)

	require.NoError(t, h.Handle(context.Background(), topics.RunningTradeBatch, raw))
	assert.EqualValues(t, 1, obs.buys)
	assert.EqualValues(t, 1, obs.sells)
}

func TestHandlerLivenessTopicsTouchProvider(t *testing.T) {
	topics := Topics{RunningTradeBatch: "rt", BestBidOffer: "bbo", IepIev: "iep", LivePrice: "lp"}
	toucher := &fakeLiveness{}
	h := NewFrameHandler(&fakeRunningSink{}, topics, nopLogger{}, WithLiveness(toucher))

	bbo := &datapricefeedv1.BestBidOfferWS{StockCode: "BBCA"}
	raw, err := proto.Marshal(bbo)
	require.NoError(t, err)
	require.NoError(t, h.Handle(context.Background(), topics.BestBidOffer, raw))

	iep := &datapricefeedv2.IEPIEV{StockCode: "BBRI"}
	raw, err = proto.Marshal(iep)
	require.NoError(t, err)
	require.NoError(t, h.Handle(context.Background(), topics.IepIev, raw))

	lp := &consumerv1.LivePrice{StockCode: "GULA"}
	raw, err = proto.Marshal(lp)
	require.NoError(t, err)
	require.NoError(t, h.Handle(context.Background(), topics.LivePrice, raw))

	assert.Equal(t, []string{"BBCA", "BBRI", "GULA"}, toucher.symbols)
}

type countingPersister struct {
	n int
}

func (c *countingPersister) Store(context.Context, *questdb.BookPair) error { c.n++; return nil }
func (c *countingPersister) Close(context.Context) error                    { return nil }

// TestPipelineRateCapsPersistence asserts the durable sink is throttled per
// symbol while the detector still observes every snapshot.
func TestPipelineRateCapsPersistence(t *testing.T) {
	store := &fakeStore{}
	pers := &countingPersister{}
	sinkCap := nopAlertSink{}
	engine := detection.NewEngine(detection.DefaultConfig(), sinkCap)
	pipe, err := NewBookPipeline(BookDeps{
		Combiner:           questdb.NewCombiner(25),
		Store:              store,
		Persister:          pers,
		Engine:             engine,
		Logger:             nopLogger{},
		MinPersistInterval: 600 * time.Millisecond,
	})
	require.NoError(t, err)

	mk := func(side string, px string, seq int64) *consumerv1.Orderbook {
		return &consumerv1.Orderbook{
			StockCode:      "BBCA",
			Body:           "#O|BBCA|" + side + "|" + px + ";1;100",
			SequenceNumber: seq,
		}
	}

	// Bid half first: the combiner emits only once both sides exist.
	require.NoError(t, pipe.Process(context.Background(), mk("BID", "770", 0)))
	require.NoError(t, pipe.Process(context.Background(), mk("OFFER", "7801", 1)))
	require.NoError(t, pipe.Process(context.Background(), mk("OFFER", "7802", 2))) // < interval

	assert.Equal(t, 2, store.updates, "hot state still tracks every change synchronously")

	// Exactly one pair was queued (the capped one never enqueued); the async
	// writer must drain it.
	deadline := time.Now().Add(2 * time.Second)
	for pipe.Pending() > 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	assert.Equal(t, 0, pipe.Pending())
	assert.Equal(t, 1, pers.n, "only the uncapped snapshot reaches questdb")
}
