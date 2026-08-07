package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm-server/internal/mocks"
)

func TestHealthUsecaseGetHealth(t *testing.T) {
	tests := []struct {
		name               string
		pingErr            error
		pingRedisErr       error
		wantDBConnected    bool
		wantRedisConnected bool
	}{
		{
			name:               "database and redis connected",
			wantDBConnected:    true,
			wantRedisConnected: true,
		},
		{
			name:               "database unavailable",
			pingErr:            errors.New("connection refused"),
			wantDBConnected:    false,
			wantRedisConnected: true,
		},
		{
			name:               "redis unavailable",
			pingRedisErr:       errors.New("connection refused"),
			wantDBConnected:    true,
			wantRedisConnected: false,
		},
		{
			name:               "database and redis unavailable",
			pingErr:            errors.New("db down"),
			pingRedisErr:       errors.New("redis down"),
			wantDBConnected:    false,
			wantRedisConnected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := mocks.NewMockHealthRepository(ctrl)
			repo.EXPECT().Ping(gomock.Any()).Return(tt.pingErr)
			repo.EXPECT().PingRedis(gomock.Any()).Return(tt.pingRedisErr)

			uc := NewHealthUsecase(repo)
			status, err := uc.GetHealth(context.Background())
			require.NoError(t, err)
			assert.Equal(t, statusOK, status.Status)
			assert.Equal(t, tt.wantDBConnected, status.DBConnected)
			assert.Equal(t, tt.wantRedisConnected, status.RedisConnected)
		})
	}
}
