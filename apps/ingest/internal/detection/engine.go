// Package detection implements the bandarmology signal evaluator: four
// deterministic rules over order book snapshots and the trade stream, with
// evidence-bearing alerts.
//
// Clock domain: every window, cooldown, and session gap operates on EXCHANGE
// event time (Book.ExchangeTS, Trade.TS). ReceiveTS is transport metadata and
// never enters the evaluator's arithmetic — mixing domains silently disables
// signals whenever one clock drifts past a window boundary. The evaluator is
// deterministic and replayable against recorded data.
//
// Signals:
//
//	PULL_BID/PULL_ASK — a fat level disappears without executions at its
//	price, repeatedly inside a lookback window.
//	ICEBERG           — a level is consumed by real trades and refilled to
//	near its reference size, n times.
//	AKUMULASI         — sustained net buying while passive bid support holds
//	and the mid barely moves.
//	DISTRIBUSI        — the mirror image for quiet selling.
package detection

import (
	"context"
	"strconv"
	"time"
)

// SignalType names an emitted bandarmology signal.
type SignalType string

const (
	SignalPullBid      SignalType = "PULL_BID"
	SignalPullAsk      SignalType = "PULL_ASK"
	SignalIceberg      SignalType = "ICEBERG"
	SignalAccumulation SignalType = "ACCUMULASI"
	SignalDistribution SignalType = "DISTRIBUSI"
)

// BookSide is one side of a snapshot: best-first price/quantity arrays.
type BookSide struct {
	Seq    int64
	Prices []float64
	Qtys   []int64
}

// Book pairs both sides of one symbol's book.
type Book struct {
	Symbol                string
	Bid, Ask              *BookSide
	ExchangeTS, ReceiveTS time.Time
}

// Trade is one executed trade feeding the net-flow windows.
type Trade struct {
	Symbol        string
	TS            time.Time
	Price, Volume float64
	Value         float64
	Buy           bool
}

// Alert is an emitted signal with its evidence payload.
type Alert struct {
	Symbol string
	Type   SignalType
	Side   string
	TS     time.Time
	Detail map[string]any
}

// Sink receives emitted alerts.
type Sink interface {
	Emit(ctx context.Context, a Alert) error
}

// Config carries every tunable threshold. Use DefaultConfig then adjust.
type Config struct {
	TopLevels      int           // book depth considered by pull/iceberg scans
	TradeBufferTTL time.Duration // temporal-join buffer for executions
	SessionGap     time.Duration // silence longer than this resets per-symbol state
	Cooldown       time.Duration // per symbol+signal anti-spam gate

	Pull struct {
		MinQty  int64         // level size worth caring about
		RepeatK int           // removal events needed inside Window
		Window  time.Duration // lookback for repeated removals
	}

	Iceberg struct {
		MinQty        int64   // refill size floor
		N             int     // refills before firing
		UniformityPct float64 // refill must match reference size within this percent
	}

	Accum struct {
		NetMin       float64       // net buy value over Window
		Window       time.Duration // net-flow window
		MidDriftMax  float64       // max |mid change| relative to baseline
		ConfirmFor   time.Duration // price-support must hold this long
		SupportGamma float64       // current bid depth must stay >= gamma * baseline
	}

	Distrib struct {
		NetMin       float64
		Window       time.Duration
		MidDriftMax  float64
		ConfirmFor   time.Duration
		SupportGamma float64
	}
}

// DefaultConfig returns the plan's starting thresholds; tune via config at
// the wiring layer, not here.
func DefaultConfig() Config {
	cfg := Config{
		TopLevels:      5,
		TradeBufferTTL: 90 * time.Second,
		SessionGap:     5 * time.Minute,
		Cooldown:       15 * time.Minute,
	}
	cfg.Pull.MinQty = 500
	cfg.Pull.RepeatK = 2
	cfg.Pull.Window = 10 * time.Minute

	cfg.Iceberg.MinQty = 300
	cfg.Iceberg.N = 3
	cfg.Iceberg.UniformityPct = 80

	cfg.Accum.NetMin = 100_000_000
	cfg.Accum.Window = 20 * time.Minute
	cfg.Accum.MidDriftMax = 0.01
	cfg.Accum.ConfirmFor = 5 * time.Minute
	cfg.Accum.SupportGamma = 1.0

	cfg.Distrib.NetMin = cfg.Accum.NetMin
	cfg.Distrib.Window = cfg.Accum.Window
	cfg.Distrib.MidDriftMax = cfg.Accum.MidDriftMax
	cfg.Distrib.ConfirmFor = cfg.Accum.ConfirmFor
	cfg.Distrib.SupportGamma = cfg.Accum.SupportGamma
	return cfg
}

