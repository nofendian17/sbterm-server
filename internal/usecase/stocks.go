package usecase

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=stocks.go -destination=../mocks/mock_stocks_usecase.go -package=mocks -typed
type StocksUsecase interface {
	GetStocks(ctx context.Context) ([]domain.Stock, error)
}

type stocksUsecase struct {
	repo repository.StocksRepository
}

func NewStocksUsecase(repo repository.StocksRepository) *stocksUsecase {
	return &stocksUsecase{repo: repo}
}

func (u *stocksUsecase) GetStocks(ctx context.Context) ([]domain.Stock, error) {
	return u.repo.GetStocks(ctx)
}
