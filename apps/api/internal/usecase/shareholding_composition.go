package usecase

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=shareholding_composition.go -destination=../mocks/mock_shareholding_composition_usecase.go -package=mocks -typed
type ShareholdingCompositionUsecase interface {
	GetShareholdingComposition(ctx context.Context, symbol, periodStart, periodEnd string) ([]domain.ShareholdingCompositionPeriod, error)
}

type shareholdingCompositionUsecase struct {
	repo repository.ShareholdingCompositionRepository
}

func NewShareholdingCompositionUsecase(repo repository.ShareholdingCompositionRepository) *shareholdingCompositionUsecase {
	return &shareholdingCompositionUsecase{repo: repo}
}

func (u *shareholdingCompositionUsecase) GetShareholdingComposition(ctx context.Context, symbol, periodStart, periodEnd string) ([]domain.ShareholdingCompositionPeriod, error) {
	return u.repo.GetShareholdingComposition(ctx, symbol, periodStart, periodEnd)
}
