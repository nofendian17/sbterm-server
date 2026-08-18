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

const brokerTopRepoBody = `{"message":"Successfully get top broker","data":{"date":{"from":"2026-08-11","to":"2026-08-11","idx":"2026-08-11"},"list":[{"code":"XL","name":"Stockbit Sekuritas Digital","investor_type":"INVESTOR_TYPE_UNSPECIFIED","total_value":"3954882296950","net_value":"31002582100","buy_value":"1992942439525","sell_value":"1961939857425","total_volume":"16818162166","total_frequency":"1431582","group":"BROKER_GROUP_LOCAL"},{"code":"AK","name":"UBS Sekuritas Indonesia","investor_type":"INVESTOR_TYPE_UNSPECIFIED","total_value":"2972454430720","net_value":"-364958830900","buy_value":"1303747799910","sell_value":"1668706630810","total_volume":"4282349294","total_frequency":"271644","group":"BROKER_GROUP_FOREIGN"}]}}`

func TestBrokerTopRepositoryGetBrokerTop(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{
			name:   "returns mapped broker top",
			status: http.StatusOK,
			body:   brokerTopRepoBody,
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
				assert.Equal(t, "/order-trade/broker/top", r.URL.Path)
				assert.Equal(t, "TB_SORT_BY_TOTAL_VALUE", r.URL.Query().Get("sort"))
				assert.Equal(t, "ORDER_BY_DESC", r.URL.Query().Get("order"))
				assert.Equal(t, "TB_PERIOD_LAST_1_DAY", r.URL.Query().Get("period"))
				assert.Equal(t, "MARKET_TYPE_ALL", r.URL.Query().Get("market_type"))
				assert.Equal(t, "true", r.URL.Query().Get("eod_only"))
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewBrokerTopRepository(client)

			got, err := repo.GetBrokerTop(context.Background(), "TB_SORT_BY_TOTAL_VALUE", "ORDER_BY_DESC", "TB_PERIOD_LAST_1_DAY", "MARKET_TYPE_ALL", true)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "2026-08-11", got.Date.Idx)
			require.Len(t, got.List, 2)
			first := got.List[0]
			assert.Equal(t, "XL", first.Code)
			assert.Equal(t, "3954882296950", first.TotalValue)
			assert.Equal(t, "BROKER_GROUP_LOCAL", first.Group)
		})
	}
}
