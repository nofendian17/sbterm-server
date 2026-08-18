package usecase

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=foreigndomestic.go -destination=../mocks/mock_foreigndomestic_usecase.go -package=mocks -typed
type ForeignDomesticUsecase interface {
	GetForeignDomesticHistorical(ctx context.Context, symbol, marketType, period, from, to string) (*domain.ForeignDomesticData, error)
}

type foreignDomesticUsecase struct {
	repo repository.ForeignDomesticRepository
}

func NewForeignDomesticUsecase(repo repository.ForeignDomesticRepository) *foreignDomesticUsecase {
	return &foreignDomesticUsecase{repo: repo}
}

func (u *foreignDomesticUsecase) GetForeignDomesticHistorical(ctx context.Context, symbol, marketType, period, from, to string) (*domain.ForeignDomesticData, error) {
	return u.repo.GetForeignDomesticHistorical(ctx, symbol, marketType, period, from, to)
}
