package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
	"github.com/nofendian17/sbterm/libs/stockbit"
)

// NotificationRepository fetches notifications from the Stockbit API.
type NotificationRepository struct {
	client *stockbit.Client
}

func NewNotificationRepository(client *stockbit.Client) *NotificationRepository {
	return &NotificationRepository{client: client}
}

func (r *NotificationRepository) GetNotifications(ctx context.Context, lastID int64, limit int, types []string) (*domain.NotificationList, error) {
	req := stockbit.NotificationRequest{LastID: lastID, Limit: limit}
	for _, t := range types {
		req.Types = append(req.Types, stockbit.NotificationType(t))
	}
	resp, err := r.client.GetNotifications(ctx, req)
	if err != nil {
		return nil, err
	}
	items := make([]domain.Notification, 0, len(resp.Data.Result))
	for _, n := range resp.Data.Result {
		masks := make([]domain.NotificationMask, 0, len(n.Data.MessageMask))
		for _, m := range n.Data.MessageMask {
			masks = append(masks, domain.NotificationMask{
				Key:  m.Key,
				Text: m.Payload.Text,
				Tag:  m.Payload.Tag,
				Type: m.Payload.Type,
			})
		}
		items = append(items, domain.Notification{
			ID:      n.ID,
			Type:    string(n.Type),
			Avatar:  n.Data.Avatar,
			Message: n.Data.Message,
			Masks:   masks,
			LinkTo:  domain.NotificationLink{Key: n.Data.LinkTo.Key, Value: n.Data.LinkTo.Value},
			Created: n.Created,
			IsRead:  n.IsRead,
		})
	}
	return &domain.NotificationList{Unread: resp.Data.Unread, Items: items}, nil
}

var _ repository.NotificationRepository = (*NotificationRepository)(nil)
