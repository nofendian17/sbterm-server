package marketdetector

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

func TestMarketDetectorHandlerMarketDetector(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		setup       func(uc *mocks.MockMarketDetectorUsecase)
		wantStatus  int
		wantSymbol  string
		wantAccdist string
		wantErrCode string
	}{
		{
			name: "returns market detector data with defaults",
			path: "/api/v1/market-detector/BRPT?from=2026-08-03&to=2026-08-10",
			setup: func(uc *mocks.MockMarketDetectorUsecase) {
				uc.EXPECT().GetMarketDetector(gomock.Any(), "BRPT", "2026-08-03", "2026-08-10", "TRANSACTION_TYPE_NET", "MARKET_BOARD_REGULER", "INVESTOR_TYPE_ALL", 0).Return(&domain.MarketDetectorData{
					BandarDetector: domain.BandarDetector{BrokerAccdist: "Dist", Top1: domain.BandarAccdist{Accdist: "Normal Acc", Amount: 13157295000}},
					BrokerSummary:  domain.BrokerSummary{Symbol: "BRPT"},
				}, nil)
			},
			wantStatus:  http.StatusOK,
			wantSymbol:  "BRPT",
			wantAccdist: "Dist",
		},
		{
			name: "passes explicit filter params",
			path: "/api/v1/market-detector/BRPT?from=2026-08-03&to=2026-08-10&transaction_type=TRANSACTION_TYPE_GROSS&market_board=MARKET_BOARD_TUNAI&investor_type=INVESTOR_TYPE_FOREIGN&limit=10",
			setup: func(uc *mocks.MockMarketDetectorUsecase) {
				uc.EXPECT().GetMarketDetector(gomock.Any(), "BRPT", "2026-08-03", "2026-08-10", "TRANSACTION_TYPE_GROSS", "MARKET_BOARD_TUNAI", "INVESTOR_TYPE_FOREIGN", 10).Return(&domain.MarketDetectorData{}, nil)
			},
			wantStatus: http.StatusOK,
			wantSymbol: "BRPT",
		},
		{
			name:        "missing from returns 422",
			path:        "/api/v1/market-detector/BRPT?to=2026-08-10",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid from date returns 422",
			path:        "/api/v1/market-detector/BRPT?from=not-a-date&to=2026-08-10",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "non-numeric limit returns 422",
			path:        "/api/v1/market-detector/BRPT?from=2026-08-03&to=2026-08-10&limit=abc",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "negative limit returns 422",
			path:        "/api/v1/market-detector/BRPT?from=2026-08-03&to=2026-08-10&limit=-5",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid transaction_type returns 422",
			path:        "/api/v1/market-detector/BRPT?from=2026-08-03&to=2026-08-10&transaction_type=BAD",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid market_board returns 422",
			path:        "/api/v1/market-detector/BRPT?from=2026-08-03&to=2026-08-10&market_board=BAD",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid investor_type returns 422",
			path:        "/api/v1/market-detector/BRPT?from=2026-08-03&to=2026-08-10&investor_type=BAD",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name: "usecase error returns 500",
			path: "/api/v1/market-detector/BRPT?from=2026-08-03&to=2026-08-10",
			setup: func(uc *mocks.MockMarketDetectorUsecase) {
				uc.EXPECT().GetMarketDetector(gomock.Any(), "BRPT", "2026-08-03", "2026-08-10", "TRANSACTION_TYPE_NET", "MARKET_BOARD_REGULER", "INVESTOR_TYPE_ALL", 0).Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
		{
			name: "upstream 400 maps to 422",
			path: "/api/v1/market-detector/BRPT?from=2026-08-03&to=2026-08-10",
			setup: func(uc *mocks.MockMarketDetectorUsecase) {
				uc.EXPECT().GetMarketDetector(gomock.Any(), "BRPT", "2026-08-03", "2026-08-10", "TRANSACTION_TYPE_NET", "MARKET_BOARD_REGULER", "INVESTOR_TYPE_ALL", 0).Return(nil, &domain.UpstreamError{Status: http.StatusBadRequest})
			},
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockMarketDetectorUsecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			r := chi.NewRouter()
			h := NewMarketDetectorHandler(uc, validator.New())
			r.Get("/api/v1/market-detector/{symbol}", h.MarketDetector)

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    struct {
					Symbol         string `json:"symbol"`
					BandarDetector struct {
						BrokerAccdist string `json:"broker_accdist"`
					} `json:"bandar_detector"`
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
			assert.Equal(t, tt.wantSymbol, env.Data.Symbol)
			if tt.wantAccdist != "" {
				assert.Equal(t, tt.wantAccdist, env.Data.BandarDetector.BrokerAccdist)
			}
		})
	}
}
