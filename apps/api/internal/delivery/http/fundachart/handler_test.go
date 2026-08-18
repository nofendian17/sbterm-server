package fundachart

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

func TestFundaChartHandlerFundaChart(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		setup       func(uc *mocks.MockFundaChartUsecase)
		wantStatus  int
		wantLen     int
		wantItem    string
		wantErrCode string
	}{
		{
			name: "returns funda chart",
			path: "/api/v1/company/BUVA/fundachart?item=12148&timeframe=10y",
			setup: func(uc *mocks.MockFundaChartUsecase) {
				uc.EXPECT().GetFundaChart(gomock.Any(), "BUVA", "12148", "10y").Return([]domain.FundaChartCompany{
					{CompanyName: "BUVA", Ratios: []domain.FundaChartRatio{{ItemName: "Current PE Ratio (Annualised)"}}},
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
			wantItem:   "Current PE Ratio (Annualised)",
		},
		{
			name:        "missing item returns 422",
			path:        "/api/v1/company/BUVA/fundachart",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name: "usecase error returns 500",
			path: "/api/v1/company/BUVA/fundachart?item=12148",
			setup: func(uc *mocks.MockFundaChartUsecase) {
				uc.EXPECT().GetFundaChart(gomock.Any(), "BUVA", "12148", "10y").Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockFundaChartUsecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			r := chi.NewRouter()
			h := NewFundaChartHandler(uc, validator.New())
			r.Get("/api/v1/company/{symbol}/fundachart", h.FundaChart)

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    []struct {
					CompanyName string `json:"company_name"`
					Ratios      []struct {
						ItemName string `json:"item_name"`
					} `json:"ratios"`
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
			assert.Equal(t, "BUVA", env.Data[0].CompanyName)
			require.Len(t, env.Data[0].Ratios, 1)
			assert.Equal(t, tt.wantItem, env.Data[0].Ratios[0].ItemName)
		})
	}
}
