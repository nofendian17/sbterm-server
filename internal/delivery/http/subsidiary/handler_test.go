package subsidiary

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

func TestSubsidiaryHandlerSubsidiaries(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		setup       func(uc *mocks.MockSubsidiaryUsecase)
		wantStatus  int
		wantLen     int
		wantName    string
		wantErrCode string
	}{
		{
			name: "returns subsidiaries",
			path: "/api/v1/company/DSSA/subsidiaries",
			setup: func(uc *mocks.MockSubsidiaryUsecase) {
				uc.EXPECT().GetSubsidiaries(gomock.Any(), "DSSA").Return(&domain.SubsidiaryData{
					Currency:          "CURRENCY_USD",
					LastUpdatedPeriod: "Q1 2026",
					Unit:              "UNIT_FULL",
					Subsidiaries:      []domain.Subsidiary{{CompanyName: "PT DSST Mas Gemilang", Percentage: "99.99"}},
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
			wantName:   "PT DSST Mas Gemilang",
		},
		{
			name:        "missing path param returns 422",
			path:        "/api/v1/company//subsidiaries",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name: "usecase error returns 500",
			path: "/api/v1/company/DSSA/subsidiaries",
			setup: func(uc *mocks.MockSubsidiaryUsecase) {
				uc.EXPECT().GetSubsidiaries(gomock.Any(), "DSSA").Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockSubsidiaryUsecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			r := chi.NewRouter()
			h := NewSubsidiaryHandler(uc, validator.New())
			r.Get("/api/v1/company/{symbol}/subsidiaries", h.Subsidiaries)

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    struct {
					Currency          string `json:"currency"`
					LastUpdatedPeriod string `json:"last_updated_period"`
					Unit              string `json:"unit"`
					Subsidiaries      []struct {
						CompanyName string `json:"company_name"`
					} `json:"subsidiaries"`
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
			assert.Equal(t, "CURRENCY_USD", env.Data.Currency)
			assert.Equal(t, "Q1 2026", env.Data.LastUpdatedPeriod)
			assert.Equal(t, "UNIT_FULL", env.Data.Unit)
			require.Len(t, env.Data.Subsidiaries, tt.wantLen)
			assert.Equal(t, tt.wantName, env.Data.Subsidiaries[0].CompanyName)
		})
	}
}
