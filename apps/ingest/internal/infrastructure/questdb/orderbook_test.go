package questdb

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	consumerv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/consumer/entity/v1"
)

func TestSplitBody(t *testing.T) {
	t.Run("parses price frequency shares triplets", func(t *testing.T) {
		_, side, levels, err := splitBody("#O|BBCA|BID|7750;12;340000|7745;3;125000")
		require.NoError(t, err)
		assert.Equal(t, "BID", side)
		require.Len(t, levels, 2)
		assert.Equal(t, int64(7750), levels[0].Price)
		assert.Equal(t, int64(12), levels[0].Frequency)
		assert.Equal(t, int64(340000), levels[0].Shares)
		assert.Equal(t, int64(7745), levels[1].Price)
	})

	t.Run("skips malformed levels but keeps valid ones", func(t *testing.T) {
		_, side, levels, err := splitBody("#O|BBCA|BID|bad;stuff|7700;1;90000|;|")
		require.NoError(t, err)
		assert.Equal(t, "BID", side)
		require.Len(t, levels, 1)
		assert.Equal(t, int64(7700), levels[0].Price)
	})

	t.Run("rejects non order book bodies", func(t *testing.T) {
		for _, body := range []string{"", "#X|BBCA|x", "#O|BBCA"} {
			_, _, _, err := splitBody(body)
			require.Error(t, err, "body %q must be rejected", body)
		}
	})
}

func TestSideOf(t *testing.T) {
	assert.Equal(t, SideBid, sideOf("BID"))
	assert.Equal(t, SideBid, sideOf("b"))
	assert.Equal(t, SideAsk, sideOf("OFFER"))
	assert.Equal(t, SideAsk, sideOf("sell"))
	assert.Equal(t, SideUnknown, sideOf("wat"))
}

func TestCombiner(t *testing.T) {
	ts := time.Date(2026, 8, 27, 9, 15, 0, 0, time.UTC)

	ob := func(code, side string, seq int64, levels ...string) *consumerv1.Orderbook {
		return &consumerv1.Orderbook{
			StockCode:      code,
			Body:           "#O|" + code + "|" + side + "|" + joinLevels(levels),
			SequenceNumber: seq,
			Datetime:       ts.Format(time.RFC3339Nano),
			Board:          consumerv1.BoardType_BOARD_TYPE_RG,
		}
	}

	t.Run("emits nothing until both sides have been seen", func(t *testing.T) {
		c := NewCombiner(25)
		pair, ok := c.Observe(ob("BBCA", "BID", 10, "7750;1;100"), ts)
		assert.False(t, ok)
		assert.Nil(t, pair)
	})

	t.Run("pairs sides and re-emits on every subsequent half update", func(t *testing.T) {
		c := NewCombiner(25)

		pair, ok := c.Observe(ob("BBCA", "BID", 10, "7750;1;100|7740;2;50"), ts)
		assert.False(t, ok)

		pair, ok = c.Observe(ob("BBCA", "OFFER", 11, "7760;3;200"), ts.Add(time.Millisecond))
		require.True(t, ok)
		require.NotNil(t, pair)
		assert.Equal(t, "BBCA", pair.Symbol)
		require.NotNil(t, pair.Bid)
		require.NotNil(t, pair.Ask)
		assert.Equal(t, []float64{7750, 7740}, pair.Bid.Prices)
		assert.Equal(t, []int64{100, 50}, pair.Bid.Qtys)
		assert.Equal(t, int64(10), pair.Bid.Seq)
		assert.Equal(t, []float64{7760}, pair.Ask.Prices)
		assert.Equal(t, int64(11), pair.Ask.Seq)

		// An updated bid re-emits with the last known offer attached.
		pair, ok = c.Observe(ob("BBCA", "BID", 12, "7755;4;80"), ts.Add(2*time.Millisecond))
		require.True(t, ok)
		assert.Equal(t, []float64{7755}, pair.Bid.Prices)
		assert.Equal(t, []float64{7760}, pair.Ask.Prices, "stale offer side must ride along")
	})

	t.Run("caps stored levels", func(t *testing.T) {
		c := NewCombiner(3)
		c.Observe(ob("BBCA", "OFFER", 11, "8000;1;1"), ts)

		levels := make([]string, 0, 10)
		for p := 10; p >= 1; p-- {
			levels = append(levels, itoa(p*100)+";1;"+itoa(p))
		}
		caps, ok := c.Observe(ob("BBCA", "BID", 12, levels...), ts)
		require.True(t, ok)
		require.NotNil(t, caps)
		assert.Len(t, caps.Bid.Prices, 3, "only the top-3 levels are kept")
		assert.Equal(t, []float64{1000, 900, 800}, caps.Bid.Prices)
	})

	t.Run("keeps symbols separate", func(t *testing.T) {
		c := NewCombiner(25)
		_, _ = c.Observe(ob("BBCA", "BID", 1, "7750;1;100"), ts)
		pair, ok := c.Observe(ob("BBRI", "OFFER", 2, "4200;1;100"), ts)
		assert.False(t, ok, "sides of different symbols must not pair")
		_, ok = c.Observe(ob("BBRI", "BID", 3, "4190;1;90"), ts)
		require.True(t, ok)
		_ = pair
	})
}

func joinLevels(levels []string) string {
	s := ""
	for i, l := range levels {
		if i > 0 {
			s += "|"
		}
		s += l
	}
	return s
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	digits := ""
	for v > 0 {
		digits = string(rune('0'+v%10)) + digits
		v /= 10
	}
	return digits
}
