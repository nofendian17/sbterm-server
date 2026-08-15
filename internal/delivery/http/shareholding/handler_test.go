package shareholding

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

func TestShareholdingHandlerShareholdingComposition(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		setup       func(uc *mocks.MockShareholdingCompositionUsecase)
		wantStatus  int
		wantLen     int
		wantLabel   string
		wantErrCode string
	}{
		{
			name: "returns composition periods",
			path: "/api/v1/company/DSSA/shareholding-composition?period_start=2026-06-01&period_end=2026-06-30",
			setup: func(uc *mocks.MockShareholdingCompositionUsecase) {
				uc.EXPECT().GetShareholdingComposition(gomock.Any(), "DSSA", "2026-06-01", "2026-06-30").Return([]domain.ShareholdingCompositionPeriod{{
					ReportDate:   "2026-06-30",
					Compositions: []domain.ShareholdingComposition{{Label: "SINAR MAS TUNGGAL"}},
				}}, nil)
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
			wantLabel:  "SINAR MAS TUNGGAL",
		},
		{
			name:        "missing path param returns 422",
			path:        "/api/v1/company//shareholding-composition",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name: "usecase error returns 500",
			path: "/api/v1/company/DSSA/shareholding-composition",
			setup: func(uc *mocks.MockShareholdingCompositionUsecase) {
				uc.EXPECT().GetShareholdingComposition(gomock.Any(), "DSSA", "", "").Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockShareholdingCompositionUsecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			r := chi.NewRouter()
			h := NewShareholdingHandler(uc, validator.New())
			r.Get("/api/v1/company/{symbol}/shareholding-composition", h.ShareholdingComposition)

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    []struct {
					ReportDate   string `json:"report_date"`
					Compositions []struct {
						Label string `json:"label"`
					} `json:"compositions"`
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
			assert.Equal(t, "2026-06-30", env.Data[0].ReportDate)
			require.Len(t, env.Data[0].Compositions, 1)
			assert.Equal(t, tt.wantLabel, env.Data[0].Compositions[0].Label)
		})
	}
}
