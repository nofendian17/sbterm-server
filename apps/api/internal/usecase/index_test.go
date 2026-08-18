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

func TestIndexUsecaseGetIndexes(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr bool
	}{
		{name: "returns indexes"},
		{name: "propagates repository error", repoErr: errors.New("boom"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			want := &domain.Indexes{
				Main: []domain.Index{{Symbol: "IDX30", Name: "IDX30"}},
				All:  []domain.Index{{Symbol: "ABX", Name: "Papan Akselerasi"}},
			}
			repo := mocks.NewMockIndexRepository(ctrl)
			repo.EXPECT().GetIndexes(gomock.Any()).Return(want, tt.repoErr)

			uc := NewIndexUsecase(repo)
			got, err := uc.GetIndexes(context.Background())
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}
