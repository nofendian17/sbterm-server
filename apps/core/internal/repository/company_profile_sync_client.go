package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=company_profile_sync_client.go -destination=../mocks/mock_company_profile_sync_client.go -package=mocks -typed

// CompanyProfileSyncClient is the upstream-data port the CompanyProfileUsecase
// depends on to refresh one stock's profile cluster. The concrete
// implementation calls the apps/api endpoints documented in docs/api.md
// (GET /api/v1/company/{symbol}/profile and .../subsidiaries).
type CompanyProfileSyncClient interface {
	// FetchCompanyProfile fetches the full upstream profile aggregate for a
	// symbol (header + executives + holdings + shareholder numbers +
	// subsidiaries + addresses). Implementations apply a request-scoped
	// timeout and must NOT swallow upstream errors.
	FetchCompanyProfile(ctx context.Context, symbol string) (domain.CompanyProfile, error)
}
