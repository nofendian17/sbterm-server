package repository

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=trending.go -destination=../mocks/mock_trending_repository.go -package=mocks -typed
type TrendingRepository interface {
	GetTrending(ctx context.Context) ([]domain.TrendingStock, error)
}
