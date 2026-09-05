// Package usecase implements the business logic for the core domain.

package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=company_profile.go -destination=../mocks/mock_company_profile_usecase.go -package=mocks -typed

// CompanyProfileUsecase manages the per-stock company profile aggregate.
type CompanyProfileUsecase interface {
	// Get returns the full profile aggregate for a symbol, or
	// domain.ErrCompanyProfileNotFound.
	Get(ctx context.Context, symbol string) (domain.CompanyProfile, error)
	// Save replaces the whole profile cluster for a symbol atomically.
	Save(ctx context.Context, profile domain.CompanyProfile) error
	// SyncProfile fetches the symbol's profile from the configured upstream
	// (apps/api) and atomically replaces the local cluster. Returns the
	// persisted aggregate, or domain.ErrCompanyProfileSyncFailed when the
	// upstream call failed and domain.ErrStockNotFound when the symbol is
	// not in the stocks master.
	SyncProfile(ctx context.Context, symbol string) (domain.CompanyProfile, error)
}

type companyProfileUsecase struct {
	repo repository.CompanyProfileRepository
	sync repository.CompanyProfileSyncClient
}

// NewCompanyProfileUsecase wires up the company profile usecase. sync may be
// nil when profile sync is not configured; SyncProfile then fails cleanly.
func NewCompanyProfileUsecase(
	repo repository.CompanyProfileRepository,
	sync repository.CompanyProfileSyncClient,
) CompanyProfileUsecase {
	return &companyProfileUsecase{repo: repo, sync: sync}
}

func (u *companyProfileUsecase) Get(ctx context.Context, symbol string) (domain.CompanyProfile, error) {
	p, err := u.repo.GetBySymbol(ctx, strings.ToUpper(strings.TrimSpace(symbol)))
	if err != nil {
		return domain.CompanyProfile{}, fmt.Errorf("company profile get: %w", err)
	}
	return p, nil
}

func (u *companyProfileUsecase) Save(ctx context.Context, profile domain.CompanyProfile) error {
	profile.Symbol = strings.ToUpper(strings.TrimSpace(profile.Symbol))
	if profile.Symbol == "" {
		return fmt.Errorf("company profile save: %w", domain.ErrInvalidInput)
	}
	if err := u.repo.Save(ctx, profile); err != nil {
		return fmt.Errorf("company profile save: %w", err)
	}
	return nil
}

func (u *companyProfileUsecase) SyncProfile(ctx context.Context, symbol string) (domain.CompanyProfile, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return domain.CompanyProfile{}, fmt.Errorf("company profile sync: %w", domain.ErrInvalidInput)
	}
	if u.sync == nil {
		return domain.CompanyProfile{}, fmt.Errorf("company profile sync: %w", domain.ErrCompanyProfileSyncFailed)
	}

	upstream, err := u.sync.FetchCompanyProfile(ctx, symbol)
	if err != nil {
		return domain.CompanyProfile{}, fmt.Errorf("company profile sync: %w", domain.ErrCompanyProfileSyncFailed)
	}

	if err := u.repo.Save(ctx, upstream); err != nil {
		// Preserve domain sentinels coming out of the repository (e.g.
		// ErrStockNotFound for a symbol absent from the stocks master).
		if errors.Is(err, domain.ErrStockNotFound) {
			return domain.CompanyProfile{}, fmt.Errorf("company profile sync: %w", err)
		}
		return domain.CompanyProfile{}, fmt.Errorf("company profile sync: %w", err)
	}

	saved, err := u.repo.GetBySymbol(ctx, symbol)
	if err != nil {
		return domain.CompanyProfile{}, fmt.Errorf("company profile sync: reload: %w", err)
	}
	return saved, nil
}
