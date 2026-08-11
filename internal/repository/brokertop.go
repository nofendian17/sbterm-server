package repository

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=brokertop.go -destination=../mocks/mock_brokertop_repository.go -package=mocks -typed
type BrokerTopRepository interface {
	GetBrokerTop(ctx context.Context, sort, order, period, marketType string, eodOnly bool) (*domain.BrokerTopData, error)
}
