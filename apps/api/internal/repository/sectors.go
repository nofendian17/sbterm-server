package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=sectors.go -destination=../mocks/mock_sectors_repository.go -package=mocks -typed
type SectorsRepository interface {
	GetSectors(ctx context.Context) ([]domain.Sector, error)
}