// lvl tracks one watched price level across snapshots using its AGGREGATED
// quantity: an order refill is a large aggregate drop followed by a restore
// toward the pre-drop size, with executions proving real consumption.
type lvl struct {
	base     int64     // pre-drop reference quantity
	refills  int       // confirmed restorations
	consumed bool      // significant drop observed, awaiting restore
	execFrom time.Time // execution-evidence window starts here (pre-drop)
	lastSeen time.Time // timestamp of the snapshot used for phase anchor
}

// flowPt is one signed net-flow sample.
type flowPt struct {
	ts  time.Time
	val float64 // buy positive, sell negative
}

// biasTracker holds one bias signal's stability state. Accumulation and
// distribution each own a private tracker: shared baselines or hold timers
// let one side's instability rewrite the other's reference and reset its
// confirmation progress.
type biasTracker struct {
	active      bool
	baselineSet bool
	baseMid     float64 // reference mid when the quiet stretch began
	baseDep     float64 // reference depth on this signal's support side
	holdSince   time.Time
}

type symState struct {
	prevBid, prevAsk *BookSide
	prevTS           time.Time

	bidLv, askLv map[string]*lvl

	trades []Trade
	flow   []flowPt

	accum, dist biasTracker

	pullLogBid, pullLogAsk []time.Time
	lastEmit               map[SignalType]time.Time
}

// Engine evaluates the rule set. It is safe for one consumer goroutine per
// topic group; guard with external serialization if shared.
type Engine struct {
	cfg  Config
	sink Sink
	syms map[string]*symState
}

// NewEngine builds an evaluator.
func NewEngine(cfg Config, sink Sink) *Engine {
	return &Engine{cfg: cfg, sink: sink, syms: make(map[string]*symState)}
}

func (e *Engine) state(symbol string) *symState {
	st := e.syms[symbol]
	if st == nil {
		st = &symState{
			bidLv:    make(map[string]*lvl),
			askLv:    make(map[string]*lvl),
			lastEmit: make(map[SignalType]time.Time),
		}
		e.syms[symbol] = st
	}
	return st
}

func pxKey(p float64) string { return strconv.FormatFloat(p, 'f', -1, 64) }

// ObserveBook ingests one snapshot: pull detection, iceberg tracking, and
// accumulation/distribution evaluation all hang off it.
func (e *Engine) ObserveBook(ctx context.Context, b Book) error {
	if b.Symbol == "" || b.Bid == nil || b.Ask == nil {
		return nil
	}
	st := e.state(b.Symbol)
	// Single clock domain: exchange event time for every rule below.
	now := b.ExchangeTS
	if !st.prevTS.IsZero() && now.Sub(st.prevTS) > e.cfg.SessionGap {
		e.reset(st)
	}

	e.detectPull(st, b)
	e.trackIceberg(st, b, now)
	e.evaluateBias(ctx, st, b, now)

	st.prevBid, st.prevAsk = cloneSide(b.Bid), cloneSide(b.Ask)
	st.prevTS = now
	pruneTrades(st, now.Add(-e.cfg.TradeBufferTTL))
	return nil
}

// reset wipes per-symbol windows after a session gap so pre-gap flow never
// leaks into post-gap signals. Cooldowns survive: they are anti-spam gates.
func (e *Engine) reset(st *symState) {
	lastEmit := st.lastEmit
	*st = symState{
		bidLv:    make(map[string]*lvl),
		askLv:    make(map[string]*lvl),
		lastEmit: lastEmit,
	}
}

