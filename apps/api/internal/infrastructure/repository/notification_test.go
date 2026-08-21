package repository

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nofendian17/sbterm/libs/stockbit"
)

const notificationBody = `{"message":"ok","data":{"unread":2,"result":[{"id":123,"type":"NOTIF_TYPE_NEW_REPORT","data":{"avatar":"https://assets.stockbit.com/a.png","message":"%company% released a report","message_mask":[{"key":"company","payload":{"id":143,"tag":"PAYLOAD_MASK_TAG_BOLD","text":"BBRI","type":"PAYLOAD_MASK_TYPE_COMPANY","actors":[]}}],"link_to":{"key":143,"value":"BBRI"}},"created":"2026-08-20T09:00:00Z","is_read":false}]}}`

func TestNotificationRepositoryGetNotifications(t *testing.T) {
	tests := []struct {
		name    string
		lastID  int64
		limit   int
		types   []string
		status  int
		body    string
		wantErr bool
	}{
		{
			name:   "returns mapped notifications",
			limit:  20,
			types:  []string{"NOTIF_TYPE_NEW_REPORT"},
			status: http.StatusOK,
			body:   notificationBody,
		},
		{
			name:    "propagates upstream error",
			status:  http.StatusInternalServerError,
			body:    `{"message":"boom"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/notification", r.URL.Path)
				if tt.limit > 0 {
					assert.Equal(t, "20", r.URL.Query().Get("limit"))
				}
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewNotificationRepository(client)

			got, err := repo.GetNotifications(context.Background(), tt.lastID, tt.limit, tt.types)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, 2, got.Unread)
			require.Len(t, got.Items, 1)
			item := got.Items[0]
			assert.Equal(t, int64(123), item.ID)
			assert.Equal(t, "NOTIF_TYPE_NEW_REPORT", item.Type)
			assert.Equal(t, "BBRI", item.LinkTo.Value)
			require.Len(t, item.Masks, 1)
			assert.Equal(t, "company", item.Masks[0].Key)
			assert.Equal(t, "BBRI", item.Masks[0].Text)
		})
	}
}
