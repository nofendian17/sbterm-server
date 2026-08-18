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

const shareholdingNetworkRepoBody = `{"message":"Successfully built shareholding network","data":{"root_id":"investor:8824","root_type":"SHAREHOLDING_NETWORK_NODE_TYPE_INVESTOR","report_date":"31 Jul 26","nodes":[{"id":"company:1000000003","node_type":"SHAREHOLDING_NETWORK_NODE_TYPE_COMPANY","metadata":{"company":{"id":1000000003,"symbol":"NICK","name":"Charnic Capital Tbk.","icon_url":"i"},"investor":null},"min_depth":3,"is_rendered":false}],"edges":[{"from_id":"investor:1000000141","to_id":"company:1000000022","shareholding":{"scripless":{"raw":"894310500","formatted":"894.31 M"},"scrip":{"raw":"0","formatted":"0.00"},"total_shares":{"raw":"894310500","formatted":"894.31 M"},"percentage":{"raw":1.47,"formatted":"1.47%"}},"is_rendered":false}]}}`

func TestShareholdingNetworkRepositoryGetShareholdingNetwork(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{
			name:   "returns mapped shareholding network",
			status: http.StatusOK,
			body:   shareholdingNetworkRepoBody,
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
				assert.Equal(t, "/insider/shareholding/network", r.URL.Path)
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewShareholdingNetworkRepository(client)

			got, err := repo.GetShareholdingNetwork(context.Background(), "8824", "SHAREHOLDING_NETWORK_NODE_TYPE_INVESTOR", 3, 20)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "31 Jul 26", got.ReportDate)
			require.Len(t, got.Nodes, 1)
			assert.Equal(t, "NICK", got.Nodes[0].Metadata.Company.Symbol)
			require.Len(t, got.Edges, 1)
			assert.Equal(t, 1.47, got.Edges[0].Shareholding.Percentage.Raw)
		})
	}
}