// ObserveTrade ingests one execution into the buffers and re-evaluates the
// bias signals.
func (e *Engine) ObserveTrade(ctx context.Context, t Trade) error {
	if t.Symbol == "" {
		return nil
	}
	st := e.state(t.Symbol)
	cutoff := t.TS.Add(-e.cfg.TradeBufferTTL)
	pruneTrades(st, cutoff)
	st.trades = append(st.trades, t)

	sign := -1.0
	if t.Buy {
		sign = 1.0
	}
	st.flow = append(st.flow, flowPt{ts: t.TS, val: sign * t.Value})
	e.evaluateBias(ctx, st, Book{Symbol: t.Symbol, ExchangeTS: t.TS, ReceiveTS: t.TS}, t.TS)
	return nil
}

// detectPull diffs the previous and current book per side, logging unexecuted
// fat-level removals and firing once the window holds enough of them.
func (e *Engine) detectPull(st *symState, b Book) {
	pairs := []struct {
		side   string
		typ    SignalType
		cur    *BookSide
		prev   *BookSide
		logRef *[]time.Time
	}{
		{"BID", SignalPullBid, b.Bid, st.prevBid, &st.pullLogBid},
		{"ASK", SignalPullAsk, b.Ask, st.prevAsk, &st.pullLogAsk},
	}
	for _, p := range pairs {
		if p.cur == nil || p.prev == nil {
			continue
		}
		cut := b.ExchangeTS.Add(-e.cfg.Pull.Window)
		log := (*p.logRef)[:0]
		for _, ts := range *p.logRef {
			if ts.After(cut) {
				log = append(log, ts)
			}
		}
		*p.logRef = log

		cur := levelMap(p.cur, e.cfg.TopLevels)
		prev := levelMap(p.prev, e.cfg.TopLevels)
		// Every qualifying removal counts as its own event, even when
		// several vanish inside one snapshot.
		events := 0
		for key, pq := range prev {
			if _, still := cur[key]; still {
				continue // partial drops are ignored in v1; only full removals
			}
			if pq < e.cfg.Pull.MinQty {
				continue
			}
			if st.executedBetween(pxKeyToFloat(key), st.prevTS, b.ReceiveTS) > 0 {
				continue // consumed by real demand — legitimate
			}
			events++
			*p.logRef = append(*p.logRef, b.ExchangeTS)
		}
		if events == 0 {
			continue
		}
		if len(*p.logRef) >= e.cfg.Pull.RepeatK && e.cooldownOK(st, p.typ, b.ExchangeTS) {
			detail := map[string]any{
				"events": len(*p.logRef), "window": e.cfg.Pull.Window.String(),
				"min_qty": e.cfg.Pull.MinQty,
			}
			e.emit(st, p.typ, b.Symbol, p.side, b.ExchangeTS, detail)
			*p.logRef = (*p.logRef)[:0]
		}
	}
}

