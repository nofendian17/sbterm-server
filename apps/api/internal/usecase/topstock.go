package usecase

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=topstock.go -destination=../mocks/mock_topstock_usecase.go -package=mocks -typed
type TopStockUsecase interface {
	GetTopStock(ctx context.Context, start, end, investorType, marketType, valueType string, page int) (*domain.TopStockData, error)
}

type topStockUsecase struct {
	repo repository.TopStockRepository
}

func NewTopStockUsecase(repo repository.TopStockRepository) *topStockUsecase {
	return &topStockUsecase{repo: repo}
}

func (u *topStockUsecase) GetTopStock(ctx context.Context, start, end, investorType, marketType, valueType string, page int) (*domain.TopStockData, error) {
	return u.repo.GetTopStock(ctx, start, end, investorType, marketType, valueType, page)
}
