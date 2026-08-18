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

const chartRepoBody = `{"data":{"allow_decimal":0,"chartbit":[{"date":"2026-08-10","unixdate":1786294800,"open":990,"high":1075,"low":975,"close":985,"volume":1255876400,"foreignbuy":363995640500,"foreignsell":258659313500,"frequency":122452,"foreignflow":-1624835353268,"soxclose":189748508800000,"dividend":0,"value":1279403011000,"shareoutstanding":192638080000,"freq_analyzer":0.6839892103258433,"lot":0}]}}`

func TestChartbitRepositoryGetChartPrice(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{
			name:   "returns mapped chart price",
			status: http.StatusOK,
			body:   chartRepoBody,
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
				assert.Equal(t, "/chartbit/DSSA/price/daily", r.URL.Path)
				assert.Equal(t, "2026-08-10", r.URL.Query().Get("to"))
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewChartbitRepository(client)

			got, err := repo.GetChartPrice(context.Background(), "DSSA", "daily", "2025-08-10", "2026-08-10", 0)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, got.Chartbit, 1)
			p := got.Chartbit[0]
			assert.Equal(t, "2026-08-10", p.Date)
			assert.Equal(t, int64(1786294800), p.Unixdate)
			assert.Equal(t, float64(985), p.Close)
			assert.Equal(t, float64(258659313500), p.ForeignSell)
			assert.Equal(t, float64(0.6839892103258433), p.FreqAnalyzer)
		})
	}
}
