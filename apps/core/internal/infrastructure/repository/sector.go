package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	contract "github.com/nofendian17/sbterm/apps/core/internal/repository"
)

// SectorRepository is the pgx implementation of contract.SectorRepository.
type SectorRepository struct {
	q contract.Querier
}

// NewSectorRepository builds a SectorRepository backed by the given Querier.
func NewSectorRepository(q contract.Querier) *SectorRepository {
	return &SectorRepository{q: q}
}

// List returns all non-deleted sectors ordered by name.
func (r *SectorRepository) List(ctx context.Context) ([]domain.Sector, error) {
	rows, err := r.q.Query(ctx,
		`SELECT id, name FROM sectors WHERE deleted_at IS NULL ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("sector list: %w", err)
	}
	defer rows.Close()

	out := []domain.Sector{}
	for rows.Next() {
		var s domain.Sector
		if err := rows.Scan(&s.ID, &s.Name); err != nil {
			return nil, fmt.Errorf("sector list scan: %w", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sector list rows: %w", err)
	}
	return out, nil
}

// GetByID returns one non-deleted sector.
func (r *SectorRepository) GetByID(ctx context.Context, id string) (domain.Sector, error) {
	var s domain.Sector
	err := r.q.QueryRow(ctx,
		`SELECT id, name FROM sectors WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&s.ID, &s.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Sector{}, fmt.Errorf("sector get: %w", domain.ErrSectorNotFound)
		}
		return domain.Sector{}, fmt.Errorf("sector get: %w", err)
	}
	return s, nil
}

// GetByName returns the non-deleted sector with the given name.
func (r *SectorRepository) GetByName(ctx context.Context, name string) (domain.Sector, error) {
	var s domain.Sector
	err := r.q.QueryRow(ctx,
		`SELECT id, name FROM sectors WHERE name = $1 AND deleted_at IS NULL`, name,
	).Scan(&s.ID, &s.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Sector{}, fmt.Errorf("sector get by name: %w", domain.ErrSectorNotFound)
		}
		return domain.Sector{}, fmt.Errorf("sector get by name: %w", err)
	}
	return s, nil
}

// Create inserts a sector and returns it.
func (r *SectorRepository) Create(ctx context.Context, name string) (domain.Sector, error) {
	var s domain.Sector
	err := r.q.QueryRow(ctx,
		`INSERT INTO sectors (name) VALUES ($1) RETURNING id, name`, name,
	).Scan(&s.ID, &s.Name)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Sector{}, fmt.Errorf("sector create: %w", domain.ErrSectorNameTaken)
		}
		return domain.Sector{}, fmt.Errorf("sector create: %w", err)
	}
	return s, nil
}

// Update renames a sector.
func (r *SectorRepository) Update(ctx context.Context, id, name string) error {
	tag, err := r.q.Exec(ctx,
		`UPDATE sectors SET name = $2, updated_at = now() WHERE id = $1 AND deleted_at IS NULL`,
		id, name)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("sector update: %w", domain.ErrSectorNameTaken)
		}
		return fmt.Errorf("sector update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("sector update: %w", domain.ErrSectorNotFound)
	}
	return nil
}

// SoftDelete marks the sector deleted. It refuses when live stocks still
// reference it, mirroring StockRepository.SoftDelete's watchlist guard.
func (r *SectorRepository) SoftDelete(ctx context.Context, id string) error {
	tag, err := r.q.Exec(ctx,
		`UPDATE sectors SET deleted_at = now()
		 WHERE id = $1 AND deleted_at IS NULL
		 AND NOT EXISTS (
			SELECT 1 FROM stocks
			WHERE stocks.sector_id = sectors.id
			AND stocks.deleted_at IS NULL
		 )`, id)
	if err != nil {
		return fmt.Errorf("sector soft delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		if qerr := r.q.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM stocks WHERE sector_id = $1 AND deleted_at IS NULL)`,
			id,
		).Scan(&exists); qerr == nil && exists {
			return fmt.Errorf("sector soft delete: %w", domain.ErrSectorHasStocks)
		}
		return fmt.Errorf("sector soft delete: %w", domain.ErrSectorNotFound)
	}
	return nil
}
