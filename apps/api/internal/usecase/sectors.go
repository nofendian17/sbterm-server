package usecase

import (
	"context"
	"errors"
	"sync"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=sectors.go -destination=../mocks/mock_sectors_usecase.go -package=mocks -typed
type SectorsUsecase interface {
	GetSectors(ctx context.Context) ([]domain.Sector, error)
}

// sectorID is the parent sector of the IDX sector indexes returned by the
// sectors endpoint. It is the sector part of the subsector companies path.
//
// ponytail: hardcoded parent; if the API ever returns per-entry parents,
// use that instead.
const sectorID = "70"

type sectorsUsecase struct {
	repo          repository.SectorsRepository
	subsectorRepo repository.SubsectorRepository
}

func NewSectorsUsecase(repo repository.SectorsRepository, subsectorRepo repository.SubsectorRepository) *sectorsUsecase {
	return &sectorsUsecase{repo: repo, subsectorRepo: subsectorRepo}
}

func (u *sectorsUsecase) GetSectors(ctx context.Context) ([]domain.Sector, error) {
	sectors, err := u.repo.GetSectors(ctx)
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	errs := make([]error, len(sectors))
	for i := range sectors {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			companies, err := u.subsectorRepo.GetCompanies(ctx, sectorID, sectors[i].ID)
			if err != nil {
				errs[i] = err
				return
			}
			sectors[i].Companies = companies
		}(i)
	}
	wg.Wait()

	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return sectors, nil
}
