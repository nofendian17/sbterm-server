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

func TestCorpActionUsecaseGetCorpActions(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr bool
	}{
		{name: "returns corp actions"},
		{name: "propagates repository error", repoErr: errors.New("boom"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			want := []domain.CompanyCorpAction{
				{ActionType: "rups", Rups: &domain.RupsInfo{RupsID: "1460868", RupsDate: "2026-06-11"}},
			}
			repo := mocks.NewMockCorpActionRepository(ctrl)
			repo.EXPECT().GetCorpActions(gomock.Any(), "BUVA", 30).Return(want, tt.repoErr)

			uc := NewCorpActionUsecase(repo)
			got, err := uc.GetCorpActions(context.Background(), "BUVA", 30)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}
