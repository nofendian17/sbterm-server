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
		name          string
		pingErr       error
		wantConnected bool
	}{
		{
			name:          "database connected",
			pingErr:       nil,
			wantConnected: true,
		},
		{
			name:          "database unavailable",
			pingErr:       errors.New("connection refused"),
			wantConnected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := mocks.NewMockHealthRepository(ctrl)
			repo.EXPECT().Ping(gomock.Any()).Return(tt.pingErr)

			uc := NewHealthUsecase(repo)
			status, err := uc.GetHealth(context.Background())
			require.NoError(t, err)
			assert.Equal(t, statusOK, status.Status)
			assert.Equal(t, tt.wantConnected, status.DBConnected)
		})
	}
}
