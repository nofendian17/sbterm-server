package usecase

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=subsidiary.go -destination=../mocks/mock_subsidiary_usecase.go -package=mocks -typed
type SubsidiaryUsecase interface {
	GetSubsidiaries(ctx context.Context, symbol string) (*domain.SubsidiaryData, error)
}

type subsidiaryUsecase struct {
	repo repository.SubsidiaryRepository
}

func NewSubsidiaryUsecase(repo repository.SubsidiaryRepository) *subsidiaryUsecase {
	return &subsidiaryUsecase{repo: repo}
}

func (u *subsidiaryUsecase) GetSubsidiaries(ctx context.Context, symbol string) (*domain.SubsidiaryData, error) {
	return u.repo.GetSubsidiaries(ctx, symbol)
}
