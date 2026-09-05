package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=company_profile.go -destination=../mocks/mock_company_profile_repository.go -package=mocks -typed

// CompanyProfileRepository persists the 1:1 company profile aggregate
// (header + normalized children) for a stock.
type CompanyProfileRepository interface {
	// GetBySymbol returns the full profile aggregate for the symbol, or
	// domain.ErrCompanyProfileNotFound when no profile row exists (a stock
	// may exist without a profile).
	GetBySymbol(ctx context.Context, symbol string) (domain.CompanyProfile, error)

	// Save replaces the whole profile cluster for the symbol in one
	// transaction: the header row is upserted and all children are
	// deleted and re-inserted from the given aggregate.
	Save(ctx context.Context, profile domain.CompanyProfile) error
}
