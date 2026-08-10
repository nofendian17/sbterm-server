package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const majorHolderBody = `{"message":"Successfully majorholder data","data":{"is_more":false,"movement":[{"id":"10199","name":"DIAN SWASTATIKA SENTOSA","symbol":"DSSA","date":"22 Apr 26","previous":{"value":"37,605,956,750","percentage":"19.52","formatted_value":""},"current":{"value":"37,905,956,750","percentage":"19.68","formatted_value":""},"changes":{"value":"+300,000,000","percentage":"+0.16","formatted_value":"300,000,000"},"marker":"","is_posted":false,"cmh_id":"0","nationality":"NATIONALITY_TYPE_LOCAL","action_type":"ACTION_TYPE_BUY","data_source":{"label":"Sumber: KSEI","type":"SOURCE_TYPE_KSEI"},"price_formatted":"0","broker_detail":{"code":"SS","group":"BROKER_GROUP_LOCAL"},"badges":[]}]}}`

func TestGetMajorHolder(t *testing.T) {
	tests := []struct {
		name       string
		symbols    string
		actionType string
		sourceType string
		page       int
		limit      int
		handler    func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check      func(t *testing.T, resp *MajorHolderResponse)
	}{
		{
			name:       "returns major holder movements",
			symbols:    "DSSA",
			actionType: "ACTION_TYPE_UNSPECIFIED",
			sourceType: "SOURCE_TYPE_UNSPECIFIED",
			page:       1,
			limit:      20,
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/insider/company/majorholder", r.URL.Path)
				assert.Equal(t, "DSSA", r.URL.Query().Get("symbols"))
				assert.Equal(t, "ACTION_TYPE_UNSPECIFIED", r.URL.Query().Get("action_type"))
				assert.Equal(t, "SOURCE_TYPE_UNSPECIFIED", r.URL.Query().Get("source_type"))
				assert.Equal(t, "1", r.URL.Query().Get("page"))
				assert.Equal(t, "20", r.URL.Query().Get("limit"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(majorHolderBody))
			},
			check: func(t *testing.T, resp *MajorHolderResponse) {
				assert.False(t, resp.Data.IsMore)
				require.Len(t, resp.Data.Movement, 1)
				m := resp.Data.Movement[0]
				assert.Equal(t, "DIAN SWASTATIKA SENTOSA", m.Name)
				assert.Equal(t, "ACTION_TYPE_BUY", m.ActionType)
				assert.Equal(t, "SOURCE_TYPE_KSEI", m.DataSource.Type)
				assert.Equal(t, "+300,000,000", m.Changes.Value)
				assert.Equal(t, "SS", m.BrokerDetail.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.handler(t, w, r)
			}))
			defer srv.Close()

			resp, err := New(WithBaseURL(srv.URL)).GetMajorHolder(context.Background(), tt.symbols, tt.actionType, tt.sourceType, tt.page, tt.limit)
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}
