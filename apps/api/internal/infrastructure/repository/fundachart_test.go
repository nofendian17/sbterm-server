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

const fundaChartRepoBody = `{"data":[{"company_id":105,"company_name":"BUVA","ratios":[{"decimal_point":2,"group_data":false,"item_id":12148,"item_name":"Current PE Ratio (Annualised)","item_type":6,"suffix":"","xaxis_id":7,"yaxis_id":3,"chart_data":[{"date":1470762000,"formated_date":"2016-08-10","value":-31.62,"ratio_value":-31.62}]}]}]}`

func TestFundaChartRepositoryGetFundaChart(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{
			name:   "returns mapped raw ratio series",
			status: http.StatusOK,
			body:   fundaChartRepoBody,
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
				assert.Equal(t, "/fundachart", r.URL.Path)
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewFundaChartRepository(client)

			got, err := repo.GetFundaChart(context.Background(), "BUVA", "12148", "10y")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, "BUVA", got[0].CompanyName)
			assert.Equal(t, "Current PE Ratio (Annualised)", got[0].Ratios[0].ItemName)
			assert.Equal(t, float64(-31.62), got[0].Ratios[0].ChartData[0].Value)
		})
	}
}
