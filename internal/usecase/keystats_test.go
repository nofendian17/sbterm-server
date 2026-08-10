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

func TestKeystatsUsecaseGetKeystats(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr bool
	}{
		{name: "returns keystats"},
		{name: "propagates repository error", repoErr: errors.New("boom"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			want := &domain.Keystats{
				Stats: domain.KeystatsStats{MarketCap: "19,324 B"},
				Info:  "",
			}
			repo := mocks.NewMockKeystatsRepository(ctrl)
			repo.EXPECT().GetKeystats(gomock.Any(), "BUVA", 10).Return(want, tt.repoErr)

			uc := NewKeystatsUsecase(repo)
			got, err := uc.GetKeystats(context.Background(), "BUVA", 10)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}
