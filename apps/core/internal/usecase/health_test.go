package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm/apps/core/internal/mocks"
)

func TestHealthUsecase_GetHealth(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(repo *mocks.MockHealthRepository)
		want      string
		wantDB    bool
		wantCache bool
	}{
		{
			name: "all healthy",
			setup: func(repo *mocks.MockHealthRepository) {
				repo.EXPECT().Ping(gomock.Any()).Return(nil)
				repo.EXPECT().PingCache(gomock.Any()).Return(nil)
			},
			want:      "ok",
			wantDB:    true,
			wantCache: true,
		},
		{
			name: "db down",
			setup: func(repo *mocks.MockHealthRepository) {
				repo.EXPECT().Ping(gomock.Any()).Return(errors.New("db error"))
				repo.EXPECT().PingCache(gomock.Any()).Return(nil)
			},
			want:      "degraded",
			wantDB:    false,
			wantCache: true,
		},
		{
			name: "cache down",
			setup: func(repo *mocks.MockHealthRepository) {
				repo.EXPECT().Ping(gomock.Any()).Return(nil)
				repo.EXPECT().PingCache(gomock.Any()).Return(errors.New("cache error"))
			},
			want:      "degraded",
			wantDB:    true,
			wantCache: false,
		},
		{
			name: "both down",
			setup: func(repo *mocks.MockHealthRepository) {
				repo.EXPECT().Ping(gomock.Any()).Return(errors.New("db error"))
				repo.EXPECT().PingCache(gomock.Any()).Return(errors.New("cache error"))
			},
			want:      "degraded",
			wantDB:    false,
			wantCache: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			repo := mocks.NewMockHealthRepository(ctrl)
			tt.setup(repo)

			uc := NewHealthUsecase(repo)
			got, err := uc.GetHealth(context.Background())
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.Status)
			assert.Equal(t, tt.wantDB, got.DBConnected)
			assert.Equal(t, tt.wantCache, got.CacheConnected)
		})
	}
}
