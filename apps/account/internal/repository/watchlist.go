package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/account/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=watchlist.go -destination=../mocks/mock_watchlist_repository.go -package=mocks -typed

// WatchlistRepository persists and retrieves user watchlists.
type WatchlistRepository interface {
	// ListByUser returns all watchlist entries for the given user.
	ListByUser(ctx context.Context, userID string) ([]domain.Watchlist, error)
	// Add inserts a new watchlist entry. A conflicting (user_id, symbol)
	// unique violation is mapped to domain.ErrDuplicateWatchlist.
	Add(ctx context.Context, w domain.Watchlist) error
	// RemoveBySymbol deletes the watchlist entry for the given user and symbol.
	RemoveBySymbol(ctx context.Context, userID, symbol string) error
}
