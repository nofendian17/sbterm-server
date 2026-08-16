package container

import (
	"context"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/nofendian17/sbterm-server/internal/infrastructure/config"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit"
	datafeedv1 "github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit/proto/securities/transactional/datafeed/v1"
	"github.com/nofendian17/sbterm-server/pkg/log"
)

// wsService runs the Stockbit datafeed websocket client for the lifetime of
// the server and stops it on injector shutdown.
type wsService struct {
	client    *stockbit.WSClient
	refresher *stockbit.Refresher
	cfg       *config.Config
	logger    log.Logger

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

func newWSService(client *stockbit.WSClient, refresher *stockbit.Refresher, cfg *config.Config, logger log.Logger) *wsService {
	return &wsService{client: client, refresher: refresher, cfg: cfg, logger: logger}
}

// wsMessageJSON renders a decoded datafeed frame as JSON for debug logging,
// capped so a busy order-book feed does not flood the log.
func wsMessageJSON(m *datafeedv1.WebsocketWrapMessageChannel) string {
	out, err := protojson.Marshal(m)
	if err != nil {
		return "?"
	}
	const max = 4096
	if len(out) > max {
		out = append(out[:max], []byte("... (truncated)")...)
	}
	return string(out)
}

// start dials the datafeed and subscribes to the configured symbols in the
// background. It is idempotent.
func (s *wsService) start() {
	if s.cancel != nil {
		return
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.done = make(chan struct{})

	userID, err := s.refresher.UserID(context.Background())
	if err != nil {
		s.logger.Warn("container: resolve stockbit user id failed", "error", err)
	}
	sub := stockbit.WSSubscription{
		UserID:  userID,
		Channel: stockbit.WSChannelAll(s.cfg.Stockbit.WSSymbols...),
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
func (s *wsService) Shutdown() error {
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
