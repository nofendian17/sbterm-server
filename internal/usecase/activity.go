package usecase

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=activity.go -destination=../mocks/mock_activity_usecase.go -package=mocks -typed
type ActivityUsecase interface {
	GetActivityChart(ctx context.Context, symbols, brokersCode []string, from, to, period, investorType, marketBoard string) (*domain.ActivityChartData, error)
	GetActivity(ctx context.Context, brokerCode []string, transactionType, investorType, marketBoard string, limit, page int, from, to, netValPeriod string) (*domain.ActivityData, error)
	GetActivityHistorical(ctx context.Context, interval, dateFrom, dateTo string, brokerCodes, symbols []string, marketBoard, investorType, netInterval string) (*domain.ActivityHistoricalData, error)
}

type activityUsecase struct {
	repo repository.ActivityRepository
}

func NewActivityUsecase(repo repository.ActivityRepository) *activityUsecase {
	return &activityUsecase{repo: repo}
}

func (u *activityUsecase) GetActivityChart(ctx context.Context, symbols, brokersCode []string, from, to, period, investorType, marketBoard string) (*domain.ActivityChartData, error) {
	return u.repo.GetActivityChart(ctx, symbols, brokersCode, from, to, period, investorType, marketBoard)
}

func (u *activityUsecase) GetActivity(ctx context.Context, brokerCode []string, transactionType, investorType, marketBoard string, limit, page int, from, to, netValPeriod string) (*domain.ActivityData, error) {
	return u.repo.GetActivity(ctx, brokerCode, transactionType, investorType, marketBoard, limit, page, from, to, netValPeriod)
}

func (u *activityUsecase) GetActivityHistorical(ctx context.Context, interval, dateFrom, dateTo string, brokerCodes, symbols []string, marketBoard, investorType, netInterval string) (*domain.ActivityHistoricalData, error) {
	return u.repo.GetActivityHistorical(ctx, interval, dateFrom, dateTo, brokerCodes, symbols, marketBoard, investorType, netInterval)
}
