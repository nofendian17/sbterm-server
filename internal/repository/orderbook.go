package repository

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=orderbook.go -destination=../mocks/mock_orderbook_repository.go -package=mocks -typed
type OrderBookRepository interface {
	GetOrderBook(ctx context.Context, symbol string) (*domain.OrderBookData, error)
}
