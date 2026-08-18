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

const marketDetectorRepoBody = `{"data":{"bandar_detector":{"average":1917.1906,"avg":{"accdist":"Neutral","amount":-628582850,"percent":-0.60751265,"vol":-3278.6667},"avg5":{"accdist":"Neutral","amount":2635370000,"percent":2.5470319,"vol":13746},"broker_accdist":"Dist","number_broker_buysell":12,"top1":{"accdist":"Normal Acc","amount":13157295000,"percent":12.71626,"vol":68628},"top3":{"accdist":"Neutral","amount":1623285200,"percent":1.5688723,"vol":8467},"top5":{"accdist":"Neutral","amount":-6144404000,"percent":-5.938442,"vol":-32049},"top10":{"accdist":"Neutral","amount":-5922968600,"percent":-5.724429,"vol":-30894},"total_buyer":40,"total_seller":28,"value":103468280000,"volume":539687},"broker_summary":{"brokers_buy":[{"blot":"166709.99999999997","blotv":"1.6770999999999998e+07","bval":"3.18497415e+10","bvalv":"3.20427415e+10","netbs_broker_code":"BB","netbs_buy_avg_price":"1910.604108282154","netbs_date":"20260810","netbs_stock_code":"BRPT","type":"Lokal","freq":"1562"}],"brokers_sell":[{"netbs_broker_code":"ZP","netbs_date":"20260810","netbs_sell_avg_price":"1924.946788568702","netbs_stock_code":"BRPT","slot":"-98082","slotv":"1.17362e+07","sval":"-1.8920775e+10","svalv":"2.25915605e+10","type":"Asing","freq":"2374"}],"symbol":"BRPT"},"from":"2026-08-03","to":"2026-08-10"}}`

func TestMarketDetectorRepositoryGetMarketDetector(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{
			name:   "returns mapped market detector data",
			status: http.StatusOK,
			body:   marketDetectorRepoBody,
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
				assert.Equal(t, "/marketdetectors/BRPT", r.URL.Path)
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewMarketDetectorRepository(client)

			got, err := repo.GetMarketDetector(context.Background(), "BRPT", "2026-08-03", "2026-08-10", "TRANSACTION_TYPE_NET", "MARKET_BOARD_REGULER", "INVESTOR_TYPE_ALL", 25)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "Dist", got.BandarDetector.BrokerAccdist)
			assert.Equal(t, "Normal Acc", got.BandarDetector.Top1.Accdist)
			assert.Equal(t, int64(13157295000), got.BandarDetector.Top1.Amount)
			require.Len(t, got.BrokerSummary.BrokersBuy, 1)
			assert.Equal(t, "BB", got.BrokerSummary.BrokersBuy[0].NetbsBrokerCode)
			assert.Equal(t, "Lokal", got.BrokerSummary.BrokersBuy[0].Type)
			require.Len(t, got.BrokerSummary.BrokersSell, 1)
			assert.Equal(t, "ZP", got.BrokerSummary.BrokersSell[0].NetbsBrokerCode)
			assert.Equal(t, "-98082", got.BrokerSummary.BrokersSell[0].Slot)
			assert.Equal(t, "BRPT", got.BrokerSummary.Symbol)
			assert.Equal(t, "2026-08-03", got.From)
			assert.Equal(t, "2026-08-10", got.To)
		})
	}
}
