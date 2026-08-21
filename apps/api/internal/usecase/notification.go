package usecase

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=notification.go -destination=../mocks/mock_notification_usecase.go -package=mocks -typed
type NotificationUsecase interface {
	GetNotifications(ctx context.Context, lastID int64, limit int, types []string) (*domain.NotificationList, error)
}

type notificationUsecase struct {
	repo repository.NotificationRepository
}

func NewNotificationUsecase(repo repository.NotificationRepository) *notificationUsecase {
	return &notificationUsecase{repo: repo}
}

func (u *notificationUsecase) GetNotifications(ctx context.Context, lastID int64, limit int, types []string) (*domain.NotificationList, error) {
	return u.repo.GetNotifications(ctx, lastID, limit, types)
}
