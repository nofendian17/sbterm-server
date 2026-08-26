package service

import (
	"context"
	"fmt"
	"hash/fnv"
	"log/slog"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nofendian17/sbterm/libs/pkg/log"
	consumerv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/consumer/entity/v1"
	datafeedv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/v1"

	"github.com/nofendian17/sbterm/apps/ingest/internal/detection"
	"github.com/nofendian17/sbterm/apps/ingest/internal/infrastructure/hotstate"
	"github.com/nofendian17/sbterm/apps/ingest/internal/infrastructure/questdb"
)

// TradeFeed is one execution fed into the detector.
type TradeFeed = detection.Trade

// BookPipeline consumes decoded order book frames end-to-end: dedup, side
// combining, hot-state mirroring, persistence, and signal evaluation.
type BookPipeline interface {
	Process(ctx context.Context, ob *consumerv1.Orderbook) error
	ObserveTrade(ctx context.Context, t detection.Trade) error
	Pending() int
}

// TradeObserver receives every decoded running trade.
type TradeObserver interface {
	ObserveTrade(ctx context.Context, t TradeFeed) error
}

// LivenessToucher refreshes per-symbol presence without rewriting state.
type LivenessToucher interface {
	TouchBook(ctx context.Context, symbol string, ts time.Time) error
}

// HandlerOption augments the frame handler with optional pipelines.
type HandlerOption func(*FrameHandler)

// WithBookPipeline attaches the order book pipeline.
func WithBookPipeline(bp BookPipeline) HandlerOption {
	return func(h *FrameHandler) { h.bookPipe = bp }
}

// WithTradeObserver attaches the trade observer.
func WithTradeObserver(to TradeObserver) HandlerOption {
	return func(h *FrameHandler) { h.tradeObserver = to }
}

// WithLiveness attaches the touch provider for auxiliary channels.
func WithLiveness(l LivenessToucher) HandlerOption {
	return func(h *FrameHandler) { h.liveness = l }
}

// BookDeps bundles the collaborators the default pipeline needs.
type BookDeps struct {
	Store     BookStorer
	Persister BookPersister
	Logger    log.Logger
	// EngineFactory builds one detector per shard worker.
	EngineFactory func() *detection.Engine
	// Workers adalah jumlah shard; 0 berarti 1. Simbol dipetakan stabil ke
	// shard via hash sehingga urutan antar-frame satu simbol terjaga.
	Workers int
	// MinPersistInterval throttles durable writes per symbol; zero disables
	// the cap. The detector and hot state still observe every snapshot.
	MinPersistInterval time.Duration
}

// BookStorer is the hot-state slice used by the pipeline.
type BookStorer interface {
	SetBook(context.Context, hotstate.BookUpdate) error
	TouchBook(context.Context, string, time.Time) error
	DedupBook(context.Context, string, string) (bool, error)
}

// BookPersister is the durable slice used by the pipeline.
type BookPersister interface {
	Store(context.Context, *questdb.BookPair) error
	Close(context.Context) error
}

// BookPipeline is the production pipeline over the given deps.
type bookShard struct {
	combiner    *questdb.Combiner
	engine      *detection.Engine
	lastPersist map[string]time.Time
	ch          chan shardJob
}

type bookPipeline struct {
	store      BookStorer
	persister  BookPersister
	logger     log.Logger
	minPersist time.Duration
	shards     []bookShard
	drops      atomic.Uint64

	done      chan struct{} // closed once by Shutdown
	wg        sync.WaitGroup
	closeOnce sync.Once
}

type shardJob struct {
	ob    *consumerv1.Orderbook
	trade *detection.Trade
	recv  time.Time
}

// NewBookPipeline wires the production order book pipeline.
func NewBookPipeline(deps BookDeps) (BookPipeline, error) {
	if deps.EngineFactory == nil || deps.Store == nil || deps.Logger == nil {
		return nil, fmt.Errorf("service: book pipeline requires engine factory, store, and logger")
	}
	w := deps.Workers
	if w <= 0 {
		w = 1
	}
	p := &bookPipeline{
		store:      deps.Store,
		persister:  deps.Persister,
		logger:     deps.Logger,
		minPersist: deps.MinPersistInterval,
	}
	p.done = make(chan struct{})
	for i := 0; i < w; i++ {
		shard := bookShard{
			combiner:    questdb.NewCombiner(25),
			engine:      deps.EngineFactory(),
			lastPersist: make(map[string]time.Time),
			ch:          make(chan shardJob, 2048),
		}
		p.shards = append(p.shards, shard)
		p.wg.Add(1)
		go p.shardLoop(&p.shards[i])
	}
	return p, nil
}

