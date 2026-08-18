package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/mocks"
)

func TestShareholdingNetworkUsecaseGetShareholdingNetwork(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr bool
	}{
		{name: "returns shareholding network"},
		{name: "propagates repository error", repoErr: errors.New("boom"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			want := &domain.ShareholdingNetwork{
				RootID: "investor:8824",
				Nodes:  []domain.ShareholdingNetworkNode{{ID: "company:1000000003"}},
				Edges:  []domain.ShareholdingNetworkEdge{{FromID: "investor:1000000141"}},
			}
			repo := mocks.NewMockShareholdingNetworkRepository(ctrl)
			repo.EXPECT().GetShareholdingNetwork(gomock.Any(), "8824", "SHAREHOLDING_NETWORK_NODE_TYPE_INVESTOR", 3, 20).Return(want, tt.repoErr)

			uc := NewShareholdingNetworkUsecase(repo)
			got, err := uc.GetShareholdingNetwork(context.Background(), "8824", "SHAREHOLDING_NETWORK_NODE_TYPE_INVESTOR", 3, 20)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}
