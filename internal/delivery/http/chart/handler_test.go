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

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/mocks"
	"github.com/nofendian17/sbterm-server/pkg/validator"
)

func TestChartbitHandlerChartPrice(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		setup       func(uc *mocks.MockChartbitUsecase)
		wantStatus  int
		wantClose   float64
		wantErrCode string
	}{
		{
			name: "returns chart price",
			path: "/v1/company/DSSA/chart?timeframe=daily&from=2025-08-10&to=2026-08-10&limit=0",
			setup: func(uc *mocks.MockChartbitUsecase) {
				uc.EXPECT().GetChartPrice(gomock.Any(), "DSSA", "daily", "2025-08-10", "2026-08-10", 0).Return(&domain.ChartPriceData{
					Chartbit: []domain.ChartPrice{{Close: 985, High: 1075, Low: 975, Open: 990}},
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantClose:  985,
		},
		{
			name:        "missing symbol returns 422",
			path:        "/v1/company//chart?timeframe=daily",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid timeframe returns 422",
			path:        "/v1/company/DSSA/chart?timeframe=hourly",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name: "usecase error returns 500",
			path: "/v1/company/DSSA/chart?timeframe=daily",
			setup: func(uc *mocks.MockChartbitUsecase) {
				uc.EXPECT().GetChartPrice(gomock.Any(), "DSSA", "daily", "", "", 0).Return(nil, errors.New("boom"))
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
			r.Get("/v1/company/{symbol}/chart", h.ChartPrice)

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
					Code string `json:"code"`
				} `json:"error"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))

			if tt.wantErrCode != "" {
				require.NotNil(t, env.Error)
				assert.Equal(t, tt.wantErrCode, env.Error.Code)
				return
			}
			require.Len(t, env.Data.Chartbit, 1)
			assert.Equal(t, tt.wantClose, env.Data.Chartbit[0].Close)
		})
	}
}
