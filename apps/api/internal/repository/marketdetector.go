package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=marketdetector.go -destination=../mocks/mock_marketdetector_repository.go -package=mocks -typed
type MarketDetectorRepository interface {
	GetMarketDetector(ctx context.Context, symbol, from, to, transactionType, marketBoard, investorType string, limit int) (*domain.MarketDetectorData, error)
}
