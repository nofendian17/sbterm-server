package network

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/mocks"
	"github.com/nofendian17/sbterm-server/pkg/validator"
)

func TestShareholdingNetworkHandlerShareholdingNetwork(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		setup       func(uc *mocks.MockShareholdingNetworkUsecase)
		wantStatus  int
		wantNodeID  string
		wantEdge    string
		wantErrCode string
	}{
		{
			name: "returns shareholding network",
			path: "/v1/insider/shareholding-network?root_id=8824&root_type=SHAREHOLDING_NETWORK_NODE_TYPE_INVESTOR&max_depth=3&max_edge_per_node=20",
			setup: func(uc *mocks.MockShareholdingNetworkUsecase) {
				uc.EXPECT().GetShareholdingNetwork(gomock.Any(), "8824", "SHAREHOLDING_NETWORK_NODE_TYPE_INVESTOR", 3, 20).Return(&domain.ShareholdingNetwork{
					RootID: "investor:8824",
					Nodes: []domain.ShareholdingNetworkNode{
						{
							ID:       "company:1000000003",
							NodeType: "SHAREHOLDING_NETWORK_NODE_TYPE_COMPANY",
							Metadata: domain.ShareholdingNetworkMetadata{
								Company: &domain.ShareholdingNetworkCompany{ID: 1000000003, Symbol: "NICK", Name: "Charnic Capital Tbk.", IconURL: "i"},
							},
						},
						{
							ID:       "investor:1000000141",
							NodeType: "SHAREHOLDING_NETWORK_NODE_TYPE_INVESTOR",
							Metadata: domain.ShareholdingNetworkMetadata{
								Investor: &domain.ShareholdingNetworkInvestor{
									ID:                     1000000141,
									Name:                   "MNC TOURISM",
									InvestorType:           domain.ShareholdingRawFormatted{Raw: "CP", Formatted: "CP"},
									InvestorClassification: domain.ShareholdingRawFormatted{Raw: "Firm", Formatted: "Firm"},
								},
							},
						},
					},
					Edges: []domain.ShareholdingNetworkEdge{{FromID: "investor:1000000141"}},
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantNodeID: "company:1000000003",
			wantEdge:   "investor:1000000141",
		},
		{
			name:        "missing root_id returns 422",
			path:        "/v1/insider/shareholding-network?root_type=SHAREHOLDING_NETWORK_NODE_TYPE_INVESTOR",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name: "usecase error returns 500",
			path: "/v1/insider/shareholding-network?root_id=8824&root_type=SHAREHOLDING_NETWORK_NODE_TYPE_INVESTOR",
			setup: func(uc *mocks.MockShareholdingNetworkUsecase) {
				uc.EXPECT().GetShareholdingNetwork(gomock.Any(), "8824", "SHAREHOLDING_NETWORK_NODE_TYPE_INVESTOR", 0, 0).Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockShareholdingNetworkUsecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			h := NewShareholdingNetworkHandler(uc, validator.New())
			rec := httptest.NewRecorder()
			h.ShareholdingNetwork(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    struct {
					Nodes []struct {
						ID       string `json:"id"`
						Metadata struct {
							Company *struct {
								Symbol string `json:"symbol"`
							} `json:"company"`
							Investor *struct {
								ID                     int64 `json:"id"`
								InvestorClassification struct {
									Formatted string `json:"formatted"`
								} `json:"investor_classification"`
							} `json:"investor"`
						} `json:"metadata"`
					} `json:"nodes"`
					Edges []struct {
						FromID string `json:"from_id"`
					} `json:"edges"`
				} `json:"data"`
				Error *struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))

			if tt.wantErrCode != "" {
				require.NotNil(t, env.Error)
				assert.Equal(t, tt.wantErrCode, env.Error.Code)
				return
			}
			require.Len(t, env.Data.Nodes, 2)
			assert.Equal(t, tt.wantNodeID, env.Data.Nodes[0].ID)
			require.NotNil(t, env.Data.Nodes[0].Metadata.Company)
			assert.Equal(t, "NICK", env.Data.Nodes[0].Metadata.Company.Symbol)
			require.NotNil(t, env.Data.Nodes[1].Metadata.Investor)
			assert.Equal(t, int64(1000000141), env.Data.Nodes[1].Metadata.Investor.ID)
			assert.Equal(t, "Firm", env.Data.Nodes[1].Metadata.Investor.InvestorClassification.Formatted)
			require.Len(t, env.Data.Edges, 1)
			assert.Equal(t, tt.wantEdge, env.Data.Edges[0].FromID)
		})
	}
}
