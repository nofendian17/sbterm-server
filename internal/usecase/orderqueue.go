package usecase

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=orderqueue.go -destination=../mocks/mock_orderqueue_usecase.go -package=mocks -typed
type OrderQueueUsecase interface {
	GetOrderQueue(ctx context.Context, stockCode, actionType, boardType, orderStatus, sortBy, sortDirection string, limit int, price int64) (*domain.OrderQueueData, error)
}

type orderQueueUsecase struct {
	repo repository.OrderQueueRepository
}

func NewOrderQueueUsecase(repo repository.OrderQueueRepository) *orderQueueUsecase {
	return &orderQueueUsecase{repo: repo}
}

func (u *orderQueueUsecase) GetOrderQueue(ctx context.Context, stockCode, actionType, boardType, orderStatus, sortBy, sortDirection string, limit int, price int64) (*domain.OrderQueueData, error) {
	return u.repo.GetOrderQueue(ctx, stockCode, actionType, boardType, orderStatus, sortBy, sortDirection, limit, price)
}
