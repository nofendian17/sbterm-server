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

func TestTrendingUsecaseGetTrending(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr bool
	}{
		{name: "returns trending stocks"},
		{name: "propagates repository error", repoErr: errors.New("boom"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			want := []domain.TrendingStock{{Symbol: "DSSA", Name: "Dian Swastatika Sentosa Tbk"}}
			repo := mocks.NewMockTrendingRepository(ctrl)
			repo.EXPECT().GetTrending(gomock.Any()).Return(want, tt.repoErr)

			uc := NewTrendingUsecase(repo)
			got, err := uc.GetTrending(context.Background())
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}