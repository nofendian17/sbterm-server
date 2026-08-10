package repository

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=indexsummary.go -destination=../mocks/mock_indexsummary_repository.go -package=mocks -typed
type IndexSummaryRepository interface {
	GetIndexSummary(ctx context.Context, symbol, from, to, interval string) (*domain.IndexSummaryData, error)
}
