package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const marketDetectorBody = `{"message":"Successfully retrieved market detector data","data":{"bandar_detector":{"average":1917.1906,"avg":{"accdist":"Neutral","amount":-628582850,"percent":-0.60751265,"vol":-3278.6667},"avg5":{"accdist":"Neutral","amount":2635370000,"percent":2.5470319,"vol":13746},"broker_accdist":"Dist","number_broker_buysell":12,"top1":{"accdist":"Normal Acc","amount":13157295000,"percent":12.71626,"vol":68628},"top3":{"accdist":"Neutral","amount":1623285200,"percent":1.5688723,"vol":8467},"top5":{"accdist":"Neutral","amount":-6144404000,"percent":-5.938442,"vol":-32049},"top10":{"accdist":"Neutral","amount":-5922968600,"percent":-5.724429,"vol":-30894},"total_buyer":40,"total_seller":28,"value":103468280000,"volume":539687},"broker_summary":{"brokers_buy":[{"blot":"166709.99999999997","blotv":"1.6770999999999998e+07","bval":"3.18497415e+10","bvalv":"3.20427415e+10","netbs_broker_code":"BB","netbs_buy_avg_price":"1910.604108282154","netbs_date":"20260810","netbs_stock_code":"BRPT","type":"Lokal","freq":"1562"}],"brokers_sell":[{"netbs_broker_code":"ZP","netbs_date":"20260810","netbs_sell_avg_price":"1924.946788568702","netbs_stock_code":"BRPT","slot":"-98082","slotv":"1.17362e+07","sval":"-1.8920775e+10","svalv":"2.25915605e+10","type":"Asing","freq":"2374"}],"symbol":"BRPT"},"from":"2026-08-03","to":"2026-08-10"}}`

func TestGetMarketDetector(t *testing.T) {
	tests := []struct {
		name         string
		symbol       string
		from         string
		to           string
		transaction  string
		marketBoard  string
		investorType string
		limit        int
		handler      func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check        func(t *testing.T, resp *MarketDetectorResponse)
	}{
		{
			name:         "returns market detector data",
			symbol:       "BRPT",
			from:         "2026-08-03",
			to:           "2026-08-10",
			transaction:  "TRANSACTION_TYPE_NET",
			marketBoard:  "MARKET_BOARD_REGULER",
			investorType: "INVESTOR_TYPE_ALL",
			limit:        25,
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/marketdetectors/BRPT", r.URL.Path)
				assert.Equal(t, "2026-08-03", r.URL.Query().Get("from"))
				assert.Equal(t, "2026-08-10", r.URL.Query().Get("to"))
				assert.Equal(t, "TRANSACTION_TYPE_NET", r.URL.Query().Get("transaction_type"))
				assert.Equal(t, "MARKET_BOARD_REGULER", r.URL.Query().Get("market_board"))
				assert.Equal(t, "INVESTOR_TYPE_ALL", r.URL.Query().Get("investor_type"))
				assert.Equal(t, "25", r.URL.Query().Get("limit"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(marketDetectorBody))
			},
			check: func(t *testing.T, resp *MarketDetectorResponse) {
				bd := resp.Data.BandarDetector
				assert.Equal(t, 1917.1906, bd.Average)
				assert.Equal(t, "Dist", bd.BrokerAccdist)
				assert.Equal(t, "Normal Acc", bd.Top1.Accdist)
				assert.Equal(t, int64(13157295000), bd.Top1.Amount)
				assert.Equal(t, 12, bd.NumberBrokerBuysell)
				assert.Equal(t, int64(539687), bd.Volume)
				require.Len(t, resp.Data.BrokerSummary.BrokersBuy, 1)
				assert.Equal(t, "BB", resp.Data.BrokerSummary.BrokersBuy[0].NetbsBrokerCode)
				assert.Equal(t, "Lokal", resp.Data.BrokerSummary.BrokersBuy[0].Type)
				require.Len(t, resp.Data.BrokerSummary.BrokersSell, 1)
				assert.Equal(t, "ZP", resp.Data.BrokerSummary.BrokersSell[0].NetbsBrokerCode)
				assert.Equal(t, "-98082", resp.Data.BrokerSummary.BrokersSell[0].Slot)
				assert.Equal(t, "BRPT", resp.Data.BrokerSummary.Symbol)
				assert.Equal(t, "2026-08-03", resp.Data.From)
				assert.Equal(t, "2026-08-10", resp.Data.To)
			},
		},
		{
			name:         "omits zero limit from query",
			symbol:       "BRPT",
			from:         "2026-08-03",
			to:           "2026-08-10",
			transaction:  "TRANSACTION_TYPE_GROSS",
			marketBoard:  "MARKET_BOARD_TUNAI",
			investorType: "INVESTOR_TYPE_FOREIGN",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "", r.URL.Query().Get("limit"))
				assert.Equal(t, "TRANSACTION_TYPE_GROSS", r.URL.Query().Get("transaction_type"))
				assert.Equal(t, "MARKET_BOARD_TUNAI", r.URL.Query().Get("market_board"))
				assert.Equal(t, "INVESTOR_TYPE_FOREIGN", r.URL.Query().Get("investor_type"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(marketDetectorBody))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.handler(t, w, r)
			}))
			defer srv.Close()

			resp, err := New(WithBaseURL(srv.URL)).GetMarketDetector(context.Background(), tt.symbol, tt.from, tt.to, tt.transaction, tt.marketBoard, tt.investorType, tt.limit)
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}
