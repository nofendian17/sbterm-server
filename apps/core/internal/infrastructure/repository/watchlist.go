// Package repository implements the repository contracts using PostgreSQL/Redis backends.

package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/repository"
)

// watchlistRepository is the pgx implementation of repository.WatchlistRepository.
type WatchlistRepository struct {
	q repository.Querier
}

// NewWatchlistRepository builds a WatchlistRepository backed by the given Querier.
func NewWatchlistRepository(q repository.Querier) *WatchlistRepository {
	return &WatchlistRepository{q: q}
}

// ListByUser returns all non-deleted watchlist entries for the given user.
func (r *WatchlistRepository) ListByUser(ctx context.Context, userID string) ([]domain.Watchlist, error) {
	rows, err := r.q.Query(ctx,
		`SELECT id, user_id, symbol, label, created_at
		 FROM watchlists WHERE user_id = $1 AND deleted_at IS NULL ORDER BY created_at`, userID)
	if err != nil {
		return nil, fmt.Errorf("watchlist list: %w", err)
	}
	defer rows.Close()

	items := []domain.Watchlist{}
	for rows.Next() {
		var w domain.Watchlist
		if err := rows.Scan(&w.ID, &w.UserID, &w.Symbol, &w.Label, &w.CreatedAt); err != nil {
			return nil, fmt.Errorf("watchlist list scan: %w", err)
		}
		items = append(items, w)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("watchlist list rows: %w", err)
	}
	return items, nil
}

// Add inserts a new watchlist entry. A live conflict (the user already has
// a non-deleted entry for the same symbol) returns domain.ErrDuplicateWatchlist.
// A soft-deleted conflict is reactivated — a user who removed BBCA and
// re-adds it gets the row back, not a 409.
func (r *WatchlistRepository) Add(ctx context.Context, w domain.Watchlist) error {
	// Step 1: try to reactivate a soft-deleted row at this slot. If one
	// exists, we're done.
	res, err := r.q.Exec(ctx,
		`UPDATE watchlists
		 SET deleted_at = NULL, label = $3, updated_at = now()
		 WHERE user_id = $1 AND symbol = $2 AND deleted_at IS NOT NULL`,
		w.UserID, w.Symbol, w.Label,
	)
	if err != nil {
		return fmt.Errorf("watchlist add: %w", err)
	}
	if res.RowsAffected() > 0 {
		return nil
	}

	// Step 2: no soft-deleted row to reactivate. Try to insert a fresh one.
	// A live row at the same (user_id, symbol) triggers a unique violation
	// (the constraint treats (user_id, symbol) as the key, regardless of
	// deleted_at), which we map to ErrDuplicateWatchlist.
	const q = `INSERT INTO watchlists (user_id, symbol, label) VALUES ($1, $2, $3)`
	if _, err := r.q.Exec(ctx, q, w.UserID, w.Symbol, w.Label); err != nil {
		if isPgErrorCode(err, uniqueViolationCode) {
			return fmt.Errorf("watchlist add: %w", domain.ErrDuplicateWatchlist)
		}
		if isPgErrorCode(err, foreignKeyViolationCode) {
			// The stocks master table (000004) has no row for this symbol.
			// Under the authenticated flow the user_id FK can never fire (the
			// middleware loads the user first), so a FK breach here means an
			// unknown symbol.
			return fmt.Errorf("watchlist add: %w", domain.ErrStockNotFound)
		}
		return fmt.Errorf("watchlist add: %w", err)
	}
	return nil
}

// RemoveBySymbol soft-deletes the watchlist entry for the given user and
// symbol.
func (r *WatchlistRepository) RemoveBySymbol(ctx context.Context, userID, symbol string) error {
	const q = `
		UPDATE watchlists
		SET deleted_at = now()
		WHERE user_id = $1 AND symbol = $2 AND deleted_at IS NULL
	`
	if _, err := r.q.Exec(ctx, q, userID, symbol); err != nil {
		return fmt.Errorf("watchlist remove: %w", err)
	}
	return nil
}

// isPgErrorCode reports whether err is (or wraps) a Postgres error with the
// given SQLSTATE code.
func isPgErrorCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == code
	}
	return false
}
