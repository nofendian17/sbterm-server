package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool is the narrow interface *Postgres depends on. It is satisfied by
// *pgxpool.Pool in production and by *pgxmock.PgxPool in tests.
// Begin starts a transaction; *pgxpool.Pool and *pgxmock.PgxPool both satisfy it.
type Pool interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
	Ping(ctx context.Context) error
	Close()
}

type Postgres struct {
	pool Pool
}

type Option func(*options)

type options struct {
	maxConns        int32
	minConns        int32
	maxConnLifetime time.Duration
	maxConnIdleTime time.Duration
}

func WithMaxConns(n int32) Option {
	return func(o *options) { o.maxConns = n }
}

func WithMinConns(n int32) Option {
	return func(o *options) { o.minConns = n }
}

func WithMaxConnLifetime(d time.Duration) Option {
	return func(o *options) { o.maxConnLifetime = d }
}

func WithMaxConnIdleTime(d time.Duration) Option {
	return func(o *options) { o.maxConnIdleTime = d }
}

func (o *options) apply(cfg *pgxpool.Config) {
	if o.maxConns > 0 {
		cfg.MaxConns = o.maxConns
	}
	if o.minConns > 0 {
		cfg.MinConns = o.minConns
	}
	if o.maxConnLifetime > 0 {
		cfg.MaxConnLifetime = o.maxConnLifetime
	}
	if o.maxConnIdleTime > 0 {
		cfg.MaxConnIdleTime = o.maxConnIdleTime
	}
}

func New(ctx context.Context, dsn string, opts ...Option) (*Postgres, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("database: parse dsn: %w", err)
	}

	o := &options{}
	for _, opt := range opts {
		opt(o)
	}
	o.apply(cfg)

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("database: create pool: %w", err)
	}

	return &Postgres{pool: pool}, nil
}

func NewWithPool(pool Pool) *Postgres {
	return &Postgres{pool: pool}
}

func (p *Postgres) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

func (p *Postgres) Begin(ctx context.Context) (pgx.Tx, error) {
	return p.pool.Begin(ctx)
}

func (p *Postgres) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	return p.pool.BeginTx(ctx, txOptions)
}

// HealthCheck implements the samber/do health check hook.
func (p *Postgres) HealthCheck(ctx context.Context) error {
	return p.Ping(ctx)
}

func (p *Postgres) Shutdown() error {
	p.pool.Close()
	return nil
}
