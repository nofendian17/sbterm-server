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

func TestMarketMoverUsecaseGetMarketMover(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr bool
	}{
		{name: "returns market movers"},
		{name: "propagates repository error", repoErr: errors.New("boom"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			want := []domain.MarketMover{{Symbol: "VOKS", Name: "Voksel Electric Tbk.", Price: 270}}
			repo := mocks.NewMockMarketMoverRepository(ctrl)
			repo.EXPECT().GetMarketMover(gomock.Any(), "MOVER_TYPE_TOP_GAINER", []string{"FILTER_STOCKS_TYPE_MAIN_BOARD"}).Return(want, tt.repoErr)

			uc := NewMarketMoverUsecase(repo)
			got, err := uc.GetMarketMover(context.Background(), "MOVER_TYPE_TOP_GAINER", []string{"FILTER_STOCKS_TYPE_MAIN_BOARD"})
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}
