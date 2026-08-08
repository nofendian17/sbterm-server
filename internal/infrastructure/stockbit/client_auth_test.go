package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubAuth struct {
	token     string
	refreshed atomic.Int32
}

func (s *stubAuth) EnsureToken(ctx context.Context) (string, error) { return s.token, nil }
func (s *stubAuth) Refresh(ctx context.Context) (string, error) {
	s.refreshed.Add(1)
	s.token = "at-new"
	return s.token, nil
}

func TestAuthHeaders(t *testing.T) {
	tests := []struct {
		name    string
		handler func(t *testing.T, w http.ResponseWriter, r *http.Request)
		call    func(t *testing.T, c *Client) error
	}{
		{
			name: "do attaches access token",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "Bearer at-ok", r.Header.Get("Authorization"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{}`))
			},
			call: func(t *testing.T, c *Client) error {
				return c.Get(context.Background(), "/v1/stocks", nil, nil)
			},
		},
		{
			name: "login sends no authorization",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, loginPath, r.URL.Path)
				assert.Empty(t, r.Header.Get("Authorization"))
				writeLoginResponse(w, "at-1", "rt-1")
			},
			call: func(t *testing.T, c *Client) error {
				_, err := c.Login(context.Background(), LoginRequest{User: "budi", Password: "secret"})
				return err
			},
		},
		{
			name: "refresh sends refresh token not access token",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, refreshPath, r.URL.Path)
				assert.Equal(t, "Bearer rt-1", r.Header.Get("Authorization"))
				writeRefreshResponse(w, "at-2", "rt-2")
			},
			call: func(t *testing.T, c *Client) error {
				_, err := c.Refresh(context.Background(), "rt-1")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.handler(t, w, r)
			}))
			defer srv.Close()

			err := tt.call(t, New(
				WithBaseURL(srv.URL),
				WithAuthenticator(&stubAuth{token: "at-ok"}),
			))
			require.NoError(t, err)
		})
	}
}

func TestDoHandlesUnauthorized(t *testing.T) {
	tests := []struct {
		name    string
		handler func(t *testing.T, w http.ResponseWriter, r *http.Request, hits *atomic.Int32)
		call    func(t *testing.T, c *Client) error
		wantErr error
		check   func(t *testing.T, auth *stubAuth, hits int32)
	}{
		{
			name: "refreshes and retries on unauthorized",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request, hits *atomic.Int32) {
				assert.Equal(t, "/v1/stocks", r.URL.Path)
				if hits.Add(1) == 1 {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				assert.Equal(t, "Bearer at-new", r.Header.Get("Authorization"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{}`))
			},
			call: func(t *testing.T, c *Client) error {
				return c.Get(context.Background(), "/v1/stocks", nil, nil)
			},
			check: func(t *testing.T, auth *stubAuth, hits int32) {
				assert.Equal(t, int32(1), auth.refreshed.Load())
				assert.Equal(t, int32(2), hits)
			},
		},
		{
			name: "does not retry when refresh fails",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request, hits *atomic.Int32) {
				hits.Add(1)
				w.WriteHeader(http.StatusUnauthorized)
			},
			call: func(t *testing.T, c *Client) error {
				return c.Get(context.Background(), "/v1/stocks", nil, nil)
			},
			wantErr: ErrUnauthorized,
			check: func(t *testing.T, auth *stubAuth, hits int32) {
				assert.Equal(t, int32(2), hits)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hits atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.handler(t, w, r, &hits)
			}))
			defer srv.Close()

			auth := &stubAuth{token: "at-old"}
			err := tt.call(t, New(
				WithBaseURL(srv.URL),
				WithAuthenticator(auth),
			))
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			if tt.check != nil {
				tt.check(t, auth, hits.Load())
			}
		})
	}
}
