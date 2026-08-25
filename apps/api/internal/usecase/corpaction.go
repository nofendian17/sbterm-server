package usecase

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=corpaction.go -destination=../mocks/mock_corpaction_usecase.go -package=mocks -typed
type CorpActionUsecase interface {
	GetCorpActions(ctx context.Context, symbol string, limit int) ([]domain.CompanyCorpAction, error)
	GetCorpActionCalendar(ctx context.Context, date string) (*domain.CorpActionCalendar, error)
}

type corpActionUsecase struct {
	repo repository.CorpActionRepository
}

func NewCorpActionUsecase(repo repository.CorpActionRepository) *corpActionUsecase {
	return &corpActionUsecase{repo: repo}
}

func (u *corpActionUsecase) GetCorpActions(ctx context.Context, symbol string, limit int) ([]domain.CompanyCorpAction, error) {
	return u.repo.GetCorpActions(ctx, symbol, limit)
}

func (u *corpActionUsecase) GetCorpActionCalendar(ctx context.Context, date string) (*domain.CorpActionCalendar, error) {
	return u.repo.GetCorpActionCalendar(ctx, date)
}
