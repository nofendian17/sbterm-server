package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/mocks"
	"github.com/nofendian17/sbterm-server/pkg/log"
)

func TestRouter(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		setup      func(uc *mocks.MockHealthUsecase)
		wantStatus int
	}{
		{
			name:   "get health returns 200",
			method: http.MethodGet,
			path:   "/health",
			setup: func(uc *mocks.MockHealthUsecase) {
				uc.EXPECT().GetHealth(gomock.Any()).Return(&domain.HealthStatus{Status: "ok", DBConnected: true}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "unknown route returns 404",
			method:     http.MethodGet,
			path:       "/unknown",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "wrong method returns 405",
			method:     http.MethodPost,
			path:       "/health",
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockHealthUsecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			logger := log.New(log.WithWriter(io.Discard))
			router := NewRouter(NewHealthHandler(uc), logger)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, nil)
			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

func TestRouterRateLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	uc := mocks.NewMockHealthUsecase(ctrl)
	uc.EXPECT().GetHealth(gomock.Any()).Return(&domain.HealthStatus{Status: "ok", DBConnected: true}, nil).AnyTimes()

	logger := log.New(log.WithWriter(io.Discard))
	router := NewRouter(NewHealthHandler(uc), logger, WithRateLimit(1, 1))

	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, httptest.NewRequest(http.MethodGet, "/health", nil))
	assert.Equal(t, http.StatusOK, rec1.Code)

	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/health", nil))
	assert.Equal(t, http.StatusTooManyRequests, rec2.Code)
	assert.Equal(t, "1", rec2.Header().Get("Retry-After"))
}
