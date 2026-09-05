package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=sector.go -destination=../mocks/mock_sector_repository.go -package=mocks -typed

// SectorRepository manages the manually-curated sectors master table.
// Every read filters soft-deleted rows.
type SectorRepository interface {
	// List returns all non-deleted sectors ordered by name.
	List(ctx context.Context) ([]domain.Sector, error)

	// GetByID returns one non-deleted sector, or domain.ErrSectorNotFound.
	GetByID(ctx context.Context, id string) (domain.Sector, error)

	// GetByName returns the non-deleted sector with the given name, or
	// domain.ErrSectorNotFound.
	GetByName(ctx context.Context, name string) (domain.Sector, error)

	// Create inserts a sector. A name conflict (23505) maps to
	// domain.ErrSectorNameTaken.
	Create(ctx context.Context, name string) (domain.Sector, error)

	// Update renames a sector. Returns domain.ErrSectorNotFound when missing,
	// domain.ErrSectorNameTaken on name conflict.
	Update(ctx context.Context, id, name string) error

	// SoftDelete sets deleted_at. Refused with domain.ErrSectorHasStocks
	// when live stocks still reference the sector (stocks.sector_id FK only
	// blocks hard deletes); domain.ErrSectorNotFound when missing.
	SoftDelete(ctx context.Context, id string) error
}
