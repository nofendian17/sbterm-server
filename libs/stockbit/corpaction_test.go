package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const corpActionBody = `{"message":"Successfully get corp action","data":[{"action_type":"rups","action_info":{"rups":{"company_id":"105","company_symbol":"BUVA","corp_action_active":false,"rups_created":"2026-05-20","rups_date":"2026-06-11","rups_id":"1460868","rups_time":"14:00"}}},{"action_type":"rightissue","action_info":{"rightissue":{"company_id":"105","company_symbol":"BUVA-R","rightissue_id":"11504","rightissue_price":150,"rightissue_price_formatted":"","rightissue_ratio":"225 : 44","rightissue_exdate":"2025-11-04"}}}]}`

const corpActionCalendarBody = `{"message":"Successfully retrieved corporate action events for today","data":{"bonus":[],"dividend":[{"company_id":"0","company_symbol":"BPII","corp_action_active":false,"dividend_created":"","dividend_cumdate":"2026-08-26","dividend_datahash":"","dividend_exdate":"2026-08-27","dividend_id":"118072","dividend_iqp_id":"","dividend_lastupdate":"2026-08-14","dividend_lock":0,"dividend_paydate":"2026-09-11","dividend_recdate":"2026-08-28","dividend_value":"3.54","lastprice":"","event_note":"Cum Date","dividend_value_formatted":"Rp 3.54","lastprice_formatted":"","dividend_currency":"CURRENCY_IDR","dividend_fiscal_year":0,"dividend_value_adjusted":0}],"economic":[],"ipo":[],"pubex":[],"rightissue":[],"rups":[{"company_id":"0","company_symbol":"NICE","corp_action_active":false,"rups_created":"","rups_datahash":"","rups_date":"2026-08-26","rups_id":"0","rups_time":"15:00","rups_iqp_agenda":"","rups_iqp_id":"","rups_iqp_rec_dt":"","rups_iqp_remark":"","rups_iqp_result":"","rups_iqp_revised_date":"","rups_iqp_type":"","rups_venue":"Artotel Gelora Senayan Yudhistira Room","rups_eligible_date":"2026-07-30","company_name":"","company_icon_url":""}],"stock_reverse":[],"stocksplit":[],"tender":[{"company_id":"0","company_name":"","company_symbol":"KETR","corp_action_active":false,"tender_created":"","tender_datahash":"","tender_end":"2026-08-26","tender_id":"0","tender_paydate":"2026-09-02","tender_percentage":"35","tender_price":"523","tender_shares":"994442000","tender_start":"2026-08-08","event_note":"Offering End","tender_price_formatted":"Rp 523"}],"warrant":[{"company_id":"0","company_symbol":"BRPTBQCQ6A","corp_action_active":false,"wrant_exc_end":"2026-08-31","wrant_exc_from":"2026-06-08","wrant_exc_price":"2078","wrant_id":"3202","wrant_iqp_id":"","wrant_lastupdate":"","wrant_serie":"","wrant_total":"","wrant_trading_end":"2026-08-26","wrant_trading_from":"2026-06-08","event_note":"Trading End","wrant_exc_price_formatted":"Rp 2078","wrant_foreign_percentage":0,"wrant_local_percentage":0,"wrant_number_of_securities":0,"wrant_company_id":"0"}],"stock_dividend":[],"today":["101","202"]}}`

