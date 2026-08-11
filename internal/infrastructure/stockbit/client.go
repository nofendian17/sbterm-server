// Package stockbit provides a typed REST client for the Exodus Stockbit API.
package stockbit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	pkghttpclient "github.com/nofendian17/sbterm-server/pkg/httpclient"
	"github.com/nofendian17/sbterm-server/pkg/log"
)

// defaultBaseURL is the third-party endpoint the client talks to.
const defaultBaseURL = "https://exodus.stockbit.com"

// Authenticator supplies a valid access token, refreshing as needed. It is
// implemented by *Refresher and faked in tests.
type Authenticator interface {
	EnsureToken(ctx context.Context) (string, error)
	Refresh(ctx context.Context) (string, error)
}

// ErrUnauthorized marks responses whose bearer token was rejected (HTTP 401).
var ErrUnauthorized = errors.New("stockbit: unauthorized")

// ErrRateLimited marks responses where the upstream API returned HTTP 429.
var ErrRateLimited = errors.New("stockbit: rate limited")

const maxRetries = 3

// StatusError reports a non-2xx response, carrying the upstream HTTP status
// code so callers can distinguish client errors (4xx) from upstream failures
// (5xx) without parsing the message. Method and Path preserve the request
// context in the error text.
type StatusError struct {
	Status int
	Method string
	Path   string
	Msg    string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("stockbit: %s %s: unexpected status %d: %s", e.Method, e.Path, e.Status, e.Msg)
}

// defaultHeaders are sent with every request. Individual values can be
// overridden through WithHeader.
var defaultHeaders = map[string]string{
	"User-Agent":      "Stockbit/3.22.1 (stockbit.com.stockbit; build:40970; iOS 26.5.2) Alamofire/5.9.0",
	"X-DeviceType":    "iPhone 12",
	"X-Platform":      "iOS",
	"X-AppVersion":    "3.22.1",
	"Content-Type":    "application/json",
	"Accept-Language": "ID",
}

type Option func(*options)

type options struct {
	baseURL    string
	timeout    time.Duration
	retryCount int
	doer       pkghttpclient.Doer
	headers    map[string]string
	auth       Authenticator
	logger     log.Logger
}

func WithBaseURL(u string) Option {
	return func(o *options) { o.baseURL = u }
}

func WithTimeout(d time.Duration) Option {
	return func(o *options) { o.timeout = d }
}

func WithRetryCount(n int) Option {
	return func(o *options) { o.retryCount = n }
}

func WithHTTPClient(d pkghttpclient.Doer) Option {
	return func(o *options) { o.doer = d }
}

func WithHeader(name, value string) Option {
	return func(o *options) {
		if o.headers == nil {
			o.headers = make(map[string]string)
		}
		o.headers[name] = value
	}
}

// WithAuthenticator enables automatic bearer-token management: requests get an
// Authorization header (except the login endpoint, which has no token), and a
// 401 response triggers one refresh-and-retry.
func WithAuthenticator(a Authenticator) Option {
	return func(o *options) { o.auth = a }
}

// SetAuthenticator attaches the token provider used for automatic auth. It
// exists so the authenticator can be built from the client itself.
func (c *Client) SetAuthenticator(a Authenticator) { c.auth = a }

// WithLogger enables per-request debug logging.
func WithLogger(l log.Logger) Option {
	return func(o *options) { o.logger = l }
}

type Client struct {
	h       pkghttpclient.Client
	baseURL string
	headers map[string]string
	auth    Authenticator
	logger  log.Logger
}

func New(opts ...Option) *Client {
	o := &options{
		baseURL: defaultBaseURL,
		timeout: 30 * time.Second,
		headers: maps.Clone(defaultHeaders),
	}
	for _, opt := range opts {
		opt(o)
	}

	copts := []pkghttpclient.Option{pkghttpclient.WithTimeout(o.timeout)}
	if o.doer != nil {
		copts = append(copts, pkghttpclient.WithHTTPClient(o.doer))
	}
	if o.retryCount > 0 {
		copts = append(copts,
			pkghttpclient.WithRetryCount(o.retryCount),
			pkghttpclient.WithConstantBackoff(500*time.Millisecond, 50*time.Millisecond),
		)
	}

	return &Client{
		h:       pkghttpclient.NewClient(copts...),
		baseURL: strings.TrimRight(o.baseURL, "/"),
		headers: o.headers,
		auth:    o.auth,
		logger:  o.logger,
	}
}

// Get performs a GET request against an absolute API path and decodes a 2xx
// JSON response body into out. A non-2xx status returns an error carrying the
// response body for diagnostics.
func (c *Client) Get(ctx context.Context, path string, query url.Values, out any) error {
	return c.do(ctx, http.MethodGet, path, query, nil, nil, out)
}

// Post performs a POST request with the given body and decodes a 2xx JSON
// response body into out.
func (c *Client) Post(ctx context.Context, path string, body io.Reader, out any) error {
	return c.do(ctx, http.MethodPost, path, nil, body, nil, out)
}

