package repository

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=orderqueue.go -destination=../mocks/mock_orderqueue_repository.go -package=mocks -typed
type OrderQueueRepository interface {
	GetOrderQueue(ctx context.Context, stockCode, actionType, boardType, orderStatus, sortBy, sortDirection string, limit int, price int64) (*domain.OrderQueueData, error)
}
