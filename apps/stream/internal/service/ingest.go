package service

import (
	"context"
	"encoding/json"

	"github.com/twmb/franz-go/pkg/kgo"
	"google.golang.org/protobuf/proto"

	"github.com/nofendian17/sbterm/libs/marketdata"
	"github.com/nofendian17/sbterm/libs/pkg/log"
	datafeedv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/v1"
)

// poller is the subset of the Kafka consumer the loop depends on. It is an
// interface so the loop can be exercised with an in-memory fake.
type poller interface {
	PollFetches(ctx context.Context) kgo.Fetches
}

// runningTradeEnvelope is the wire shape pushed to clients on the
// running_trade channel; Type mirrors ChannelRunningTrade. Data carries the
// canonical marketdata.Trade projection, so a client envelope matches a
// QuestDB running_trades row field-for-field.
type runningTradeEnvelope struct {
	Type   string             `json:"type"`
	Symbol string             `json:"symbol"`
	Data   []marketdata.Trade `json:"data"`
}

// Service polls Kafka and fans every decodable running trade batch out to the
// hub. A record that fails to decode is logged and skipped: fan-out is best
// effort by design — durability belongs to apps/ingest, and one poisoned
// record must not wedge live delivery for every connected client.
type Service struct {
	consumer poller
	hub      *Hub
	topic    string
	logger   log.Logger

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

// NewService builds the fan-out loop bound to one topic.
func NewService(consumer poller, hub *Hub, topic string, logger log.Logger) *Service {
	return &Service{consumer: consumer, hub: hub, topic: topic, logger: logger}
}

// Start launches the poll loop. It is idempotent.
func (s *Service) Start() {
	if s.cancel != nil {
		return
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.done = make(chan struct{})
	go func() {
		defer close(s.done)
		s.run()
	}()
}

func (s *Service) run() {
	for {
		if s.ctx.Err() != nil {
			return
		}
		fetches := s.consumer.PollFetches(s.ctx)
		if fetches.IsClientClosed() {
			return
		}
		for _, err := range fetches.Errors() {
			s.logger.Warn("kafka: fetch error", "error", err.Err)
		}
		s.processFetches(fetches)
	}
}

// processFetches hands each record to the hub. Unlike apps/ingest there is no
// commit step: offsets are never committed, records are never retried.
func (s *Service) processFetches(fetches kgo.Fetches) {
	for _, rec := range fetches.Records() {
		s.handleRecord(s.ctx, rec.Topic, rec.Value)
	}
}

// handleRecord decodes one Kafka record and broadcasts it. Every failure path
// logs and returns; none of them stops the loop or the batch.
func (s *Service) handleRecord(_ context.Context, topic string, value []byte) {
	if topic != s.topic {
		s.logger.Warn("stream: unexpected topic", "topic", topic)
		return
	}
	batch := &datafeedv1.RunningTradeBatch{}
	if err := proto.Unmarshal(value, batch); err != nil {
		s.logger.Warn("stream: decode running trade batch", "error", err)
		return
	}
	trades := batch.GetBatch()
	if len(trades) == 0 {
		return // nothing observable for clients
	}
	symbol := trades[0].GetStock()
	payload, err := json.Marshal(runningTradeEnvelope{
		Type:   string(ChannelRunningTrade),
		Symbol: symbol,
		Data:   marketdata.NewTrades(trades),
	})
	if err != nil {
		s.logger.Warn("stream: marshal running trade envelope", "error", err)
		return
	}
	s.hub.Broadcast(ChannelRunningTrade, symbol, payload)
}

// Shutdown stops the poll loop and waits for it to exit. The wait yields to
// ctx so the container can enforce the spec's total shutdown budget (≤5s)
// across loop, HTTP drain, and hub close combined.
func (s *Service) Shutdown(ctx context.Context) error {
	if s.cancel != nil {
		s.cancel()
	}
	if s.done != nil {
		select {
		case <-s.done:
		case <-ctx.Done():
			s.logger.Warn("stream: poll loop did not stop within shutdown budget")
		}
	}
	return nil
}
