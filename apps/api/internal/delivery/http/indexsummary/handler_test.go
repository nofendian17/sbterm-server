package indexsummary

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

func TestIndexSummaryHandlerIndexSummary(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		setup          func(uc *mocks.MockIndexSummaryUsecase)
		wantStatus     int
		wantValue      string
		wantErrCode    string
		wantErrDetails map[string]string
	}{
		{
			name: "returns index summary",
			path: "/api/v1/index/IHSG/summary?from=2026-08-10&to=2026-08-10&interval=INTERVAL_CHART_MINUTELY",
			setup: func(uc *mocks.MockIndexSummaryUsecase) {
				uc.EXPECT().GetIndexSummary(gomock.Any(), "IHSG", "2026-08-10", "2026-08-10", "INTERVAL_CHART_MINUTELY").Return(&domain.IndexSummaryData{
					XAxisOpt: "intraday",
					Prices:   []domain.IndexSummaryPrice{{FormattedDate: "2026-08-10 09:00:00", Value: "6442.65"}},
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantValue:  "6442.65",
		},
		{
			name:        "missing symbol returns 422",
			path:        "/api/v1/index//summary?from=2026-08-10&to=2026-08-10",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
			wantErrDetails: map[string]string{
				"symbol": "is required",
			},
		},
		{
			name: "omitted range defaults to last session",
			path: "/api/v1/index/IHSG/summary?interval=INTERVAL_CHART_MINUTELY",
			setup: func(uc *mocks.MockIndexSummaryUsecase) {
				uc.EXPECT().GetIndexSummary(gomock.Any(), "IHSG", "", "", "INTERVAL_CHART_MINUTELY").Return(&domain.IndexSummaryData{
					XAxisOpt: "intraday",
					Prices:   []domain.IndexSummaryPrice{{FormattedDate: "2026-08-10 09:00:00", Value: "6442.65"}},
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantValue:  "6442.65",
		},
		{
			name:        "from without to returns 422",
			path:        "/api/v1/index/IHSG/summary?from=2026-08-10",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
			wantErrDetails: map[string]string{
				"to": "from and to must both be provided or both omitted",
			},
		},
		{
			name:        "to without from returns 422",
			path:        "/api/v1/index/IHSG/summary?to=2026-08-10",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
			wantErrDetails: map[string]string{
				"from": "from and to must both be provided or both omitted",
			},
		},
		{
			name:        "invalid from date returns 422",
			path:        "/api/v1/index/IHSG/summary?from=10-08-2026&to=2026-08-10",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
			wantErrDetails: map[string]string{
				"from": "must match datetime format 2006-01-02",
			},
		},
		{
			name:        "impossible from date returns 422",
			path:        "/api/v1/index/IHSG/summary?from=2026-13-40&to=2026-08-10",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
			wantErrDetails: map[string]string{
				"from": "must match datetime format 2006-01-02",
			},
		},
		{
			name:        "invalid to date returns 422",
			path:        "/api/v1/index/IHSG/summary?from=2026-08-10&to=2026/08/10",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
			wantErrDetails: map[string]string{
				"to": "must match datetime format 2006-01-02",
			},
		},
		{
			name:        "reversed range returns 422",
			path:        "/api/v1/index/IHSG/summary?from=2026-08-10&to=2025-08-10",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
			wantErrDetails: map[string]string{
				"from": "must be earlier than or equal to to",
			},
		},
		{
			name: "usecase error returns 500",
			path: "/api/v1/index/IHSG/summary?from=2026-08-10&to=2026-08-10",
			setup: func(uc *mocks.MockIndexSummaryUsecase) {
				uc.EXPECT().GetIndexSummary(gomock.Any(), "IHSG", "2026-08-10", "2026-08-10", "").Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockIndexSummaryUsecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			r := chi.NewRouter()
			h := NewIndexSummaryHandler(uc, validator.New())
			r.Get("/api/v1/index/{symbol}/summary", h.IndexSummary)

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    struct {
					Prices []struct {
						Value string `json:"value"`
					} `json:"prices"`
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
			require.Len(t, env.Data.Prices, 1)
			assert.Equal(t, tt.wantValue, env.Data.Prices[0].Value)
		})
	}
}

func TestIndexSummaryHandlerIndexChart(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		setup          func(uc *mocks.MockIndexSummaryUsecase)
		wantStatus     int
		wantClose      float64
		wantErrCode    string
		wantErrDetails map[string]string
	}{
		{
			name: "returns summary and chart",
			path: "/api/v1/index/IHSG/chart?from=2026-08-10&to=2026-08-10&interval=INTERVAL_CHART_MINUTELY",
			setup: func(uc *mocks.MockIndexSummaryUsecase) {
				uc.EXPECT().GetIndexChart(gomock.Any(), "IHSG", "2026-08-10", "2026-08-10", "INTERVAL_CHART_MINUTELY").Return(&domain.IndexChartData{
					Summary: domain.IndexSummaryData{
						XAxisOpt: "intraday",
						Prices:   []domain.IndexSummaryPrice{{FormattedDate: "2026-08-10 09:00:00", Value: "6442.65"}},
					},
					Chart: domain.ChartPriceData{
						Chartbit: []domain.ChartPrice{{Close: 6365.374, High: 6462.738}},
					},
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantClose:  6365.374,
		},
		{
			name: "omitted range defaults to last session",
			path: "/api/v1/index/IHSG/chart?interval=INTERVAL_CHART_MINUTELY",
			setup: func(uc *mocks.MockIndexSummaryUsecase) {
				uc.EXPECT().GetIndexChart(gomock.Any(), "IHSG", "", "", "INTERVAL_CHART_MINUTELY").Return(&domain.IndexChartData{
					Summary: domain.IndexSummaryData{Prices: []domain.IndexSummaryPrice{{Value: "6442.65"}}},
					Chart:   domain.ChartPriceData{Chartbit: []domain.ChartPrice{{Close: 6365.374}}},
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantClose:  6365.374,
		},
		{
			name:        "from without to returns 422",
			path:        "/api/v1/index/IHSG/chart?from=2026-08-10",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
			wantErrDetails: map[string]string{
				"to": "from and to must both be provided or both omitted",
			},
		},
		{
			name:        "to without from returns 422",
			path:        "/api/v1/index/IHSG/chart?to=2026-08-10",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
			wantErrDetails: map[string]string{
				"from": "from and to must both be provided or both omitted",
			},
		},
		{
			name:        "invalid from date returns 422",
			path:        "/api/v1/index/IHSG/chart?from=10-08-2026&to=2026-08-10",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
			wantErrDetails: map[string]string{
				"from": "must match datetime format 2006-01-02",
			},
		},
		{
			name:        "reversed range returns 422",
			path:        "/api/v1/index/IHSG/chart?from=2026-08-10&to=2025-08-10",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
			wantErrDetails: map[string]string{
				"from": "must be earlier than or equal to to",
			},
		},
		{
			name: "usecase error returns 500",
			path: "/api/v1/index/IHSG/chart?from=2026-08-10&to=2026-08-10",
			setup: func(uc *mocks.MockIndexSummaryUsecase) {
				uc.EXPECT().GetIndexChart(gomock.Any(), "IHSG", "2026-08-10", "2026-08-10", "").Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockIndexSummaryUsecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			r := chi.NewRouter()
			h := NewIndexSummaryHandler(uc, validator.New())
			r.Get("/api/v1/index/{symbol}/chart", h.IndexChart)

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    struct {
					Chart struct {
						Chartbit []struct {
							Close float64 `json:"close"`
						} `json:"chartbit"`
					} `json:"chart"`
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
			require.Len(t, env.Data.Chart.Chartbit, 1)
			assert.Equal(t, tt.wantClose, env.Data.Chart.Chartbit[0].Close)
		})
	}
}
