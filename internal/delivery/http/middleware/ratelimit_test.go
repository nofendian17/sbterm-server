package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimit(t *testing.T) {
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name string
		opts []Option
		reqs []*http.Request
		want []int
	}{
		{
			name: "allows up to burst then rejects",
			opts: []Option{WithRatePerSecond(1), WithBurst(2)},
			reqs: []*http.Request{
				httptest.NewRequest(http.MethodGet, "/", nil),
				httptest.NewRequest(http.MethodGet, "/", nil),
				httptest.NewRequest(http.MethodGet, "/", nil),
			},
			want: []int{http.StatusOK, http.StatusOK, http.StatusTooManyRequests},
		},
		{
			name: "rejects exceed the configured retry-after header",
			opts: []Option{WithRatePerSecond(1), WithBurst(1)},
			reqs: []*http.Request{
				httptest.NewRequest(http.MethodGet, "/", nil),
				httptest.NewRequest(http.MethodGet, "/", nil),
			},
			want: []int{http.StatusOK, http.StatusTooManyRequests},
		},
		{
			name: "clients are limited independently by ip",
			opts: []Option{WithRatePerSecond(1), WithBurst(1)},
			reqs: []*http.Request{
				httptest.NewRequest(http.MethodGet, "/", nil),
				httptest.NewRequest(http.MethodGet, "/", nil),
				httptest.NewRequest(http.MethodGet, "/", nil),
				httptest.NewRequest(http.MethodGet, "/", nil),
			},
			want: []int{http.StatusOK, http.StatusOK, http.StatusTooManyRequests, http.StatusTooManyRequests},
		},
		{
			name: "custom key extractor isolates by header",
			opts: []Option{
				WithRatePerSecond(1),
				WithBurst(1),
				WithKeyExtractor(func(r *http.Request) string { return r.Header.Get("X-Client-Id") }),
			},
			reqs: []*http.Request{
				httptest.NewRequest(http.MethodGet, "/", nil),
				httptest.NewRequest(http.MethodGet, "/", nil),
				httptest.NewRequest(http.MethodGet, "/", nil),
			},
			want: []int{http.StatusOK, http.StatusOK, http.StatusTooManyRequests},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := NewRateLimit(tt.opts...)

			for i, req := range tt.reqs {
				if tt.name == "clients are limited independently by ip" {
					if i%2 == 0 {
						req.RemoteAddr = "1.2.3.4:1234"
					} else {
						req.RemoteAddr = "5.6.7.8:5678"
					}
				}
				if tt.name == "custom key extractor isolates by header" {
					if i%2 == 0 {
						req.Header.Set("X-Client-Id", "client-a")
					} else {
						req.Header.Set("X-Client-Id", "client-b")
					}
				}

				rec := httptest.NewRecorder()
				rl(okHandler).ServeHTTP(rec, req)
				assert.Equal(t, tt.want[i], rec.Code, "request %d", i)

				if tt.name == "rejects exceed the configured retry-after header" && i == 1 {
					assert.Equal(t, "1", rec.Header().Get("Retry-After"))
				}
			}
		})
	}
}

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		want       string
	}{
		{name: "host port", remoteAddr: "1.2.3.4:1234", want: "1.2.3.4"},
		{name: "malformed address", remoteAddr: "not-a-host-port", want: "not-a-host-port"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			assert.Equal(t, tt.want, clientIP(req))
		})
	}
}

func TestRateLimitCleanupOptions(t *testing.T) {
	rl := NewRateLimit(
		WithRatePerSecond(1),
		WithBurst(1),
		WithCleanupInterval(time.Millisecond),
		WithClientTTL(time.Millisecond),
	)
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	rec := httptest.NewRecorder()
	rl(ok).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimitCleanupExpired(t *testing.T) {
	rl := &RateLimit{
		cleanupInterval: time.Minute,
		clientTTL:       time.Minute,
		clients: map[string]*rateLimitClient{
			"fresh": {limiter: nil, lastSeen: time.Now()},
			"stale": {limiter: nil, lastSeen: time.Now().Add(-2 * time.Minute)},
		},
	}

	rl.cleanupExpired(time.Now())

	assert.Contains(t, rl.clients, "fresh")
	assert.NotContains(t, rl.clients, "stale")
}

func TestRateLimitResponseShape(t *testing.T) {
	rl := NewRateLimit(WithRatePerSecond(1), WithBurst(1))

	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)

	rec1 := httptest.NewRecorder()
	rl(ok).ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusOK, rec1.Code)

	rec2 := httptest.NewRecorder()
	rl(ok).ServeHTTP(rec2, req2)
	require.Equal(t, http.StatusTooManyRequests, rec2.Code)

	var env struct {
		Success bool `json:"success"`
		Error   *struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rec2.Body.Bytes(), &env))
	assert.False(t, env.Success)
	require.NotNil(t, env.Error)
	assert.Equal(t, "TOO_MANY_REQUESTS", env.Error.Code)
}
