package service

import (
	"context"
	"sync"
	"sync/atomic"
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

func (f *fakeBookPipe) ObserveTrade(context.Context, detection.Trade) error { return nil }

type fakeTradeObs struct {
	buys  int
	sells int
}

func (f *fakeTradeObs) ObserveTrade(_ context.Context, t DetectorTrade) error {
	if t.Buy {
		f.buys++
	} else {
		f.sells++
	}
	return nil
}

type fakeStore struct {
	updates atomic.Int64
}

func (f *fakeStore) SetBook(context.Context, hotstate.BookUpdate) error { f.updates.Add(1); return nil }
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

// TestTradeToFeedExcludesUnspecifiedFromNetFlow confirms that UNSPECIFIED
// trades on a NON-regular board (NG/TN/unknown) are zeroed so they cannot
// fabricate distribution pressure in the bias evaluator, while UNSPECIFIED on
// the regular board is preserved (there it is a closing-auction trade whose
// side resolves upstream).
func TestTradeToFeedExcludesUnspecifiedFromNetFlow(t *testing.T) {
	now := time.Now()

	// UNSPECIFIED + non-regular board (NG): zeroed, excluded from net-flow.
	ng := protoTradeToDetectorTrade(&datafeedv1.RunningTrade{
		Stock:       "BBCA",
		Price:       7750,
		Volume:      200,
		Value:       1652000,
		Action:      datafeedv1.TradeType_TRADE_TYPE_UNSPECIFIED,
		MarketBoard: consumerv1.BoardType_BOARD_TYPE_NG,
	}, now)
	assert.Equal(t, "BBCA", ng.Symbol)
	assert.Equal(t, 0.0, ng.Value) // zeroed: excluded from net-flow
	assert.Equal(t, 0.0, detectorTradeNetFlow(ng))
	assert.False(t, ng.Buy)

	// UNSPECIFIED + regular board (RG): preserved (closing-auction trade whose
	// side resolves upstream). Value is kept, but Buy stays false so net-flow
	// still treats it as a sell until the side is known.
	rg := protoTradeToDetectorTrade(&datafeedv1.RunningTrade{
		Stock:       "BBCA",
		Price:       7750,
		Volume:      200,
		Value:       1652000,
		Action:      datafeedv1.TradeType_TRADE_TYPE_UNSPECIFIED,
		MarketBoard: consumerv1.BoardType_BOARD_TYPE_RG,
	}, now)
	assert.Equal(t, 1652000.0, rg.Value) // kept, not zeroed
	assert.False(t, rg.Buy)
	assert.Equal(t, -1652000.0, detectorTradeNetFlow(rg))

	buy := protoTradeToDetectorTrade(&datafeedv1.RunningTrade{
		Stock:  "BBCA",
		Price:  7750,
		Volume: 200,
		Value:  1652000,
		Action: datafeedv1.TradeType_TRADE_TYPE_BUY,
	}, now)
	assert.True(t, buy.Buy)
	assert.Equal(t, 1652000.0, buy.Value)

	sell := protoTradeToDetectorTrade(&datafeedv1.RunningTrade{
		Stock:  "BBCA",
		Price:  7745,
		Volume: 100,
		Value:  774500,
		Action: datafeedv1.TradeType_TRADE_TYPE_SELL,
	}, now)
	assert.False(t, sell.Buy)
	assert.Equal(t, 774500.0, sell.Value)
}

// detectorTradeNetFlow mirrors how the engine folds a trade into the
// signed net-flow: sign(-1 for sell) * Value, so a zeroed Value contributes 0.
func detectorTradeNetFlow(t DetectorTrade) float64 {
	sign := -1.0
	if t.Buy {
		sign = 1.0
	}
	return sign * t.Value
}

type countingPersister struct {
	n atomic.Int64
}

func (c *countingPersister) Store(context.Context, *questdb.BookPair) error { c.n.Add(1); return nil }
func (c *countingPersister) Close(context.Context) error                    { return nil }

// TestPipelineRateCapsPersistence asserts the durable sink is throttled per
// symbol while the detector still observes every snapshot.
func TestPipelineRateCapsPersistence(t *testing.T) {
	store := &fakeStore{}
	pers := &countingPersister{}
	sinkCap := nopAlertSink{}
	pipe, err := NewBookPipeline(BookDeps{
		Store:     store,
		Persister: pers,
		Logger:    nopLogger{},
		EngineFactory: func() *detection.Engine {
			return detection.NewEngine(detection.DefaultConfig(), sinkCap)
		},
		Workers:            1,
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

	// Shards own their queues: Process only enqueues, so wait until every
	// completed pair reached hot state before judging the throttle.
	deadline := time.Now().Add(2 * time.Second)
	for store.updates.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	assert.EqualValues(t, 2, store.updates.Load(), "hot state tracks every paired snapshot")
	assert.Equal(t, 0, pipe.Pending())

	assert.EqualValues(t, 1, pers.n.Load(), "only the uncapped snapshot reaches questdb")
}

// TestPipelineConcurrentTradesAndBooksRaceFree pins the ownership rule: the
// shard worker is the only goroutine touching a detection engine. Trades used
// to bypass the shard queue and race ObserveBook; run with -race to enforce.
func TestPipelineConcurrentTradesAndBooksRaceFree(t *testing.T) {
	pipe, err := NewBookPipeline(BookDeps{
		Store:     &fakeStore{},
		Persister: nil,
		Logger:    nopLogger{},
		EngineFactory: func() *detection.Engine {
			return detection.NewEngine(detection.DefaultConfig(), nopAlertSink{})
		},
	})
	require.NoError(t, err)

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				assert.NoError(t, pipe.ObserveTrade(context.Background(), detection.Trade{
					Symbol: "BBCA", TS: time.Now(), Price: 7750, Volume: 10, Value: 77500, Buy: true,
				}))
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				assert.NoError(t, pipe.Process(context.Background(), &consumerv1.Orderbook{
					StockCode: "BBCA", Body: "#O|BBCA|BID|7750;1;100",
				}))
			}
		}()
	}
	wg.Wait()

	deadline := time.Now().Add(5 * time.Second)
	for pipe.Pending() > 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	assert.Equal(t, 0, pipe.Pending())
}
