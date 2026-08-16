package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nofendian17/sbterm-server/pkg/log"
)

const webSocketKeyBody = `{"message":"Success get websocket key","data":{"key":"test-websocket-key"}}`

func TestGetWebSocketKey(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		handler func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check   func(t *testing.T, resp *WebSocketKeyResponse, logs string)
	}{
		{
			name: "returns the websocket key",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, webSocketKeyPath, r.URL.Path)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(webSocketKeyBody))
			},
			check: func(t *testing.T, resp *WebSocketKeyResponse, logs string) {
				require.NotNil(t, resp)
				assert.Equal(t, "Success get websocket key", resp.Message)
				assert.Equal(t, "test-websocket-key", resp.Data.Key)
			},
		},
		{
			name: "uses access token",
			opts: []Option{WithAuthenticator(&stubAuth{token: "at-ok"})},
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "Bearer at-ok", r.Header.Get("Authorization"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(webSocketKeyBody))
			},
		},
		{
			name: "is not logged",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(webSocketKeyBody))
			},
			check: func(t *testing.T, resp *WebSocketKeyResponse, logs string) {
				assert.Contains(t, logs, "stockbit request")
				assert.NotContains(t, logs, "test-websocket-key")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.handler(t, w, r)
			}))
			defer srv.Close()

			var buf strings.Builder
			logger := log.New(log.WithWriter(&buf), log.WithLevel(log.LevelDebug))
			opts := append([]Option{WithBaseURL(srv.URL), WithLogger(logger)}, tt.opts...)
			resp, err := New(opts...).GetWebSocketKey(context.Background())
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resp, buf.String())
			}
		})
	}
}
