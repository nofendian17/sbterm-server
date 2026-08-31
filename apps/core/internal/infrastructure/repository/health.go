package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/core/internal/repository"
)

// DBPinger is satisfied by *database.Postgres in production.
type DBPinger interface {
	Ping(ctx context.Context) error
}

// RedisPinger is satisfied by *cache.Redis in production.
type RedisPinger interface {
	Ping(ctx context.Context) error
}

type HealthRepository struct {
	db    DBPinger
	redis RedisPinger
}

// NewHealthRepository builds a HealthRepository backed by the given pingers.
func NewHealthRepository(db DBPinger, redis RedisPinger) *HealthRepository {
	return &HealthRepository{db: db, redis: redis}
}

func (r *HealthRepository) Ping(ctx context.Context) error {
	return r.db.Ping(ctx)
}

func (r *HealthRepository) PingRedis(ctx context.Context) error {
	return r.redis.Ping(ctx)
}

var _ repository.HealthRepository = (*HealthRepository)(nil)
