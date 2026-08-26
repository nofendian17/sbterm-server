package service

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nofendian17/sbterm/apps/ingest/internal/detection"
	"github.com/nofendian17/sbterm/apps/ingest/internal/infrastructure/questdb"
	consumerv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/consumer/entity/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// blockingPersister lets the test hold one durable write mid-flight so
// shutdown semantics around an in-flight job are observable.
type blockingPersister struct {
	entered  chan struct{}
	release  chan struct{}
	n        atomic.Int64
	finished atomic.Int64
}

func newBlockingPersister() *blockingPersister {
	return &blockingPersister{entered: make(chan struct{}, 8), release: make(chan struct{})}
}

func (b *blockingPersister) Store(context.Context, *questdbBookPair) error {
	b.entered <- struct{}{}
	<-b.release
	b.n.Add(1)
	b.finished.Add(1)
	return nil
}
func (b *blockingPersister) Close(context.Context) error { return nil }

func TestPipelineShutdownStopsWorkersAndDropsLateFrames(t *testing.T) {
	pipe := newTestPipeline(t)

	require.NoError(t, pipe.Process(context.Background(), bookFrame("BBCA", "BID", "7750;1;100", 1)))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, pipe.Shutdown(ctx), "shutdown must wait for workers")

	// Late frames after shutdown: dropped silently, never a panic, and the
	// dead queues stay empty so Pending() remains meaningful.
	require.NoError(t, pipe.Process(ctx, bookFrame("BBCA", "BID", "7750;1;100", 2)))
	require.NoError(t, pipe.ObserveTrade(ctx, detection.Trade{Symbol: "BBCA"}))
	assert.Equal(t, 0, pipe.Pending())

	// Idempotent: second shutdown is a no-op success.
	require.NoError(t, pipe.Shutdown(ctx))
}

func TestPipelineShutdownWaitsForInFlightPersist(t *testing.T) {
	pers := newBlockingPersister()
	store := &fakeStore{}
	pipe := mustPipeline(t, store, pers)

	// One paired frame -> exactly one in-flight Store call.
	require.NoError(t, pipe.Process(context.Background(), bookFrame("BBCA", "BID", "7750;1;100", 1)))
	require.NoError(t, pipe.Process(context.Background(), bookFrame("BBCA", "OFFER", "7800;1;50", 2)))
	select {
	case <-pers.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("worker never reached the persister")
	}

	done := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		done <- pipe.Shutdown(ctx)
	}()

	// While the persist is blocked, shutdown must NOT complete.
	select {
	case err := <-done:
		t.Fatalf("shutdown returned before the in-flight job finished: %v", err)
	case <-time.After(80 * time.Millisecond):
	}

	close(pers.release)
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown never returned after the job finished")
	}
	assert.EqualValues(t, 1, pers.n.Load())
	assert.EqualValues(t, 0, pipe.Pending())
}

func TestNilPipelineShutdownIsSafe(t *testing.T) {
	var pipe *bookPipeline
	require.NoError(t, pipe.Shutdown(context.Background()))
}

// TestPipelineConcurrentTrafficDuringShutdown pins panic-freedom when senders
// race the shutdown signal; run under -race.
func TestPipelineConcurrentTrafficDuringShutdown(t *testing.T) {
	pipe := newTestPipeline(t)

	var wg sync.WaitGroup
	stop := make(chan struct{})
	for i := 0; i < 4; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				_ = pipe.Process(context.Background(), bookFrame("BBCA", "BID", "7750;1;100", 1))
			}
		}()
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				_ = pipe.ObserveTrade(context.Background(), detection.Trade{Symbol: "BBCA"})
			}
		}()
	}

	time.Sleep(50 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, pipe.Shutdown(ctx))
	close(stop)
	wg.Wait()

	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	require.NoError(t, pipe.Shutdown(ctx2))
}

type questdbBookPair = questdb.BookPair

func bookFrame(sym, side, level string, seq int64) *consumerv1.Orderbook {
	return &consumerv1.Orderbook{StockCode: sym, Body: "#O|" + sym + "|" + side + "|" + level, SequenceNumber: seq}
}

func mustPipeline(t *testing.T, store BookStorer, pers BookPersister) *bookPipeline {
	t.Helper()
	bp, err := NewBookPipeline(BookDeps{
		Store:     store,
		Persister: pers,
		Logger:    nopLogger{},
		EngineFactory: func() *detection.Engine {
			return detection.NewEngine(detection.DefaultConfig(), nopAlertSink{})
		},
	})
	require.NoError(t, err)
	return bp.(*bookPipeline)
}

func newTestPipeline(t *testing.T) *bookPipeline {
	return mustPipeline(t, &fakeStore{}, nil)
}
