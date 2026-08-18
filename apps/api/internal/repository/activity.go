package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=activity.go -destination=../mocks/mock_activity_repository.go -package=mocks -typed
type ActivityRepository interface {
	GetActivityChart(ctx context.Context, symbols, brokersCode []string, from, to, period, investorType, marketBoard string) (*domain.ActivityChartData, error)
	GetActivity(ctx context.Context, brokerCode []string, transactionType, investorType, marketBoard string, limit, page int, from, to, netValPeriod string) (*domain.ActivityData, error)
	GetActivityHistorical(ctx context.Context, interval, dateFrom, dateTo string, brokerCodes, symbols []string, marketBoard, investorType, netInterval string) (*domain.ActivityHistoricalData, error)
}
