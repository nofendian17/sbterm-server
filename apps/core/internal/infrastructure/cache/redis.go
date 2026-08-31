package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client is the narrow interface *Redis depends on. It is satisfied by
// *redis.Client in production and by fakes in tests.
type Client interface {
	Ping(ctx context.Context) *redis.StatusCmd
	Close() error
}

type Redis struct {
	client Client
}

type Option func(*options)

type options struct {
	maxRetries   int
	poolSize     int
	minIdleConns int
	dialTimeout  time.Duration
	readTimeout  time.Duration
	writeTimeout time.Duration
}

func WithMaxRetries(n int) Option {
	return func(o *options) { o.maxRetries = n }
}

func WithPoolSize(n int) Option {
	return func(o *options) { o.poolSize = n }
}

func WithMinIdleConns(n int) Option {
	return func(o *options) { o.minIdleConns = n }
}

func WithDialTimeout(d time.Duration) Option {
	return func(o *options) { o.dialTimeout = d }
}

func WithReadTimeout(d time.Duration) Option {
	return func(o *options) { o.readTimeout = d }
}

func WithWriteTimeout(d time.Duration) Option {
	return func(o *options) { o.writeTimeout = d }
}

func (o *options) apply(cfg *redis.Options) {
	if o.maxRetries > 0 {
		cfg.MaxRetries = o.maxRetries
	}
	if o.poolSize > 0 {
		cfg.PoolSize = o.poolSize
	}
	if o.minIdleConns > 0 {
		cfg.MinIdleConns = o.minIdleConns
	}
	if o.dialTimeout > 0 {
		cfg.DialTimeout = o.dialTimeout
	}
	if o.readTimeout > 0 {
		cfg.ReadTimeout = o.readTimeout
	}
	if o.writeTimeout > 0 {
		cfg.WriteTimeout = o.writeTimeout
	}
}

func New(ctx context.Context, url string, opts ...Option) (*Redis, error) {
	cfg, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("cache: parse redis url: %w", err)
	}

	o := &options{}
	for _, opt := range opts {
		opt(o)
	}
	o.apply(cfg)

	client := redis.NewClient(cfg)
	return &Redis{client: client}, nil
}

func NewWithClient(client Client) *Redis {
	return &Redis{client: client}
}

func (r *Redis) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Cmdable exposes the underlying Redis client for advanced operations.
// Returns nil when a non-Redis fake was injected.
func (r *Redis) Cmdable() redis.Cmdable {
	c, ok := r.client.(redis.Cmdable)
	if !ok {
		return nil
	}
	return c
}

// HealthCheck implements the samber/do health check hook.
func (r *Redis) HealthCheck(ctx context.Context) error {
	return r.Ping(ctx)
}

func (r *Redis) Shutdown() error {
	return r.client.Close()
}

// Client returns the underlying *redis.Client. This is used by the container
// to pass the client to repository constructors that need it.
func (r *Redis) Client() *redis.Client {
	c, ok := r.client.(*redis.Client)
	if !ok {
		return nil
	}
	return c
}
