package chart

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/mocks"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

func TestChartbitHandlerChartPrice(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		setup          func(uc *mocks.MockChartbitUsecase)
		wantStatus     int
		wantClose      float64
		wantErrCode    string
		wantErrDetails map[string]string
	}{
		{
			name: "returns chart price",
			path: "/api/v1/company/DSSA/chart?timeframe=daily&from=2025-08-10&to=2026-08-10&limit=0",
			setup: func(uc *mocks.MockChartbitUsecase) {
				uc.EXPECT().GetChartPrice(gomock.Any(), "DSSA", "daily", "2025-08-10", "2026-08-10", 0).Return(&domain.ChartPriceData{
					Chartbit: []domain.ChartPrice{{Close: 985, High: 1075, Low: 975, Open: 990}},
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantClose:  985,
		},
		{
			name: "intraday returns chart price",
			path: "/api/v1/company/DSSA/chart?timeframe=intraday&from=1786230000&to=1786143600&limit=5",
			setup: func(uc *mocks.MockChartbitUsecase) {
				uc.EXPECT().GetChartPrice(gomock.Any(), "DSSA", "intraday", "1786230000", "1786143600", 5).Return(&domain.ChartPriceData{
					Chartbit: []domain.ChartPrice{{Close: 3130, Symbol: "DSSA"}},
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantClose:  3130,
		},
		{
			name:        "missing symbol returns 422",
			path:        "/api/v1/company//chart?timeframe=daily",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid timeframe returns 422",
			path:        "/api/v1/company/DSSA/chart?timeframe=hourly",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "daily missing from returns 422",
			path:        "/api/v1/company/DSSA/chart?timeframe=daily&to=2026-08-10",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
			wantErrDetails: map[string]string{
				"from": "is required",
			},
		},
		{
			name:        "daily missing to returns 422",
			path:        "/api/v1/company/DSSA/chart?timeframe=daily&from=2025-08-10",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
			wantErrDetails: map[string]string{
				"to": "is required",
			},
		},
		{
			name:        "daily missing from and to returns 422",
			path:        "/api/v1/company/DSSA/chart?timeframe=daily",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
			wantErrDetails: map[string]string{
				"from": "is required",
				"to":   "is required",
			},
		},
		{
			name:        "intraday missing from returns 422",
			path:        "/api/v1/company/DSSA/chart?timeframe=intraday&to=1786143600&limit=5",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
			wantErrDetails: map[string]string{
				"from": "is required",
			},
		},
		{
			name:        "intraday missing to returns 422",
			path:        "/api/v1/company/DSSA/chart?timeframe=intraday&from=1786230000&limit=5",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
			wantErrDetails: map[string]string{
				"to": "is required",
			},
		},
		{
			name:        "intraday missing from and to returns 422",
			path:        "/api/v1/company/DSSA/chart?timeframe=intraday&limit=5",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
			wantErrDetails: map[string]string{
				"from": "is required",
				"to":   "is required",
			},
		},
		{
			name:        "intraday missing limit returns 422",
			path:        "/api/v1/company/DSSA/chart?timeframe=intraday&from=1786230000&to=1786143600",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
			wantErrDetails: map[string]string{
				"limit": "must be at least 1 for intraday timeframe",
			},
		},
		{
			name:        "intraday limit zero returns 422",
			path:        "/api/v1/company/DSSA/chart?timeframe=intraday&from=1786230000&to=1786143600&limit=0",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
			wantErrDetails: map[string]string{
				"limit": "must be at least 1 for intraday timeframe",
			},
		},
		{
			name:        "non-numeric limit returns 422",
			path:        "/api/v1/company/DSSA/chart?timeframe=daily&from=2025-08-10&to=2026-08-10&limit=abc",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "daily negative limit returns 422",
			path:        "/api/v1/company/DSSA/chart?timeframe=daily&from=2025-08-10&to=2026-08-10&limit=-5",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
			wantErrDetails: map[string]string{
				"limit": "must be at least 1",
			},
		},
		{
			name: "usecase error returns 500",
			path: "/api/v1/company/DSSA/chart?timeframe=daily&from=2025-08-10&to=2026-08-10",
			setup: func(uc *mocks.MockChartbitUsecase) {
				uc.EXPECT().GetChartPrice(gomock.Any(), "DSSA", "daily", "2025-08-10", "2026-08-10", 0).Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockChartbitUsecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			r := chi.NewRouter()
			h := NewChartbitHandler(uc, validator.New())
			r.Get("/api/v1/company/{symbol}/chart", h.ChartPrice)

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    struct {
					Chartbit []struct {
						Close float64 `json:"close"`
					} `json:"chartbit"`
				} `json:"data"`
				Error *struct {
					Code    string            `json:"code"`
					Details map[string]string `json:"details"`
				} `json:"error"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))

			if tt.wantErrCode != "" {
				require.NotNil(t, env.Error)
				assert.Equal(t, tt.wantErrCode, env.Error.Code)
				if tt.wantErrDetails != nil {
					assert.Equal(t, tt.wantErrDetails, env.Error.Details)
				}
				return
			}
			require.Len(t, env.Data.Chartbit, 1)
			assert.Equal(t, tt.wantClose, env.Data.Chartbit[0].Close)
		})
	}
}

func TestChartTimeframeRequirements(t *testing.T) {
	tests := []struct {
		name    string
		req     chartPriceRequest
		wantNil bool
		want    map[string]string
	}{
		{
			name:    "daily with from and to is valid",
			req:     chartPriceRequest{Timeframe: "daily", From: "2025-08-10", To: "2026-08-10"},
			wantNil: true,
		},
		{
			name: "daily missing from",
			req:  chartPriceRequest{Timeframe: "daily", To: "2026-08-10"},
			want: map[string]string{"from": "is required"},
		},
		{
			name: "daily missing to",
			req:  chartPriceRequest{Timeframe: "daily", From: "2025-08-10"},
			want: map[string]string{"to": "is required"},
		},
		{
			name: "daily missing both",
			req:  chartPriceRequest{Timeframe: "daily"},
			want: map[string]string{"from": "is required", "to": "is required"},
		},
		{
			name:    "intraday with from/to and limit is valid",
			req:     chartPriceRequest{Timeframe: "intraday", From: "1786230000", To: "1786143600", Limit: 5},
			wantNil: true,
		},
		{
			name: "intraday missing from",
			req:  chartPriceRequest{Timeframe: "intraday", To: "1786143600", Limit: 5},
			want: map[string]string{"from": "is required"},
		},
		{
			name: "intraday missing to",
			req:  chartPriceRequest{Timeframe: "intraday", From: "1786230000", Limit: 5},
			want: map[string]string{"to": "is required"},
		},
		{
			name: "intraday missing from/to and limit",
			req:  chartPriceRequest{Timeframe: "intraday"},
			want: map[string]string{"from": "is required", "to": "is required", "limit": "must be at least 1 for intraday timeframe"},
		},
		{
			name: "intraday missing limit only",
			req:  chartPriceRequest{Timeframe: "intraday", From: "1786230000", To: "1786143600"},
			want: map[string]string{"limit": "must be at least 1 for intraday timeframe"},
		},
		{
			name: "intraday limit zero",
			req:  chartPriceRequest{Timeframe: "intraday", From: "1786230000", To: "1786143600", Limit: 0},
			want: map[string]string{"limit": "must be at least 1 for intraday timeframe"},
		},
		{
			name: "intraday negative limit",
			req:  chartPriceRequest{Timeframe: "intraday", From: "1786230000", To: "1786143600", Limit: -2},
			want: map[string]string{"limit": "must be at least 1 for intraday timeframe"},
		},
		{
			name: "daily negative limit",
			req:  chartPriceRequest{Timeframe: "daily", From: "2025-08-10", To: "2026-08-10", Limit: -5},
			want: map[string]string{"limit": "must be at least 1"},
		},
		{
			name:    "daily with from/to but limit ignored",
			req:     chartPriceRequest{Timeframe: "daily", From: "2025-08-10", To: "2026-08-10", Limit: 0},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := chartTimeframeRequirements(tt.req)
			if tt.wantNil {
				assert.Nil(t, got)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
