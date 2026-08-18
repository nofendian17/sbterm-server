package usecase

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=orderbook.go -destination=../mocks/mock_orderbook_usecase.go -package=mocks -typed
type OrderBookUsecase interface {
	GetOrderBook(ctx context.Context, symbol string) (*domain.OrderBookData, error)
}

type orderBookUsecase struct {
	repo repository.OrderBookRepository
}

func NewOrderBookUsecase(repo repository.OrderBookRepository) *orderBookUsecase {
	return &orderBookUsecase{repo: repo}
}

func (u *orderBookUsecase) GetOrderBook(ctx context.Context, symbol string) (*domain.OrderBookData, error) {
	return u.repo.GetOrderBook(ctx, symbol)
}