func TestGetCorpActionCalendar(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/corpaction", r.URL.Path)
		assert.Equal(t, "2026-08-26", r.URL.Query().Get("date"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(corpActionCalendarBody))
	}))
	defer srv.Close()

	resp, err := New(WithBaseURL(srv.URL)).GetCorpActionCalendar(context.Background(), "2026-08-26")
	require.NoError(t, err)

	assert.Equal(t, "Successfully retrieved corporate action events for today", resp.Message)

	require.Len(t, resp.Data.Dividend, 1)
	d := resp.Data.Dividend[0]
	assert.Equal(t, "BPII", d.CompanySymbol)
	assert.Equal(t, "2026-08-27", d.DividendExdate)
	assert.Equal(t, "3.54", d.DividendValue)
	assert.Equal(t, "Rp 3.54", d.DividendValueFormatted)
	assert.Equal(t, "CURRENCY_IDR", d.DividendCurrency)
	assert.Equal(t, "Cum Date", d.EventNote)

	require.Len(t, resp.Data.Rups, 1)
	rp := resp.Data.Rups[0]
	assert.Equal(t, "NICE", rp.CompanySymbol)
	assert.Equal(t, "2026-08-26", rp.RupsDate)
	assert.Equal(t, "15:00", rp.RupsTime)

	require.Len(t, resp.Data.Tender, 1)
	td := resp.Data.Tender[0]
	assert.Equal(t, "KETR", td.CompanySymbol)
	assert.Equal(t, "2026-08-26", td.TenderEnd)
	assert.Equal(t, "523", td.TenderPrice)
	assert.Equal(t, "35", td.TenderPercentage)

	require.Len(t, resp.Data.Warrant, 1)
	wr := resp.Data.Warrant[0]
	assert.Equal(t, "BRPTBQCQ6A", wr.CompanySymbol)
	assert.Equal(t, "2026-08-26", wr.WrantTradingEnd)
	assert.Equal(t, "2078", wr.WrantExcPrice)

	assert.Empty(t, resp.Data.Bonus)
	assert.Empty(t, resp.Data.Economic)
	assert.Empty(t, resp.Data.Ipo)
	assert.Empty(t, resp.Data.Pubex)
	assert.Empty(t, resp.Data.StockReverse)
	assert.Empty(t, resp.Data.StockDividend)

	assert.Equal(t, []string{"101", "202"}, []string(resp.Data.Today))
}

// TestGetCorpActionCalendarScalarToday reproduces the upstream schema drift where
// "today" is a bare date string (e.g. "2026-08-25") instead of an array of ids.
// It must decode without error (no more 500 INTERNAL_ERROR) and normalize to a slice.
func TestGetCorpActionCalendarScalarToday(t *testing.T) {
	body := `{"message":"Successfully retrieved corporate action events for today","data":{"bonus":[],"dividend":[],"economic":[],"ipo":[],"pubex":[],"rightissue":[],"rups":[],"stock_reverse":[],"stocksplit":[],"tender":[],"warrant":[],"stock_dividend":[],"today":"2026-08-25"}}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/corpaction", r.URL.Path)
		assert.Equal(t, "2026-08-25", r.URL.Query().Get("date"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	}))
	defer srv.Close()

	resp, err := New(WithBaseURL(srv.URL)).GetCorpActionCalendar(context.Background(), "2026-08-25")
	require.NoError(t, err)
	assert.Equal(t, []string{"2026-08-25"}, []string(resp.Data.Today))
}

func TestGetCorpActions(t *testing.T) {
	tests := []struct {
		name    string
		symbol  string
		limit   int
		handler func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check   func(t *testing.T, resp *CorpActionResponse)
	}{
		{
			name:   "returns corp actions and preserves action info",
			symbol: "BUVA",
			limit:  30,
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/corpaction/BUVA", r.URL.Path)
				assert.Equal(t, "30", r.URL.Query().Get("limit"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(corpActionBody))
			},
			check: func(t *testing.T, resp *CorpActionResponse) {
				require.Len(t, resp.Data, 2)
				assert.Equal(t, "rups", resp.Data[0].ActionType)
				require.NotNil(t, resp.Data[0].Rups)
				assert.Equal(t, "2026-06-11", resp.Data[0].Rups.RupsDate)
				assert.Equal(t, "14:00", resp.Data[0].Rups.RupsTime)
				assert.Equal(t, "rightissue", resp.Data[1].ActionType)
				require.NotNil(t, resp.Data[1].RightIssue)
				assert.Equal(t, int64(150), int64(resp.Data[1].RightIssue.RightIssuePrice))
				assert.Equal(t, "225 : 44", resp.Data[1].RightIssue.RightIssueRatio)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.handler(t, w, r)
			}))
			defer srv.Close()

			resp, err := New(WithBaseURL(srv.URL)).GetCorpActions(context.Background(), tt.symbol, tt.limit)
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}
