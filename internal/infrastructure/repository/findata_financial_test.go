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

const findataFinancialV2RepoBody = `{"data":{"currency":["IDR","USD"],"default_currency":"IDR","html_report":"","rounding_value":[1000000000,1000000],"data_tables":{"periods":["12M 2025","12M 2024"],"accounts":[{"id":190,"level":1,"name":"<b>Arus Kas Dari Aktivitas Operasi</b>","values":[],"accounts":[{"id":191,"level":2,"name":"Penerimaan Kas Dari Pelanggan","values":["132,751 B","37,651 B"],"accounts":[],"is_total_exist":true,"is_default_expanded":false,"max_show_level":1}],"is_total_exist":true,"is_default_expanded":false,"max_show_level":1}],"max_show_level":1}}}`

func TestFindataFinancialRepositoryGetFindataFinancial(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{
			name:   "returns mapped structured financial report",
			status: http.StatusOK,
			body:   findataFinancialV2RepoBody,
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
				assert.Equal(t, "/findata-view/v2/financials/BRPT", r.URL.Path)
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewFindataFinancialRepository(client)

			got, err := repo.GetFindataFinancial(context.Background(), "BRPT", 1, 0, 1, 3, 2)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "IDR", got.DefaultCurrency)
			require.Len(t, got.DataTables.Periods, 2)
			assert.Equal(t, "12M 2025", got.DataTables.Periods[0])
			require.Len(t, got.DataTables.Accounts, 1)
			assert.Equal(t, int64(190), got.DataTables.Accounts[0].ID)
			assert.Equal(t, "Penerimaan Kas Dari Pelanggan", got.DataTables.Accounts[0].Accounts[0].Name)
		})
	}
}
