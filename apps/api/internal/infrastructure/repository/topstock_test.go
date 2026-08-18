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

const topStockRepoBody = `{"data":{"top_buy":[{"rank":1,"code":"DSSA","icon_url":"https://assets.stockbit.com/logos/companies/DSSA.png","value":{"raw":"1297165000000","formatted":"1,297.2B"},"lot":{"raw":"13305000","formatted":"13.3M"},"average":{"raw":"974","formatted":"974"},"foreign_value":{"raw":"0","formatted":"0"},"frequency":{"raw":"3","formatted":"3"}}],"top_sell":[{"rank":1,"code":"AMRT","icon_url":"https://assets.stockbit.com/logos/companies/AMRT.png","value":{"raw":"-28256436000","formatted":"-28.3B"},"lot":{"raw":"-204165","formatted":"-204.2K"},"average":{"raw":"1384","formatted":"1,384"},"foreign_value":{"raw":"0","formatted":"0"},"frequency":{"raw":"-4","formatted":"-4"}}],"total":[],"response_info":{"page":1,"limit":100,"max_day_duration":360,"start_date":"2026-08-09","end_date":"2026-08-10","value_type":"VALUE_TYPE_NET"},"display_option":{"banner_message":"","foreign_value_column":false,"enabled_value_type":{"gross":true,"net":true,"total":true}}}}`

func TestTopStockRepositoryGetTopStock(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{
			name:   "returns mapped top stock data",
			status: http.StatusOK,
			body:   topStockRepoBody,
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
				assert.Equal(t, "/order-trade/top-stock", r.URL.Path)
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewTopStockRepository(client)

			got, err := repo.GetTopStock(context.Background(), "2026-08-09", "2026-08-10", "INVESTOR_TYPE_ALL", "MARKET_TYPE_ALL", "VALUE_TYPE_NET", 1)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, got.TopBuy, 1)
			assert.Equal(t, 1, got.TopBuy[0].Rank)
			assert.Equal(t, "DSSA", got.TopBuy[0].Code)
			assert.Equal(t, "1297165000000", got.TopBuy[0].Value.Raw)
			assert.Equal(t, "1,297.2B", got.TopBuy[0].Value.Formatted)
			require.Len(t, got.TopSell, 1)
			assert.Equal(t, "AMRT", got.TopSell[0].Code)
			assert.Equal(t, "VALUE_TYPE_NET", got.ResponseInfo.ValueType)
			assert.Equal(t, 100, got.ResponseInfo.Limit)
			assert.True(t, got.DisplayOption.EnabledValueType.Net)
		})
	}
}
