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

	// step advances one minute per call; bid levels describe the full bid
	// side after each snapshot (a missing 7750 means the level was consumed).
	step := func(n int, bids ...[2]int64) {
		require.NoError(t, e.ObserveBook(ctx, book("BBCA",
			side(int64(n), bids...),
			side(900000+int64(n), [2]int64{7760, 400}), base.Add(time.Duration(n)*time.Minute))))
	}

	// Cycle 1: level sits at 500, gets bought away, comes back at ~500.
	step(1, [2]int64{7750, 500})
	require.NoError(t, e.ObserveTrade(ctx, trade("BBCA", base.Add(90*time.Second), 7750, 500, true)))
	step(2, [2]int64{7745, 100}) // 7750 consumed and gone
	step(3, [2]int64{7750, 495}, [2]int64{7745, 100})

	// Cycle 2: consumed again, restored again.
	require.NoError(t, e.ObserveTrade(ctx, trade("BBCA", base.Add(4*time.Minute), 7750, 495, true)))
	step(5, [2]int64{7745, 100})
	step(6, [2]int64{7750, 505}, [2]int64{7745, 100})

	// Cycle 3: third refill crosses iceberg.n and fires.
	require.NoError(t, e.ObserveTrade(ctx, trade("BBCA", base.Add(7*time.Minute), 7750, 505, true)))
	step(8, [2]int64{7745, 100})
	step(9, [2]int64{7750, 498}, [2]int64{7745, 100})

	require.NotEmpty(t, sink.alerts)
	var found bool
	for _, a := range sink.alerts {
		if a.Type == SignalIceberg {
			found = true
			assert.Equal(t, "BID", a.Side)
		}
	}
	assert.True(t, found, "expected an ICEBERG alert, got %+v", sink.alerts)
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