// do performs a request; extra headers override client-level headers.
func (c *Client) do(ctx context.Context, method, path string, query url.Values, body io.Reader, extra map[string]string, out any) error {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return fmt.Errorf("stockbit: parse url: %w", err)
	}
	if len(query) > 0 {
		u.RawQuery = query.Encode()
	}

	// Buffer the body so the request can be rebuilt for a 401 retry.
	var bodyBytes []byte
	if body != nil {
		if bodyBytes, err = io.ReadAll(body); err != nil {
			return fmt.Errorf("stockbit: read body: %w", err)
		}
	}

	build := func(access string) (*http.Request, error) {
		var rd io.Reader
		if bodyBytes != nil {
			rd = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequestWithContext(ctx, method, u.String(), rd)
		if err != nil {
			return nil, fmt.Errorf("stockbit: build request: %w", err)
		}
		for name, value := range c.headers {
			req.Header.Set(name, value)
		}
		if access != "" {
			req.Header.Set("Authorization", "Bearer "+access)
		}
		for name, value := range extra {
			req.Header.Set(name, value)
		}
		return req, nil
	}

	_, explicitAuth := extra["Authorization"]
	autoAuth := c.auth != nil && path != loginPath && !explicitAuth

	attempts := 1
	if autoAuth {
		attempts = 2
	}
	var retries int
	// The loop accounts for both the base attempt (plus 401 auth retry)
	// and up to maxRetries additional iterations for HTTP 429 rate-limit retries.
	for attempt := 0; attempt < attempts+maxRetries; attempt++ {
		var access string
		if autoAuth {
			if access, err = c.auth.EnsureToken(ctx); err != nil {
				return fmt.Errorf("stockbit: authenticate %s %s: %w", method, path, err)
			}
		}

		req, err := build(access)
		if err != nil {
			return err
		}

		start := time.Now()
		resp, err := c.h.Do(req)
		if err != nil {
			if c.logger != nil {
				c.logger.Debug("stockbit request",
					"method", method, "path", path, "status", 0, "error", err.Error())
			}
			return fmt.Errorf("stockbit: request %s %s: %w", method, path, err)
		}

		respBytes, rerr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if rerr != nil {
			return fmt.Errorf("stockbit: read response %s %s: %w", method, path, rerr)
		}

		if c.logger != nil {
			reqBody := ""
			respBody := ""
			if path != loginPath && path != refreshPath && path != websocketKeyPath {
				reqBody = truncate(string(bodyBytes))
				respBody = truncate(string(respBytes))
			}
			c.logger.Debug("stockbit request",
				"method", method, "path", path, "status", resp.StatusCode,
				"duration", time.Since(start).Round(time.Millisecond).String(),
				"request_body", reqBody,
				"response_body", respBody)
		}

		if resp.StatusCode == http.StatusUnauthorized && autoAuth && attempt == 0 {
			if _, err := c.auth.Refresh(ctx); err != nil {
				return fmt.Errorf("stockbit: %s %s: unauthorized and refresh failed: %w", method, path, err)
			}
			continue
		}

		// Honor HTTP 429 rate limiting: wait out the Retry-After hint (falling
		// back to a fixed backoff) and retry up to maxRetries times.
		if resp.StatusCode == http.StatusTooManyRequests && retries < maxRetries {
			if c.logger != nil {
				c.logger.Debug("stockbit rate limited",
					"method", method, "path", path, "retry", retries+1, "max", maxRetries)
			}
			if err := sleepRetryAfter(ctx, resp.Header.Get("Retry-After")); err != nil {
				return fmt.Errorf("stockbit: %s %s: rate limit wait cancelled: %w", method, path, err)
			}
			retries++
			continue
		}

		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			msg := truncate(string(respBytes))
			err := &StatusError{Status: resp.StatusCode, Method: method, Path: path, Msg: strings.TrimSpace(msg)}
			if resp.StatusCode == http.StatusUnauthorized {
				return errors.Join(ErrUnauthorized, err)
			}
			if resp.StatusCode == http.StatusTooManyRequests {
				return errors.Join(ErrRateLimited, err)
			}
			return err
		}

		if out != nil {
			if err := json.Unmarshal(respBytes, out); err != nil {
				return fmt.Errorf("stockbit: decode %s %s: %w", method, path, err)
			}
		}
		return nil
	}
	return nil
}

// truncate caps a debug-log payload to keep logs readable. It cuts at a rune
// boundary so truncated UTF-8 stays valid.
func truncate(s string) string {
	const max = 4096
	if len(s) <= max {
		return s
	}
	s = s[:max]
	_, size := utf8.DecodeLastRuneInString(s)
	return s[:len(s)-size] + "..."
}

// sleepRetryAfter parses the Retry-After header (integer seconds or HTTP-date)
// and sleeps until the deadline. Falls back to a fixed 1-second backoff if the
// header is absent or unparseable. Returns ctx.Err() if the context is done
// before the sleep completes.
func sleepRetryAfter(ctx context.Context, retryAfter string) error {
	const fallback = time.Second
	dur := fallback
	if retryAfter != "" {
		if secs, err := strconv.Atoi(retryAfter); err == nil && secs >= 0 {
			dur = time.Duration(secs) * time.Second
		} else if t, err := http.ParseTime(retryAfter); err == nil {
			dur = time.Until(t)
			if dur < 0 {
				dur = fallback
			}
		}
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(dur):
		return nil
	}
}
