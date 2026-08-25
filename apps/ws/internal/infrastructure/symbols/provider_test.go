package symbols

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stocksHandler(t *testing.T, hits *atomic.Int32, failAfter int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits.Add(1) > int32(failAfter) && failAfter >= 0 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		assert.Equal(t, "/api/v1/stocks", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":[` +
			`{"symbol":"BBCA","name":"Bank Central Asia"},` +
			`{"symbol":"bbri","name":"Bank Rakyat Indonesia"},` +
			`{"symbol":"","name":"no symbol"},` +
			`{"name":"missing symbol field"}]}`))
	})
}

func TestSymbols(t *testing.T) {
	t.Run("fetches uppercase non-empty symbols from the api envelope", func(t *testing.T) {
		var hits atomic.Int32
		srv := httptest.NewServer(stocksHandler(t, &hits, -1))
		t.Cleanup(srv.Close)

		p := New(srv.URL, srv.Client(), time.Hour)
		syms, err := p.Symbols(context.Background())

		require.NoError(t, err)
		assert.Equal(t, []string{"BBCA", "BBRI"}, syms)
	})

	t.Run("serves the cache while it is fresh", func(t *testing.T) {
		var hits atomic.Int32
		srv := httptest.NewServer(stocksHandler(t, &hits, -1))
		t.Cleanup(srv.Close)

		p := New(srv.URL, srv.Client(), time.Hour)
		first, err := p.Symbols(context.Background())
		require.NoError(t, err)
		second, err := p.Symbols(context.Background())
		require.NoError(t, err)

		assert.Equal(t, first, second)
		assert.EqualValues(t, 1, hits.Load(), "second read must come from the cache")
	})

	t.Run("falls back to the stale cache when the api fails", func(t *testing.T) {
		var hits atomic.Int32
		srv := httptest.NewServer(stocksHandler(t, &hits, 1))
		t.Cleanup(srv.Close)

		p := New(srv.URL, srv.Client(), time.Nanosecond)
		first, err := p.Symbols(context.Background())
		require.NoError(t, err)

		stale, err := p.Symbols(context.Background())
		require.NoError(t, err, "an expired cache must still be served when the api is down")
		assert.Equal(t, first, stale)
	})

	t.Run("errors when the api fails and nothing is cached", func(t *testing.T) {
		var hits atomic.Int32
		srv := httptest.NewServer(stocksHandler(t, &hits, 0))
		t.Cleanup(srv.Close)

		p := New(srv.URL, srv.Client(), time.Hour)
		_, err := p.Symbols(context.Background())
		require.Error(t, err)
	})
}
