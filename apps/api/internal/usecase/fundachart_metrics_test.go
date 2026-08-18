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

func TestFundaChartMetricsUsecaseGetFundaChartMetrics(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr bool
	}{
		{name: "returns metrics"},
		{name: "propagates repository error", repoErr: errors.New("boom"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			want := []domain.FundaChartMetric{{FitemID: 18, FitemName: "Size", Child: []domain.FundaChartMetric{{FitemID: 2892, FitemName: "Market Cap"}}}}
			repo := mocks.NewMockFundaChartMetricsRepository(ctrl)
			repo.EXPECT().GetFundaChartMetrics(gomock.Any(), "fundachart").Return(want, tt.repoErr)

			uc := NewFundaChartMetricsUsecase(repo)
			got, err := uc.GetFundaChartMetrics(context.Background(), "fundachart")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}
