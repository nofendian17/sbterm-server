package repository

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=search.go -destination=../mocks/mock_search_repository.go -package=mocks -typed
type SearchRepository interface {
	GetSearch(ctx context.Context, keyword string, page int, typ string) (*domain.SearchResult, error)
}
