package usecase

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=stream.go -destination=../mocks/mock_stream_usecase.go -package=mocks -typed
type StreamUsecase interface {
	GetUserStream(ctx context.Context, username, category string, lastStreamID int64, limit int) (*domain.UserStreamData, error)
	GetStreamConversation(ctx context.Context, streamID string) (*domain.StreamConversationData, error)
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

func (u *streamUsecase) GetStreamConversation(ctx context.Context, streamID string) (*domain.StreamConversationData, error) {
	return u.repo.GetStreamConversation(ctx, streamID)
}

func (u *streamUsecase) GetStreamAnnouncement(ctx context.Context, streamID string) ([]domain.StreamAnnouncement, error) {
	return u.repo.GetStreamAnnouncement(ctx, streamID)
}
