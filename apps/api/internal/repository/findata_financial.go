package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=findata_financial.go -destination=../mocks/mock_findata_financial_repository.go -package=mocks -typed
type FindataFinancialRepository interface {
	GetFindataFinancial(ctx context.Context, symbol string, dataType, isPercentage, page, reportType, statementType int) (*domain.FindataFinancial, error)
}
