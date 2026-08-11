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
			assert.Equal(t, "2026-08-02", got.From)
			assert.Equal(t, "11 Aug 2026", got.DateSessionInfo)
			require.Len(t, got.PriceChartData, 1)
			assert.Equal(t, "835", got.PriceChartData[0].Value.Raw)
			require.Len(t, got.BrokerChartData, 2)
			g := got.BrokerChartData[0]
			assert.Equal(t, "TYPE_CHART_VALUE", g.Type)
			assert.Equal(t, []string{"XL", "BK", "AK", "CC", "YU"}, g.Brokers)
			require.Len(t, g.Charts, 1)
			assert.Equal(t, "AK", g.Charts[0].BrokerCode)
			require.Len(t, g.Charts[0].Chart, 1)
			assert.Equal(t, "(11.9B)", g.Charts[0].Chart[0].Value.Formatted)
			assert.Nil(t, g.Charts[0].Chart[0].Open)
		})
	}
}

const runningTradeRepoBody = `{"data":{"from":"2026-08-02","to":"2026-08-11","data_last_updated":"2026-08-11T00:00:00Z","price_chart_data":[{"date":"2026-08-03","time":"00:00","value":{"raw":"835","formatted":"835"},"datetime_label":"03 Aug","open":{"raw":"850","formatted":"850"},"high":{"raw":"875","formatted":"875"},"low":{"raw":"830","formatted":"830"}}],"broker_chart_data":[{"type":"TYPE_CHART_VALUE","brokers":["XL","BK","AK","CC","YU"],"charts":[{"broker_code":"AK","chart":[{"date":"2026-08-03","time":"00:00","value":{"raw":"-11939190000","formatted":"(11.9B)"},"datetime_label":"03 Aug","open":null,"high":null,"low":null}]}]},{"type":"TYPE_CHART_VOLUME","brokers":["XL","BK","AK","CC","YU"],"charts":[{"broker_code":"XL","chart":[{"date":"2026-08-03","time":"00:00","value":{"raw":"-46272","formatted":"(46.3K)"},"datetime_label":"03 Aug","open":null,"high":null,"low":null}]}]}],"date_session_info":"11 Aug 2026"}}`
