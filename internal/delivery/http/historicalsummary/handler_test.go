package historicalsummary

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

func TestHistoricalSummaryHandler(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		setup       func(uc *mocks.MockHistoricalSummaryUsecase)
		wantStatus  int
		wantErrCode string
		wantItems   int
		wantNext    string
	}{
		{
			name: "returns historical summary with all params",
			path: "/api/v1/company/DSSA/historical-summary?period=HS_PERIOD_WEEKLY&start_date=2025-08-11&end_date=2026-08-11&limit=12&page=1",
			setup: func(uc *mocks.MockHistoricalSummaryUsecase) {
				uc.EXPECT().GetHistoricalSummary(gomock.Any(), "DSSA", "HS_PERIOD_WEEKLY", "2025-08-11", "2026-08-11", 12, 1).Return(&domain.HistoricalSummaryData{
					Result:   []domain.HistoricalSummaryItem{{Date: "2026-08-10", Close: 945}},
					Paginate: domain.HistoricalSummaryPaginate{NextPage: "2"},
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantItems:  1,
			wantNext:   "2",
		},
		{
			name: "defaults period/limit/page when omitted",
			path: "/api/v1/company/DSSA/historical-summary",
			setup: func(uc *mocks.MockHistoricalSummaryUsecase) {
				uc.EXPECT().GetHistoricalSummary(gomock.Any(), "DSSA", "HS_PERIOD_DAILY", "", "", 50, 1).Return(&domain.HistoricalSummaryData{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "missing symbol returns 422",
			path:        "/api/v1/company//historical-summary",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid period returns 422",
			path:        "/api/v1/company/DSSA/historical-summary?period=HS_PERIOD_YEARLY",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid start_date format returns 422",
			path:        "/api/v1/company/DSSA/historical-summary?start_date=11-08-2025",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name: "usecase error returns 500",
			path: "/api/v1/company/DSSA/historical-summary",
			setup: func(uc *mocks.MockHistoricalSummaryUsecase) {
				uc.EXPECT().GetHistoricalSummary(gomock.Any(), "DSSA", "HS_PERIOD_DAILY", "", "", 50, 1).Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockHistoricalSummaryUsecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			r := chi.NewRouter()
			h := NewHistoricalSummaryHandler(uc, validator.New())
			r.Get("/api/v1/company/{symbol}/historical-summary", h.HistoricalSummary)

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    struct {
					Result   []map[string]any `json:"result"`
					Paginate struct {
						NextPage string `json:"next_page"`
					} `json:"paginate"`
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
			if tt.wantItems > 0 {
				assert.Len(t, env.Data.Result, tt.wantItems)
				assert.Equal(t, tt.wantNext, env.Data.Paginate.NextPage)
			}
		})
	}
}
