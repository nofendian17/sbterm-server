// Package repository implements the repository contracts using PostgreSQL/Redis backends.

package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/nofendian17/sbterm/apps/core/internal/repository"
)

// PGXQuerier is the narrow structural subset of *pgxpool.Pool, pgx.Tx and
// pgxmock pools that repositories need. All three satisfy it implicitly.
// It lives in infrastructure (which may import pgx) so the port
// (internal/repository) stays driver-free.
type PGXQuerier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// querierAdapter wraps a PGXQuerier so it satisfies repository.Querier.
// pgconn.CommandTag already exposes RowsAffected() int64 and pgx.Rows/pgx.Row
// already expose Close/Err/Next/Scan, so the adaptation is structural with no
// behavior change.
type querierAdapter struct {
	q PGXQuerier
}

func (a querierAdapter) Exec(ctx context.Context, sql string, args ...any) (repository.CommandResult, error) {
	return a.q.Exec(ctx, sql, args...)
}

func (a querierAdapter) Query(ctx context.Context, sql string, args ...any) (repository.Rows, error) {
	return a.q.Query(ctx, sql, args...)
}

func (a querierAdapter) QueryRow(ctx context.Context, sql string, args ...any) repository.Row {
	return a.q.QueryRow(ctx, sql, args...)
}

var _ repository.Querier = querierAdapter{}

// AdaptQuerier converts a pgx-native querier (pool, tx or pgxmock) into a
// repository.Querier. Call it at the infrastructure boundary (container,
// TxManager, tests) — never in usecases.
func AdaptQuerier(q PGXQuerier) repository.Querier {
	return querierAdapter{q: q}
}
