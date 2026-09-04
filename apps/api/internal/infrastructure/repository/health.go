package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/repository"
)

// CachePinger is satisfied by *cache.Redis in production and by a fake in tests.
type CachePinger interface {
	Ping(ctx context.Context) error
}

type HealthRepository struct {
	cache CachePinger
}

func NewHealthRepository(cache CachePinger) *HealthRepository {
	return &HealthRepository{cache: cache}
}

func (r *HealthRepository) PingCache(ctx context.Context) error {
	return r.cache.Ping(ctx)
}

var _ repository.HealthRepository = (*HealthRepository)(nil)
