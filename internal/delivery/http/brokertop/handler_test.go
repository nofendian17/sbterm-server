package brokertop

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

func TestBrokerTopHandler(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		setup       func(uc *mocks.MockBrokerTopUsecase)
		wantStatus  int
		wantErrCode string
		wantIdx     string
		wantItems   int
	}{
		{
			name: "returns broker top with all params",
			path: "/api/v1/broker/top?sort=TB_SORT_BY_TOTAL_VALUE&order=ORDER_BY_DESC&period=TB_PERIOD_LAST_1_DAY&market_type=MARKET_TYPE_ALL&eod_only=true",
			setup: func(uc *mocks.MockBrokerTopUsecase) {
				uc.EXPECT().GetBrokerTop(gomock.Any(), "TB_SORT_BY_TOTAL_VALUE", "ORDER_BY_DESC", "TB_PERIOD_LAST_1_DAY", "MARKET_TYPE_ALL", true).Return(&domain.BrokerTopData{
					Date: domain.BrokerTopDate{From: "2026-08-11", To: "2026-08-11", Idx: "2026-08-11"},
					List: []domain.BrokerTopItem{{Code: "XL", Name: "Stockbit Sekuritas Digital", TotalValue: "3954882296950", Group: "BROKER_GROUP_LOCAL"}},
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantIdx:    "2026-08-11",
			wantItems:  1,
		},
		{
			name: "defaults params when omitted",
			path: "/api/v1/broker/top",
			setup: func(uc *mocks.MockBrokerTopUsecase) {
				uc.EXPECT().GetBrokerTop(gomock.Any(), "TB_SORT_BY_TOTAL_VALUE", "ORDER_BY_DESC", "TB_PERIOD_LAST_1_DAY", "MARKET_TYPE_ALL", true).Return(&domain.BrokerTopData{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "invalid sort returns 422",
			path:        "/api/v1/broker/top?sort=BOGUS",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid order returns 422",
			path:        "/api/v1/broker/top?order=BOGUS",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid period returns 422",
			path:        "/api/v1/broker/top?period=BOGUS",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid market_type returns 422",
			path:        "/api/v1/broker/top?market_type=MARKET_TYPE_REGULAR",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid eod_only returns 422",
			path:        "/api/v1/broker/top?eod_only=notabool",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name: "usecase error returns 500",
			path: "/api/v1/broker/top",
			setup: func(uc *mocks.MockBrokerTopUsecase) {
				uc.EXPECT().GetBrokerTop(gomock.Any(), "TB_SORT_BY_TOTAL_VALUE", "ORDER_BY_DESC", "TB_PERIOD_LAST_1_DAY", "MARKET_TYPE_ALL", true).Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockBrokerTopUsecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			r := chi.NewRouter()
			h := NewBrokerTopHandler(uc, validator.New())
			r.Get("/api/v1/broker/top", h.BrokerTop)

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    struct {
					Date struct {
						From string `json:"from"`
						To   string `json:"to"`
						Idx  string `json:"idx"`
					} `json:"date"`
					List []map[string]any `json:"list"`
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
			if tt.wantIdx != "" {
				assert.Equal(t, tt.wantIdx, env.Data.Date.Idx)
			}
			if tt.wantItems > 0 {
				assert.Len(t, env.Data.List, tt.wantItems)
			}
		})
	}
}
