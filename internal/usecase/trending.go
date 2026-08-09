package usecase

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=trending.go -destination=../mocks/mock_trending_usecase.go -package=mocks -typed
type TrendingUsecase interface {
	GetTrending(ctx context.Context) ([]domain.TrendingStock, error)
}

type trendingUsecase struct {
	repo repository.TrendingRepository
}

func NewTrendingUsecase(repo repository.TrendingRepository) *trendingUsecase {
	return &trendingUsecase{repo: repo}
}

func (u *trendingUsecase) GetTrending(ctx context.Context) ([]domain.TrendingStock, error) {
	return u.repo.GetTrending(ctx)
}