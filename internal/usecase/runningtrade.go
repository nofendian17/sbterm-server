package usecase

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=runningtrade.go -destination=../mocks/mock_runningtrade_usecase.go -package=mocks -typed
type RunningTradeUsecase interface {
	GetRunningTradeChart(ctx context.Context, symbol string, brokerCodes []string, from, to, investorType, marketBoard, period string) (*domain.RunningTradeData, error)
}

type runningTradeUsecase struct {
	repo repository.RunningTradeRepository
}

func NewRunningTradeUsecase(repo repository.RunningTradeRepository) *runningTradeUsecase {
	return &runningTradeUsecase{repo: repo}
}

func (u *runningTradeUsecase) GetRunningTradeChart(ctx context.Context, symbol string, brokerCodes []string, from, to, investorType, marketBoard, period string) (*domain.RunningTradeData, error) {
	return u.repo.GetRunningTradeChart(ctx, symbol, brokerCodes, from, to, investorType, marketBoard, period)
}
