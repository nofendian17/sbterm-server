// Package httpclient provides a thin wrapper around gojek/heimdall so that the
// underlying retry/backoff implementation can be swapped without touching
// consumers. The returned Client matches the standard net/http contract.
package httpclient

import (
	"io"
	"net/http"
	"time"

	heimdall "github.com/gojek/heimdall/v8"
	heimdallhttp "github.com/gojek/heimdall/v8/httpclient"
)

type Client interface {
	Do(req *http.Request) (*http.Response, error)
	Get(url string, headers http.Header) (*http.Response, error)
	Post(url string, headers http.Header, body io.Reader) (*http.Response, error)
}

type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

type Option func(*options)

type options struct {
	timeout    time.Duration
	retryCount int
	backoff    heimdall.Backoff
	doer       Doer
}

func WithTimeout(d time.Duration) Option {
	return func(o *options) { o.timeout = d }
}

func WithRetryCount(n int) Option {
	return func(o *options) { o.retryCount = n }
}

func WithConstantBackoff(interval, maxJitter time.Duration) Option {
	return func(o *options) {
		o.backoff = heimdall.NewConstantBackoff(interval, maxJitter)
	}
}

func WithExponentialBackoff(initialTimeout, maxTimeout time.Duration, exponentFactor float64, maxJitter time.Duration) Option {
	return func(o *options) {
		o.backoff = heimdall.NewExponentialBackoff(initialTimeout, maxTimeout, exponentFactor, maxJitter)
	}
}

func WithHTTPClient(d Doer) Option {
	return func(o *options) { o.doer = d }
}

type client struct {
	c *heimdallhttp.Client
}

func NewClient(opts ...Option) Client {
	o := &options{
		timeout: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(o)
	}

	copts := []heimdallhttp.Option{heimdallhttp.WithHTTPTimeout(o.timeout)}
	if o.doer != nil {
		copts = append(copts, heimdallhttp.WithHTTPClient(o.doer))
	}
	if o.retryCount > 0 {
		copts = append(copts, heimdallhttp.WithRetryCount(o.retryCount))
		if o.backoff == nil {
			o.backoff = heimdall.NewConstantBackoff(500*time.Millisecond, 5*time.Millisecond)
		}
		copts = append(copts, heimdallhttp.WithRetrier(heimdall.NewRetrier(o.backoff)))
	}

	return &client{c: heimdallhttp.NewClient(copts...)}
}

func (c *client) Do(req *http.Request) (*http.Response, error) {
	return c.c.Do(req)
}

func (c *client) Get(url string, headers http.Header) (*http.Response, error) {
	return c.c.Get(url, headers)
}

func (c *client) Post(url string, headers http.Header, body io.Reader) (*http.Response, error) {
	return c.c.Post(url, body, headers)
}
