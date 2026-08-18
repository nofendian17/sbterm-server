package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=stream.go -destination=../mocks/mock_stream_repository.go -package=mocks -typed
type StreamRepository interface {
	GetUserStream(ctx context.Context, username, category string, lastStreamID int64, limit int) (*domain.UserStreamData, error)
	GetStreamAnnouncement(ctx context.Context, streamID string) ([]domain.StreamAnnouncement, error)
}
