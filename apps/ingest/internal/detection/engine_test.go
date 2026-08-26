package detection

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- fixtures -------------------------------------------------------------

var base = time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC)

type captureSink struct {
	alerts []Alert
	nowFn  func() time.Time
}

func (c *captureSink) Emit(_ context.Context, a Alert) error {
	c.alerts = append(c.alerts, a)
	return nil
}

func side(seq int64, pairs ...[2]int64) *BookSide {
	s := &BookSide{Seq: seq}
	for _, p := range pairs {
		s.Prices = append(s.Prices, float64(p[0]))
		s.Qtys = append(s.Qtys, p[1])
	}
	return s
}

func book(symbol string, bid, ask *BookSide, at time.Time) Book {
	return Book{Symbol: symbol, Bid: bid, Ask: ask, ExchangeTS: at, ReceiveTS: at}
}

func trade(symbol string, at time.Time, price, vol float64, buy bool) Trade {
	return Trade{Symbol: symbol, TS: at, Price: price, Volume: vol, Value: price * vol, Buy: buy}
}

func defaultCfg() Config {
	cfg := DefaultConfig()
	// Deterministic small-scale defaults for tests.
	cfg.Pull.MinQty = 500
	cfg.Pull.RepeatK = 2
	cfg.Pull.Window = 10 * time.Minute
	cfg.Iceberg.MinQty = 300
	cfg.Iceberg.N = 3
	cfg.Iceberg.UniformityPct = 80
	cfg.Accum.NetMin = 100_000_000
	cfg.Accum.MidDriftMax = 0.01
	cfg.Accum.ConfirmFor = 5 * time.Minute
	cfg.Distrib.NetMin = 100_000_000
	cfg.Cooldown = 15 * time.Minute
	cfg.TradeBufferTTL = 90 * time.Second
	cfg.SessionGap = 5 * time.Minute
	return cfg
}

// --- tests ----------------------------------------------------------------

func TestPullEmittedAfterRepeatedUnexecutedRemovals(t *testing.T) {
	sink := &captureSink{}
	e := NewEngine(defaultCfg(), sink)
	ctx := context.Background()

	t0 := base
	// Establish a book with two fat bids.
	require.NoError(t, e.ObserveBook(ctx, book("BBCA",
		side(1, [2]int64{7750, 800}, [2]int64{7740, 700}),
		side(2, [2]int64{7760, 400}), t0)))

	// First removal: bid at 7750 vanishes with no trade there.
	require.NoError(t, e.ObserveBook(ctx, book("BBCA",
		side(3, [2]int64{7740, 700}),
		side(4, [2]int64{7760, 400}), t0.Add(time.Minute))))
	assert.Empty(t, sink.alerts, "one event alone stays under repeat_k")

	// Second removal event: the remaining fat bid disappears unexecuted.
	require.NoError(t, e.ObserveBook(ctx, book("BBCA",
		side(5, [2]int64{7730, 250}),
		side(6, [2]int64{7760, 400}), t0.Add(2*time.Minute))))

	require.Len(t, sink.alerts, 1)
	a := sink.alerts[0]
	assert.Equal(t, SignalPullBid, a.Type)
	assert.Equal(t, "BBCA", a.Symbol)
	assert.Equal(t, "BID", a.Side)
}

func TestPullSuppressedWhenLevelWasExecuted(t *testing.T) {
	sink := &captureSink{}
	e := NewEngine(defaultCfg(), sink)
	ctx := context.Background()

	t0 := base
	require.NoError(t, e.ObserveBook(ctx, book("BBCA",
		side(1, [2]int64{7750, 800}),
		side(2, [2]int64{7760, 400}), t0)))

	// A real trade consumes the level right before it disappears.
	require.NoError(t, e.ObserveTrade(ctx, trade("BBCA", t0.Add(30*time.Second), 7750, 800, true)))
	require.NoError(t, e.ObserveBook(ctx, book("BBCA",
		side(3, [2]int64{7740, 300}),
		side(4, [2]int64{7760, 400}), t0.Add(time.Minute))))

	assert.Empty(t, sink.alerts, "an executed disappearance is legitimate consumption")
}