// Process dispatches one order book frame to its symbol's shard. It never
// blocks on downstream work: shards own their queues and drop on overflow.
func (p *bookPipeline) Process(_ context.Context, ob *consumerv1.Orderbook) error {
	symbol := ob.GetStockCode()
	if symbol == "" {
		return nil
	}
	idx := shardIndex(symbol, len(p.shards))
	job := shardJob{ob: ob, recv: time.Now()}
	select {
	case <-p.done:
		return nil // shutting down: frame dropped like any overflow
	case p.shards[idx].ch <- job:
	default:
		if n := p.drops.Add(1); n%1000 == 1 {
			p.logger.Warn("book shard queue full; dropping frames", "dropped", n, "shard", idx)
		}
	}
	return nil
}

// ObserveTrade feeds one execution to the trade's symbol shard. Like book
// frames it is enqueued rather than applied inline: the shard worker owns the
// engine exclusively, so trades and books for one symbol are serialized and
// share the same bounded-queue backpressure.
func (p *bookPipeline) ObserveTrade(_ context.Context, t detection.Trade) error {
	if t.Symbol == "" {
		return nil
	}
	idx := shardIndex(t.Symbol, len(p.shards))
	select {
	case <-p.done:
		return nil // shutting down: trade dropped like any overflow
	case p.shards[idx].ch <- shardJob{trade: &t}:
	default:
		if n := p.drops.Add(1); n%1000 == 1 {
			p.logger.Warn("book shard queue full; dropping frames", "dropped", n, "shard", idx)
		}
	}
	return nil
}

func shardIndex(symbol string, n int) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(symbol))
	return int(h.Sum32() % uint32(n))
}

// shardLoop is one worker: dedup → combine → hot state → persist (throttled)
// → evaluate. Everything here touches only this shard's symbols.
func (p *bookPipeline) shardLoop(sh *bookShard) {
	defer p.wg.Done()
	for {
		select {
		case <-p.done:
			return
		case job := <-sh.ch:
			p.processOne(sh, job)
		}
	}
}

// Shutdown stops the shard workers and releases the durable sink. Workers
// finish their in-flight job; queued frames are abandoned (market data is
// worthless once stale). Implements do.ShutdownerWithContextAndError so a
// samber/do scope calls it automatically. Nil pipelines (feature disabled)
// and repeated calls are safe.
func (p *bookPipeline) Shutdown(ctx context.Context) error {
	if p == nil {
		return nil
	}
	p.closeOnce.Do(func() { close(p.done) })

	waited := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(waited)
	}()
	select {
	case <-waited:
	case <-ctx.Done():
		return fmt.Errorf("service: book pipeline shutdown timed out: %w", ctx.Err())
	}

	if p.persister != nil {
		return p.persister.Close(ctx)
	}
	return nil
}

func (p *bookPipeline) processOne(sh *bookShard, job shardJob) {
	if job.trade != nil {
		if err := sh.engine.ObserveTrade(context.Background(), *job.trade); err != nil {
			p.logger.Warn("detection: trade observation failed", "symbol", job.trade.Symbol, "error", err)
		}
		return
	}

	ob := job.ob
	symbol := ob.GetStockCode()

	dup, err := p.store.DedupBook(context.Background(), symbol, bodyHash(ob.GetBody()))
	if err != nil {
		p.logger.Warn("hotstate: dedup failed", "symbol", symbol, "error", err)
	} else if dup {
		return
	}

	pair, _ := sh.combiner.Observe(ob, job.recv)
	if pair == nil {
		return
	}
	if uerr := p.store.SetBook(context.Background(), updateFrom(pair)); uerr != nil {
		p.logger.Warn("hotstate: set book failed", "symbol", symbol, "error", uerr)
	}
	if p.persister != nil && p.persistOK(sh, symbol, time.Now()) {
		ctx, cancel := context.WithTimeout(context.Background(), handleTimeout)
		if serr := p.persister.Store(ctx, pair); serr != nil {
			cancel()
			p.logger.Warn("questdb: async order book write failed", "symbol", symbol, "error", serr)
			return
		}
		cancel()
	}
	bk := detection.Book{
		Symbol:     pair.Symbol,
		Bid:        convertSide(pair.Bid),
		Ask:        convertSide(pair.Ask),
		ExchangeTS: pair.ExchangeTS,
		ReceiveTS:  pair.ReceiveTS,
	}
	_ = sh.engine.ObserveBook(context.Background(), bk)
}

