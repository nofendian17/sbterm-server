package usecase

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=historicalsummary.go -destination=../mocks/mock_historicalsummary_usecase.go -package=mocks -typed
type HistoricalSummaryUsecase interface {
	GetHistoricalSummary(ctx context.Context, symbol, period, startDate, endDate string, limit, page int) (*domain.HistoricalSummaryData, error)
}

type historicalSummaryUsecase struct {
	repo repository.HistoricalSummaryRepository
}

func NewHistoricalSummaryUsecase(repo repository.HistoricalSummaryRepository) *historicalSummaryUsecase {
	return &historicalSummaryUsecase{repo: repo}
}

func (u *historicalSummaryUsecase) GetHistoricalSummary(ctx context.Context, symbol, period, startDate, endDate string, limit, page int) (*domain.HistoricalSummaryData, error) {
	return u.repo.GetHistoricalSummary(ctx, symbol, period, startDate, endDate, limit, page)
}
