package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nofendian17/sbterm-server/pkg/log"
)

const trendingBody = `{"message":"Successfully retrieved trending stocks list","data":[{"change":"+5","symbol":"DSSA","percent":"0.52000","name":"Dian Swastatika Sentosa Tbk","last":"975","symbol_2":"DSSA","symbol_3":"DSSA","company_id":"143","notation":[{"notation_code":"G","notation_desc":"Sanksi Administratif","icon_url":{"light_mode":"https://assets.stockbit.com/logos/notations/light/G.png","dark_mode":"https://assets.stockbit.com/logos/notations/dark/G.png"}}],"uma":false,"tradeable":1,"country":"ID","type":"Saham","corp_action":{"active":false,"icon":"","text":"","detail":null},"isexist":0,"status":"STATUS_ACTIVE","icon_url":"https://assets.stockbit.com/logos/companies/DSSA.png","is_following":false,"formatted_price":"","is_exists":false,"previous":"970","day_trade_info":{"is_show_multiplier":false,"multiplier":"0"},"trading_limit_info":{"is_trading_limit":false,"haircut_percentage":""},"margin_info":{"is_margin_trading":false,"percentage":"","percentage_raw":0}}]}`

func TestGetTrending(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		handler func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check   func(t *testing.T, resp *TrendingResponse, logs string)
	}{
		{
			name: "returns trending stocks",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, trendingPath, r.URL.Path)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(trendingBody))
			},
			check: func(t *testing.T, resp *TrendingResponse, logs string) {
				require.Len(t, resp.Data, 1)
				stock := resp.Data[0]
				assert.Equal(t, "DSSA", stock.Symbol)
				assert.Equal(t, "Dian Swastatika Sentosa Tbk", stock.Name)
				assert.Equal(t, "975", stock.Last)
				assert.Equal(t, "0.52000", stock.Percent)
				assert.Equal(t, "STATUS_ACTIVE", stock.Status)
				assert.Equal(t, "143", stock.CompanyID)
				require.Len(t, stock.Notation, 1)
				assert.Equal(t, "G", stock.Notation[0].NotationCode)
				assert.Equal(t, "Sanksi Administratif", stock.Notation[0].NotationDesc)
				assert.Equal(t, "https://assets.stockbit.com/logos/notations/light/G.png", stock.Notation[0].IconURL.LightMode)
			},
		},
		{
			name: "uses access token",
			opts: []Option{WithAuthenticator(&stubAuth{token: "at-ok"})},
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "Bearer at-ok", r.Header.Get("Authorization"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(trendingBody))
			},
		},
		{
			name: "logs request and response",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(trendingBody))
			},
			check: func(t *testing.T, resp *TrendingResponse, logs string) {
				assert.Contains(t, logs, "stockbit request")
				assert.Contains(t, logs, "DSSA")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.handler(t, w, r)
			}))
			defer srv.Close()

			var buf strings.Builder
			logger := log.New(log.WithWriter(&buf), log.WithLevel(log.LevelDebug))
			opts := append([]Option{WithBaseURL(srv.URL), WithLogger(logger)}, tt.opts...)
			resp, err := New(opts...).GetTrending(context.Background())
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resp, buf.String())
			}
		})
	}
}
