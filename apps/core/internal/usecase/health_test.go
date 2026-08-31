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
		wantRedis bool
	}{
		{
			name: "all healthy",
			setup: func(repo *mocks.MockHealthRepository) {
				repo.EXPECT().Ping(gomock.Any()).Return(nil)
				repo.EXPECT().PingRedis(gomock.Any()).Return(nil)
			},
			want:      "ok",
			wantDB:    true,
			wantRedis: true,
		},
		{
			name: "db down",
			setup: func(repo *mocks.MockHealthRepository) {
				repo.EXPECT().Ping(gomock.Any()).Return(errors.New("db error"))
				repo.EXPECT().PingRedis(gomock.Any()).Return(nil)
			},
			want:      "degraded",
			wantDB:    false,
			wantRedis: true,
		},
		{
			name: "redis down",
			setup: func(repo *mocks.MockHealthRepository) {
				repo.EXPECT().Ping(gomock.Any()).Return(nil)
				repo.EXPECT().PingRedis(gomock.Any()).Return(errors.New("redis error"))
			},
			want:      "degraded",
			wantDB:    true,
			wantRedis: false,
		},
		{
			name: "both down",
			setup: func(repo *mocks.MockHealthRepository) {
				repo.EXPECT().Ping(gomock.Any()).Return(errors.New("db error"))
				repo.EXPECT().PingRedis(gomock.Any()).Return(errors.New("redis error"))
			},
			want:      "degraded",
			wantDB:    false,
			wantRedis: false,
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
			assert.Equal(t, tt.wantRedis, got.RedisConnected)
		})
	}
}
