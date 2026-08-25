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

const corpActionRepoBody = `{"message":"Successfully get corp action","data":[{"action_type":"rups","action_info":{"rups":{"company_id":"105","company_symbol":"BUVA","corp_action_active":false,"rups_created":"2026-05-20","rups_date":"2026-06-11","rups_id":"1460868","rups_time":"14:00"}}}]}`

const corpActionCalendarRepoBody = `{"message":"ok","data":{"bonus":[],"dividend":[{"company_id":"0","company_symbol":"BPII","dividend_exdate":"2026-08-27","dividend_value":"3.54","event_note":"Cum Date"}],"economic":[],"ipo":[],"pubex":[],"rightissue":[],"rups":[{"company_id":"0","company_symbol":"NICE","rups_date":"2026-08-26","rups_time":"15:00"}],"stock_reverse":[],"stocksplit":[],"tender":[{"company_symbol":"KETR","tender_end":"2026-08-26","tender_price":"523"}],"warrant":[{"company_symbol":"BRPTBQCQ6A","wrant_trading_end":"2026-08-26","wrant_exc_price":"2078"}],"stock_dividend":[],"today":["101"]}}`

func TestCorpActionRepositoryGetCorpActions(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{
			name:   "returns mapped corp actions",
			status: http.StatusOK,
			body:   corpActionRepoBody,
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
				assert.Equal(t, "/corpaction/BUVA", r.URL.Path)
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewCorpActionRepository(client)

			got, err := repo.GetCorpActions(context.Background(), "BUVA", 30)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, "rups", got[0].ActionType)
			require.NotNil(t, got[0].Rups)
			assert.Equal(t, "BUVA", got[0].Rups.CompanySymbol)
			assert.Equal(t, "2026-06-11", got[0].Rups.RupsDate)
		})
	}
}

func TestCorpActionRepositoryGetCorpActionCalendar(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{
			name:   "returns mapped calendar",
			status: http.StatusOK,
			body:   corpActionCalendarRepoBody,
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
				assert.Equal(t, "/corpaction", r.URL.Path)
				assert.Equal(t, "2026-08-26", r.URL.Query().Get("date"))
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewCorpActionRepository(client)

			got, err := repo.GetCorpActionCalendar(context.Background(), "2026-08-26")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			require.Len(t, got.Dividend, 1)
			assert.Equal(t, "BPII", got.Dividend[0].CompanySymbol)
			assert.Equal(t, "3.54", got.Dividend[0].DividendValue)
			assert.Equal(t, "Cum Date", got.Dividend[0].EventNote)

			require.Len(t, got.Rups, 1)
			assert.Equal(t, "NICE", got.Rups[0].CompanySymbol)
			assert.Equal(t, "15:00", got.Rups[0].RupsTime)

			require.Len(t, got.Tender, 1)
			assert.Equal(t, "KETR", got.Tender[0].CompanySymbol)
			assert.Equal(t, "523", got.Tender[0].TenderPrice)

			require.Len(t, got.Warrant, 1)
			assert.Equal(t, "BRPTBQCQ6A", got.Warrant[0].CompanySymbol)
			assert.Equal(t, "2078", got.Warrant[0].WrantExcPrice)

			assert.Empty(t, got.Bonus)
			assert.Empty(t, got.StockSplit)
			assert.Equal(t, []string{"101"}, got.Today)
		})
	}
}
