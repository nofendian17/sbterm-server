package repository

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit"
)

func TestRunningTradeRepositoryGetRunningTradeChart(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		wantErr  bool
		wantUp   bool
		wantCode int
	}{
		{
			name:   "returns mapped running trade chart",
			status: http.StatusOK,
			body:   runningTradeRepoBody,
		},
		{
			name:    "propagates upstream error",
			status:  http.StatusInternalServerError,
			body:    `{"message":"boom"}`,
			wantErr: true,
		},
		{
			name:     "translates upstream 400 into domain error",
			status:   http.StatusBadRequest,
			body:     `{"message":"Please check your request"}`,
			wantErr:  true,
			wantUp:   true,
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/order-trade/running-trade/chart/DSSA", r.URL.Path)
				assert.Equal(t, []string{"DR", "AK"}, r.URL.Query()["broker_code"])
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewRunningTradeRepository(client)

			got, err := repo.GetRunningTradeChart(context.Background(), "DSSA", []string{"DR", "AK"}, "2026-07-01", "2026-08-10", "INVESTOR_TYPE_ALL", "BOARD_TYPE_ALL", "")
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantUp {
					var up *domain.UpstreamError
					require.ErrorAs(t, err, &up)
					assert.Equal(t, tt.wantCode, up.Status)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "2026-07-01", got.From)
			assert.Equal(t, "10 Aug 2026", got.DateSessionInfo)
			require.Len(t, got.PriceChartData, 1)
			assert.Equal(t, "820", got.PriceChartData[0].Value.Raw)
			require.Len(t, got.BrokerChartData, 1)
			g := got.BrokerChartData[0]
			assert.Equal(t, "TYPE_CHART_VALUE", g.Type)
			require.Len(t, g.Charts, 1)
			assert.Equal(t, "ZP", g.Charts[0].BrokerCode)
			require.Len(t, g.Charts[0].Chart, 1)
			assert.Equal(t, "(27.4B)", g.Charts[0].Chart[0].Value.Formatted)
			assert.Nil(t, g.Charts[0].Chart[0].Open)
		})
	}
}

const runningTradeRepoBody = `{"data":{"from":"2026-07-01","to":"2026-08-10","data_last_updated":"2026-08-10T00:00:00Z","price_chart_data":[{"date":"2026-07-01","time":"00:00","value":{"raw":"820","formatted":"820"},"datetime_label":"01 Jul","open":{"raw":"810","formatted":"810"},"high":{"raw":"835","formatted":"835"},"low":{"raw":"795","formatted":"795"}}],"broker_chart_data":[{"type":"TYPE_CHART_VALUE","brokers":["DR","AK","DH","ZP","HP"],"charts":[{"broker_code":"ZP","chart":[{"date":"2026-07-01","time":"00:00","value":{"raw":"-27436237000","formatted":"(27.4B)"},"datetime_label":"01 Jul","open":null,"high":null,"low":null}]}]}],"date_session_info":"10 Aug 2026"}}`
