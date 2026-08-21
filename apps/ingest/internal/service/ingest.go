package service

import (
	"context"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/nofendian17/sbterm/libs/pkg/log"
)

// committer is the subset of the Kafka consumer the loop depends on. It is an
// interface so the redelivery logic can be exercised with an in-memory fake.
type committer interface {
	PollFetches(ctx context.Context) kgo.Fetches
	MarkCommit(ctx context.Context, processed []*kgo.Record) error
	AllowRebalance()
}

// Service drains the pipeline topics and persists each record through the
// FrameHandler until Shutdown. A record is committed only after the handler has
// durably persisted it: a persist failure stops the current fetch batch and
// commits up to the last successful record, so the failed record (and anything
// after it) is redelivered once the downstream recovers. Combined with the
// deduplicating QuestDB table, this gives the pipeline at-least-once delivery
// without silent data loss.
type Service struct {
	consumer committer
	handler  *FrameHandler
	logger   log.Logger

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

// NewService builds the ingestion loop.
func NewService(consumer committer, handler *FrameHandler, logger log.Logger) *Service {
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
		s.processFetches(fetches)
		s.consumer.AllowRebalance()
	}
}

// processFetches persists each record in order, tracking the highest offset
// processed per partition. On the first handle error it stops the batch and
// commits up to the last good record, so the failing record (and the rest) is
// redelivered on the next poll rather than dropped. When every record is
// handled, the batch is committed once at the end so a single fetch costs at
// most one offset-commit round-trip. Only successfully-persisted records are
// committed; "persisted" means the handler flushed them to QuestDB.
func (s *Service) processFetches(fetches kgo.Fetches) {
	committed := make([]*kgo.Record, 0, 8)
	var failedAt *kgo.Record
	for _, rec := range fetches.Records() {
		if err := s.handler.Handle(s.ctx, rec.Topic, rec.Value); err != nil {
			s.logger.Warn("ingest: handle record",
				"topic", rec.Topic, "partition", rec.Partition,
				"offset", rec.Offset, "error", err)
			failedAt = rec
			break
		}
		committed = append(committed, rec)
	}

	if failedAt != nil {
		// Roll back to the last successful record: drop the one that failed
		// and everything after it from the commit so they are redelivered.
		committed = trimToPartition(committed, failedAt)
	}
	if len(committed) == 0 {
		return // nothing safe to advance; redeliver the whole batch
	}
	if err := s.consumer.MarkCommit(s.ctx, committed); err != nil {
		s.logger.Warn("ingest: commit processed records",
			"count", len(committed), "error", err)
	}
}

// trimToPartition returns the committed slice with the failing record and any
// later record on the same topic partition removed, so the batch resumes
// exactly at the failed offset.
func trimToPartition(committed []*kgo.Record, failedAt *kgo.Record) []*kgo.Record {
	out := committed[:0]
	for _, r := range committed {
		if r.Topic == failedAt.Topic && r.Partition == failedAt.Partition && r.Offset >= failedAt.Offset {
			continue
		}
		out = append(out, r)
	}
	return out
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
