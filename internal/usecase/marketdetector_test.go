package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/mocks"
)

func TestMarketDetectorUsecaseGetMarketDetector(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr bool
	}{
		{name: "returns market detector data"},
		{name: "propagates repository error", repoErr: errors.New("boom"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			want := &domain.MarketDetectorData{
				BrokerSummary: domain.BrokerSummary{Symbol: "BRPT"},
			}
			repo := mocks.NewMockMarketDetectorRepository(ctrl)
			repo.EXPECT().GetMarketDetector(gomock.Any(), "BRPT", "2026-08-03", "2026-08-10", "TRANSACTION_TYPE_NET", "MARKET_BOARD_REGULER", "INVESTOR_TYPE_ALL", 25).Return(want, tt.repoErr)

			uc := NewMarketDetectorUsecase(repo)
			got, err := uc.GetMarketDetector(context.Background(), "BRPT", "2026-08-03", "2026-08-10", "TRANSACTION_TYPE_NET", "MARKET_BOARD_REGULER", "INVESTOR_TYPE_ALL", 25)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}