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

func TestChartbitUsecaseGetChartPrice(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr bool
	}{
		{name: "returns chart price"},
		{name: "propagates repository error", repoErr: errors.New("boom"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			want := &domain.ChartPriceData{
				Chartbit: []domain.ChartPrice{{Close: 985, High: 1075, Low: 975, Open: 990}},
			}
			repo := mocks.NewMockChartbitRepository(ctrl)
			repo.EXPECT().GetChartPrice(gomock.Any(), "DSSA", "daily", "2025-08-10", "2026-08-10", 0).Return(want, tt.repoErr)

			uc := NewChartbitUsecase(repo)
			got, err := uc.GetChartPrice(context.Background(), "DSSA", "daily", "2025-08-10", "2026-08-10", 0)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}
