package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/mocks"
)

func TestHealthHandlerHealth(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(uc *mocks.MockHealthUsecase)
		wantStatus    int
		wantStatusVal string
		wantDatabase  string
		wantRedis     string
		wantErrCode   string
	}{
		{
			name: "database and redis up returns 200 ok",
			setup: func(uc *mocks.MockHealthUsecase) {
				uc.EXPECT().GetHealth(gomock.Any()).Return(&domain.HealthStatus{Status: "ok", DBConnected: true, RedisConnected: true}, nil)
			},
			wantStatus:    http.StatusOK,
			wantStatusVal: "ok",
			wantDatabase:  "up",
			wantRedis:     "up",
		},
		{
			name: "database down returns 503 degraded",
			setup: func(uc *mocks.MockHealthUsecase) {
				uc.EXPECT().GetHealth(gomock.Any()).Return(&domain.HealthStatus{Status: "degraded", DBConnected: false, RedisConnected: true}, nil)
			},
			wantStatus:    http.StatusServiceUnavailable,
			wantStatusVal: "degraded",
			wantDatabase:  "down",
			wantRedis:     "up",
		},
		{
			name: "redis down returns 503 degraded",
			setup: func(uc *mocks.MockHealthUsecase) {
				uc.EXPECT().GetHealth(gomock.Any()).Return(&domain.HealthStatus{Status: "degraded", DBConnected: true, RedisConnected: false}, nil)
			},
			wantStatus:    http.StatusServiceUnavailable,
			wantStatusVal: "degraded",
			wantDatabase:  "up",
			wantRedis:     "down",
		},
		{
			name: "usecase error returns 500",
			setup: func(uc *mocks.MockHealthUsecase) {
				uc.EXPECT().GetHealth(gomock.Any()).Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
		{
			name: "both dependencies down returns 503 degraded",
			setup: func(uc *mocks.MockHealthUsecase) {
				uc.EXPECT().GetHealth(gomock.Any()).Return(&domain.HealthStatus{Status: "degraded", DBConnected: false, RedisConnected: false}, nil)
			},
			wantStatus:    http.StatusServiceUnavailable,
			wantStatusVal: "degraded",
			wantDatabase:  "down",
			wantRedis:     "down",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockHealthUsecase(ctrl)
			tt.setup(uc)

			h := NewHealthHandler(uc)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			h.Health(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    struct {
					Status   string `json:"status"`
					Database string `json:"database"`
					Redis    string `json:"redis"`
				} `json:"data"`
				Error *struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))

			if tt.wantStatusVal != "" {
				assert.Equal(t, tt.wantStatusVal, env.Data.Status)
			}
			if tt.wantDatabase != "" {
				assert.Equal(t, tt.wantDatabase, env.Data.Database)
			}
			if tt.wantRedis != "" {
				assert.Equal(t, tt.wantRedis, env.Data.Redis)
			}
			if tt.wantErrCode != "" {
				require.NotNil(t, env.Error)
				assert.Equal(t, tt.wantErrCode, env.Error.Code)
			}
		})
	}
}

func TestDBStatus(t *testing.T) {
	tests := []struct {
		name      string
		connected bool
		want      string
	}{
		{name: "connected", connected: true, want: "up"},
		{name: "disconnected", connected: false, want: "down"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, dbStatus(tt.connected))
		})
	}
}
