package usecase

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=search.go -destination=../mocks/mock_search_usecase.go -package=mocks -typed
type SearchUsecase interface {
	GetSearch(ctx context.Context, keyword string, page int, typ string) (*domain.SearchResult, error)
}

type searchUsecase struct {
	repo repository.SearchRepository
}

func NewSearchUsecase(repo repository.SearchRepository) *searchUsecase {
	return &searchUsecase{repo: repo}
}

func (u *searchUsecase) GetSearch(ctx context.Context, keyword string, page int, typ string) (*domain.SearchResult, error) {
	return u.repo.GetSearch(ctx, keyword, page, typ)
}