// trackIceberg watches aggregated-level dynamics per side. A refill is
// counted when the aggregate quantity at a price drops sharply (real
// consumption evidenced by trades), then restores toward its pre-drop size.
func (e *Engine) trackIceberg(st *symState, b Book, now time.Time) {
	sides := []struct {
		name      string
		cur, prev *BookSide
		lv        map[string]*lvl
		typ       SignalType
	}{
		{"BID", b.Bid, st.prevBid, st.bidLv, SignalIceberg},
		{"ASK", b.Ask, st.prevAsk, st.askLv, SignalIceberg},
	}
	for _, s := range sides {
		if s.cur == nil || s.prev == nil {
			continue
		}
		cur := levelMap(s.cur, e.cfg.TopLevels)
		prev := levelMap(s.prev, e.cfg.TopLevels)
		tolLow := 1 - e.cfg.Iceberg.UniformityPct/100

		seen := make(map[string]bool, len(cur))
		for key, cq := range cur {
			seen[key] = true // survivors of this snapshot are never cleaned up
			pq := prev[key]  // 0 when newly appeared
			tr := s.lv[key]
			if tr == nil {
				tr = &lvl{lastSeen: now}
				// Born mid-cycle: a fresh tracker that already shows a sharp
				// aggregate drop must arm its consumed phase immediately.
				if pq > 0 && float64(cq) <= float64(pq)*iceDropRatio {
					tr.consumed = true
					tr.base = pq
					tr.execFrom = st.prevTS
				} else {
					tr.base = cq
				}
				s.lv[key] = tr
				continue
			}

			switch {
			case !tr.consumed && pq > 0 && float64(cq) <= float64(pq)*iceDropRatio:
				// Significant consumption of the aggregate level.
				tr.consumed = true
				tr.base = pq
				tr.execFrom = st.prevTS
				tr.lastSeen = now
			case tr.consumed && cq >= e.cfg.Iceberg.MinQty &&
				float64(cq) >= float64(tr.base)*tolLow &&
				float64(cq) <= float64(tr.base)*(1+tolLow) &&
				st.executedBetween(pxKeyToFloat(key), tr.execFrom, now) > 0:
				// Restored toward pre-drop size after real consumption.
				tr.refills++
				tr.consumed = false
				tr.base = cq
				if tr.refills >= e.cfg.Iceberg.N && e.cooldownOK(st, s.typ, b.ExchangeTS) {
					e.emit(st, s.typ, b.Symbol, s.name, b.ExchangeTS, map[string]any{
						"price": key, "refills": tr.refills, "base_qty": tr.base,
					})
					tr.refills = 0
				}
			default:
				tr.lastSeen = now
			}
		}
		// Prices absent from BOTH sides' current view age out of tracking.
		for key := range s.lv {
			if !seen[key] {
				delete(s.lv, key)
			}
		}
	}
}

const iceDropRatio = 0.5

// evaluateBias runs the akumulasi/distribusi composite over the rolling flow
// window using the current book (when present) for support and mid checks.
func (e *Engine) evaluateBias(ctx context.Context, st *symState, b Book, now time.Time) {
	mid, bidDepth, askDepth := currentBookShape(st, b)

	specs := []struct {
		tr       *biasTracker
		netMin   float64
		window   time.Duration
		driftMax float64
		confirm  time.Duration
		gamma    float64
		depth    float64
		typ      SignalType
		side     string
	}{
		{&st.accum, e.cfg.Accum.NetMin, e.cfg.Accum.Window, e.cfg.Accum.MidDriftMax,
			e.cfg.Accum.ConfirmFor, e.cfg.Accum.SupportGamma, bidDepth,
			SignalAccumulation, "BID"},
		{&st.dist, e.cfg.Distrib.NetMin, e.cfg.Distrib.Window, e.cfg.Distrib.MidDriftMax,
			e.cfg.Distrib.ConfirmFor, e.cfg.Distrib.SupportGamma, askDepth,
			SignalDistribution, "ASK"},
	}

	// Flow samples outlive the longest configured window only; every spec
	// then sums over its own window so independently tuned configs behave.
	maxWin := specs[0].window
	if specs[1].window > maxWin {
		maxWin = specs[1].window
	}
	st.pruneFlowTo(now.Add(-maxWin))
	for i := range specs {
		sp := &specs[i]
		tr := sp.tr
		buy, sell := st.flowSum(now, sp.window)
		net := buy
		if i == 1 {
			net = sell
		}
		if sp.depth <= 0 || mid <= 0 {
			continue // need a live book to judge stability
		}
		if !tr.baselineSet {
			tr.baseMid, tr.baseDep = mid, sp.depth
			tr.baselineSet = true
		}
		drift := absFloat(mid-tr.baseMid) / tr.baseMid
		support := sp.depth >= sp.gamma*tr.baseDep
		stable := drift <= sp.driftMax && support

		if stable {
			if tr.holdSince.IsZero() {
				tr.holdSince = now
			}
		} else {
			tr.holdSince = time.Time{}
			tr.active = false
			// Re-baseline THIS signal only, so one side's instability never
			// rewrites the other side's reference or resets its confirm timer.
			tr.baseMid, tr.baseDep = mid, sp.depth
		}

		held := !tr.holdSince.IsZero() && now.Sub(tr.holdSince) >= sp.confirm
		pressure := net >= sp.netMin
		if held && pressure {
			if e.cooldownOK(st, sp.typ, now) {
				e.emit(st, sp.typ, b.Symbol, sp.side, now, map[string]any{
					"net": net, "mid": mid, "drift": drift,
					"held_for": now.Sub(tr.holdSince).String(),
				})
				tr.active = true
			}
		}
		if net < sp.netMin {
			tr.active = false
		}
	}
}

