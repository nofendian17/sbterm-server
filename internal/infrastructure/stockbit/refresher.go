package stockbit

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/nofendian17/sbterm-server/pkg/log"
)

// Credentials are the account details used to obtain the first token pair.
type Credentials struct {
	PlayerID string
	Username string
	Password string
}

// refreshSkew keeps the client refreshing before the access token actually
// expires, so requests never hit the server with a token already stale.
const refreshSkew = time.Minute

const (
	backoffInitial = time.Second
	backoffMax     = 30 * time.Second
)

// Refresher keeps the Stockbit access token fresh. It refreshes proactively
// ahead of expiry and falls back to a full login when the refresh token is
// rejected. Concurrent callers (request path and background timer) are
// serialized so a rotating refresh token is never used twice.
type Refresher struct {
	client *Client
	store  TokenStore
	creds  Credentials
	logger log.Logger

	mu     sync.Mutex
	skew   time.Duration
	ctx    context.Context
	cancel context.CancelFunc
}

func NewRefresher(client *Client, store TokenStore, creds Credentials, logger log.Logger) *Refresher {
	ctx, cancel := context.WithCancel(context.Background())
	return &Refresher{
		client: client,
		store:  store,
		creds:  creds,
		logger: logger,
		skew:   refreshSkew,
		ctx:    ctx,
		cancel: cancel,
	}
}

// EnsureToken returns a valid access token, refreshing or logging in if needed.
func (r *Refresher) EnsureToken(ctx context.Context) (string, error) {
	td, err := r.store.Get(ctx)
	if err != nil {
		return "", err
	}
	if td != nil && time.Now().Before(td.accessExpiry().Add(-r.skew)) {
		return td.Access.Token, nil
	}
	return r.refresh(ctx, false)
}

// Refresh forces a token refresh (falling back to login) and returns the new
// access token. It is used after a request is rejected with 401 even though the
// token looked valid.
func (r *Refresher) Refresh(ctx context.Context) (string, error) {
	return r.refresh(ctx, true)
}

// refresh serializes refreshes and, unless forced, skips when another caller
// already produced a valid token.
func (r *Refresher) refresh(ctx context.Context, force bool) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	td, err := r.store.Get(ctx)
	if err != nil {
		return "", err
	}
	if !force && td != nil && time.Now().Before(td.accessExpiry().Add(-r.skew)) {
		return td.Access.Token, nil
	}
	if td != nil && td.Refresh.Token != "" {
		resp, err := r.client.Refresh(ctx, td.Refresh.Token)
		if err == nil {
			fresh := &TokenData{Access: resp.Data.Access, Refresh: resp.Data.Refresh}
			if err := r.store.Set(ctx, fresh); err != nil {
				return "", err
			}
			r.logger.Info("stockbit access token refreshed",
				"expires_at", fresh.Access.ExpiredAt)
			return fresh.Access.Token, nil
		}
		if !errors.Is(err, ErrUnauthorized) {
			return "", err
		}
	}
	// No tokens yet, or the refresh token was rejected: log in fresh.
	resp, err := r.client.Login(ctx, LoginRequest{
		PlayerID: r.creds.PlayerID,
		User:     r.creds.Username,
		Password: r.creds.Password,
	})
	if err != nil {
		return "", err
	}
	td = &TokenData{
		Access:  resp.Data.Login.TokenData.Access,
		Refresh: resp.Data.Login.TokenData.Refresh,
	}
	if err := r.store.Set(ctx, td); err != nil {
		return "", err
	}
	r.logger.Info("stockbit logged in",
		"username", r.creds.Username,
		"access_expires_at", td.Access.ExpiredAt,
		"refresh_expires_at", td.Refresh.ExpiredAt)
	return td.Access.Token, nil
}

// Start runs the proactive refresh loop until Shutdown is called.
func (r *Refresher) Start() {
	go func() {
		backoff := backoffInitial
		for {
			if _, err := r.EnsureToken(r.ctx); err != nil {
				r.logger.Warn("stockbit token refresh failed",
					"error", err, "retry_in", backoff.String())
				if !sleepCtx(r.ctx, backoff) {
					return
				}
				backoff = min(backoff*2, backoffMax)
				continue
			}
			backoff = backoffInitial

			td, err := r.store.Get(r.ctx)
			if err != nil {
				r.logger.Warn("stockbit load token failed", "error", err)
				if !sleepCtx(r.ctx, backoffMax) {
					return
				}
				continue
			}
			wait := backoffInitial
			if td != nil {
				if w := time.Until(td.accessExpiry().Add(-r.skew)); w > 0 {
					wait = w
				}
			}
			if !sleepCtx(r.ctx, wait) {
				return
			}
		}
	}()
}

// Shutdown stops the background refresh loop.
func (r *Refresher) Shutdown() error {
	r.cancel()
	return nil
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	}
}
