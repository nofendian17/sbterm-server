package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=notification.go -destination=../mocks/mock_notification_repository.go -package=mocks -typed
type NotificationRepository interface {
	GetNotifications(ctx context.Context, lastID int64, limit int, types []string) (*domain.NotificationList, error)
}
