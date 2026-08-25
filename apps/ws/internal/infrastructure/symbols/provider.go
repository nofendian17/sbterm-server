// Package symbols resolves the tradable symbol universe from the sbterm API.
// The provider caches successful responses and keeps serving the last good
// list when the API is down: a stale subscription list beats an empty one,
// but a failing provider must never silently subscribe nothing.
package symbols

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

const stocksPath = "/api/v1/stocks"

type envelope struct {
	Success bool `json:"success"`
	Data    []struct {
		Symbol string `json:"symbol"`
	} `json:"data"`
}

// Provider fetches the IHSG constituent list from GET {base}/api/v1/stocks
// and caches it for ttl. It is safe for concurrent use.
type Provider struct {
	base string
	hc   *http.Client
	ttl  time.Duration

	mu       sync.Mutex
	cached   []string
	cachedAt time.Time
}

// NewProvider builds a provider against an sbterm api base URL. A non-positive
// http client or ttl falls back to defaults (10s client, 10m cache).
func New(base string, hc *http.Client, ttl time.Duration) *Provider {
	if hc == nil {
		hc = &http.Client{Timeout: 10 * time.Second}
	}
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	return &Provider{base: strings.TrimRight(base, "/"), hc: hc, ttl: ttl}
}

// Symbols returns the current symbol universe, serving the cache while fresh
// and falling back to a stale cache when refresh fails. An empty universe is
// never cached or returned as success without data.
func (p *Provider) Symbols(ctx context.Context) ([]string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.cached) > 0 && time.Since(p.cachedAt) < p.ttl {
		return cloneSymbols(p.cached), nil
	}

	syms, err := p.fetch(ctx)
	if err != nil {
		if len(p.cached) > 0 {
			return cloneSymbols(p.cached), nil
		}
		return nil, err
	}
	if len(syms) == 0 {
		if len(p.cached) > 0 {
			return cloneSymbols(p.cached), nil
		}
		return nil, fmt.Errorf("symbols: %s returned no symbols", p.base)
	}

	p.cached = syms
	p.cachedAt = time.Now()
	return cloneSymbols(syms), nil
}

func (p *Provider) fetch(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.base+stocksPath, nil)
	if err != nil {
		return nil, fmt.Errorf("symbols: build request: %w", err)
	}
	resp, err := p.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("symbols: get %s: %w", stocksPath, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("symbols: get %s: status %d", stocksPath, resp.StatusCode)
	}

	var env envelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, fmt.Errorf("symbols: decode %s response: %w", stocksPath, err)
	}
	syms := make([]string, 0, len(env.Data))
	for _, s := range env.Data {
		if sym := strings.ToUpper(strings.TrimSpace(s.Symbol)); sym != "" {
			syms = append(syms, sym)
		}
	}
	return syms, nil
}

func cloneSymbols(in []string) []string {
	out := make([]string, len(in))
	copy(out, in)
	return out
}
