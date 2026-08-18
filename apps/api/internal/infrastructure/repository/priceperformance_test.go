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

const pricePerformanceRepoBody = `{"data":{"prices":[{"close":{"raw":785,"formatted":"785"},"high":{"raw":805,"formatted":"805"},"low":{"raw":780,"formatted":"780"},"percentage":{"raw":-1.2578616,"formatted":"(-1.26%)"},"timeframe":"1D"}]}}`

func TestPricePerformanceRepositoryGetPricePerformance(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{
			name:   "returns mapped price performance",
			status: http.StatusOK,
			body:   pricePerformanceRepoBody,
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
				assert.Equal(t, "/company-price-feed/price-performance/BUVA", r.URL.Path)
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewPricePerformanceRepository(client)

			got, err := repo.GetPricePerformance(context.Background(), "BUVA")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, got.Prices, 1)
			p := got.Prices[0]
			assert.Equal(t, "1D", p.Timeframe)
			assert.Equal(t, float64(785), p.Close.Raw)
			assert.Equal(t, "(-1.26%)", p.Percentage.Formatted)
		})
	}
}
