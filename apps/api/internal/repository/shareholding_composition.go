package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=shareholding_composition.go -destination=../mocks/mock_shareholding_composition_repository.go -package=mocks -typed
type ShareholdingCompositionRepository interface {
	GetShareholdingComposition(ctx context.Context, symbol, periodStart, periodEnd string) ([]domain.ShareholdingCompositionPeriod, error)
}
