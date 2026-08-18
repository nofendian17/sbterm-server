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

const sessionBody = `{"data":{"datetime":"2026-08-09 18:54:49","detail":{"fca":{"session":0,"state_name":"STATE_NAME_MARKET_CLOSED","is_last_session":false,"is_end_of_day":true,"state_start_time":"","state_end_time":"","time_left":{"raw":49811,"formatted":"13 jam 50 menit 11 detik"},"suspend_info":""},"regular":{"session":0,"state_name":"STATE_NAME_MARKET_CLOSED","is_last_session":false,"is_end_of_day":true,"state_start_time":"","state_end_time":"","time_left":{"raw":49811,"formatted":"13 jam 50 menit 11 detik"},"suspend_info":""}}}}`

func TestMarketSessionRepositoryGetMarketSession(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{
			name:   "returns mapped market session",
			status: http.StatusOK,
			body:   sessionBody,
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
				assert.Equal(t, "/company-price-feed/market-time/session", r.URL.Path)
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewMarketSessionRepository(client)

			got, err := repo.GetMarketSession(context.Background())
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "2026-08-09 18:54:49", got.Datetime)
			assert.Equal(t, "STATE_NAME_MARKET_CLOSED", got.Regular.StateName)
			assert.True(t, got.Regular.IsEndOfDay)
			assert.Equal(t, "13 jam 50 menit 11 detik", got.FCA.TimeLeft)
		})
	}
}
