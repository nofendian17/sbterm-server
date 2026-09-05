// Package stockapi is the HTTP adapter for the apps/api stock catalog
// endpoint (docs/api.md, GET /api/v1/stocks). It implements
// repository.StockSyncClient so the StockUsecase can refresh the local
// catalog without depending on any HTTP library.
package stockapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
)

// Client is a thin HTTP client for the apps/api "list stocks" endpoint.
// It is safe for concurrent use.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient builds a client with the given base URL (e.g.
// "http://localhost:8080") and a per-request timeout shared by all calls.
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: timeout},
	}
}

// upstreamStock is the subset of the apps/api /stocks payload that the core
// catalog persists. The market-data fields (last/change/percent/volume/...)
// are ignored; company_status drives IsActive.
type upstreamStock struct {
	Symbol        string `json:"symbol"`
	Name          string `json:"name"`
	IconURL       string `json:"icon_url"`
	CompanyStatus string `json:"company_status"`
}

// upstreamEnvelope mirrors the libs/pkg/response envelope used by apps/api.
type upstreamEnvelope struct {
	Success bool            `json:"success"`
	Data    []upstreamStock `json:"data"`
}

// ListSymbols fetches the catalog from GET {baseURL}/api/v1/stocks and maps
// it into domain.Stock values ready for upsert.
func (c *Client) ListSymbols(ctx context.Context) ([]domain.Stock, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("stockapi: parse base url: %w", err)
	}
	u.Path = "/api/v1/stocks"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("stockapi: new request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stockapi: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("stockapi: upstream status %d", resp.StatusCode)
	}

	var env upstreamEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, fmt.Errorf("stockapi: decode response: %w", err)
	}

	out := make([]domain.Stock, 0, len(env.Data))
	for _, r := range env.Data {
		if r.Symbol == "" || r.Name == "" {
			// Skip malformed rows rather than fail the whole sync.
			continue
		}
		out = append(out, domain.Stock{
			Symbol:   r.Symbol,
			Name:     r.Name,
			IconURL:  nullable(r.IconURL),
			IsActive: r.CompanyStatus == "STATUS_ACTIVE",
		})
	}
	return out, nil
}

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
