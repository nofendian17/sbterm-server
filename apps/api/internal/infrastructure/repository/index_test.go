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

const indexBody = `{"data":{"main":[{"parent":70,"id":"559","symbol":"IDX30","name":"IDX30","percent":"1.45","change":"5.14","last":"359.4049987792969","marketcap":"5901884776.00","valuema20":"0.00"}],"all":[{"parent":70,"id":"1000003448","symbol":"ABX","name":"Papan Akselerasi","percent":"2.68","change":"71.43","last":"2737.52001953125","marketcap":"409564574.00","valuema20":"0.00"}]}}`

func TestIndexRepositoryGetIndexes(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{
			name:   "returns mapped indexes",
			status: http.StatusOK,
			body:   indexBody,
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
				assert.Equal(t, "/emitten/indexes/mobile", r.URL.Path)
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewIndexRepository(client)

			got, err := repo.GetIndexes(context.Background())
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, got.Main, 1)
			assert.Equal(t, "IDX30", got.Main[0].Symbol)
			assert.Equal(t, "359.4049987792969", got.Main[0].Last)
			assert.Equal(t, "5901884776.00", got.Main[0].MarketCap)
			require.Len(t, got.All, 1)
			assert.Equal(t, "ABX", got.All[0].Symbol)
		})
	}
}
