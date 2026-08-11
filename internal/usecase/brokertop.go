package usecase

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=brokertop.go -destination=../mocks/mock_brokertop_usecase.go -package=mocks -typed
type BrokerTopUsecase interface {
	GetBrokerTop(ctx context.Context, sort, order, period, marketType string, eodOnly bool) (*domain.BrokerTopData, error)
}

type brokerTopUsecase struct {
	repo repository.BrokerTopRepository
}

func NewBrokerTopUsecase(repo repository.BrokerTopRepository) *brokerTopUsecase {
	return &brokerTopUsecase{repo: repo}
}

func (u *brokerTopUsecase) GetBrokerTop(ctx context.Context, sort, order, period, marketType string, eodOnly bool) (*domain.BrokerTopData, error) {
	return u.repo.GetBrokerTop(ctx, sort, order, period, marketType, eodOnly)
}
