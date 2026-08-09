package usecase

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=company_profile.go -destination=../mocks/mock_company_profile_usecase.go -package=mocks -typed
type CompanyProfileUsecase interface {
	GetProfile(ctx context.Context, symbol string) (*domain.CompanyProfile, error)
}

type companyProfileUsecase struct {
	repo repository.CompanyProfileRepository
}

func NewCompanyProfileUsecase(repo repository.CompanyProfileRepository) *companyProfileUsecase {
	return &companyProfileUsecase{repo: repo}
}

func (u *companyProfileUsecase) GetProfile(ctx context.Context, symbol string) (*domain.CompanyProfile, error) {
	return u.repo.GetProfile(ctx, symbol)
}
