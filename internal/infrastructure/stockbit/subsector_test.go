package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const subsectorBody = `{"message":"Found 113 companies in subsector","data":[{"avgvolume":"151621732.00","change":"+80.00","company_id":"26","last":"3160","marketcap":"75937216531000.00","name":"Aneka Tambang Tbk.","symbol":"ANTM","value":1403166616,"volume":108524600,"icon_url":"https://assets.stockbit.com/logos/companies/ANTM.png","percent":"2.60"}]}`

func TestGetSubsectorCompanies(t *testing.T) {
	tests := []struct {
		name        string
		sectorID    string
		subsectorID string
		opts        []Option
		handler     func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check       func(t *testing.T, resp *SubsectorCompaniesResponse)
	}{
		{
			name:        "returns subsector companies",
			sectorID:    "70",
			subsectorID: "1000003292",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/emitten/v3/sector/70/subsector/1000003292/company", r.URL.Path)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(subsectorBody))
			},
			check: func(t *testing.T, resp *SubsectorCompaniesResponse) {
				require.Len(t, resp.Data, 1)
				c := resp.Data[0]
				assert.Equal(t, "ANTM", c.Symbol)
				assert.Equal(t, "Aneka Tambang Tbk.", c.Name)
				assert.Equal(t, "3160", c.Last)
				assert.Equal(t, int64(108524600), c.Volume)
				assert.Equal(t, int64(1403166616), c.Value)
			},
		},
		{
			name:        "uses access token",
			sectorID:    "70",
			subsectorID: "1000003292",
			opts:        []Option{WithAuthenticator(&stubAuth{token: "at-ok"})},
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "Bearer at-ok", r.Header.Get("Authorization"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(subsectorBody))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.handler(t, w, r)
			}))
			defer srv.Close()

			opts := append([]Option{WithBaseURL(srv.URL)}, tt.opts...)
			resp, err := New(opts...).GetSubsectorCompanies(context.Background(), tt.sectorID, tt.subsectorID)
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}
