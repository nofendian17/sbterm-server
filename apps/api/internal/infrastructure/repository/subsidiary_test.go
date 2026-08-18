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

const subsidiaryRepoBody = `{"message":"Successfully retrieved subsidiary data","data":{"currency":"CURRENCY_USD","last_updated_period":"Q1 2026","unit":"UNIT_FULL","subsidiaries":[{"company_name":"PT DSST Mas Gemilang","business_type":"Penyertaan Saham","location":"Jakarta","commercial_year":"","total_assets":"1,836,569,189","percentage":"99.99","id":0,"operational_status":"","period":"","raw":"x"}]}}`

func TestSubsidiaryRepositoryGetSubsidiaries(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{
			name:   "returns mapped subsidiaries",
			status: http.StatusOK,
			body:   subsidiaryRepoBody,
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
				assert.Equal(t, "/emitten-metadata/subsidiary/DSSA", r.URL.Path)
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewSubsidiaryRepository(client)

			got, err := repo.GetSubsidiaries(context.Background(), "DSSA")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "CURRENCY_USD", got.Currency)
			assert.Equal(t, "Q1 2026", got.LastUpdatedPeriod)
			assert.Equal(t, "UNIT_FULL", got.Unit)
			require.Len(t, got.Subsidiaries, 1)
			s := got.Subsidiaries[0]
			assert.Equal(t, "PT DSST Mas Gemilang", s.CompanyName)
			assert.Equal(t, "99.99", s.Percentage)
			require.NotNil(t, s.Raw)
			assert.Equal(t, "x", *s.Raw)
		})
	}
}
