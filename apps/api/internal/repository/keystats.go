package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=keystats.go -destination=../mocks/mock_keystats_repository.go -package=mocks -typed
type KeystatsRepository interface {
	GetKeystats(ctx context.Context, symbol string, yearLimit int) (*domain.Keystats, error)
}
