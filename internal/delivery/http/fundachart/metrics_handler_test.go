package fundachart

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
	"github.com/nofendian17/sbterm-server/pkg/validator"
)

func TestFundaChartMetricsHandlerMetrics(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		setup       func(uc *mocks.MockFundaChartMetricsUsecase)
		wantStatus  int
		wantLen     int
		wantName    string
		wantErrCode string
	}{
		{
			name: "returns metrics",
			path: "/api/v1/fundachart/metrics?metric_name=fundachart",
			setup: func(uc *mocks.MockFundaChartMetricsUsecase) {
				uc.EXPECT().GetFundaChartMetrics(gomock.Any(), "fundachart").Return([]domain.FundaChartMetric{
					{FitemID: 18, FitemName: "Size", Child: []domain.FundaChartMetric{{FitemID: 2892, FitemName: "Market Cap"}}},
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
			wantName:   "Size",
		},
		{
			name:        "missing metric_name returns 422",
			path:        "/api/v1/fundachart/metrics",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name: "usecase error returns 500",
			path: "/api/v1/fundachart/metrics?metric_name=fundachart",
			setup: func(uc *mocks.MockFundaChartMetricsUsecase) {
				uc.EXPECT().GetFundaChartMetrics(gomock.Any(), "fundachart").Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockFundaChartMetricsUsecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			h := NewFundaChartMetricsHandler(uc, validator.New())
			rec := httptest.NewRecorder()
			h.Metrics(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    []struct {
					FitemName string `json:"fitem_name"`
					Child     []struct {
						FitemName string `json:"fitem_name"`
					} `json:"child"`
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
			assert.Equal(t, tt.wantName, env.Data[0].FitemName)
			require.Len(t, env.Data[0].Child, 1)
			assert.Equal(t, "Market Cap", env.Data[0].Child[0].FitemName)
		})
	}
}
