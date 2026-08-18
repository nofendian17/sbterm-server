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

const corpActionRepoBody = `{"message":"Successfully get corp action","data":[{"action_type":"rups","action_info":{"rups":{"company_id":"105","company_symbol":"BUVA","corp_action_active":false,"rups_created":"2026-05-20","rups_date":"2026-06-11","rups_id":"1460868","rups_time":"14:00"}}}]}`

func TestCorpActionRepositoryGetCorpActions(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{
			name:   "returns mapped corp actions",
			status: http.StatusOK,
			body:   corpActionRepoBody,
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
				assert.Equal(t, "/corpaction/BUVA", r.URL.Path)
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewCorpActionRepository(client)

			got, err := repo.GetCorpActions(context.Background(), "BUVA", 30)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, "rups", got[0].ActionType)
			require.NotNil(t, got[0].Rups)
			assert.Equal(t, "BUVA", got[0].Rups.CompanySymbol)
			assert.Equal(t, "2026-06-11", got[0].Rups.RupsDate)
		})
	}
}
