package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=subsidiary.go -destination=../mocks/mock_subsidiary_repository.go -package=mocks -typed
type SubsidiaryRepository interface {
	GetSubsidiaries(ctx context.Context, symbol string) (*domain.SubsidiaryData, error)
}
