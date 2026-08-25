package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildMicrostructureChannel(t *testing.T) {
	syms := []string{"BBCA", "BBRI"}

	t.Run("fills exactly the requested channels with the given symbols", func(t *testing.T) {
		ch, err := BuildMicrostructureChannel(
			[]string{"order_book", "liveprice", "iepiev", "best_bid_offer"}, syms)

		require.NoError(t, err)
		assert.Equal(t, syms, ch.GetOrderBook())
		assert.Equal(t, syms, ch.GetLiveprice())
		assert.Equal(t, syms, ch.GetIepiev())
		assert.Equal(t, syms, ch.GetBestBidOffer())
		assert.Empty(t, ch.GetWatchlist(), "unrequested channels must stay unset")
		assert.Empty(t, ch.GetRunningTradeBatch())
	})

	t.Run("rejects unknown channel names", func(t *testing.T) {
		_, err := BuildMicrostructureChannel([]string{"order_book", "watchlist"}, syms)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "watchlist")
	})

	t.Run("accepts nothing gracefully", func(t *testing.T) {
		ch, err := BuildMicrostructureChannel(nil, syms)
		require.NoError(t, err)
		assert.NotNil(t, ch)
		assert.Empty(t, ch.GetOrderBook())
	})
}

func TestNextRefreshAt(t *testing.T) {
	wib := time.FixedZone("WIB", 7*3600)

	t.Run("returns today's slot when it is still ahead", func(t *testing.T) {
		now := time.Date(2026, 8, 26, 7, 0, 0, 0, wib)
		target, err := NextRefreshAt(now, "08:45", wib)
		require.NoError(t, err)
		assert.Equal(t, time.Date(2026, 8, 26, 8, 45, 0, 0, wib), target)
	})

	t.Run("rolls to tomorrow once the slot has passed", func(t *testing.T) {
		now := time.Date(2026, 8, 26, 8, 45, 0, 1, wib)
		target, err := NextRefreshAt(now, "08:45", wib)
		require.NoError(t, err)
		assert.Equal(t, time.Date(2026, 8, 27, 8, 45, 0, 0, wib), target)
	})

	t.Run("converts from another timezone", func(t *testing.T) {
		now := time.Date(2026, 8, 26, 2, 0, 0, 0, time.UTC) // 09:00 WIB
		target, err := NextRefreshAt(now, "08:45", wib)
		require.NoError(t, err)
		assert.Equal(t, time.Date(2026, 8, 27, 8, 45, 0, 0, wib), target)
	})

	t.Run("rejects malformed times", func(t *testing.T) {
		_, err := NextRefreshAt(time.Now(), "25:00", wib)
		require.Error(t, err)
		_, err = NextRefreshAt(time.Now(), "bad", wib)
		require.Error(t, err)
	})
}
