// Package usecase implements the business logic for the core domain.

package usecase

import (
	"context"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=watchlist.go -destination=../mocks/mock_watchlist_usecase.go -package=mocks -typed

// WatchlistUsecase manages per-user symbol watchlists.
type WatchlistUsecase interface {
	// List returns the user's watchlist.
	List(ctx context.Context, userID string) ([]domain.Watchlist, error)
	// Add adds a symbol to the user's watchlist.
	Add(ctx context.Context, userID string, input domain.AddWatchlistInput) error
	// Remove removes a symbol from the user's watchlist.
	Remove(ctx context.Context, userID, symbol string) error
}

type watchlistUsecase struct {
	repo repository.WatchlistRepository
}

// NewWatchlistUsecase wires up the watchlist usecase.
func NewWatchlistUsecase(repo repository.WatchlistRepository) WatchlistUsecase {
	return &watchlistUsecase{repo: repo}
}

func (u *watchlistUsecase) List(ctx context.Context, userID string) ([]domain.Watchlist, error) {
	return u.repo.ListByUser(ctx, userID)
}

func (u *watchlistUsecase) Add(ctx context.Context, userID string, input domain.AddWatchlistInput) error {
	w := domain.Watchlist{
		UserID: userID,
		Symbol: input.Symbol,
		Label:  input.Label,
	}
	return u.repo.Add(ctx, w)
}

func (u *watchlistUsecase) Remove(ctx context.Context, userID, symbol string) error {
	return u.repo.RemoveBySymbol(ctx, userID, symbol)
}
