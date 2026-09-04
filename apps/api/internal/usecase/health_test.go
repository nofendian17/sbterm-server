package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm/apps/api/internal/mocks"
)

func TestHealthUsecaseGetHealth(t *testing.T) {
	tests := []struct {
		name               string
		pingCacheErr       error
		wantStatus         string
		wantCacheConnected bool
	}{
		{
			name:               "cache connected",
			wantStatus:         statusOK,
			wantCacheConnected: true,
		},
		{
			name:               "cache unavailable",
			pingCacheErr:       errors.New("connection refused"),
			wantStatus:         statusDegraded,
			wantCacheConnected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := mocks.NewMockHealthRepository(ctrl)
			repo.EXPECT().PingCache(gomock.Any()).Return(tt.pingCacheErr)

			uc := NewHealthUsecase(repo)
			status, err := uc.GetHealth(context.Background())
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, status.Status)
			assert.Equal(t, tt.wantCacheConnected, status.CacheConnected)
		})
	}
}
