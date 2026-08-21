package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nofendian17/sbterm/libs/pkg/log"
)

const notificationBody = `{"message":"Success retrieved notifications","data":{"unread":3,"result":[{"id":123,"type":"NOTIF_TYPE_NEW_REPORT","data":{"avatar":"https://assets.stockbit.com/avatar.png","message":"%company% released a new report","message_mask":[{"key":"company","payload":{"id":143,"tag":"PAYLOAD_MASK_TAG_BOLD","text":"BBRI","type":"PAYLOAD_MASK_TYPE_COMPANY","actors":[]}}],"link_to":{"key":143,"value":"BBRI"}},"created":"2026-08-20T09:00:00Z","is_read":false}]}}`

func TestGetNotifications(t *testing.T) {
	tests := []struct {
		name    string
		req     NotificationRequest
		handler func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check   func(t *testing.T, resp *NotificationResponse)
		wantErr bool
	}{
		{
			name: "sends last_id, limit and repeated types",
			req: NotificationRequest{
				LastID: 42,
				Limit:  20,
				Types:  []NotificationType{NotifTypeNewReport, NotifTypeNewsfeed},
			},
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, notificationPath, r.URL.Path)
				assert.Equal(t, "42", r.URL.Query().Get("last_id"))
				assert.Equal(t, "20", r.URL.Query().Get("limit"))
				assert.Equal(t, []string{"NOTIF_TYPE_NEW_REPORT", "NOTIF_TYPE_NEWSFEED"}, r.URL.Query()["types"])
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(notificationBody))
			},
			check: func(t *testing.T, resp *NotificationResponse) {
				assert.Equal(t, 3, resp.Data.Unread)
				require.Len(t, resp.Data.Result, 1)
				n := resp.Data.Result[0]
				assert.Equal(t, int64(123), n.ID)
				assert.Equal(t, NotifTypeNewReport, n.Type)
				assert.False(t, n.IsRead)
				require.Len(t, n.Data.MessageMask, 1)
				assert.Equal(t, "BBRI", n.Data.MessageMask[0].Payload.Text)
				assert.Equal(t, int64(143), n.Data.LinkTo.Key)
				assert.Equal(t, "BBRI", n.Data.LinkTo.Value)
			},
		},
		{
			name: "omits empty params",
			req:  NotificationRequest{},
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Empty(t, r.URL.Query().Get("last_id"))
				assert.Empty(t, r.URL.Query().Get("limit"))
				assert.Empty(t, r.URL.Query()["types"])
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(notificationBody))
			},
			check: func(t *testing.T, resp *NotificationResponse) {},
		},
		{
			name: "propagates upstream error",
			req:  NotificationRequest{},
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"message":"boom"}`))
			},
			check:   func(t *testing.T, resp *NotificationResponse) {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logs strings.Builder
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.handler(t, w, r)
			}))
			defer srv.Close()

			client := New(WithBaseURL(srv.URL), WithLogger(log.New(log.WithWriter(&logs))))
			resp, err := client.GetNotifications(context.Background(), tt.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			tt.check(t, resp)
		})
	}
}
