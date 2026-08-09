package usecase

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=majorholder.go -destination=../mocks/mock_majorholder_usecase.go -package=mocks -typed
type MajorHolderUsecase interface {
	GetMajorHolder(ctx context.Context, symbols, actionType, sourceType string, page, limit int) (*domain.MajorHolderData, error)
}

type majorHolderUsecase struct {
	repo repository.MajorHolderRepository
}

func NewMajorHolderUsecase(repo repository.MajorHolderRepository) *majorHolderUsecase {
	return &majorHolderUsecase{repo: repo}
}

func (u *majorHolderUsecase) GetMajorHolder(ctx context.Context, symbols, actionType, sourceType string, page, limit int) (*domain.MajorHolderData, error) {
	return u.repo.GetMajorHolder(ctx, symbols, actionType, sourceType, page, limit)
}