// Pending reports how many frames await processing across shards. A shut
// down pipeline always reports zero: its queues are abandoned by design.
func (p *bookPipeline) Pending() int {
	select {
	case <-p.done:
		return 0
	default:
	}
	n := 0
	for i := range p.shards {
		n += len(p.shards[i].ch)
	}
	return n
}

// persistOK reports whether enough time has passed since this symbol's last
// durable write. Skipped snapshots still reach the detector and hot state.
func (p *bookPipeline) persistOK(sh *bookShard, symbol string, now time.Time) bool {
	if p.minPersist <= 0 {
		return true
	}
	if last, ok := sh.lastPersist[symbol]; ok && now.Sub(last) < p.minPersist {
		return false
	}
	sh.lastPersist[symbol] = now
	return true
}

func updateFrom(pair *questdb.BookPair) hotstate.BookUpdate {
	u := hotstate.BookUpdate{
		Symbol:    pair.Symbol,
		Board:     pair.Board.String(),
		ReceiveTS: pair.ReceiveTS,
	}
	if pair.Bid != nil {
		u.Bid = &hotstate.Side{Seq: pair.Bid.Seq, Prices: pair.Bid.Prices, Qtys: pair.Bid.Qtys}
	}
	if pair.Ask != nil {
		u.Ask = &hotstate.Side{Seq: pair.Ask.Seq, Prices: pair.Ask.Prices, Qtys: pair.Ask.Qtys}
	}
	return u
}

func convertSide(h *questdb.BookHalf) *detection.BookSide {
	if h == nil {
		return nil
	}
	return &detection.BookSide{Seq: h.Seq, Prices: h.Prices, Qtys: h.Qtys}
}

// bodyHash fingerprints an order book body for cross-restart dedup.
func bodyHash(body string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(body))
	return strconv.FormatUint(h.Sum64(), 16)
}

// nopLogger discards everything; tests and disabled pipelines use it.
type nopLogger struct{}

func (nopLogger) Enabled(context.Context, log.Level) bool      { return false }
func (n nopLogger) Slog() *slog.Logger                         { return slog.Default() }
func (nopLogger) Debug(string, ...any)                         {}
func (nopLogger) DebugContext(context.Context, string, ...any) {}
func (nopLogger) Info(string, ...any)                          {}
func (nopLogger) InfoContext(context.Context, string, ...any)  {}
func (nopLogger) Warn(string, ...any)                          {}
func (nopLogger) WarnContext(context.Context, string, ...any)  {}
func (nopLogger) Error(string, ...any)                         {}
func (nopLogger) ErrorContext(context.Context, string, ...any) {}
func (n nopLogger) With(...any) log.Logger                     { return n }
func (n nopLogger) WithGroup(string) log.Logger                { return n }

var (
	_ log.Logger   = nopLogger{}
	_ BookPipeline = (*bookPipeline)(nil)
)

// tradeToFeed converts a decoded proto trade into the detector input.
func tradeToFeed(tr *datafeedv1.RunningTrade, fallback time.Time) TradeFeed {
	ts := fallback
	if t := tr.GetTime(); t != nil {
		ts = t.AsTime()
	}
	return TradeFeed{
		Symbol: tr.GetStock(),
		TS:     ts,
		Price:  tr.GetPrice(),
		Volume: tr.GetVolume(),
		Value:  tr.GetValue(),
		Buy:    tr.GetAction() == datafeedv1.TradeType_TRADE_TYPE_BUY,
	}
}
