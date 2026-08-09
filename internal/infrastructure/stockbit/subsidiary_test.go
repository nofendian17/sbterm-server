package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const subsidiaryBody = `{"message":"Successfully retrieved subsidiary data","data":{"currency":"CURRENCY_USD","last_updated_period":"Q1 2026","unit":"UNIT_FULL","subsidiaries":[{"company_name":"PT DSST Mas Gemilang","business_type":"Penyertaan Saham","location":"Jakarta","commercial_year":"","total_assets":"1,836,569,189","percentage":"99.99","id":0,"operational_status":"","period":"","raw":null}]}}`

func TestGetSubsidiaries(t *testing.T) {
	tests := []struct {
		name    string
		symbol  string
		handler func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check   func(t *testing.T, resp *SubsidiaryResponse)
	}{
		{
			name:   "returns subsidiaries",
			symbol: "DSSA",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/emitten-metadata/subsidiary/DSSA", r.URL.Path)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(subsidiaryBody))
			},
			check: func(t *testing.T, resp *SubsidiaryResponse) {
				assert.Equal(t, "CURRENCY_USD", resp.Data.Currency)
				assert.Equal(t, "Q1 2026", resp.Data.LastUpdatedPeriod)
				assert.Equal(t, "UNIT_FULL", resp.Data.Unit)
				require.Len(t, resp.Data.Subsidiaries, 1)
				s := resp.Data.Subsidiaries[0]
				assert.Equal(t, "PT DSST Mas Gemilang", s.CompanyName)
				assert.Equal(t, "99.99", s.Percentage)
				assert.Nil(t, s.Raw)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.handler(t, w, r)
			}))
			defer srv.Close()

			resp, err := New(WithBaseURL(srv.URL)).GetSubsidiaries(context.Background(), tt.symbol)
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}