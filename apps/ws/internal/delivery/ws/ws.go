// Package ws runs one Stockbit datafeed websocket client per configured
// subscription for the lifetime of the server, publishing decoded frames to
// Kafka, and stops them on container shutdown.
package ws

import (
	"context"
	"sync"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/nofendian17/sbterm/apps/ws/internal/infrastructure/config"
	stockbitws "github.com/nofendian17/sbterm/apps/ws/internal/infrastructure/stockbit"
	"github.com/nofendian17/sbterm/libs/pkg/log"
	datafeedv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/v1"
	"github.com/nofendian17/sbterm/libs/stockbit"
)

// Subscription couples a dedicated datafeed websocket client with the channel
// that the connection subscribes to on connect.
type Subscription struct {
	Name    string
	Client  *stockbitws.WSClient
	Channel *datafeedv1.WebsocketChannel
}

// Service runs one Stockbit datafeed websocket client per subscription,
// routing every decoded frame to the ingestion pipeline through Kafka, and
// stops them all on container shutdown.
type Service struct {
	subs      []*Subscription
	refresher *stockbit.Refresher
	router    *FrameRouter
	logger    log.Logger

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

// New builds a Service around the configured subscriptions.
func New(subs []*Subscription, refresher *stockbit.Refresher, router *FrameRouter, logger log.Logger) *Service {
	return &Service{subs: subs, refresher: refresher, router: router, logger: logger}
}

// BuildChannel maps a channel config onto the corresponding datafeed channel
// by composing the per-service builders. Empty arrays subscribe nothing on
// that channel.
func BuildChannel(ch config.WSChannelConfig) *datafeedv1.WebsocketChannel {
	return stockbitws.MergeWSChannels(
		stockbitws.WSChannelWatchlist(ch.Watchlist...),
		stockbitws.WSChannelOrderBook(ch.OrderBook...),
		stockbitws.WSChannelRunningTrade(ch.RunningTrade...),
		stockbitws.WSChannelRunningTradeBatch(ch.RunningTradeBatch...),
		stockbitws.WSChannelLiveprice(ch.Liveprice...),
		stockbitws.WSChannelIepiev(ch.Iepiev...),
		stockbitws.WSChannelIntraday(ch.Intraday...),
		stockbitws.WSChannelBestBidOffer(ch.BestBidOffer...),
		stockbitws.WSChannelLivepriceV3(ch.LivepriceV3...),
		stockbitws.WSChannelOrderBookV3(ch.OrderBookV3...),
		stockbitws.WSChannelIntradayV3(ch.IntradayV3...),
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
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.run(sub, userID)
		}()
	}
	go func() {
		wg.Wait()
		close(s.done)
	}()
}

// run drives one subscription's connection until the shared context is
// cancelled.
func (s *Service) run(sub *Subscription, userID int64) {
	request := stockbitws.WSSubscription{UserID: userID, Channel: sub.Channel}

	err := sub.Client.Run(s.ctx, request, func(ctx context.Context, m *datafeedv1.WebsocketWrapMessageChannel) error {
		s.logger.Debug("stockbit ws frame", "subscription", sub.Name, "message", wsMessageJSON(m))
		if err := s.router.Route(ctx, m); err != nil {
			s.logger.Warn("ws: publish frame failed", "subscription", sub.Name, "error", err)
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
