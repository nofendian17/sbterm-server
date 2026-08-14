package usecase

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=stream.go -destination=../mocks/mock_stream_usecase.go -package=mocks -typed
type StreamUsecase interface {
	GetUserStream(ctx context.Context, username, category string, lastStreamID int64, limit int) (*domain.UserStreamData, error)
	GetStreamAnnouncement(ctx context.Context, streamID string) ([]domain.StreamAnnouncement, error)
}

type streamUsecase struct {
	repo repository.StreamRepository
}

func NewStreamUsecase(repo repository.StreamRepository) *streamUsecase {
	return &streamUsecase{repo: repo}
}

func (u *streamUsecase) GetUserStream(ctx context.Context, username, category string, lastStreamID int64, limit int) (*domain.UserStreamData, error) {
	return u.repo.GetUserStream(ctx, username, category, lastStreamID, limit)
}

func (u *streamUsecase) GetStreamAnnouncement(ctx context.Context, streamID string) ([]domain.StreamAnnouncement, error) {
	return u.repo.GetStreamAnnouncement(ctx, streamID)
}
