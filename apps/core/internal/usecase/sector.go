// Package usecase implements the business logic for the core domain.

package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=sector.go -destination=../mocks/mock_sector_usecase.go -package=mocks -typed

// SectorUsecase manages the sectors master table. Reads are user-facing;
// writes are admin-gated by the router.
type SectorUsecase interface {
	List(ctx context.Context) ([]domain.Sector, error)
	GetByID(ctx context.Context, id string) (domain.Sector, error)
	Create(ctx context.Context, name string) (domain.Sector, error)
	Update(ctx context.Context, id, name string) error
	SoftDelete(ctx context.Context, id string) error
}

type sectorUsecase struct {
	repo repository.SectorRepository
}

// NewSectorUsecase wires up the sector usecase.
func NewSectorUsecase(repo repository.SectorRepository) SectorUsecase {
	return &sectorUsecase{repo: repo}
}

func (u *sectorUsecase) List(ctx context.Context) ([]domain.Sector, error) {
	sectors, err := u.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("sector list: %w", err)
	}
	return sectors, nil
}

func (u *sectorUsecase) GetByID(ctx context.Context, id string) (domain.Sector, error) {
	s, err := u.repo.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.Sector{}, fmt.Errorf("sector get: %w", err)
	}
	return s, nil
}

func (u *sectorUsecase) Create(ctx context.Context, name string) (domain.Sector, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Sector{}, fmt.Errorf("sector create: %w", domain.ErrInvalidInput)
	}
	s, err := u.repo.Create(ctx, name)
	if err != nil {
		return domain.Sector{}, fmt.Errorf("sector create: %w", err)
	}
	return s, nil
}

func (u *sectorUsecase) Update(ctx context.Context, id, name string) error {
	name = strings.TrimSpace(name)
	if strings.TrimSpace(id) == "" || name == "" {
		return fmt.Errorf("sector update: %w", domain.ErrInvalidInput)
	}
	if err := u.repo.Update(ctx, strings.TrimSpace(id), name); err != nil {
		return fmt.Errorf("sector update: %w", err)
	}
	return nil
}

func (u *sectorUsecase) SoftDelete(ctx context.Context, id string) error {
	if err := u.repo.SoftDelete(ctx, strings.TrimSpace(id)); err != nil {
		return fmt.Errorf("sector soft delete: %w", err)
	}
	return nil
}
