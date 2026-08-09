package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm-server/internal/delivery/http/health"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/index"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/mover"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/sectors"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/session"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/trending"
	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/mocks"
	"github.com/nofendian17/sbterm-server/pkg/log"
	"github.com/nofendian17/sbterm-server/pkg/validator"
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
				uc.EXPECT().GetHealth(gomock.Any()).Return(&domain.HealthStatus{Status: "ok", DBConnected: true, RedisConnected: true}, nil)
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
			handler := health.NewHealthHandler(uc)
			trendingHandler := trending.NewTrendingHandler(mocks.NewMockTrendingUsecase(ctrl))
			moverHandler := mover.NewMarketMoverHandler(mocks.NewMockMarketMoverUsecase(ctrl), validator.New())
			sessionHandler := session.NewMarketSessionHandler(mocks.NewMockMarketSessionUsecase(ctrl))
			indexHandler := index.NewIndexHandler(mocks.NewMockIndexUsecase(ctrl))
			sectorsHandler := sectors.NewSectorsHandler(mocks.NewMockSectorsUsecase(ctrl))
			router := NewRouter(handler, trendingHandler, moverHandler, sessionHandler, indexHandler, sectorsHandler, logger)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, nil)
			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

func TestRouterRateLimit(t *testing.T) {
	tests := []struct {
		name      string
		wantCodes []int
	}{
		{
			name:      "requests beyond burst are rejected",
			wantCodes: []int{http.StatusOK, http.StatusTooManyRequests},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockHealthUsecase(ctrl)
			uc.EXPECT().GetHealth(gomock.Any()).Return(&domain.HealthStatus{Status: "ok", DBConnected: true, RedisConnected: true}, nil).AnyTimes()

			logger := log.New(log.WithWriter(io.Discard))
			router := NewRouter(health.NewHealthHandler(uc), trending.NewTrendingHandler(mocks.NewMockTrendingUsecase(ctrl)), mover.NewMarketMoverHandler(mocks.NewMockMarketMoverUsecase(ctrl), validator.New()), session.NewMarketSessionHandler(mocks.NewMockMarketSessionUsecase(ctrl)), index.NewIndexHandler(mocks.NewMockIndexUsecase(ctrl)), sectors.NewSectorsHandler(mocks.NewMockSectorsUsecase(ctrl)), logger, WithRateLimit(1, 1))

			for _, want := range tt.wantCodes {
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
				assert.Equal(t, want, rec.Code)
				if want == http.StatusTooManyRequests {
					assert.Equal(t, "1", rec.Header().Get("Retry-After"))
				}
			}
		})
	}
}
