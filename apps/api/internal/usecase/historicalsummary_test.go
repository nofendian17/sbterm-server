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

func TestHistoricalSummaryUsecaseGetHistoricalSummary(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr bool
	}{
		{name: "returns historical summary"},
		{name: "propagates repository error", repoErr: errors.New("boom"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			want := &domain.HistoricalSummaryData{}
			repo := mocks.NewMockHistoricalSummaryRepository(ctrl)
			repo.EXPECT().GetHistoricalSummary(gomock.Any(), "DSSA", "HS_PERIOD_WEEKLY", "2025-08-11", "2026-08-11", 12, 1).Return(want, tt.repoErr)

			uc := NewHistoricalSummaryUsecase(repo)
			got, err := uc.GetHistoricalSummary(context.Background(), "DSSA", "HS_PERIOD_WEEKLY", "2025-08-11", "2026-08-11", 12, 1)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}
