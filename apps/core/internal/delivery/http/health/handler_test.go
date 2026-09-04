package health

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/mocks"
)

func TestHealthHandler_Health(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(uc *mocks.MockHealthUsecase)
		wantCode   int
		wantStatus string
	}{
		{
			name: "all healthy",
			setup: func(uc *mocks.MockHealthUsecase) {
				uc.EXPECT().GetHealth(gomock.Any()).Return(&domain.HealthStatus{
					Status:         "ok",
					DBConnected:    true,
					CacheConnected: true,
				}, nil)
			},
			wantCode:   http.StatusOK,
			wantStatus: "ok",
		},
		{
			name: "db down",
			setup: func(uc *mocks.MockHealthUsecase) {
				uc.EXPECT().GetHealth(gomock.Any()).Return(&domain.HealthStatus{
					Status:         "degraded",
					DBConnected:    false,
					CacheConnected: true,
				}, nil)
			},
			wantCode:   http.StatusServiceUnavailable,
			wantStatus: "degraded",
		},
		{
			name: "error",
			setup: func(uc *mocks.MockHealthUsecase) {
				uc.EXPECT().GetHealth(gomock.Any()).Return(nil, errors.New("failed"))
			},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			uc := mocks.NewMockHealthUsecase(ctrl)
			tt.setup(uc)

			handler := NewHealthHandler(uc)
			req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			rec := httptest.NewRecorder()
			handler.Health(rec, req)

			assert.Equal(t, tt.wantCode, rec.Code)
			if tt.wantStatus != "" {
				var resp map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
				data := resp["data"].(map[string]any)
				assert.Equal(t, tt.wantStatus, data["status"])
			}
		})
	}
}
