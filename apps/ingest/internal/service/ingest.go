package service

import (
	"context"
	"time"

	"github.com/nofendian17/sbterm/apps/ingest/internal/infrastructure/kafka"
	"github.com/nofendian17/sbterm/libs/pkg/log"
)

// Service drains the pipeline topics and persists each record through the
// FrameHandler until Shutdown. Records that fail to decode or persist are
// logged and redelivered thanks to at-least-once semantics and QuestDB dedup.
type Service struct {
	consumer *kafka.Consumer
	handler  *FrameHandler
	logger   log.Logger

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

// NewService builds the ingestion loop.
func NewService(consumer *kafka.Consumer, handler *FrameHandler, logger log.Logger) *Service {
	return &Service{consumer: consumer, handler: handler, logger: logger}
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
		if errs := fetches.Errors(); len(errs) > 0 {
			for _, err := range errs {
				s.logger.Warn("kafka: fetch error", "error", err.Err)
			}
		}
		for _, rec := range fetches.Records() {
			if err := s.handler.Handle(s.ctx, rec.Topic, rec.Value); err != nil {
				s.logger.Warn("ingest: handle record", "topic", rec.Topic, "partition", rec.Partition, "offset", rec.Offset, "error", err)
			}
		}
		s.consumer.AllowRebalance()
	}
}

// Shutdown stops the poll loop and closes the handler sinks.
func (s *Service) Shutdown() error {
	if s.cancel != nil {
		s.cancel()
	}
	if s.done != nil {
		select {
		case <-s.done:
		case <-time.After(5 * time.Second):
			s.logger.Warn("ingest: poll loop did not stop within 5s")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.handler.Close(ctx)
}
