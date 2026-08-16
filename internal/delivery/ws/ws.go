// Package ws runs the Stockbit datafeed websocket client for the lifetime of
// the server and stops it on container shutdown.
package ws

import (
	"context"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/nofendian17/sbterm-server/internal/infrastructure/config"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit"
	datafeedv1 "github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit/proto/securities/transactional/datafeed/v1"
	"github.com/nofendian17/sbterm-server/pkg/log"
)

// Service runs the Stockbit datafeed websocket client for the lifetime of the
// server and stops it on container shutdown.
type Service struct {
	client    *stockbit.WSClient
	refresher *stockbit.Refresher
	cfg       *config.Config
	logger    log.Logger

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

// New builds a Service around an already-configured datafeed client.
func New(client *stockbit.WSClient, refresher *stockbit.Refresher, cfg *config.Config, logger log.Logger) *Service {
	return &Service{client: client, refresher: refresher, cfg: cfg, logger: logger}
}

// wsMessageJSON renders a decoded datafeed frame as JSON for debug logging.
func wsMessageJSON(m *datafeedv1.WebsocketWrapMessageChannel) string {
	out, err := protojson.Marshal(m)
	if err != nil {
		return "?"
	}
	return string(out)
}

// subscribeAllSymbols subscribes the given symbols on every symbol-array
// datafeed service (watchlist, order book, running trade, live price, best bid
// offer, and their v3 variants) by composing the per-service builders.
func subscribeAllSymbols(symbols ...string) *datafeedv1.WebsocketChannel {
	return stockbit.MergeWSChannels(
		stockbit.WSChannelWatchlist(symbols...),
		stockbit.WSChannelOrderBook(symbols...),
		stockbit.WSChannelRunningTrade(symbols...),
		stockbit.WSChannelRunningTradeBatch(symbols...),
		stockbit.WSChannelLiveprice(symbols...),
		stockbit.WSChannelIepiev(symbols...),
		stockbit.WSChannelIntraday(symbols...),
		stockbit.WSChannelBestBidOffer(symbols...),
		stockbit.WSChannelLivepriceV3(symbols...),
		stockbit.WSChannelOrderBookV3(symbols...),
		stockbit.WSChannelIntradayV3(symbols...),
	)
}

// Start dials the datafeed and subscribes to the configured symbols in the
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
	sub := stockbit.WSSubscription{
		UserID:  userID,
		Channel: subscribeAllSymbols(s.cfg.Stockbit.WSSymbols...),
	}

	go func() {
		defer close(s.done)
		if err := s.client.Run(s.ctx, sub, func(ctx context.Context, m *datafeedv1.WebsocketWrapMessageChannel) error {
			s.logger.Debug("stockbit ws frame", "message", wsMessageJSON(m))
			return nil
		}); err != nil {
			s.logger.Warn("stockbit ws client stopped", "error", err)
		}
	}()
}

// Shutdown cancels the run context and waits for the client to stop.
func (s *Service) Shutdown() error {
	if s.cancel != nil {
		s.cancel()
	}
	if s.done != nil {
		select {
		case <-s.done:
		case <-time.After(5 * time.Second):
			s.logger.Warn("stockbit ws client did not stop within 5s")
		}
	}
	return nil
}
