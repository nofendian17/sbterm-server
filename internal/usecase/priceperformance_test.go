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

func TestPricePerformanceUsecaseGetPricePerformance(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr bool
	}{
		{name: "returns price performance"},
		{name: "propagates repository error", repoErr: errors.New("boom"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			want := &domain.PricePerformanceData{
				Prices: []domain.PricePerformance{{Timeframe: "1D", Close: domain.PriceRawFormatted{Raw: 785, Formatted: "785"}}},
			}
			repo := mocks.NewMockPricePerformanceRepository(ctrl)
			repo.EXPECT().GetPricePerformance(gomock.Any(), "BUVA").Return(want, tt.repoErr)

			uc := NewPricePerformanceUsecase(repo)
			got, err := uc.GetPricePerformance(context.Background(), "BUVA")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}