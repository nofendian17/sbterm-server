package trending

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

func TestTrendingHandlerTrending(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(uc *mocks.MockTrendingUsecase)
		wantStatus  int
		wantLen     int
		wantSymbol  string
		wantErrCode string
	}{
		{
			name: "returns trending stocks",
			setup: func(uc *mocks.MockTrendingUsecase) {
				uc.EXPECT().GetTrending(gomock.Any()).Return([]domain.TrendingStock{
					{Symbol: "DSSA", Name: "Dian Swastatika Sentosa Tbk", Last: "975", Change: "+5", Percent: "0.52000", Previous: "970", LogoURL: "https://assets.stockbit.com/logos/companies/DSSA.png", Status: "STATUS_ACTIVE"},
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
			wantSymbol: "DSSA",
		},
		{
			name: "usecase error returns 500",
			setup: func(uc *mocks.MockTrendingUsecase) {
				uc.EXPECT().GetTrending(gomock.Any()).Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockTrendingUsecase(ctrl)
			tt.setup(uc)

			h := NewTrendingHandler(uc)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/v1/trending", nil)
			h.Trending(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    []struct {
					Symbol string `json:"symbol"`
					Name   string `json:"name"`
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
		})
	}
}
