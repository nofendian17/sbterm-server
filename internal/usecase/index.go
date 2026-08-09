package usecase

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=index.go -destination=../mocks/mock_index_usecase.go -package=mocks -typed
type IndexUsecase interface {
	GetIndexes(ctx context.Context) (*domain.Indexes, error)
}

type indexUsecase struct {
	repo repository.IndexRepository
}

func NewIndexUsecase(repo repository.IndexRepository) *indexUsecase {
	return &indexUsecase{repo: repo}
}

func (u *indexUsecase) GetIndexes(ctx context.Context) (*domain.Indexes, error) {
	return u.repo.GetIndexes(ctx)
}
