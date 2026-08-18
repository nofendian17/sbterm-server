package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=foreigndomestic.go -destination=../mocks/mock_foreigndomestic_repository.go -package=mocks -typed
type ForeignDomesticRepository interface {
	GetForeignDomesticHistorical(ctx context.Context, symbol, marketType, period, from, to string) (*domain.ForeignDomesticData, error)
}