func TestCooldownSuppressesImmediateRepeat(t *testing.T) {
	sink := &captureSink{}
	e := NewEngine(defaultCfg(), sink)
	ctx := context.Background()

	t0 := base
	require.NoError(t, e.ObserveBook(ctx, book("BBCA",
		side(1, [2]int64{7750, 800}, [2]int64{7740, 700}),
		side(2, [2]int64{7760, 400}), t0)))
	require.NoError(t, e.ObserveBook(ctx, book("BBCA",
		side(3, [2]int64{7730, 200}),
		side(4, [2]int64{7760, 400}), t0.Add(time.Minute))))
	require.Len(t, sink.alerts, 1)

	// Two more removals inside the pull window but inside the cooldown too:
	// the events re-arm, yet emission stays gated.
	require.NoError(t, e.ObserveBook(ctx, book("BBCA",
		side(5, [2]int64{7750, 800}, [2]int64{7740, 700}),
		side(6, [2]int64{7760, 400}), t0.Add(10*time.Minute))))
	require.NoError(t, e.ObserveBook(ctx, book("BBCA",
		side(7, [2]int64{7730, 200}),
		side(8, [2]int64{7760, 400}), t0.Add(11*time.Minute))))

	assert.Len(t, sink.alerts, 1, "cooldown must gate re-emission")
}

func TestIcebergDetectedOnUniformRefills(t *testing.T) {
	sink := &captureSink{}
	e := NewEngine(defaultCfg(), sink)
	ctx := context.Background()

	n := int64(0)
	step := func(bidQty int64, execVol float64) {
		n++
		prevQty := bidQty
		if n == 1 {
			// Establish baseline first without any consumption.
			require.NoError(t, e.ObserveBook(ctx, book("BBCA",
				side(n, [2]int64{7750, bidQty}),
				side(900000+n, [2]int64{7760, 400}), base.Add(time.Duration(n)*time.Minute))))
			if st := e.syms["BBCA"]; st != nil {
				t.Logf("n=%d bidLv=%d", n, len(st.bidLv))
			}
			return
		}
		if execVol > 0 {
			require.NoError(t, e.ObserveTrade(ctx, trade("BBCA",
				base.Add(time.Duration(n)*time.Minute-time.Second), 7750, execVol, true)))
		}
		require.NoError(t, e.ObserveBook(ctx, book("BBCA",
			side(n, [2]int64{7750, bidQty}),
			side(900000+n, [2]int64{7760, 400}), base.Add(time.Duration(n)*time.Minute))))
		if st := e.syms["BBCA"]; st != nil {
			t.Logf("n=%d bidLv=%d consumed=%v", n, len(st.bidLv), func() bool { tr, ok := st.bidLv["7750"]; return ok && tr.consumed }())
		}
		_ = prevQty
	}

	// Baseline 500 lot.
	step(500, 0)
	// Cycle 1: aggregate crushed to 40% with real prints, restored in-band.
	step(180, 320)
	step(495, 0) // refill 1
	// Cycle 2.
	step(190, 305)
	step(505, 0) // refill 2
	// Cycle 3 crosses iceberg.N.
	step(170, 330)
	step(498, 0) // refill 3 -> EMIT

	if st := e.syms["BBCA"]; st != nil {
		for k, tr := range st.bidLv {
			t.Logf("DEBUG lvl %v base=%.0f refills=%d consumed=%v", k, float64(tr.base), tr.refills, tr.consumed)
		}
	}
	t.Logf("DEBUG alerts=%d", len(sink.alerts))
	found := false
	for _, a := range sink.alerts {
		if a.Type == SignalIceberg {
			found = true
			assert.Equal(t, "BID", a.Side)
		}
	}
	assert.True(t, found, "expected ICEBERG after three consume/restore cycles")
}

// TestIcebergIgnoresAggregateNoise mirrors the live NICL case: an aggregated
// level hovering around ~690k lots that only wiggles a few percent must never
// count as consumption/refill cycles, even with constant small trades.
func TestIcebergIgnoresAggregateNoise(t *testing.T) {
	sink := &captureSink{}
	e := NewEngine(defaultCfg(), sink)
	ctx := context.Background()

	qty := 680_000.0
	for i := 1; i <= 12; i++ {
		at := base.Add(time.Duration(i) * time.Minute)
		wiggle := 5_000.0 * float64((i%3)-1) // -5k / 0 / +5k
		require.NoError(t, e.ObserveTrade(ctx, trade("NICL", at, 482, 5_000, true)))
		newQty := qty + wiggle
		require.NoError(t, e.ObserveBook(ctx, book("NICL",
			side(int64(i), [2]int64{482, int64(newQty)}),
			side(int64(100+i), [2]int64{486, 30_000}), at)))
		qty = newQty
	}

	for _, a := range sink.alerts {
		assert.NotEqual(t, SignalIceberg, a.Type,
			"sub-percent aggregate wiggles are not iceberg refills")
	}
}

