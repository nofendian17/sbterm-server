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

func TestTopStockUsecaseGetTopStock(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr bool
	}{
		{name: "returns top stock data"},
		{name: "propagates repository error", repoErr: errors.New("boom"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			want := &domain.TopStockData{
				ResponseInfo: domain.TopStockResponseInfo{ValueType: "VALUE_TYPE_NET"},
			}
			repo := mocks.NewMockTopStockRepository(ctrl)
			repo.EXPECT().GetTopStock(gomock.Any(), "2026-08-09", "2026-08-10", "INVESTOR_TYPE_ALL", "MARKET_TYPE_ALL", "VALUE_TYPE_NET", 1).Return(want, tt.repoErr)

			uc := NewTopStockUsecase(repo)
			got, err := uc.GetTopStock(context.Background(), "2026-08-09", "2026-08-10", "INVESTOR_TYPE_ALL", "MARKET_TYPE_ALL", "VALUE_TYPE_NET", 1)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}
