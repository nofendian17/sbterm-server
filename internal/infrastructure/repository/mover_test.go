package repository

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit"
)

const moverBody = `{"data":{"mover_list":[{"stock_detail":{"code":"VOKS","name":"Voksel Electric Tbk.","icon_url":"","has_uma":false,"notations":[],"corpaction":{"active":false}},"price":270,"change":{"value":70,"percentage":35},"value":{"raw":4795797800,"formatted":"4.80 B"},"volume":{"raw":183517,"formatted":"183.52 K"},"frequency":{"raw":2743,"formatted":"2.74 K"},"net_foreign_buy":{"raw":0,"formatted":"-"},"net_foreign_sell":{"raw":164914400,"formatted":"164.91 M"},"iepiev_detail":{"iep":{"raw":250,"formatted":"250"},"iev":{"raw":260,"formatted":"260"},"ieval":{"raw":255,"formatted":"255"},"iep_change":{"raw":5,"formatted":"0.00%"},"iep_change_prev":{"raw":75,"formatted":"75%"},"iep_price_diff":{"raw":0,"formatted":"0"},"iep_prev_price_diff":{"raw":0,"formatted":"0"}}}]}}`

func TestMarketMoverRepositoryGetMarketMover(t *testing.T) {
	tests := []struct {
		name      string
		moverType string
		filters   []string
		status    int
		body      string
		wantLen   int
		wantErr   bool
	}{
		{
			name:      "returns mapped market movers",
			moverType: "MOVER_TYPE_TOP_GAINER",
			filters:   []string{"FILTER_STOCKS_TYPE_MAIN_BOARD"},
			status:    http.StatusOK,
			body:      moverBody,
			wantLen:   1,
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
				assert.Equal(t, "/order-trade/market-mover", r.URL.Path)
				assert.Equal(t, tt.moverType, r.URL.Query().Get("mover_type"))
				assert.Equal(t, tt.filters, r.URL.Query()["filter_stocks"])
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewMarketMoverRepository(client)

			got, err := repo.GetMarketMover(context.Background(), tt.moverType, tt.filters)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, got, tt.wantLen)
			m := got[0]
			assert.Equal(t, "VOKS", m.Symbol)
			assert.Equal(t, "Voksel Electric Tbk.", m.Name)
			assert.Equal(t, 270.0, m.Price)
			assert.Equal(t, 70.0, m.ChangeValue)
			assert.Equal(t, 35.0, m.ChangePercent)
			assert.Equal(t, 4795797800.0, m.Value)
			assert.Equal(t, 183517.0, m.Volume)
			assert.Equal(t, 2743.0, m.Frequency)
			assert.Equal(t, 0.0, m.NetForeignBuy)
			assert.Equal(t, 164914400.0, m.NetForeignSell)
			assert.Equal(t, 250.0, m.IEP)
			assert.Equal(t, 260.0, m.IEV)
			assert.Equal(t, 255.0, m.IEVAL)
			assert.Equal(t, 75.0, m.IEPChangePrev)
		})
	}
}
