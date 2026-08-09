package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sessionBody = `{"message":"Successfully get session info data","data":{"datetime":"2026-08-09 18:54:49","detail":{"fca":{"session":0,"state_name":"STATE_NAME_MARKET_CLOSED","is_last_session":false,"is_end_of_day":true,"state_start_time":"","state_end_time":"","time_left":{"raw":49811,"formatted":"13 jam 50 menit 11 detik"},"suspend_info":""},"regular":{"session":0,"state_name":"STATE_NAME_MARKET_CLOSED","is_last_session":false,"is_end_of_day":true,"state_start_time":"","state_end_time":"","time_left":{"raw":49811,"formatted":"13 jam 50 menit 11 detik"},"suspend_info":""}}}}`

func TestGetMarketSession(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		handler func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check   func(t *testing.T, resp *MarketSessionResponse)
	}{
		{
			name: "returns session state",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, marketSessionPath, r.URL.Path)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(sessionBody))
			},
			check: func(t *testing.T, resp *MarketSessionResponse) {
				assert.Equal(t, "2026-08-09 18:54:49", resp.Data.Datetime)
				assert.Equal(t, "STATE_NAME_MARKET_CLOSED", resp.Data.Detail.Regular.StateName)
				assert.True(t, resp.Data.Detail.Regular.IsEndOfDay)
				assert.Equal(t, "13 jam 50 menit 11 detik", resp.Data.Detail.Regular.TimeLeft.Formatted)
				assert.Equal(t, 49811.0, resp.Data.Detail.FCA.TimeLeft.Raw)
			},
		},
		{
			name: "uses access token",
			opts: []Option{WithAuthenticator(&stubAuth{token: "at-ok"})},
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "Bearer at-ok", r.Header.Get("Authorization"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(sessionBody))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.handler(t, w, r)
			}))
			defer srv.Close()

			opts := append([]Option{WithBaseURL(srv.URL)}, tt.opts...)
			resp, err := New(opts...).GetMarketSession(context.Background())
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}
