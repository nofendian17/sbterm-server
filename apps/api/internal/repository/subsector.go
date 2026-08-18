package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=subsector.go -destination=../mocks/mock_subsector_repository.go -package=mocks -typed
type SubsectorRepository interface {
	GetCompanies(ctx context.Context, sectorID, subsectorID string) ([]domain.SubsectorCompany, error)
}
