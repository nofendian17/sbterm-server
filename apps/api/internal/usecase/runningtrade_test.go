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

func TestRunningTradeUsecaseGetRunningTradeChart(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr bool
	}{
		{name: "returns running trade chart"},
		{name: "propagates repository error", repoErr: errors.New("boom"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			want := &domain.RunningTradeData{
				From:            "2026-07-01",
				To:              "2026-08-10",
				DateSessionInfo: "10 Aug 2026",
			}
			repo := mocks.NewMockRunningTradeRepository(ctrl)
			repo.EXPECT().GetRunningTradeChart(gomock.Any(), "DSSA", []string{"DR"}, "2026-07-01", "2026-08-10", "INVESTOR_TYPE_ALL", "BOARD_TYPE_ALL", "").Return(want, tt.repoErr)

			uc := NewRunningTradeUsecase(repo)
			got, err := uc.GetRunningTradeChart(context.Background(), "DSSA", []string{"DR"}, "2026-07-01", "2026-08-10", "INVESTOR_TYPE_ALL", "BOARD_TYPE_ALL", "")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}
