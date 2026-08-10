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

const majorHolderRepoBody = `{"data":{"is_more":false,"movement":[{"id":"10199","name":"DIAN SWASTATIKA SENTOSA","symbol":"DSSA","date":"22 Apr 26","previous":{"value":"37,605,956,750","percentage":"19.52","formatted_value":""},"current":{"value":"37,905,956,750","percentage":"19.68","formatted_value":""},"changes":{"value":"+300,000,000","percentage":"+0.16","formatted_value":"300,000,000"},"marker":"","is_posted":false,"cmh_id":"0","nationality":"NATIONALITY_TYPE_LOCAL","action_type":"ACTION_TYPE_BUY","data_source":{"label":"Sumber: KSEI","type":"SOURCE_TYPE_KSEI"},"price_formatted":"0","broker_detail":{"code":"SS","group":"BROKER_GROUP_LOCAL"},"badges":[]}]}}`

func TestMajorHolderRepositoryGetMajorHolder(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{
			name:   "returns mapped major holder data",
			status: http.StatusOK,
			body:   majorHolderRepoBody,
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
				assert.Equal(t, "/insider/company/majorholder", r.URL.Path)
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewMajorHolderRepository(client)

			got, err := repo.GetMajorHolder(context.Background(), "DSSA", "ACTION_TYPE_UNSPECIFIED", "SOURCE_TYPE_UNSPECIFIED", 1, 20)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.False(t, got.IsMore)
			require.Len(t, got.Movement, 1)
			m := got.Movement[0]
			assert.Equal(t, "DIAN SWASTATIKA SENTOSA", m.Name)
			assert.Equal(t, "+0.16", m.Changes.Percentage)
			assert.Equal(t, "SOURCE_TYPE_KSEI", m.DataSource.Type)
			assert.Equal(t, "BROKER_GROUP_LOCAL", m.BrokerDetail.Group)
		})
	}
}
