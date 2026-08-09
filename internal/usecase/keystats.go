package usecase

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=keystats.go -destination=../mocks/mock_keystats_usecase.go -package=mocks -typed
type KeystatsUsecase interface {
	GetKeystats(ctx context.Context, symbol string, yearLimit int) (*domain.Keystats, error)
}

type keystatsUsecase struct {
	repo repository.KeystatsRepository
}

func NewKeystatsUsecase(repo repository.KeystatsRepository) *keystatsUsecase {
	return &keystatsUsecase{repo: repo}
}

func (u *keystatsUsecase) GetKeystats(ctx context.Context, symbol string, yearLimit int) (*domain.Keystats, error) {
	return u.repo.GetKeystats(ctx, symbol, yearLimit)
}
