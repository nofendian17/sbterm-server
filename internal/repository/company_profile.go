package repository

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=company_profile.go -destination=../mocks/mock_company_profile_repository.go -package=mocks -typed
type CompanyProfileRepository interface {
	GetProfile(ctx context.Context, symbol string) (*domain.CompanyProfile, error)
}
