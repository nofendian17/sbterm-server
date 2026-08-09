package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const shareholdingNetworkBody = `{"message":"Successfully built shareholding network","data":{"root_id":"investor:8824","root_type":"SHAREHOLDING_NETWORK_NODE_TYPE_INVESTOR","report_date":"31 Jul 26","nodes":[{"id":"company:1000000003","node_type":"SHAREHOLDING_NETWORK_NODE_TYPE_COMPANY","metadata":{"company":{"id":1000000003,"symbol":"NICK","name":"Charnic Capital Tbk.","icon_url":"https://assets.stockbit.com/logos/companies/NICK.png"},"investor":null},"min_depth":3,"is_rendered":false},{"id":"investor:1000000141","node_type":"SHAREHOLDING_NETWORK_NODE_TYPE_INVESTOR","metadata":{"company":null,"investor":{"id":1000000141,"name":"MNC TOURISM INDONESIA","investor_type":{"raw":"CP","formatted":"CP"},"location":{"raw":"L","formatted":"Local"},"nationality":{"raw":"","formatted":"-"},"domicile":{"raw":"INDONESIA","formatted":"INDONESIA"},"investor_classification":{"raw":"Firm","formatted":"Firm"}}},"min_depth":2,"is_rendered":true}],"edges":[{"from_id":"investor:1000000141","to_id":"company:1000000022","shareholding":{"scripless":{"raw":"894310500","formatted":"894.31 M"},"scrip":{"raw":"0","formatted":"0.00"},"total_shares":{"raw":"894310500","formatted":"894.31 M"},"percentage":{"raw":1.47,"formatted":"1.47%"}},"is_rendered":false}]}}`

func TestGetShareholdingNetwork(t *testing.T) {
	tests := []struct {
		name           string
		rootID         string
		rootType       string
		maxDepth       int
		maxEdgePerNode int
		handler        func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check          func(t *testing.T, resp *ShareholdingNetworkResponse)
	}{
		{
			name:           "returns shareholding network",
			rootID:         "8824",
			rootType:       "SHAREHOLDING_NETWORK_NODE_TYPE_INVESTOR",
			maxDepth:       3,
			maxEdgePerNode: 20,
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/insider/shareholding/network", r.URL.Path)
				assert.Equal(t, "8824", r.URL.Query().Get("root_id"))
				assert.Equal(t, "SHAREHOLDING_NETWORK_NODE_TYPE_INVESTOR", r.URL.Query().Get("root_type"))
				assert.Equal(t, "3", r.URL.Query().Get("max_depth"))
				assert.Equal(t, "20", r.URL.Query().Get("max_edge_per_node"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(shareholdingNetworkBody))
			},
			check: func(t *testing.T, resp *ShareholdingNetworkResponse) {
				require.Len(t, resp.Data.Nodes, 2)
				assert.Equal(t, "company:1000000003", resp.Data.Nodes[0].ID)
				assert.Equal(t, "NICK", resp.Data.Nodes[0].Metadata.Company.Symbol)
				assert.Equal(t, int64(1000000141), resp.Data.Nodes[1].Metadata.Investor.ID)
				assert.Equal(t, "Firm", resp.Data.Nodes[1].Metadata.Investor.InvestorClassification.Formatted)
				require.Len(t, resp.Data.Edges, 1)
				assert.Equal(t, "investor:1000000141", resp.Data.Edges[0].FromID)
				assert.Equal(t, 1.47, resp.Data.Edges[0].Shareholding.Percentage.Raw)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.handler(t, w, r)
			}))
			defer srv.Close()

			resp, err := New(WithBaseURL(srv.URL)).GetShareholdingNetwork(context.Background(), tt.rootID, tt.rootType, tt.maxDepth, tt.maxEdgePerNode)
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}