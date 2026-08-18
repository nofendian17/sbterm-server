package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const profileBody = `{"data":{"background":"PT Dian Swastatika Sentosa Tbk menjalankan kegiatan usaha utama","history":{"amount":"150 B","board":"Papan Utama","date":"10 Dec 2009","price":"1500","registrar":"","shares":"100,000,000","underwriters":["PT. HD Capital"],"free_float":"19.35%"},"key_executive":{"commissioner":[{"id":"35897650","key":"Commissioner","value":"HANDHIANTO SURYO KENTJONO, PH.D."}],"director":[{"id":"35897639","key":"Director","value":"HERMAWAN TARJONO"}],"independent_commissioner":[]},"address":[{"office":"Gedung Sinar Mas Land Plaza","phone":"021-31990258","fax":"021-31990259","email":["corsec@dss.co.id"],"website":"www.dssa.co.id","npwp":"01.785.257.5-054.000"}],"subsidiary":[{"company":"PT Marga Buana Bumi Mulia","percentage":"86.87%","types":"Pengolahan Bubur Kertas","value":"538884"}],"beneficiary":[{"name":"FRANKY OESMAN WIDJAJA"}],"shareholder":[{"percentage":"59.9%","name":"PT SINAR MAS TUNGGAL","value":"115.39 B","badges":["pengendali"]}],"shareholder_director_commissioner":[{"percentage":"0.0025%","name":"VIVIANA DYAH AYU","value":"3.82 M","badges":["direksi"]}],"shareholder_numbers":[{"shareholder_date":"30 Jun 2026","total_share":"86,926","change":48123,"change_formatted":"(+48,123)","change_value":"48123"}],"shareholder_one_percent":{"shareholder":[{"id":"1000004641","percentage":"59.90%","name":"SINAR MAS TUNGGAL","value":"115,388,080,000","badges":[],"type":"CP","location":"Local","nationality":"-","domicile":"INDONESIA","scripless":"0","scrip":"115,388,080,000","value_formatted":"115.39 B","classification":"Corporate"}]},"badges":["direktur","komisaris","pengendali"],"listing_information":{"exercise_start_date":"","exercise_end_date":"","exercise_price":0,"expire_date":"","listing_date":"","foreign_percentage":{"raw":0,"formatted":""},"local_percentage":{"raw":0,"formatted":""},"number_of_securities":0,"total_shares":0},"profile":{"fund_type":{"value":"","info":""},"inception_date":"","fund_manager":"","fund_manager_ico":"","custodian_bank":"","custodian_ico":"","risk_level":{"value":"","info":""},"aum":{"value":"","info":""},"maxdrawdown":{"value":"","info":""},"cagr5year":{"value":"","info":""},"expense_ratio":{"value":"","info":""},"average_yield":{"value":"","info":""},"prospectus":{"name":"","file":"","dir":"","url":""},"fund_fact_sheet":[],"redemption_bank_name":"","min_buy":"","buy_fee":"","sell_fee":""},"secretary":[{"id":"12529373","key":"","lastupdate":"2021-12-28T14:00:40+07:00","value":"{\"name\":\"Susan Chandra\"}"}],"asset_allocation":[],"classification":null,"fee":[],"pdf":[],"shareholder_reksa":[],"top_holdings":[]}}`

func TestGetProfile(t *testing.T) {
	tests := []struct {
		name    string
		symbol  string
		handler func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check   func(t *testing.T, resp *CompanyProfileResponse)
	}{
		{
			name:   "returns curated profile sections",
			symbol: "DSSA",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/emitten/DSSA/profile", r.URL.Path)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(profileBody))
			},
			check: func(t *testing.T, resp *CompanyProfileResponse) {
				require.NotNil(t, resp.Data.History)
				assert.Equal(t, "10 Dec 2009", resp.Data.History.Date)
				assert.Equal(t, "19.35%", resp.Data.History.FreeFloat)
				require.Len(t, resp.Data.History.Underwriters, 1)
				assert.Equal(t, "PT. HD Capital", resp.Data.History.Underwriters[0])
				require.NotNil(t, resp.Data.KeyExecutive)
				assert.Equal(t, "HANDHIANTO SURYO KENTJONO, PH.D.", resp.Data.KeyExecutive.Commissioner[0].Value)
				require.Len(t, resp.Data.Address, 1)
				assert.Equal(t, "corsec@dss.co.id", resp.Data.Address[0].Email[0])
				assert.Equal(t, "PT Marga Buana Bumi Mulia", resp.Data.Subsidiary[0].Company)
				assert.Equal(t, "FRANKY OESMAN WIDJAJA", resp.Data.Beneficiary[0].Name)
				require.Len(t, resp.Data.Shareholder, 1)
				assert.Equal(t, "PT SINAR MAS TUNGGAL", resp.Data.Shareholder[0].Name)
				assert.Equal(t, "pengendali", resp.Data.Shareholder[0].Badges[0])
				require.Len(t, resp.Data.ShareholderDirectorCommissioner, 1)
				assert.Equal(t, "VIVIANA DYAH AYU", resp.Data.ShareholderDirectorCommissioner[0].Name)
				require.Len(t, resp.Data.ShareholderNumbers, 1)
				assert.Equal(t, "30 Jun 2026", resp.Data.ShareholderNumbers[0].ShareholderDate)
				require.NotNil(t, resp.Data.ShareholderOnePercent)
				assert.Equal(t, "SINAR MAS TUNGGAL", resp.Data.ShareholderOnePercent.Shareholder[0].Name)
				assert.Equal(t, "1000004641", resp.Data.ShareholderOnePercent.Shareholder[0].ID)
				assert.Equal(t, "Corporate", resp.Data.ShareholderOnePercent.Shareholder[0].Classification)
				require.Len(t, resp.Data.Badges, 3)
				assert.Equal(t, "direktur", resp.Data.Badges[0])
				require.NotNil(t, resp.Data.ListingInformation)
				assert.Equal(t, int64(0), resp.Data.ListingInformation.TotalShares)
				require.NotNil(t, resp.Data.Profile)
				assert.Equal(t, "", resp.Data.Profile.FundManager)
				require.Len(t, resp.Data.Secretary, 1)
				assert.Equal(t, "12529373", resp.Data.Secretary[0].ID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.handler(t, w, r)
			}))
			defer srv.Close()

			resp, err := New(WithBaseURL(srv.URL)).GetProfile(context.Background(), tt.symbol)
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}
