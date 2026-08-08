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

const websocketKeyBody = `{"message":"Success get websocket key","data":{"key":"test-websocket-key"}}`

func TestGetWebsocketKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, websocketKeyPath, r.URL.Path)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(websocketKeyBody))
	}))
	defer srv.Close()

	resp, err := New(WithBaseURL(srv.URL)).GetWebsocketKey(context.Background())
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "Success get websocket key", resp.Message)
	assert.Equal(t, "test-websocket-key", resp.Data.Key)
}

func TestGetWebsocketKeyUsesAccessToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer at-ok", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(websocketKeyBody))
	}))
	defer srv.Close()

	_, err := New(
		WithBaseURL(srv.URL),
		WithAuthenticator(&stubAuth{token: "at-ok"}),
	).GetWebsocketKey(context.Background())
	require.NoError(t, err)
}

func TestGetWebsocketKeyIsNotLogged(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(websocketKeyBody))
	}))
	defer srv.Close()

	var buf strings.Builder
	logger := log.New(log.WithWriter(&buf), log.WithLevel(log.LevelDebug))
	_, err := New(WithBaseURL(srv.URL), WithLogger(logger)).GetWebsocketKey(context.Background())
	require.NoError(t, err)

	line := buf.String()
	assert.Contains(t, line, "stockbit request")
	assert.NotContains(t, line, "test-websocket-key")
}
