package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/mocks"
)

func TestNotificationUsecaseGetNotifications(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr bool
	}{
		{name: "returns notification list"},
		{name: "propagates repository error", repoErr: errors.New("boom"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			want := &domain.NotificationList{Unread: 2, Items: []domain.Notification{{ID: 123, Type: "NOTIF_TYPE_NEW_REPORT"}}}
			repo := mocks.NewMockNotificationRepository(ctrl)
			repo.EXPECT().GetNotifications(gomock.Any(), int64(7), 20, []string{"NOTIF_TYPE_NEW_REPORT"}).Return(want, tt.repoErr)

			uc := NewNotificationUsecase(repo)
			got, err := uc.GetNotifications(context.Background(), 7, 20, []string{"NOTIF_TYPE_NEW_REPORT"})
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}
