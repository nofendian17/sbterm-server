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

func TestMarketSessionUsecaseGetMarketSession(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr bool
	}{
		{name: "returns market session"},
		{name: "propagates repository error", repoErr: errors.New("boom"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			want := &domain.MarketSession{
				Datetime: "2026-08-09 18:54:49",
				Regular:  domain.MarketSessionSegment{StateName: "STATE_NAME_MARKET_CLOSED", IsEndOfDay: true},
			}
			repo := mocks.NewMockMarketSessionRepository(ctrl)
			repo.EXPECT().GetMarketSession(gomock.Any()).Return(want, tt.repoErr)

			uc := NewMarketSessionUsecase(repo)
			got, err := uc.GetMarketSession(context.Background())
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}
