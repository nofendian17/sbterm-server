package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=corpaction.go -destination=../mocks/mock_corpaction_repository.go -package=mocks -typed
type CorpActionRepository interface {
	GetCorpActions(ctx context.Context, symbol string, limit int) ([]domain.CompanyCorpAction, error)
}