func TestAkumulasiRequiresStablePriceAndSupport(t *testing.T) {
	sink := &captureSink{}
	cfg := defaultCfg()
	e := NewEngine(cfg, sink)
	ctx := context.Background()

	// Baseline book.
	require.NoError(t, e.ObserveBook(ctx, book("NICL",
		side(1, [2]int64{482, 5000}),
		side(2, [2]int64{488, 5000}), base)))

	// Steady net buying for > confirm horizon while mid barely moves.
	minute := 0
	for minute = 1; minute <= 6; minute++ {
		at := base.Add(time.Duration(minute) * time.Minute)
		require.NoError(t, e.ObserveTrade(ctx, trade("NICL", at, 486, 40_000, true)))
		// Refresh book so support depth stays alive and mid stays put.
		require.NoError(t, e.ObserveBook(ctx, book("NICL",
			side(int64(10+minute), [2]int64{482, 5000}),
			side(int64(20+minute), [2]int64{488, 5000}), at)))
	}

	found := false
	for _, a := range sink.alerts {
		if a.Type == SignalAccumulation {
			found = true
		}
	}
	assert.True(t, found, "steady quiet buying must trip ACCUMULASI")
}

func TestAkumulasiSuppressedWhenPriceRunsAway(t *testing.T) {
	sink := &captureSink{}
	e := NewEngine(defaultCfg(), sink)
	ctx := context.Background()

	require.NoError(t, e.ObserveBook(ctx, book("GULA",
		side(1, [2]int64{750, 5000}),
		side(2, [2]int64{756, 5000}), base)))

	// Same buying pressure, but the mid runs up 4% — momentum, not stealth.
	for m := 1; m <= 6; m++ {
		at := base.Add(time.Duration(m) * time.Minute)
		mid := 753.0 * (1 + 0.04*float64(m)/6)
		half := 3.0
		require.NoError(t, e.ObserveTrade(ctx, trade("GULA", at, mid, 40_000, true)))
		require.NoError(t, e.ObserveBook(ctx, book("GULA",
			side(int64(10+m), [2]int64{int64(mid - half), 5000}),
			side(int64(20+m), [2]int64{int64(mid + half), 5000}), at)))
	}

	for _, a := range sink.alerts {
		assert.NotEqual(t, SignalAccumulation, a.Type, "a running price is momentum, not akumulasi")
	}
}

func TestDistribusiMirrorsAkumulasi(t *testing.T) {
	sink := &captureSink{}
	e := NewEngine(defaultCfg(), sink)
	ctx := context.Background()

	require.NoError(t, e.ObserveBook(ctx, book("ANTM",
		side(1, [2]int64{3140, 5000}),
		side(2, [2]int64{3150, 5000}), base)))

	for m := 1; m <= 6; m++ {
		at := base.Add(time.Duration(m) * time.Minute)
		require.NoError(t, e.ObserveTrade(ctx, trade("ANTM", at, 3145, 40_000, false)))
		require.NoError(t, e.ObserveBook(ctx, book("ANTM",
			side(int64(10+m), [2]int64{3140, 5000}),
			side(int64(20+m), [2]int64{3150, 5000}), at)))
	}

	found := false
	for _, a := range sink.alerts {
		if a.Type == SignalDistribution {
			found = true
		}
	}
	assert.True(t, found, "steady quiet selling must trip DISTRIBUSI")
}

