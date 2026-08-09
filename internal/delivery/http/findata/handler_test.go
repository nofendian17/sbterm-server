package findata

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

func TestFindataFinancialHandlerFinancial(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		setup       func(uc *mocks.MockFindataFinancialUsecase)
		wantStatus  int
		wantErrCode string
	}{
		{
			name: "returns structured financial report",
			path: "/v1/company/BRPT/financial?data_type=1&is_percentage=0&page=1&report_type=3&statement_type=2",
			setup: func(uc *mocks.MockFindataFinancialUsecase) {
				uc.EXPECT().GetFindataFinancial(gomock.Any(), "BRPT", 1, 0, 1, 3, 2).Return(&domain.FindataFinancial{
					DefaultCurrency: "IDR",
					DataTables: domain.FindataDataTables{
						Periods:  []string{"12M 2025"},
						Accounts: []domain.FindataAccount{{ID: 190, Name: "Arus Kas Dari Aktivitas Operasi"}},
					},
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "missing page returns 422",
			path:        "/v1/company/BRPT/financial?report_type=3&statement_type=2",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid statement_type returns 422",
			path:        "/v1/company/BRPT/financial?page=1&report_type=3&statement_type=99",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name: "usecase error returns 500",
			path: "/v1/company/BRPT/financial?page=1&report_type=3&statement_type=2",
			setup: func(uc *mocks.MockFindataFinancialUsecase) {
				uc.EXPECT().GetFindataFinancial(gomock.Any(), "BRPT", 0, 0, 1, 3, 2).Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockFindataFinancialUsecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			r := chi.NewRouter()
			h := NewFindataFinancialHandler(uc, validator.New())
			r.Get("/v1/company/{symbol}/financial", h.Financial)

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    struct {
					DefaultCurrency string `json:"default_currency"`
					DataTables      struct {
						Periods  []string `json:"periods"`
						Accounts []struct {
							ID   int64  `json:"id"`
							Name string `json:"name"`
						} `json:"accounts"`
					} `json:"data_tables"`
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
			assert.Equal(t, "IDR", env.Data.DefaultCurrency)
			assert.Equal(t, "12M 2025", env.Data.DataTables.Periods[0])
			require.Len(t, env.Data.DataTables.Accounts, 1)
			assert.Equal(t, int64(190), env.Data.DataTables.Accounts[0].ID)
		})
	}
}
