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

const fundaChartMetricsRepoBody = `{"data":[{"fitem_id":18,"fitem_name":"Size","show_chart_icon":0,"child":[{"fitem_id":2892,"fitem_name":"Market Cap","show_chart_icon":0,"child":[]}]}]}`

func TestFundaChartMetricsRepositoryGetFundaChartMetrics(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{
			name:   "returns mapped metric tree",
			status: http.StatusOK,
			body:   fundaChartMetricsRepoBody,
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
				assert.Equal(t, "/fundachart/metrics", r.URL.Path)
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewFundaChartMetricsRepository(client)

			got, err := repo.GetFundaChartMetrics(context.Background(), "fundachart")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, "Size", got[0].FitemName)
			assert.Equal(t, "Market Cap", got[0].Child[0].FitemName)
		})
	}
}