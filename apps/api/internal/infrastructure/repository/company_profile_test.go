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

const companyProfileBody = `{"data":{"background":"PT Dian Swastatika Sentosa Tbk","history":{"amount":"150 B","board":"Papan Utama","date":"10 Dec 2009","price":"1500","registrar":"","shares":"100,000,000","underwriters":["PT. HD Capital"],"free_float":"19.35%"},"key_executive":{"commissioner":[{"id":"1","key":"Commissioner","value":"HANDHIANTO"}],"director":[],"independent_commissioner":[]},"address":[{"office":"Gedung Sinar Mas","phone":"021-31990258","fax":"021-31990259","email":["corsec@dss.co.id"],"website":"www.dssa.co.id","npwp":"01.785.257"}],"subsidiary":[{"company":"PT Marga","percentage":"86.87%","types":"Kertas","value":"538884"}],"beneficiary":[{"name":"FRANKY"}],"shareholder":[{"percentage":"59.9%","name":"PT SINAR MAS TUNGGAL","value":"115.39 B","badges":["pengendali"]}],"shareholder_director_commissioner":[{"percentage":"0.0025%","name":"VIVIANA","value":"3.82 M","badges":["direksi"]}],"shareholder_numbers":[{"shareholder_date":"30 Jun 2026","total_share":"86,926","change":48123,"change_formatted":"(+48,123)","change_value":"48123"}],"shareholder_one_percent":{"shareholder":[{"id":"1000004641","percentage":"59.90%","name":"SINAR MAS TUNGGAL","value":"115,388,080,000","badges":[],"type":"CP","location":"Local","nationality":"-","domicile":"INDONESIA","scripless":"0","scrip":"115,388,080,000","value_formatted":"115.39 B","classification":"Corporate"}]}}}`

func TestCompanyProfileRepositoryGetProfile(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{
			name:   "returns mapped company profile",
			status: http.StatusOK,
			body:   companyProfileBody,
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
				assert.Equal(t, "/emitten/DSSA/profile", r.URL.Path)
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewCompanyProfileRepository(client)

			got, err := repo.GetProfile(context.Background(), "DSSA")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got.History)
			assert.Equal(t, "10 Dec 2009", got.History.Date)
			assert.Equal(t, "19.35%", got.History.FreeFloat)
			require.NotNil(t, got.KeyExecutive)
			assert.Equal(t, "HANDHIANTO", got.KeyExecutive.Commissioner[0].Value)
			require.Len(t, got.Address, 1)
			assert.Equal(t, "corsec@dss.co.id", got.Address[0].Email[0])
			assert.Equal(t, "PT Marga", got.Subsidiary[0].Company)
			assert.Equal(t, "FRANKY", got.Beneficiary[0].Name)
			require.Len(t, got.Shareholder, 1)
			assert.Equal(t, "PT SINAR MAS TUNGGAL", got.Shareholder[0].Name)
			assert.Equal(t, "pengendali", got.Shareholder[0].Badges[0])
			require.Len(t, got.ShareholderDirectorCommissioner, 1)
			assert.Equal(t, "VIVIANA", got.ShareholderDirectorCommissioner[0].Name)
			require.Len(t, got.ShareholderNumbers, 1)
			assert.Equal(t, "30 Jun 2026", got.ShareholderNumbers[0].ShareholderDate)
			require.Len(t, got.ShareholderOnePercent, 1)
			assert.Equal(t, "SINAR MAS TUNGGAL", got.ShareholderOnePercent[0].Name)
			assert.Equal(t, "1000004641", got.ShareholderOnePercent[0].ID)
			assert.Equal(t, "INDONESIA", got.ShareholderOnePercent[0].Domicile)
		})
	}
}
