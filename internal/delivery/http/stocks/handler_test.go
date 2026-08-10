package stocks

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

func TestStocksHandlerStocks(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(uc *mocks.MockStocksUsecase)
		wantStatus  int
		wantLen     int
		wantSymbol  string
		wantErrCode string
	}{
		{
			name: "returns IHSG stocks",
			setup: func(uc *mocks.MockStocksUsecase) {
				uc.EXPECT().GetStocks(gomock.Any()).Return([]domain.Stock{
					{Symbol: "BBCA", Name: "Bank Central Asia Tbk.", Last: "6375", CompanyStatus: "STATUS_ACTIVE", IsUMA: false},
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
			wantSymbol: "BBCA",
		},
		{
			name: "usecase error returns 500",
			setup: func(uc *mocks.MockStocksUsecase) {
				uc.EXPECT().GetStocks(gomock.Any()).Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockStocksUsecase(ctrl)
			tt.setup(uc)

			h := NewStocksHandler(uc)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/v1/stocks", nil)
			h.Stocks(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    []struct {
					Symbol        string `json:"symbol"`
					CompanyStatus string `json:"company_status"`
					IsUMA         bool   `json:"is_uma"`
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
			require.Len(t, env.Data, tt.wantLen)
			assert.Equal(t, tt.wantSymbol, env.Data[0].Symbol)
			if tt.wantSymbol == "BBCA" {
				assert.Equal(t, "STATUS_ACTIVE", env.Data[0].CompanyStatus)
				assert.False(t, env.Data[0].IsUMA)
			}
		})
	}
}
