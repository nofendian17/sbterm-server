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

func TestFundaChartUsecaseGetFundaChart(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr bool
	}{
		{name: "returns funda chart"},
		{name: "propagates repository error", repoErr: errors.New("boom"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			want := []domain.FundaChartCompany{{
				CompanyID:   105,
				CompanyName: "BUVA",
				Ratios: []domain.FundaChartRatio{{
					ItemID:    12148,
					ChartData: []domain.FundaChartPoint{{Date: 1470762000, Value: -31.62}},
				}},
			}}
			repo := mocks.NewMockFundaChartRepository(ctrl)
			repo.EXPECT().GetFundaChart(gomock.Any(), "BUVA", "12148", "10y").Return(want, tt.repoErr)

			uc := NewFundaChartUsecase(repo)
			got, err := uc.GetFundaChart(context.Background(), "BUVA", "12148", "10y")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}
