package usecase

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=findata_financial.go -destination=../mocks/mock_findata_financial_usecase.go -package=mocks -typed
type FindataFinancialUsecase interface {
	GetFindataFinancial(ctx context.Context, symbol string, dataType, isPercentage, page, reportType, statementType int) (*domain.FindataFinancial, error)
}

type findataFinancialUsecase struct {
	repo repository.FindataFinancialRepository
}

func NewFindataFinancialUsecase(repo repository.FindataFinancialRepository) *findataFinancialUsecase {
	return &findataFinancialUsecase{repo: repo}
}

func (u *findataFinancialUsecase) GetFindataFinancial(ctx context.Context, symbol string, dataType, isPercentage, page, reportType, statementType int) (*domain.FindataFinancial, error) {
	return u.repo.GetFindataFinancial(ctx, symbol, dataType, isPercentage, page, reportType, statementType)
}
