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

const trendingBody = `{"data":[{"symbol":"DSSA","name":"Dian Swastatika Sentosa Tbk","last":"975","change":"+5","percent":"0.52000","previous":"970","company_id":"143","icon_url":"https://assets.stockbit.com/logos/companies/DSSA.png","is_following":true,"status":"STATUS_ACTIVE"}]}`

func TestTrendingRepositoryGetTrending(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantLen int
		wantErr bool
	}{
		{
			name:    "returns mapped trending stocks",
			status:  http.StatusOK,
			body:    trendingBody,
			wantLen: 1,
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
				assert.Equal(t, "/emitten/trending", r.URL.Path)
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewTrendingRepository(client)

			got, err := repo.GetTrending(context.Background())
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, got, tt.wantLen)
			stock := got[0]
			assert.Equal(t, "DSSA", stock.Symbol)
			assert.Equal(t, "Dian Swastatika Sentosa Tbk", stock.Name)
			assert.Equal(t, "975", stock.Last)
			assert.Equal(t, "+5", stock.Change)
			assert.Equal(t, "0.52000", stock.Percent)
			assert.Equal(t, "970", stock.Previous)
			assert.Equal(t, "https://assets.stockbit.com/logos/companies/DSSA.png", stock.LogoURL)
			assert.Equal(t, "STATUS_ACTIVE", stock.Status)
		})
	}
}
