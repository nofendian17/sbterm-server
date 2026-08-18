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

const subsectorBody = `{"data":[{"avgvolume":"151621732.00","change":"+80.00","company_id":"26","company_status":"STATUS_ACTIVE","last":"3160","marketcap":"75937216531000.00","name":"Aneka Tambang Tbk.","symbol":"ANTM","value":1403166616,"volume":108524600,"icon_url":"https://assets.stockbit.com/logos/companies/ANTM.png","percent":"2.60","uma":true}]}`

func TestSubsectorRepositoryGetCompanies(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{
			name:   "returns mapped subsector companies",
			status: http.StatusOK,
			body:   subsectorBody,
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
				assert.Equal(t, "/emitten/v3/sector/70/subsector/1000003292/company", r.URL.Path)
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewSubsectorRepository(client)

			got, err := repo.GetCompanies(context.Background(), "70", "1000003292")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, got, 1)
			c := got[0]
			assert.Equal(t, "ANTM", c.Symbol)
			assert.Equal(t, "Aneka Tambang Tbk.", c.Name)
			assert.Equal(t, "3160", c.Last)
			assert.Equal(t, int64(108524600), c.Volume)
			assert.Equal(t, int64(1403166616), c.Value)
			assert.Equal(t, "75937216531000.00", c.MarketCap)
			assert.Equal(t, "https://assets.stockbit.com/logos/companies/ANTM.png", c.IconURL)
			assert.Equal(t, "STATUS_ACTIVE", c.CompanyStatus)
			assert.True(t, c.IsUMA)
		})
	}
}
