package stockbit

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nofendian17/sbterm-server/pkg/log"
)

func TestGet(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		query   url.Values
		headers [][2]string
		handler func(t *testing.T, w http.ResponseWriter, r *http.Request)
		want    any
	}{
		{
			name:  "sends a request and parses json body",
			path:  "/v1/stocks/bbca",
			query: url.Values{},
			want:  map[string]any{"name": "Bank BCA", "price": 8750.0},
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]any{"name": "Bank BCA", "price": 8750.0})
			},
		},
		{
			name:  "encodes query params",
			path:  "/v1/screen",
			query: url.Values{"q": {"bca"}, "page": {"1"}},
			want:  map[string]any{},
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, url.Values{"q": {"bca"}, "page": {"1"}}.Encode(), r.URL.RawQuery)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{}`))
			},
		},
		{
			name:    "applies configured and default headers",
			path:    "/v1/stocks",
			query:   url.Values{},
			headers: [][2]string{{"Authorization", "Bearer secret"}},
			want:    map[string]any{},
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "Bearer secret", r.Header.Get("Authorization"))
				assert.Equal(t, "Google Chrome", r.Header.Get("X-DeviceType"))
				assert.Equal(t, "PC", r.Header.Get("X-Platform"))
				assert.Equal(t, "3.17.2", r.Header.Get("X-AppVersion"))
				assert.Equal(t, "ID", r.Header.Get("Accept-Language"))
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				assert.Contains(t, r.Header.Get("User-Agent"), "Chrome/143.0.0.0")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{}`))
			},
		},
		{
			name:    "overrides a default header",
			path:    "/v1/stocks",
			query:   url.Values{},
			headers: [][2]string{{"Accept-Language", "EN"}},
			want:    map[string]any{},
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "EN", r.Header.Get("Accept-Language"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{}`))
			},
		},
		{
			name:  "non-2xx returns error with status",
			path:  "/v1/denied",
			query: url.Values{},
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"nope"}`))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tt.path, r.URL.Path)
				tt.handler(t, w, r)
			}))
			defer srv.Close()

			var opts []Option
			opts = append(opts, WithBaseURL(srv.URL))
			for _, h := range tt.headers {
				opts = append(opts, WithHeader(h[0], h[1]))
			}

			var out map[string]any
			err := New(opts...).Get(context.Background(), tt.path, tt.query, &out)
			if tt.want == nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unexpected status")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, out)
		})
	}
}

func TestPost(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/watchlist", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		body, _ := io.ReadAll(r.Body)
		assert.JSONEq(t, `{"name":"growth"}`, string(body))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"1"}`))
	}))
	defer srv.Close()

	var out struct {
		ID string `json:"id"`
	}
	err := New(WithBaseURL(srv.URL)).Post(
		context.Background(), "/v1/watchlist", strings.NewReader(`{"name":"growth"}`), &out,
	)
	require.NoError(t, err)
	assert.Equal(t, "1", out.ID)
}

func TestDefaultBaseURL(t *testing.T) {
	assert.Equal(t, "https://exodus.stockbit.com", New().baseURL)
}

func TestDoLogsRequestsAtDebugLevel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	var buf strings.Builder
	logger := log.New(log.WithWriter(&buf), log.WithLevel(log.LevelDebug))
	err := New(WithBaseURL(srv.URL), WithLogger(logger)).Get(
		context.Background(), "/v1/stocks/bbca", nil, nil)
	require.NoError(t, err)

	line := buf.String()
	assert.Contains(t, line, "stockbit request")
	assert.Contains(t, line, "path=/v1/stocks/bbca")
	assert.Contains(t, line, "status=200")
}

func TestDoLogsRequestsAtInfoLevelSkips(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	var buf strings.Builder
	logger := log.New(log.WithWriter(&buf), log.WithLevel(log.LevelInfo))
	err := New(WithBaseURL(srv.URL), WithLogger(logger)).Get(
		context.Background(), "/v1/stocks/bbca", nil, nil)
	require.NoError(t, err)
	assert.NotContains(t, buf.String(), "stockbit request")
}

func TestGetUnauthorizedReturnsSentinel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"expired"}`))
	}))
	defer srv.Close()

	err := New(WithBaseURL(srv.URL)).Get(context.Background(), "/v1/stocks", nil, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthorized)
}