package repository

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/repository"
)

// DBPinger is satisfied by *database.Postgres in production and by
// *pgxmock.PgxPoolIface in tests.
type DBPinger interface {
	Ping(ctx context.Context) error
}

type HealthRepository struct {
	db DBPinger
}

func NewHealthRepository(db DBPinger) *HealthRepository {
	return &HealthRepository{db: db}
}

func (r *HealthRepository) Ping(ctx context.Context) error {
	return r.db.Ping(ctx)
}

var _ repository.HealthRepository = (*HealthRepository)(nil)
