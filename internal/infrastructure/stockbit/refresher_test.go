package stockbit

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nofendian17/sbterm-server/pkg/log"
)

const (
	expired  = "2020-01-01T00:00:00Z"
	notAfter = "2030-01-01T00:00:00Z"
)

func testLogger() log.Logger {
	return log.New(log.WithWriter(io.Discard))
}

type authServer struct {
	srv       *httptest.Server
	logins    atomic.Int32
	refreshes atomic.Int32
}

// newAuthServer wires /login/v6/username and /login/refresh to the given
// handlers and counts how often each is hit.
func newAuthServer(t *testing.T, login, refresh http.HandlerFunc) *authServer {
	t.Helper()
	as := &authServer{}
	as.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case loginPath:
			as.logins.Add(1)
			login(w, r)
		case refreshPath:
			as.refreshes.Add(1)
			refresh(w, r)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	t.Cleanup(as.srv.Close)
	return as
}

func writeRefreshResponse(w http.ResponseWriter, access, refresh string) {
	w.Write([]byte(`{"message":"ok","data":{"access":{"token":"` + access +
		`","expired_at":"` + notAfter + `"},"refresh":{"token":"` + refresh +
		`","expired_at":"` + notAfter + `"}}}`))
}

func writeLoginResponse(w http.ResponseWriter, access, refresh string) {
	w.Write([]byte(`{"message":"ok","data":{"login":{"user":{},"token_data":{"access":{"token":"` +
		access + `","expired_at":"` + notAfter + `"},"refresh":{"token":"` + refresh +
		`","expired_at":"` + notAfter + `"}}}}}`))
}

func newRefresher(t *testing.T, as *authServer, store TokenStore) *Refresher {
	t.Helper()
	r := NewRefresher(New(WithBaseURL(as.srv.URL)), store, Credentials{
		PlayerID: "p1", Username: "budi", Password: "secret",
	}, testLogger())
	r.skew = 0
	return r
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met before deadline")
}

func TestEnsureToken(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name            string
		setup           func(t *testing.T, store TokenStore)
		login           http.HandlerFunc
		refresh         http.HandlerFunc
		wantToken       string
		wantLogins      int32
		wantRefreshes   int32
		wantStoredToken string
	}{
		{
			name: "uses valid stored token",
			setup: func(t *testing.T, store TokenStore) {
				require.NoError(t, store.Set(ctx, &TokenData{
					Access:  TokenPair{Token: "at-ok", ExpiredAt: notAfter},
					Refresh: TokenPair{Token: "rt-ok", ExpiredAt: notAfter},
				}))
			},
			login:     func(w http.ResponseWriter, r *http.Request) { t.Error("login must not be called") },
			refresh:   func(w http.ResponseWriter, r *http.Request) { t.Error("refresh must not be called") },
			wantToken: "at-ok",
		},
		{
			name: "refreshes expired access token",
			setup: func(t *testing.T, store TokenStore) {
				require.NoError(t, store.Set(ctx, &TokenData{
					Access:  TokenPair{Token: "at-old", ExpiredAt: expired},
					Refresh: TokenPair{Token: "rt-1", ExpiredAt: notAfter},
				}))
			},
			login:           func(w http.ResponseWriter, r *http.Request) { t.Error("login must not be called") },
			refresh:         func(w http.ResponseWriter, r *http.Request) { writeRefreshResponse(w, "at-2", "rt-2") },
			wantToken:       "at-2",
			wantRefreshes:   1,
			wantStoredToken: "rt-2",
		},
		{
			name:       "logs in when no tokens",
			login:      func(w http.ResponseWriter, r *http.Request) { writeLoginResponse(w, "at-1", "rt-1") },
			refresh:    func(w http.ResponseWriter, r *http.Request) { t.Error("refresh must not be called") },
			wantToken:  "at-1",
			wantLogins: 1,
		},
		{
			name: "falls back to login when refresh rejected",
			setup: func(t *testing.T, store TokenStore) {
				require.NoError(t, store.Set(ctx, &TokenData{
					Access:  TokenPair{Token: "at-old", ExpiredAt: expired},
					Refresh: TokenPair{Token: "rt-old", ExpiredAt: notAfter},
				}))
			},
			login:         func(w http.ResponseWriter, r *http.Request) { writeLoginResponse(w, "at-3", "rt-3") },
			refresh:       func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusUnauthorized) },
			wantToken:     "at-3",
			wantLogins:    1,
			wantRefreshes: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, _ := newTestStore(t)
			if tt.setup != nil {
				tt.setup(t, store)
			}

			as := newAuthServer(t, tt.login, tt.refresh)
			tok, err := newRefresher(t, as, store).EnsureToken(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantToken, tok)
			assert.Equal(t, tt.wantLogins, as.logins.Load())
			assert.Equal(t, tt.wantRefreshes, as.refreshes.Load())

			if tt.wantStoredToken != "" {
				td, err := store.Get(ctx)
				require.NoError(t, err)
				assert.Equal(t, tt.wantStoredToken, td.Refresh.Token)
			}
		})
	}
}

func TestConcurrentEnsureTokenRefreshesOnce(t *testing.T) {
	tests := []struct {
		name          string
		concurrency   int
		wantRefreshes int32
		wantLogins    int32
	}{
		{name: "ten concurrent calls refresh once", concurrency: 10, wantRefreshes: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			store, _ := newTestStore(t)
			require.NoError(t, store.Set(ctx, &TokenData{
				Access:  TokenPair{Token: "at-old", ExpiredAt: expired},
				Refresh: TokenPair{Token: "rt-1", ExpiredAt: notAfter},
			}))

			as := newAuthServer(t,
				func(w http.ResponseWriter, r *http.Request) { t.Error("login must not be called") },
				func(w http.ResponseWriter, r *http.Request) { writeRefreshResponse(w, "at-2", "rt-2") },
			)

			r := newRefresher(t, as, store)

			var wg sync.WaitGroup
			errs := make(chan error, tt.concurrency)
			for i := 0; i < tt.concurrency; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					if _, err := r.EnsureToken(ctx); err != nil {
						errs <- err
					}
				}()
			}
			wg.Wait()
			close(errs)
			for err := range errs {
				t.Errorf("EnsureToken: %v", err)
			}

			assert.Equal(t, tt.wantRefreshes, as.refreshes.Load())
			assert.Equal(t, tt.wantLogins, as.logins.Load())
		})
	}
}

func TestStartRefreshesAheadOfExpiryAndStopsOnShutdown(t *testing.T) {
	tests := []struct {
		name string
	}{{name: "refreshes ahead of expiry and stops on shutdown"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			store, _ := newTestStore(t)
			require.NoError(t, store.Set(ctx, &TokenData{
				Access:  TokenPair{Token: "at-old", ExpiredAt: time.Now().Add(50 * time.Millisecond).Format(time.RFC3339)},
				Refresh: TokenPair{Token: "rt-1", ExpiredAt: notAfter},
			}))

			as := newAuthServer(t,
				func(w http.ResponseWriter, r *http.Request) { t.Error("login must not be called") },
				func(w http.ResponseWriter, r *http.Request) { writeRefreshResponse(w, "at-2", "rt-2") },
			)

			r := newRefresher(t, as, store)
			r.Start()
			defer r.Shutdown()

			waitFor(t, func() bool {
				td, err := store.Get(ctx)
				return err == nil && td != nil && td.Access.Token == "at-2"
			})
			assert.Equal(t, int32(1), as.refreshes.Load())
		})
	}
}
