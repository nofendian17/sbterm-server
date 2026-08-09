package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const shareholdingCompositionBody = `{"message":"Successfully fetched composition by company symbol","data":{"periods":[{"report_date":"2026-07-31","total_shares":{"raw":"192638080000","formatted":"192.64B"},"compositions":[{"label":"SINAR MAS TUNGGAL","shares":{"raw":"115388080000","formatted":"115.39B"},"percentage":{"raw":59.89889434113961,"formatted":"59.90%"},"colors":{"light":"#0BA16B","dark":"#0BA16B"}}]}]}}`

func TestGetShareholdingComposition(t *testing.T) {
	tests := []struct {
		name        string
		symbol      string
		periodStart string
		periodEnd   string
		handler     func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check       func(t *testing.T, resp *ShareholdingCompositionResponse)
	}{
		{
			name:   "returns composition periods",
			symbol: "DSSA",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/insider/shareholding/composition/companies/DSSA", r.URL.Path)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(shareholdingCompositionBody))
			},
			check: func(t *testing.T, resp *ShareholdingCompositionResponse) {
				require.Len(t, resp.Data.Periods, 1)
				p := resp.Data.Periods[0]
				assert.Equal(t, "2026-07-31", p.ReportDate)
				assert.Equal(t, "192.64B", p.TotalShares.Formatted)
				require.Len(t, p.Compositions, 1)
				c := p.Compositions[0]
				assert.Equal(t, "SINAR MAS TUNGGAL", c.Label)
				assert.Equal(t, "115.39B", c.Shares.Formatted)
				assert.Equal(t, 59.89889434113961, c.Percentage.Raw)
				assert.Equal(t, "#0BA16B", c.Colors.Light)
			},
		},
		{
			name:        "forwards period filter query params",
			symbol:      "DSSA",
			periodStart: "2026-06-01",
			periodEnd:   "2026-06-30",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "2026-06-01", r.URL.Query().Get("period_start"))
				assert.Equal(t, "2026-06-30", r.URL.Query().Get("period_end"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(shareholdingCompositionBody))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.handler(t, w, r)
			}))
			defer srv.Close()

			resp, err := New(WithBaseURL(srv.URL)).GetShareholdingComposition(context.Background(), tt.symbol, tt.periodStart, tt.periodEnd)
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}