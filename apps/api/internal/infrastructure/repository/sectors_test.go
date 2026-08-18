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

const sectorsBody = `{"data":{"pchange_info":[{"icon":"https://assets.stockbit.com/images/IDXCYCLIC.png","prices":["964.192","962.716"],"previous":966.498,"last":959.874,"change":"-6.62","percent":-0.69,"type":"Index","symbol":"IDXCYCLIC","symbol_2":"CYCLICAL","id":"1000003293"}]}}`

func TestSectorsRepositoryGetSectors(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{
			name:   "returns mapped sector entries",
			status: http.StatusOK,
			body:   sectorsBody,
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
				assert.Equal(t, "/emitten/company/catalog", r.URL.Path)
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewSectorsRepository(client)

			got, err := repo.GetSectors(context.Background())
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, got, 1)
			c := got[0]
			assert.Equal(t, "IDXCYCLIC", c.Symbol)
			assert.Equal(t, "1000003293", c.ID)
			assert.Equal(t, 959.874, c.Last)
			assert.Equal(t, "-6.62", c.Change)
			assert.Equal(t, -0.69, c.Percent)
		})
	}
}
