package repository

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=historicalsummary.go -destination=../mocks/mock_historicalsummary_repository.go -package=mocks -typed
type HistoricalSummaryRepository interface {
	GetHistoricalSummary(ctx context.Context, symbol, period, startDate, endDate string, limit, page int) (*domain.HistoricalSummaryData, error)
}
