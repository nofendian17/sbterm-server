package repository

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=fundachart.go -destination=../mocks/mock_fundachart_repository.go -package=mocks -typed
type FundaChartRepository interface {
	GetFundaChart(ctx context.Context, symbol, item, timeframe string) ([]domain.FundaChartCompany, error)
}
