package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=majorholder.go -destination=../mocks/mock_majorholder_repository.go -package=mocks -typed
type MajorHolderRepository interface {
	GetMajorHolder(ctx context.Context, symbols, actionType, sourceType string, page, limit int) (*domain.MajorHolderData, error)
}
