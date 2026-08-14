package stockbit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// foreignDomesticBody mirrors the real upstream response captured from
// /order-trade/foreign-domestic/historical (see /tmp/opencode/fd.json),
// trimmed to one price and one net point.
const foreignDomesticBody = `{"message":"Successfully get historical foreign data","data":{"historical_price":[{"date":"2026-08-14","datetime_label":"14 Aug","open":{"raw":"820","formatted":"820"},"high":{"raw":"920","formatted":"920"},"low":{"raw":"810","formatted":"810"},"close":{"raw":"885","formatted":"885"}}],"historical_net":[{"date":"2026-08-14","datetime_label":"14 Aug","datetime_label_table":"14 Aug 26","net_foreign":{"raw":24004280500,"formatted":"24.00B"},"foreign_buy":{"raw":45454808500,"formatted":"45.45B"},"foreign_sell":{"raw":21450528000,"formatted":"21.45B"},"foreign_flow":{"raw":268726637000,"formatted":"268.73B"},"net_lot":{"raw":274408,"formatted":"274.41K"},"net_frequency":{"raw":"2060","formatted":"2.06K"},"average_price":{"raw":878,"formatted":"878"},"percentage_foreign_value":{"raw":18.864513397216797,"formatted":"18.86%"},"percentage_domestic_value":{"raw":81.13548278808594,"formatted":"81.14%"}}],"last_updated":"14 Aug 26","from":"2026-07-14","to":"2026-08-14"}}`

func TestGetForeignDomesticHistorical(t *testing.T) {
	tests := []struct {
		name       string
		symbol     string
		marketType string
		period     string
		from, to   string
		opts       []Option
		check      func(t *testing.T, r *http.Request)
		verify     func(t *testing.T, resp *ForeignDomesticResponse)
	}{
		{
			name:       "returns foreign domestic with period",
			symbol:     "VKTR",
			marketType: "MARKET_TYPE_ALL",
			period:     "TB_PERIOD_LAST_1_MONTH",
			check: func(t *testing.T, r *http.Request) {
				assert.Equal(t, "/order-trade/foreign-domestic/historical", r.URL.Path)
				q := r.URL.Query()
				assert.Equal(t, "VKTR", q.Get("symbols"))
				assert.Equal(t, "MARKET_TYPE_ALL", q.Get("market_type"))
				assert.Equal(t, "TB_PERIOD_LAST_1_MONTH", q.Get("period"))
				assert.Equal(t, "", q.Get("from"))
				assert.Equal(t, "", q.Get("to"))
			},
			verify: func(t *testing.T, resp *ForeignDomesticResponse) {
				d := resp.Data
				assert.Equal(t, "14 Aug 26", d.LastUpdated)
				assert.Equal(t, "2026-07-14", d.From)
				assert.Equal(t, "2026-08-14", d.To)
				require.Len(t, d.HistoricalPrice, 1)
				p := d.HistoricalPrice[0]
				assert.Equal(t, "2026-08-14", p.Date)
				assert.Equal(t, "820", p.Open.Raw)
				assert.Equal(t, "885", p.Close.Raw)
				require.Len(t, d.HistoricalNet, 1)
				n := d.HistoricalNet[0]
				assert.Equal(t, json.Number("24004280500"), n.NetForeign.Raw)
				assert.Equal(t, "24.00B", n.NetForeign.Formatted)
				assert.Equal(t, json.Number("2060"), n.NetFrequency.Raw)
				assert.Equal(t, json.Number("18.864513397216797"), n.PercentageForeignValue.Raw)
				assert.Equal(t, "81.14%", n.PercentageDomesticValue.Formatted)
			},
		},
		{
			name:       "from/to range wins over period",
			symbol:     "VKTR",
			marketType: "MARKET_TYPE_ALL",
			period:     "TB_PERIOD_LAST_1_MONTH",
			from:       "2026-07-01",
			to:         "2026-08-14",
			check: func(t *testing.T, r *http.Request) {
				q := r.URL.Query()
				assert.Equal(t, "2026-07-01", q.Get("from"))
				assert.Equal(t, "2026-08-14", q.Get("to"))
				assert.Equal(t, "", q.Get("period"))
			},
		},
		{
			name:       "uses access token",
			symbol:     "VKTR",
			marketType: "MARKET_TYPE_ALL",
			period:     "TB_PERIOD_LAST_1_MONTH",
			opts:       []Option{WithAuthenticator(&stubAuth{token: "at-ok"})},
			check: func(t *testing.T, r *http.Request) {
				assert.Equal(t, "Bearer at-ok", r.Header.Get("Authorization"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.check(t, r)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(foreignDomesticBody))
			}))
			defer srv.Close()

			opts := append([]Option{WithBaseURL(srv.URL)}, tt.opts...)
			resp, err := New(opts...).GetForeignDomesticHistorical(context.Background(), tt.symbol, tt.marketType, tt.period, tt.from, tt.to)
			require.NoError(t, err)
			if tt.verify != nil {
				tt.verify(t, resp)
			}
		})
	}
}
