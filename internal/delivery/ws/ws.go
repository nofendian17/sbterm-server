// Package ws runs one Stockbit datafeed websocket client per configured
// subscription for the lifetime of the server and stops them on container
// shutdown.
package ws

import (
	"context"
	"sync"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/nofendian17/sbterm-server/internal/infrastructure/config"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/questdb"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit"
	datafeedv1 "github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit/proto/securities/transactional/datafeed/v1"
	"github.com/nofendian17/sbterm-server/pkg/log"
)

// Subscription couples a dedicated datafeed websocket client with the channel
// that the connection subscribes to on connect.
type Subscription struct {
	Name    string
	Client  *stockbit.WSClient
	Channel *datafeedv1.WebsocketChannel
}

// Service runs one Stockbit datafeed websocket client per subscription and
// stops them all on container shutdown.
type Service struct {
	subs      []*Subscription
	refresher *stockbit.Refresher
	store     questdb.RunningTradeBatchStore
	obStore   questdb.OrderBookStore
	logger    log.Logger

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

// New builds a Service around the configured subscriptions.
func New(subs []*Subscription, refresher *stockbit.Refresher, store questdb.RunningTradeBatchStore, obStore questdb.OrderBookStore, logger log.Logger) *Service {
	return &Service{subs: subs, refresher: refresher, store: store, obStore: obStore, logger: logger}
}

// BuildChannel maps a channel config onto the corresponding datafeed channel
// by composing the per-service builders. Empty arrays subscribe nothing on
// that channel.
func BuildChannel(ch config.WSChannelConfig) *datafeedv1.WebsocketChannel {
	return stockbit.MergeWSChannels(
		stockbit.WSChannelWatchlist(ch.Watchlist...),
		stockbit.WSChannelOrderBook(ch.OrderBook...),
		stockbit.WSChannelRunningTrade(ch.RunningTrade...),
		stockbit.WSChannelRunningTradeBatch(ch.RunningTradeBatch...),
		stockbit.WSChannelLiveprice(ch.Liveprice...),
		stockbit.WSChannelIepiev(ch.Iepiev...),
		stockbit.WSChannelIntraday(ch.Intraday...),
		stockbit.WSChannelBestBidOffer(ch.BestBidOffer...),
		stockbit.WSChannelLivepriceV3(ch.LivepriceV3...),
		stockbit.WSChannelOrderBookV3(ch.OrderBookV3...),
		stockbit.WSChannelIntradayV3(ch.IntradayV3...),
	)
}

// wsMessageJSON renders a decoded datafeed frame as JSON for debug logging.
func wsMessageJSON(m *datafeedv1.WebsocketWrapMessageChannel) string {
	out, err := protojson.Marshal(m)
	if err != nil {
		return "?"
	}
	return string(out)
}

// Start dials a websocket connection for every configured subscription in the
// background. It is idempotent.
func (s *Service) Start() {
	if s.cancel != nil {
		return
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.done = make(chan struct{})

	userID, err := s.refresher.UserID(context.Background())
	if err != nil {
		s.logger.Warn("ws: resolve stockbit user id failed", "error", err)
	}

	var wg sync.WaitGroup
	for _, sub := range s.subs {
		wg.Go(func() {
			s.run(sub, userID)
		})
	}
	go func() {
		wg.Wait()
		close(s.done)
	}()
}

// run drives one subscription's connection until the shared context is
// cancelled.
func (s *Service) run(sub *Subscription, userID int64) {
	request := stockbit.WSSubscription{UserID: userID, Channel: sub.Channel}

	sink, err := s.store.NewRunningTradeBatchSink(s.ctx)
	if err != nil {
		s.logger.Warn("questdb sink: borrow running trade sink failed", "subscription", sub.Name, "error", err)
		return
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := sink.Close(closeCtx); err != nil {
			s.logger.Warn("questdb sink: close running trade sink failed", "subscription", sub.Name, "error", err)
		}
	}()

	obSink, err := s.obStore.NewOrderBookSink(s.ctx)
	if err != nil {
		s.logger.Warn("questdb sink: borrow order book sink failed", "subscription", sub.Name, "error", err)
		return
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := obSink.Close(closeCtx); err != nil {
			s.logger.Warn("questdb sink: close order book sink failed", "subscription", sub.Name, "error", err)
		}
	}()

	err = sub.Client.Run(s.ctx, request, func(ctx context.Context, m *datafeedv1.WebsocketWrapMessageChannel) error {
		s.logger.Debug("stockbit ws frame", "subscription", sub.Name, "message", wsMessageJSON(m))
		if batch := m.GetRunningTradeBatch(); batch != nil {
			if err := sink.Store(ctx, batch); err != nil {
				s.logger.Warn("questdb sink: store running trade batch failed", "subscription", sub.Name, "error", err)
			}
		}
		if ob := m.GetOrderbook(); ob != nil {
			if err := obSink.Store(ctx, ob); err != nil {
				s.logger.Warn("questdb sink: store order book failed", "subscription", sub.Name, "error", err)
			}
		}
		return nil
	})
	if err != nil {
		s.logger.Warn("stockbit ws client stopped", "subscription", sub.Name, "error", err)
	}
}

// Shutdown cancels the run context and waits for every client to stop.
func (s *Service) Shutdown() error {
	if s.cancel != nil {
		s.cancel()
	}
	if s.done != nil {
		select {
		case <-s.done:
		case <-time.After(5 * time.Second):
			s.logger.Warn("stockbit ws clients did not stop within 5s")
		}
	}
	return nil
}
