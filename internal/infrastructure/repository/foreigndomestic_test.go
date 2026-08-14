package repository

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit"
)

const foreignDomesticRepoBody = `{"message":"Successfully get historical foreign data","data":{"historical_price":[{"date":"2026-08-14","datetime_label":"14 Aug","open":{"raw":"820","formatted":"820"},"high":{"raw":"920","formatted":"920"},"low":{"raw":"810","formatted":"810"},"close":{"raw":"885","formatted":"885"}}],"historical_net":[{"date":"2026-08-14","datetime_label":"14 Aug","datetime_label_table":"14 Aug 26","net_foreign":{"raw":24004280500,"formatted":"24.00B"},"foreign_buy":{"raw":45454808500,"formatted":"45.45B"},"foreign_sell":{"raw":21450528000,"formatted":"21.45B"},"foreign_flow":{"raw":268726637000,"formatted":"268.73B"},"net_lot":{"raw":274408,"formatted":"274.41K"},"net_frequency":{"raw":"2060","formatted":"2.06K"},"average_price":{"raw":878,"formatted":"878"},"percentage_foreign_value":{"raw":18.864513397216797,"formatted":"18.86%"},"percentage_domestic_value":{"raw":81.13548278808594,"formatted":"81.14%"}}],"last_updated":"14 Aug 26","from":"2026-07-14","to":"2026-08-14"}}`

func TestForeignDomesticRepositoryGetForeignDomesticHistorical(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		wantErr  bool
		wantUp   bool
		wantCode int
	}{
		{
			name:   "returns mapped foreign domestic historical",
			status: http.StatusOK,
			body:   foreignDomesticRepoBody,
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
				assert.Equal(t, "/order-trade/foreign-domestic/historical", r.URL.Path)
				assert.Equal(t, "VKTR", r.URL.Query().Get("symbols"))
				assert.Equal(t, "MARKET_TYPE_ALL", r.URL.Query().Get("market_type"))
				assert.Equal(t, "TB_PERIOD_LAST_1_MONTH", r.URL.Query().Get("period"))
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewForeignDomesticRepository(client)

			got, err := repo.GetForeignDomesticHistorical(context.Background(), "VKTR", "MARKET_TYPE_ALL", "TB_PERIOD_LAST_1_MONTH", "", "")
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
			assert.Equal(t, "14 Aug 26", got.LastUpdated)
			assert.Equal(t, "2026-07-14", got.From)
			assert.Equal(t, "2026-08-14", got.To)
			require.Len(t, got.HistoricalPrice, 1)
			p := got.HistoricalPrice[0]
			assert.Equal(t, "2026-08-14", p.Date)
			assert.Equal(t, "885", p.Close.Raw)
			require.Len(t, got.HistoricalNet, 1)
			n := got.HistoricalNet[0]
			assert.Equal(t, json.Number("24004280500"), n.NetForeign.Raw)
			assert.Equal(t, "24.00B", n.NetForeign.Formatted)
			assert.Equal(t, json.Number("2060"), n.NetFrequency.Raw)
			assert.Equal(t, json.Number("18.864513397216797"), n.PercentageForeignValue.Raw)
			assert.Equal(t, "81.14%", n.PercentageDomesticValue.Formatted)
		})
	}
}