// executedBetween sums traded volume at an exact price inside [from,to].
func (s *symState) executedBetween(price float64, from, to time.Time) float64 {
	sum := 0.0
	for _, t := range s.trades {
		if t.Price == price && !t.TS.Before(from) && !t.TS.After(to) {
			sum += t.Volume
		}
	}
	return sum
}

// pruneFlowTo drops flow samples at or before cutoff.
func (s *symState) pruneFlowTo(cutoff time.Time) {
	out := s.flow[:0]
	for _, pt := range s.flow {
		if pt.ts.After(cutoff) {
			out = append(out, pt)
		}
	}
	s.flow = out
}

// flowSum sums signed net flow inside [now-window, now] without mutating.
func (s *symState) flowSum(now time.Time, window time.Duration) (buy, sell float64) {
	cut := now.Add(-window)
	for _, pt := range s.flow {
		if !pt.ts.After(cut) {
			continue
		}
		if pt.val >= 0 {
			buy += pt.val
		} else {
			sell -= pt.val
		}
	}
	return buy, sell
}

func pruneTrades(st *symState, cutoff time.Time) {
	out := st.trades[:0]
	for _, t := range st.trades {
		if t.TS.After(cutoff) {
			out = append(out, t)
		}
	}
	st.trades = out
}

func (e *Engine) cooldownOK(st *symState, typ SignalType, now time.Time) bool {
	if last, ok := st.lastEmit[typ]; ok && now.Sub(last) < e.cfg.Cooldown {
		return false
	}
	return true
}

func (e *Engine) emit(st *symState, typ SignalType, symbol, side string, ts time.Time, detail map[string]any) {
	st.lastEmit[typ] = ts
	_ = e.sink.Emit(context.Background(), Alert{
		Symbol: symbol, Type: typ, Side: side, TS: ts, Detail: detail,
	})
}

// currentBookShape derives mid and depth sums from the freshest known sides,
// preferring the incoming snapshot's levels.
func currentBookShape(st *symState, b Book) (mid, bidDepth, askDepth float64) {
	var bidPx, askPx float64
	if b.Bid != nil && len(b.Bid.Prices) > 0 {
		bidPx = b.Bid.Prices[0]
		bidDepth = sumQtys(b.Bid.Qtys)
	} else if st.prevBid != nil && len(st.prevBid.Prices) > 0 {
		bidPx = st.prevBid.Prices[0]
		bidDepth = sumQtys(st.prevBid.Qtys)
	}
	if b.Ask != nil && len(b.Ask.Prices) > 0 {
		askPx = b.Ask.Prices[0]
		askDepth = sumQtys(b.Ask.Qtys)
	} else if st.prevAsk != nil && len(st.prevAsk.Prices) > 0 {
		askPx = st.prevAsk.Prices[0]
		askDepth = sumQtys(st.prevAsk.Qtys)
	}
	if bidPx > 0 && askPx > 0 {
		mid = (bidPx + askPx) / 2
	}
	return mid, bidDepth, askDepth
}

func sumQtys(qtys []int64) float64 {
	sum := 0.0
	for _, q := range qtys {
		sum += float64(q)
	}
	return sum
}

func levelMap(s *BookSide, top int) map[string]int64 {
	m := make(map[string]int64, len(s.Qtys))
	for i, p := range s.Prices {
		if i >= top {
			break
		}
		if i < len(s.Qtys) {
			m[pxKey(p)] = s.Qtys[i]
		}
	}
	return m
}

func cloneSide(s *BookSide) *BookSide {
	out := &BookSide{Seq: s.Seq, Prices: append([]float64(nil), s.Prices...), Qtys: append([]int64(nil), s.Qtys...)}
	return out
}

func pxKeyToFloat(key string) float64 {
	v, _ := strconv.ParseFloat(key, 64)
	return v
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
