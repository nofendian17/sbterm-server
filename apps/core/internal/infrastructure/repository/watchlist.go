package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/repository"
)

// watchlistRepository is the pgx implementation of repository.WatchlistRepository.
type watchlistRepository struct {
	q repository.Querier
}

// NewWatchlistRepository builds a WatchlistRepository backed by the given Querier.
func NewWatchlistRepository(q repository.Querier) repository.WatchlistRepository {
	return &watchlistRepository{q: q}
}

// ListByUser returns all watchlist entries for the given user.
func (r *watchlistRepository) ListByUser(ctx context.Context, userID string) ([]domain.Watchlist, error) {
	rows, err := r.q.Query(ctx,
		`SELECT id, user_id, symbol, label, created_at
		 FROM watchlists WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, fmt.Errorf("watchlist list: %w", err)
	}
	defer rows.Close()

	var items []domain.Watchlist
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

// Add inserts a new watchlist entry. A conflicting (user_id, symbol) maps to
// domain.ErrDuplicateWatchlist.
func (r *watchlistRepository) Add(ctx context.Context, w domain.Watchlist) error {
	const q = `
		INSERT INTO watchlists (user_id, symbol, label)
		VALUES ($1, $2, $3)
	`
	_, err := r.q.Exec(ctx, q, w.UserID, w.Symbol, w.Label)
	if err != nil {
		if isWatchlistUniqueViolation(err) {
			return fmt.Errorf("watchlist add: %w", domain.ErrDuplicateWatchlist)
		}
		return fmt.Errorf("watchlist add: %w", err)
	}
	return nil
}

// RemoveBySymbol deletes the watchlist entry for the given user and symbol.
func (r *watchlistRepository) RemoveBySymbol(ctx context.Context, userID, symbol string) error {
	const q = `DELETE FROM watchlists WHERE user_id = $1 AND symbol = $2`
	if _, err := r.q.Exec(ctx, q, userID, symbol); err != nil {
		return fmt.Errorf("watchlist remove: %w", err)
	}
	return nil
}

// isWatchlistUniqueViolation checks if err is a Postgres unique violation.
func isWatchlistUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == uniqueViolationCode
	}
	return false
}

// Ensure sql.ErrNoRows is used (for potential future use).
var _ = sql.ErrNoRows
