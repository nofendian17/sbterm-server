package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=market_session.go -destination=../mocks/mock_market_session_repository.go -package=mocks -typed
type MarketSessionRepository interface {
	GetMarketSession(ctx context.Context) (*domain.MarketSession, error)
}
