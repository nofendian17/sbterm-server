package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const topStockBody = `{"message":"Successfully loaded top stock data","data":{"top_buy":[{"rank":1,"code":"DSSA","icon_url":"https://assets.stockbit.com/logos/companies/DSSA.png","value":{"raw":"1297165000000","formatted":"1,297.2B"},"lot":{"raw":"13305000","formatted":"13.3M"},"average":{"raw":"974","formatted":"974"},"foreign_value":{"raw":"0","formatted":"0"},"frequency":{"raw":"3","formatted":"3"}}],"top_sell":[{"rank":1,"code":"AMRT","icon_url":"https://assets.stockbit.com/logos/companies/AMRT.png","value":{"raw":"-28256436000","formatted":"-28.3B"},"lot":{"raw":"-204165","formatted":"-204.2K"},"average":{"raw":"1384","formatted":"1,384"},"foreign_value":{"raw":"0","formatted":"0"},"frequency":{"raw":"-4","formatted":"-4"}}],"total":[],"response_info":{"page":1,"limit":100,"max_day_duration":360,"start_date":"2026-08-09","end_date":"2026-08-10","value_type":"VALUE_TYPE_NET"},"display_option":{"banner_message":"","foreign_value_column":false,"enabled_value_type":{"gross":true,"net":true,"total":true}}}}`

func TestGetTopStock(t *testing.T) {
	tests := []struct {
		name         string
		start        string
		end          string
		investorType string
		marketType   string
		valueType    string
		page         int
		handler      func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check        func(t *testing.T, resp *TopStockResponse)
	}{
		{
			name:         "returns top stock data",
			start:        "2026-08-09",
			end:          "2026-08-10",
			investorType: "INVESTOR_TYPE_ALL",
			marketType:   "MARKET_TYPE_NEGO",
			valueType:    "VALUE_TYPE_NET",
			page:         1,
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/order-trade/top-stock", r.URL.Path)
				assert.Equal(t, "2026-08-09", r.URL.Query().Get("start"))
				assert.Equal(t, "2026-08-10", r.URL.Query().Get("end"))
				assert.Equal(t, "INVESTOR_TYPE_ALL", r.URL.Query().Get("investor_type"))
				assert.Equal(t, "MARKET_TYPE_NEGO", r.URL.Query().Get("market_type"))
				assert.Equal(t, "VALUE_TYPE_NET", r.URL.Query().Get("value_type"))
				assert.Equal(t, "1", r.URL.Query().Get("page"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(topStockBody))
			},
			check: func(t *testing.T, resp *TopStockResponse) {
				require.Len(t, resp.Data.TopBuy, 1)
				assert.Equal(t, 1, resp.Data.TopBuy[0].Rank)
				assert.Equal(t, "DSSA", resp.Data.TopBuy[0].Code)
				assert.Equal(t, "1297165000000", resp.Data.TopBuy[0].Value.Raw)
				assert.Equal(t, "1,297.2B", resp.Data.TopBuy[0].Value.Formatted)
				require.Len(t, resp.Data.TopSell, 1)
				assert.Equal(t, "AMRT", resp.Data.TopSell[0].Code)
				assert.Equal(t, "-28256436000", resp.Data.TopSell[0].Value.Raw)
				ri := resp.Data.ResponseInfo
				assert.Equal(t, 1, ri.Page)
				assert.Equal(t, 100, ri.Limit)
				assert.Equal(t, "VALUE_TYPE_NET", ri.ValueType)
				assert.True(t, resp.Data.DisplayOption.EnabledValueType.Net)
			},
		},
		{
			name:         "omits zero page from query",
			start:        "2026-08-09",
			end:          "2026-08-10",
			investorType: "INVESTOR_TYPE_FOREIGN",
			marketType:   "MARKET_TYPE_REGULER",
			valueType:    "VALUE_TYPE_GROSS",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "", r.URL.Query().Get("page"))
				assert.Equal(t, "INVESTOR_TYPE_FOREIGN", r.URL.Query().Get("investor_type"))
				assert.Equal(t, "MARKET_TYPE_REGULER", r.URL.Query().Get("market_type"))
				assert.Equal(t, "VALUE_TYPE_GROSS", r.URL.Query().Get("value_type"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(topStockBody))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.handler(t, w, r)
			}))
			defer srv.Close()

			resp, err := New(WithBaseURL(srv.URL)).GetTopStock(context.Background(), tt.start, tt.end, tt.investorType, tt.marketType, tt.valueType, tt.page)
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}