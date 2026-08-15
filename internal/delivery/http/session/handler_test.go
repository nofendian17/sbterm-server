package session

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

func TestMarketSessionHandlerMarketSession(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(uc *mocks.MockMarketSessionUsecase)
		wantStatus   int
		wantDatetime string
		wantErrCode  string
	}{
		{
			name: "returns market session",
			setup: func(uc *mocks.MockMarketSessionUsecase) {
				uc.EXPECT().GetMarketSession(gomock.Any()).Return(&domain.MarketSession{
					Datetime: "2026-08-09 18:54:49",
					Regular:  domain.MarketSessionSegment{StateName: "STATE_NAME_MARKET_CLOSED", IsEndOfDay: true, TimeLeft: "13 jam 50 menit 11 detik"},
				}, nil)
			},
			wantStatus:   http.StatusOK,
			wantDatetime: "2026-08-09 18:54:49",
		},
		{
			name: "usecase error returns 500",
			setup: func(uc *mocks.MockMarketSessionUsecase) {
				uc.EXPECT().GetMarketSession(gomock.Any()).Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockMarketSessionUsecase(ctrl)
			tt.setup(uc)

			h := NewMarketSessionHandler(uc)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/market-session", nil)
			h.MarketSession(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    struct {
					Datetime string `json:"datetime"`
					Regular  struct {
						StateName string `json:"state_name"`
					} `json:"regular"`
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
			assert.Equal(t, tt.wantDatetime, env.Data.Datetime)
			assert.Equal(t, "STATE_NAME_MARKET_CLOSED", env.Data.Regular.StateName)
		})
	}
}
