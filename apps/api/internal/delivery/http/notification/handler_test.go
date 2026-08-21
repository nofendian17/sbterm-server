package notification

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/mocks"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

func TestNotificationHandlerGetNotifications(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		setup       func(uc *mocks.MockNotificationUsecase)
		wantStatus  int
		wantErrCode string
		wantFields  map[string]string
	}{
		{
			name:  "returns notifications with defaults",
			query: "",
			setup: func(uc *mocks.MockNotificationUsecase) {
				uc.EXPECT().GetNotifications(gomock.Any(), int64(0), 20, []string(nil)).Return(&domain.NotificationList{
					Unread: 2,
					Items: []domain.Notification{{
						ID:      123,
						Type:    "NOTIF_TYPE_NEW_REPORT",
						Message: "%company% released a report",
						Masks:   []domain.NotificationMask{{Key: "company", Text: "BBRI", Tag: "PAYLOAD_MASK_TAG_BOLD", Type: "PAYLOAD_MASK_TYPE_COMPANY"}},
						LinkTo:  domain.NotificationLink{Key: 143, Value: "BBRI"},
						Created: "2026-08-20T09:00:00Z",
					}},
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:  "passes query params through",
			query: "?last_id=42&limit=10&types=NOTIF_TYPE_NEWSFEED&types=NOTIF_TYPE_NEW_REPORT",
			setup: func(uc *mocks.MockNotificationUsecase) {
				uc.EXPECT().GetNotifications(gomock.Any(), int64(42), 10, []string{"NOTIF_TYPE_NEWSFEED", "NOTIF_TYPE_NEW_REPORT"}).Return(&domain.NotificationList{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "invalid last_id returns validation error",
			query:       "?last_id=abc",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
			wantFields:  map[string]string{"last_id": "must be a valid integer"},
		},
		{
			name:        "limit above upstream cap returns validation error",
			query:       "?limit=26",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "unknown type returns validation error",
			query:       "?types=NOTIF_TYPE_NOPE",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name: "usecase error returns 500",
			setup: func(uc *mocks.MockNotificationUsecase) {
				uc.EXPECT().GetNotifications(gomock.Any(), int64(0), 20, []string(nil)).Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockNotificationUsecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			h := NewNotificationHandler(uc, validator.New())
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications"+tt.query, nil)
			h.GetNotifications(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    *struct {
					Unread int `json:"unread"`
					Items  []struct {
						ID     int64  `json:"id"`
						Type   string `json:"type"`
						LinkTo struct {
							Value string `json:"value"`
						} `json:"link_to"`
					} `json:"items"`
				} `json:"data"`
				Error *struct {
					Code    string            `json:"code"`
					Details map[string]string `json:"details"`
				} `json:"error"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))

			if tt.wantErrCode != "" {
				require.NotNil(t, env.Error)
				assert.Equal(t, tt.wantErrCode, env.Error.Code)
				if tt.wantFields != nil {
					assert.Equal(t, tt.wantFields, env.Error.Details)
				}
				return
			}
			require.NotNil(t, env.Data)
		})
	}
}

func TestNotificationHandlerMapsResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	uc := mocks.NewMockNotificationUsecase(ctrl)
	uc.EXPECT().GetNotifications(gomock.Any(), int64(0), 20, []string(nil)).Return(&domain.NotificationList{
		Unread: 2,
		Items: []domain.Notification{{
			ID:      123,
			Type:    "NOTIF_TYPE_NEW_REPORT",
			Message: "%company% released a report",
			Masks:   []domain.NotificationMask{{Key: "company", Text: "BBRI", Tag: "PAYLOAD_MASK_TAG_BOLD", Type: "PAYLOAD_MASK_TYPE_COMPANY"}},
			LinkTo:  domain.NotificationLink{Key: 143, Value: "BBRI"},
			Created: "2026-08-20T09:00:00Z",
		}},
	}, nil)

	h := NewNotificationHandler(uc, validator.New())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
	h.GetNotifications(rec, req)

	var env struct {
		Data struct {
			Unread int `json:"unread"`
			Items  []struct {
				ID      int64  `json:"id"`
				Type    string `json:"type"`
				Message string `json:"message"`
				Masks   []struct {
					Key  string `json:"key"`
					Text string `json:"text"`
				} `json:"masks"`
				LinkTo struct {
					Key   int64  `json:"key"`
					Value string `json:"value"`
				} `json:"link_to"`
				Created string `json:"created"`
				IsRead  bool   `json:"is_read"`
			} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Equal(t, 2, env.Data.Unread)
	require.Len(t, env.Data.Items, 1)
	item := env.Data.Items[0]
	assert.Equal(t, int64(123), item.ID)
	assert.Equal(t, "BBRI", item.LinkTo.Value)
	require.Len(t, item.Masks, 1)
	assert.Equal(t, "company", item.Masks[0].Key)
}
