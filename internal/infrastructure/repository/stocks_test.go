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

const ihsgBody = `{"data":[{"avgvolume":"271964770.00","change":"+25.00","company_id":"54","company_status":"STATUS_ACTIVE","last":"6375","marketcap":"785878443750000.00","name":"Bank Central Asia Tbk.","symbol":"BBCA","value":1542723864,"volume":111930800,"icon_url":"https://assets.stockbit.com/logos/companies/BBCA.png","percent":"0.39","uma":false}]}`

func TestStocksRepositoryGetStocks(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{
			name:   "returns mapped IHSG stocks",
			status: http.StatusOK,
			body:   ihsgBody,
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
				assert.Equal(t, "/emitten/v3/sector/88/subsector/467/company", r.URL.Path)
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewStocksRepository(client)

			got, err := repo.GetStocks(context.Background())
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, got, 1)
			s := got[0]
			assert.Equal(t, "BBCA", s.Symbol)
			assert.Equal(t, "Bank Central Asia Tbk.", s.Name)
			assert.Equal(t, "6375", s.Last)
			assert.Equal(t, int64(111930800), s.Volume)
			assert.Equal(t, int64(1542723864), s.Value)
			assert.Equal(t, "785878443750000.00", s.MarketCap)
			assert.Equal(t, "https://assets.stockbit.com/logos/companies/BBCA.png", s.IconURL)
			assert.Equal(t, "STATUS_ACTIVE", s.CompanyStatus)
			assert.False(t, s.IsUMA)
		})
	}
}
