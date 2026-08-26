package service

import (
	"context"
	"testing"
	"time"

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
