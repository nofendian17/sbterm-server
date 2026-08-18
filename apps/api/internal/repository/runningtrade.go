package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=runningtrade.go -destination=../mocks/mock_runningtrade_repository.go -package=mocks -typed
type RunningTradeRepository interface {
	GetRunningTradeChart(ctx context.Context, symbol string, brokerCodes []string, from, to, investorType, marketBoard, period string) (*domain.RunningTradeData, error)
	GetRunningTrade(ctx context.Context, symbol, sort, orderBy, date string, limit int, tradeNumber int64) (*domain.RunningTradeFeed, error)
}
