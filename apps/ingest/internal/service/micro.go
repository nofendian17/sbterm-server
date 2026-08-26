package service

import (
	"context"
	"fmt"
	"hash/fnv"
	"log/slog"
	"strconv"
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
	Combiner  *questdb.Combiner
	Store     BookStorer
	Persister BookPersister
	Engine    *detection.Engine
	Logger    log.Logger
}

// BookStorer is the hot-state slice used by the pipeline.
type BookStorer interface {
	SetBook(context.Context, hotstate.BookUpdate) error
	TouchBook(context.Context, string, time.Time) error
	SeenBefore(context.Context, string, string) (bool, error)
	MarkSeen(context.Context, string, string) error
}

// BookPersister is the durable slice used by the pipeline.
type BookPersister interface {
	Store(context.Context, *questdb.BookPair) error
	Close(context.Context) error
}

// BookPipeline is the production pipeline over the given deps.
type bookPipeline struct {
	combiner *questdb.Combiner
	store    BookStorer
	sink     BookPersister
	engine   *detection.Engine
	logger   log.Logger
}

// NewBookPipeline wires the production order book pipeline.
func NewBookPipeline(deps BookDeps) (BookPipeline, error) {
	if deps.Combiner == nil || deps.Store == nil || deps.Engine == nil || deps.Logger == nil {
		return nil, fmt.Errorf("service: book pipeline requires combiner, store, engine, and logger")
	}
	return &bookPipeline{
		combiner: deps.Combiner,
		store:    deps.Store,
		sink:     deps.Persister,
		engine:   deps.Engine,
		logger:   deps.Logger,
	}, nil
}

// Process runs one order book frame through the chain. QuestDB persistence
// errors propagate (the consumer redelivers); hot-state errors are logged and
// swallowed so a Redis outage never stalls durable ingestion.
func (p *bookPipeline) Process(ctx context.Context, ob *consumerv1.Orderbook) error {
	symbol := ob.GetStockCode()
	body := ob.GetBody()
	receiveTS := time.Now()

	seen, err := p.store.SeenBefore(ctx, symbol, bodyHash(body))
	if err != nil {
		p.logger.Warn("hotstate: seen check failed", "symbol", symbol, "error", err)
	} else if seen {
		return p.store.TouchBook(ctx, symbol, receiveTS)
	}

	pair, _ := p.combiner.Observe(ob, receiveTS)
	if terr := p.store.TouchBook(ctx, symbol, receiveTS); terr != nil {
		p.logger.Warn("hotstate: touch failed", "symbol", symbol, "error", terr)
	}
	if pair == nil {
		return nil // half snapshot or stale replay
	}
	if merr := p.store.MarkSeen(ctx, symbol, bodyHash(body)); merr != nil {
		p.logger.Warn("hotstate: mark seen failed", "symbol", symbol, "error", merr)
	}

	if uerr := p.store.SetBook(ctx, updateFrom(pair)); uerr != nil {
		p.logger.Warn("hotstate: set book failed", "symbol", symbol, "error", uerr)
	}

	if p.sink != nil {
		if serr := p.sink.Store(ctx, pair); serr != nil {
			return fmt.Errorf("service: persist order book pair: %w", serr)
		}
	}

	b := detection.Book{
		Symbol:     pair.Symbol,
		Bid:        convertSide(pair.Bid),
		Ask:        convertSide(pair.Ask),
		ExchangeTS: pair.ExchangeTS,
		ReceiveTS:  pair.ReceiveTS,
	}
	return p.engine.ObserveBook(ctx, b)
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
