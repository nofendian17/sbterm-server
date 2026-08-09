package priceperformance

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

func TestPricePerformanceHandlerPricePerformance(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		setup       func(uc *mocks.MockPricePerformanceUsecase)
		wantStatus  int
		wantLen     int
		wantFrame   string
		wantErrCode string
	}{
		{
			name: "returns price performance",
			path: "/v1/company/BUVA/price-performance",
			setup: func(uc *mocks.MockPricePerformanceUsecase) {
				uc.EXPECT().GetPricePerformance(gomock.Any(), "BUVA").Return(&domain.PricePerformanceData{
					Prices: []domain.PricePerformance{{Timeframe: "1D", Close: domain.PriceRawFormatted{Raw: 785, Formatted: "785"}}},
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
			wantFrame:  "1D",
		},
		{
			name:        "missing path param returns 422",
			path:        "/v1/company//price-performance",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name: "usecase error returns 500",
			path: "/v1/company/BUVA/price-performance",
			setup: func(uc *mocks.MockPricePerformanceUsecase) {
				uc.EXPECT().GetPricePerformance(gomock.Any(), "BUVA").Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockPricePerformanceUsecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			r := chi.NewRouter()
			h := NewPricePerformanceHandler(uc, validator.New())
			r.Get("/v1/company/{symbol}/price-performance", h.PricePerformance)

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    struct {
					Prices []struct {
						Timeframe string `json:"timeframe"`
						Close     struct {
							Raw float64 `json:"raw"`
						} `json:"close"`
					} `json:"prices"`
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
			require.Len(t, env.Data.Prices, tt.wantLen)
			assert.Equal(t, tt.wantFrame, env.Data.Prices[0].Timeframe)
			assert.Equal(t, float64(785), env.Data.Prices[0].Close.Raw)
		})
	}
}
