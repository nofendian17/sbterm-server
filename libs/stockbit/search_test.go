package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nofendian17/sbterm/libs/pkg/log"
)

const searchBody = `{"message":"Success retrieved search results","data":{"chat":[],"company":[{"id":"59","name":"BBRI","country":"ID","desc":"Bank Rakyat Indonesia (Persero) Tbk.","exchange":"IDX","is_following":false,"img":"","is_verified":false,"other":"saham","status":"0","symbol_2":"BBRI","symbol_3":"BBRI","total_followers":0,"is_tradeable":true,"type":"Saham","url":"symbol/BBRI","icon_url":"https://assets.stockbit.com/logos/companies/BBRI.png"}],"insider":[],"people":[],"sector":[],"industries":[],"pagination":{"has_more_companies":false,"has_more_insiders":false,"has_more_users":false}}}`

func TestGetSearch(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		handler func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check   func(t *testing.T, resp *SearchResponse, logs string)
	}{
		{
			name: "sends keyword, page and type",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, searchPath, r.URL.Path)
				assert.Equal(t, "BBRI", r.URL.Query().Get("keyword"))
				assert.Equal(t, "1", r.URL.Query().Get("page"))
				assert.Equal(t, "company", r.URL.Query().Get("type"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(searchBody))
			},
			check: func(t *testing.T, resp *SearchResponse, logs string) {
				require.Len(t, resp.Data.Company, 1)
				company := resp.Data.Company[0]
				assert.Equal(t, "59", company.ID)
				assert.Equal(t, "BBRI", company.Name)
				assert.Equal(t, "Bank Rakyat Indonesia (Persero) Tbk.", company.Desc)
				assert.Equal(t, "IDX", company.Exchange)
				assert.True(t, company.IsTradeable)
				assert.Equal(t, "Saham", company.Type)
				assert.Equal(t, "symbol/BBRI", company.URL)
				assert.Equal(t, "https://assets.stockbit.com/logos/companies/BBRI.png", company.IconURL)
			},
		},
		{
			name: "uses access token",
			opts: []Option{WithAuthenticator(&stubAuth{token: "at-ok"})},
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "Bearer at-ok", r.Header.Get("Authorization"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(searchBody))
			},
		},
		{
			name: "logs request and response",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(searchBody))
			},
			check: func(t *testing.T, resp *SearchResponse, logs string) {
				assert.Contains(t, logs, "stockbit request")
				assert.Contains(t, logs, "BBRI")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.handler(t, w, r)
			}))
			defer srv.Close()

			var buf strings.Builder
			logger := log.New(log.WithWriter(&buf), log.WithLevel(log.LevelDebug))
			opts := append([]Option{WithBaseURL(srv.URL), WithLogger(logger)}, tt.opts...)
			resp, err := New(opts...).GetSearch(context.Background(), "BBRI", 1, "company")
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resp, buf.String())
			}
		})
	}
}
