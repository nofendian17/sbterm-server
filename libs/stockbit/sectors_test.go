package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sectorsBody = `{"message":"Successfully retrieved Catalog Company","data":{"pchange_info":[{"icon":"https://assets.stockbit.com/images/IDXCYCLIC.png","prices":["964.192","962.716"],"previous":966.498,"last":959.874,"change":"-6.62","percent":-0.69,"type":"Index","symbol":"IDXCYCLIC","symbol_2":"CYCLICAL","id":"1000003293"}]}}`

func TestGetSectors(t *testing.T) {
	tests := []struct {
		name    string
		req     SectorsRequest
		opts    []Option
		handler func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check   func(t *testing.T, resp *SectorsResponse)
	}{
		{
			name: "sends query params and returns sectors",
			req:  SectorsRequest{SetPrice: "1", SortBy: "pchange"},
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, sectorsPath, r.URL.Path)
				assert.Equal(t, "pchange", r.URL.Query().Get("sortby"))
				assert.Equal(t, "1", r.URL.Query().Get("setprice"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(sectorsBody))
			},
			check: func(t *testing.T, resp *SectorsResponse) {
				require.Len(t, resp.Data.PChangeInfo, 1)
				c := resp.Data.PChangeInfo[0]
				assert.Equal(t, "IDXCYCLIC", c.Symbol)
				assert.Equal(t, 959.874, c.Last)
				assert.Equal(t, -0.69, c.Percent)
				assert.Equal(t, []string{"964.192", "962.716"}, c.Prices)
			},
		},
		{
			name: "uses access token",
			req:  SectorsRequest{SortBy: "pchange"},
			opts: []Option{WithAuthenticator(&stubAuth{token: "at-ok"})},
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "Bearer at-ok", r.Header.Get("Authorization"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(sectorsBody))
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
			resp, err := New(opts...).GetSectors(context.Background(), tt.req)
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}
