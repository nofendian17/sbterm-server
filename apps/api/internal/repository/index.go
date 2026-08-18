package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=index.go -destination=../mocks/mock_index_repository.go -package=mocks -typed
type IndexRepository interface {
	GetIndexes(ctx context.Context) (*domain.Indexes, error)
}
