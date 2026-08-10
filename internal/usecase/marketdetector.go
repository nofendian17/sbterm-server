package usecase

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=marketdetector.go -destination=../mocks/mock_marketdetector_usecase.go -package=mocks -typed
type MarketDetectorUsecase interface {
	GetMarketDetector(ctx context.Context, symbol, from, to, transactionType, marketBoard, investorType string, limit int) (*domain.MarketDetectorData, error)
}

type marketDetectorUsecase struct {
	repo repository.MarketDetectorRepository
}

func NewMarketDetectorUsecase(repo repository.MarketDetectorRepository) *marketDetectorUsecase {
	return &marketDetectorUsecase{repo: repo}
}

func (u *marketDetectorUsecase) GetMarketDetector(ctx context.Context, symbol, from, to, transactionType, marketBoard, investorType string, limit int) (*domain.MarketDetectorData, error) {
	return u.repo.GetMarketDetector(ctx, symbol, from, to, transactionType, marketBoard, investorType, limit)
}
