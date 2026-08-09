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

const shareholdingCompositionRepoBody = `{"message":"Successfully fetched composition","data":{"periods":[{"report_date":"2026-07-31","total_shares":{"raw":"192638080000","formatted":"192.64B"},"compositions":[{"label":"SINAR MAS TUNGGAL","shares":{"raw":"115388080000","formatted":"115.39B"},"percentage":{"raw":59.89889434113961,"formatted":"59.90%"},"colors":{"light":"#0BA16B","dark":"#0BA16B"}}]}]}}`

func TestShareholdingCompositionRepositoryGetShareholdingComposition(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{
			name:   "returns mapped composition periods",
			status: http.StatusOK,
			body:   shareholdingCompositionRepoBody,
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
				assert.Equal(t, "/insider/shareholding/composition/companies/DSSA", r.URL.Path)
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewShareholdingCompositionRepository(client)

			got, err := repo.GetShareholdingComposition(context.Background(), "DSSA", "2026-06-01", "2026-06-30")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, got, 1)
			p := got[0]
			assert.Equal(t, "2026-07-31", p.ReportDate)
			assert.Equal(t, "192.64B", p.TotalShares.Formatted)
			require.Len(t, p.Compositions, 1)
			c := p.Compositions[0]
			assert.Equal(t, "SINAR MAS TUNGGAL", c.Label)
			assert.Equal(t, 59.89889434113961, c.Percentage.Raw)
			assert.Equal(t, "#0BA16B", c.Colors.Dark)
		})
	}
}