func TestSessionGapResetsWindows(t *testing.T) {
	sink := &captureSink{}
	e := NewEngine(defaultCfg(), sink)
	ctx := context.Background()

	require.NoError(t, e.ObserveBook(ctx, book("BBCA",
		side(1, [2]int64{7750, 800}),
		side(2, [2]int64{7760, 400}), base)))

	// Big buy burst, then a long silence crossing the session gap.
	require.NoError(t, e.ObserveTrade(ctx, trade("BBCA", base.Add(time.Minute), 7755, 60_000, true)))
	require.NoError(t, e.ObserveBook(ctx, book("BBCA",
		side(3, [2]int64{7750, 5000}),
		side(4, [2]int64{7760, 5000}), base.Add(2*time.Minute))))

	gapLater := base.Add(2*time.Minute + 6*time.Minute) // > session gap
	require.NoError(t, e.ObserveBook(ctx, book("BBCA",
		side(5, [2]int64{7750, 5000}),
		side(6, [2]int64{7760, 5000}), gapLater)))

	// After the gap, old buys must not count toward akumulasi.
	require.NoError(t, e.ObserveTrade(ctx, trade("BBCA", gapLater.Add(time.Minute), 7755, 1_000, true)))
	for m := 2; m <= 6; m++ {
		at := gapLater.Add(time.Duration(m) * time.Minute)
		require.NoError(t, e.ObserveTrade(ctx, trade("BBCA", at, 7755, 1_000, true)))
		require.NoError(t, e.ObserveBook(ctx, book("BBCA",
			side(int64(10+m), [2]int64{7750, 5000}),
			side(int64(20+m), [2]int64{7760, 5000}), at)))
	}

	for _, a := range sink.alerts {
		assert.NotEqual(t, SignalAccumulation, a.Type,
			"pre-gap volume must not leak across the session boundary")
	}
}

// TestIcebergIgnoresDeepLevelFlicker guards the false-positive mode found on
// live data: a deep level entering/leaving the scanned window must not count
// as consumption+refill, and the reference size must itself clear MinQty.
func TestIcebergIgnoresDeepLevelFlicker(t *testing.T) {
	sink := &captureSink{}
	cfg := defaultCfg()
	cfg.Iceberg.MinQty = 300
	e := NewEngine(cfg, sink)
	ctx := context.Background()

	t0 := base
	// Level 482 first appears DEEP (position 11+) with a tiny 10k remainder.
	deep := func(seq int64, pairs ...[2]int64) *BookSide {
		s := &BookSide{Seq: seq}
		for _, p := range pairs {
			s.Prices = append(s.Prices, float64(p[0]))
			s.Qtys = append(s.Qtys, p[1])
		}
		return s
	}
	bid := func(levels ...[2]int64) *BookSide { return deep(0, levels...) }

	// Snapshot A: 10 fat levels above, 482 sits at position 11 (outside top-10
	// scan) holding only 10_000.
	a := make([][2]int64, 0, 11)
	for i := 0; i < 10; i++ {
		a = append(a, [2]int64{int64(500 - i), 1000})
	}
	a = append(a, [2]int64{482, 10_000})
	require.NoError(t, e.ObserveBook(ctx, book("NICL", bid(a...),
		side(99, [2]int64{510, 1000}), t0)))

	// Trades at 482 (would satisfy the execution requirement if tracked).
	require.NoError(t, e.ObserveTrade(ctx, trade("NICL", t0.Add(time.Second), 482, 5_000, true)))

	// Snapshot B: the ten fat levels vanish; 482 now sits INSIDE the scan
	// window having "refilled" three times worth of appearances.
	for round := 0; round < 3; round++ {
		b := append([][2]int64{{480, 900}}, [2]int64{482, 10_200})
		require.NoError(t, e.ObserveBook(ctx, book("NICL", bid(b...),
			side(98, [2]int64{510, 1000}), t0.Add(time.Duration(round+1)*time.Minute))))
		c := append([][2]int64{{479, 900}}, [2]int64{482, 10_400})
		require.NoError(t, e.ObserveBook(ctx, book("NICL", bid(c...),
			side(97, [2]int64{510, 1000}), t0.Add(time.Duration(round+1)*time.Minute+30*time.Second))))
	}

	for _, a := range sink.alerts {
		assert.NotEqual(t, SignalIceberg, a.Type,
			"a deep flickering level must not produce an ICEBERG signal")
	}
}

