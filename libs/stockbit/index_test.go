package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const indexBody = `{"message":"Successfully retrieved company sector data","data":{"main":[{"parent":70,"id":"559","symbol":"IDX30","name":"IDX30","percent":"1.45","change":"5.14","last":"359.4049987792969","marketcap":"5901884776.00","valuema20":"0.00"}],"all":[{"parent":70,"id":"1000003448","symbol":"ABX","name":"Papan Akselerasi","percent":"2.68","change":"71.43","last":"2737.52001953125","marketcap":"409564574.00","valuema20":"0.00"}]}}`

func TestGetIndexes(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		handler func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check   func(t *testing.T, resp *IndexResponse)
	}{
		{
			name: "returns main and all indexes",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, indexesPath, r.URL.Path)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(indexBody))
			},
			check: func(t *testing.T, resp *IndexResponse) {
				require.Len(t, resp.Data.Main, 1)
				assert.Equal(t, "IDX30", resp.Data.Main[0].Symbol)
				assert.Equal(t, "359.4049987792969", resp.Data.Main[0].Last)
				assert.Equal(t, int64(70), resp.Data.Main[0].Parent)
				require.Len(t, resp.Data.All, 1)
				assert.Equal(t, "ABX", resp.Data.All[0].Symbol)
			},
		},
		{
			name: "uses access token",
			opts: []Option{WithAuthenticator(&stubAuth{token: "at-ok"})},
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "Bearer at-ok", r.Header.Get("Authorization"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(indexBody))
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
			resp, err := New(opts...).GetIndexes(context.Background())
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}
