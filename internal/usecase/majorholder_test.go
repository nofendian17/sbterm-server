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

func TestMajorHolderUsecaseGetMajorHolder(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr bool
	}{
		{name: "returns major holder data"},
		{name: "propagates repository error", repoErr: errors.New("boom"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			want := &domain.MajorHolderData{
				Movement: []domain.MajorHolderMovement{{Name: "DIAN SWASTATIKA SENTOSA", Symbol: "DSSA"}},
			}
			repo := mocks.NewMockMajorHolderRepository(ctrl)
			repo.EXPECT().GetMajorHolder(gomock.Any(), "DSSA", "ACTION_TYPE_BUY", "SOURCE_TYPE_KSEI", 1, 20).Return(want, tt.repoErr)

			uc := NewMajorHolderUsecase(repo)
			got, err := uc.GetMajorHolder(context.Background(), "DSSA", "ACTION_TYPE_BUY", "SOURCE_TYPE_KSEI", 1, 20)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}