// TestDistribWindowIsIndependent pins that distribution sums over its OWN
// window: stale sells outside it must not fire the signal, fresh ones must.
func TestDistribWindowIsIndependent(t *testing.T) {
	sink := &captureSink{}
	cfg := defaultCfg()
	cfg.Accum.Window = 20 * time.Minute
	cfg.Distrib.Window = 2 * time.Minute
	e := NewEngine(cfg, sink)
	ctx := context.Background()

	start := base.Add(-12 * time.Minute)
	stableBook := func(at time.Time) {
		require.NoError(t, e.ObserveBook(ctx, book("BBCA",
			side(1, [2]int64{7750, 1000}),
			side(2, [2]int64{7800, 1000}), at)))
	}

	// Phase A: a big sell 10 minutes back sits inside the 20m accumulation
	// window but outside the 2m distribution window. It must never fire.
	stableBook(start)
	require.NoError(t, e.ObserveTrade(ctx, trade("BBCA", start.Add(2*time.Minute), 7760, 51_546.4, false)))
	for m := 1; m <= 24; m++ { // every 30s up to t0
		stableBook(start.Add(time.Duration(m*30) * time.Second))
	}
	for _, a := range sink.alerts {
		assert.NotEqual(t, SignalDistribution, a.Type,
			"sells older than Distrib.Window must not fire")
	}

	// Phase B: fresh selling pressure inside the short window fires normally.
	require.NoError(t, e.ObserveTrade(ctx, trade("BBCA", base.Add(30*time.Second), 7760, 12_886.6, false)))
	require.NoError(t, e.ObserveTrade(ctx, trade("BBCA", base.Add(45*time.Second), 7760, 12_886.6, false)))
	stableBook(base.Add(time.Minute))

	fired := false
	for _, a := range sink.alerts {
		if a.Type == SignalDistribution {
			fired = true
		}
	}
	assert.True(t, fired, "distribution within its own window must fire")
}

// TestBiasTrackersAreIndependent pins that one side's instability must not
// rewrite the other signal's baseline or reset its confirmation timer. Ask
// depth collapses mid-hold; accumulation (bid side) still confirms on time.
func TestBiasTrackersAreIndependent(t *testing.T) {
	sink := &captureSink{}
	cfg := defaultCfg()
	cfg.Accum.ConfirmFor = 5 * time.Minute
	e := NewEngine(cfg, sink)
	ctx := context.Background()

	stable := func(at time.Time, askQty int64) {
		require.NoError(t, e.ObserveBook(ctx, book("BBCA",
			side(1, [2]int64{7750, 1000}),
			side(2, [2]int64{7800, askQty}), at)))
	}

	// Baseline + buying pressure early so the accumulation gate is armed.
	stable(base, 1000)
	require.NoError(t, e.ObserveTrade(ctx, trade("BBCA", base.Add(10*time.Second), 7775, 15_437, true)))

	// Quiet stretch from t0; at t=4m45s the ASK side collapses, which under
	// shared state would re-baseline and reset the shared hold timer.
	for m := 1; m <= 9; m++ { // every 30s up to t=4m30s
		stable(base.Add(time.Duration(m*30)*time.Second), 1000)
	}
	stable(base.Add(4*time.Minute+45*time.Second), 100) // ask collapse
	stable(base.Add(5*time.Minute), 100)
	stable(base.Add(5*time.Minute+30*time.Second), 100)

	fired := false
	for _, a := range sink.alerts {
		if a.Type == SignalAccumulation {
			fired = true
		}
		assert.NotEqual(t, SignalDistribution, a.Type, "no sell pressure exists")
	}
	assert.True(t, fired,
		"accumulation must confirm despite ask-side instability")
}

// TestPullSurvivesExchangeClockLag pins the single clock domain: removal
// events are stamped AND pruned on exchange time, so a large lag between the
// exchange clock and arrival can no longer prune events before they
// accumulate (which silently disabled PULL_BID/ASK).
func TestPullSurvivesExchangeClockLag(t *testing.T) {
	sink := &captureSink{}
	cfg := defaultCfg()
	e := NewEngine(cfg, sink)
	ctx := context.Background()

	lag := 45 * time.Minute // far beyond Pull.Window (10m)
	lagged := func(seq int64, bidPx int64, bidQty int64, at time.Time) {
		require.NoError(t, e.ObserveBook(ctx, Book{
			Symbol: "BBCA",
			Bid:    side(seq, [2]int64{bidPx, bidQty}),
			Ask:    side(900_000+seq, [2]int64{7800, 400}),
			// Exchange clock trails reception by `lag`.
			ExchangeTS: at.Add(-lag), ReceiveTS: at,
		}))
	}

	r0 := base.Add(time.Hour)
	lagged(1, 7750, 1000, r0)
	lagged(2, 7740, 800, r0.Add(time.Second)) // 7750 vanishes unexecuted -> event 1
	lagged(3, 7730, 900, r0.Add(2*time.Second))

	fired := false
	for _, a := range sink.alerts {
		if a.Type == SignalPullBid {
			fired = true
		}
	}
	assert.True(t, fired, "pull detection must work under exchange-clock lag")
}
