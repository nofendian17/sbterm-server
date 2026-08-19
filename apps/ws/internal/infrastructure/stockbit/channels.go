package stockbit

import (
	ordertradev1 "github.com/nofendian17/sbterm/libs/proto/financial/order_trade/entity/v1"
	datafeedv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/v1"
)

// WSChannelWatchlist subscribes the given symbols on the watchlist channel.
func WSChannelWatchlist(symbols ...string) *datafeedv1.WebsocketChannel {
	return &datafeedv1.WebsocketChannel{Watchlist: symbols}
}

// WSChannelOrderBook subscribes the given symbols on the order book channel.
func WSChannelOrderBook(symbols ...string) *datafeedv1.WebsocketChannel {
	return &datafeedv1.WebsocketChannel{OrderBook: symbols}
}

// WSChannelRunningTrade subscribes the given symbols on the running trade
// channel.
func WSChannelRunningTrade(symbols ...string) *datafeedv1.WebsocketChannel {
	return &datafeedv1.WebsocketChannel{RunningTrade: symbols}
}

// WSChannelRunningTradeBatch subscribes the given symbols on the running trade
// batch channel.
func WSChannelRunningTradeBatch(symbols ...string) *datafeedv1.WebsocketChannel {
	return &datafeedv1.WebsocketChannel{RunningTradeBatch: symbols}
}

// WSChannelLiveprice subscribes the given symbols on the live price channel.
func WSChannelLiveprice(symbols ...string) *datafeedv1.WebsocketChannel {
	return &datafeedv1.WebsocketChannel{Liveprice: symbols}
}

// WSChannelIepiev subscribes the given symbols on the IEP/IEV channel.
func WSChannelIepiev(symbols ...string) *datafeedv1.WebsocketChannel {
	return &datafeedv1.WebsocketChannel{Iepiev: symbols}
}

// WSChannelIntraday subscribes the given symbols on the intraday channel.
func WSChannelIntraday(symbols ...string) *datafeedv1.WebsocketChannel {
	return &datafeedv1.WebsocketChannel{Intraday: symbols}
}

// WSChannelBestBidOffer subscribes the given symbols on the best bid offer
// channel.
func WSChannelBestBidOffer(symbols ...string) *datafeedv1.WebsocketChannel {
	return &datafeedv1.WebsocketChannel{BestBidOffer: symbols}
}

// WSChannelLivepriceV3 subscribes the given symbols on the live price v3
// channel.
func WSChannelLivepriceV3(symbols ...string) *datafeedv1.WebsocketChannel {
	return &datafeedv1.WebsocketChannel{LivepriceV3: symbols}
}

// WSChannelOrderBookV3 subscribes the given symbols on the order book v3
// channel.
func WSChannelOrderBookV3(symbols ...string) *datafeedv1.WebsocketChannel {
	return &datafeedv1.WebsocketChannel{OrderBookV3: symbols}
}

// WSChannelIntradayV3 subscribes the given symbols on the intraday v3 channel.
func WSChannelIntradayV3(symbols ...string) *datafeedv1.WebsocketChannel {
	return &datafeedv1.WebsocketChannel{IntradayV3: symbols}
}

// WSChannelMarketMover subscribes the market mover channel with the given typed
// requests.
func WSChannelMarketMover(reqs ...*ordertradev1.MarketMoverWebsocketRequest) *datafeedv1.WebsocketChannel {
	return &datafeedv1.WebsocketChannel{MarketMover: reqs}
}

// WSChannelOrderQueue subscribes the order queue channel with the given typed
// requests.
func WSChannelOrderQueue(reqs ...*ordertradev1.OrderQueueWebsocketRequest) *datafeedv1.WebsocketChannel {
	return &datafeedv1.WebsocketChannel{OrderQueue: reqs}
}

// WSChannelTradebook subscribes the tradebook channel with the given typed
// requests.
func WSChannelTradebook(reqs ...*ordertradev1.TradebookWebsocketRequest) *datafeedv1.WebsocketChannel {
	return &datafeedv1.WebsocketChannel{Tradebook: reqs}
}

// MergeWSChannels combines symbol-array channels into a single subscription
// channel. Typed channels (market mover, order queue, tradebook) are ignored
// because they carry structured requests instead of shared symbols.
func MergeWSChannels(chans ...*datafeedv1.WebsocketChannel) *datafeedv1.WebsocketChannel {
	out := &datafeedv1.WebsocketChannel{}
	for _, ch := range chans {
		out.Watchlist = append(out.Watchlist, ch.Watchlist...)
		out.OrderBook = append(out.OrderBook, ch.OrderBook...)
		out.RunningTrade = append(out.RunningTrade, ch.RunningTrade...)
		out.RunningTradeBatch = append(out.RunningTradeBatch, ch.RunningTradeBatch...)
		out.Liveprice = append(out.Liveprice, ch.Liveprice...)
		out.Iepiev = append(out.Iepiev, ch.Iepiev...)
		out.Intraday = append(out.Intraday, ch.Intraday...)
		out.BestBidOffer = append(out.BestBidOffer, ch.BestBidOffer...)
		out.LivepriceV3 = append(out.LivepriceV3, ch.LivepriceV3...)
		out.OrderBookV3 = append(out.OrderBookV3, ch.OrderBookV3...)
		out.IntradayV3 = append(out.IntradayV3, ch.IntradayV3...)
	}
	return out
}
