package keystats

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

func TestKeystatsHandlerKeystats(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		setup       func(uc *mocks.MockKeystatsUsecase)
		wantStatus  int
		wantCap     string
		wantErrCode string
	}{
		{
			name: "returns keystats",
			path: "/v1/company/BUVA/keystats?year_limit=10",
			setup: func(uc *mocks.MockKeystatsUsecase) {
				uc.EXPECT().GetKeystats(gomock.Any(), "BUVA", 10).Return(&domain.Keystats{
					Stats:                   domain.KeystatsStats{MarketCap: "19,324 B", CurrentShareOutstanding: "24.62 B"},
					FinancialReportCurrency: []string{"IDR"},
					ClosureFinItemsResults: []domain.KeystatsFinGroup{{
						KeystatsName:   "Current Valuation",
						FinNameResults: []domain.KeystatsItem{{Fitem: domain.KeystatsFitem{Name: "Current PE Ratio (Annualised)", Value: "1,187.45"}}},
					}},
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantCap:    "19,324 B",
		},
		{
			name:        "missing path param returns 422",
			path:        "/v1/company//keystats",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name: "usecase error returns 500",
			path: "/v1/company/BUVA/keystats",
			setup: func(uc *mocks.MockKeystatsUsecase) {
				uc.EXPECT().GetKeystats(gomock.Any(), "BUVA", 0).Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockKeystatsUsecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			r := chi.NewRouter()
			h := NewKeystatsHandler(uc, validator.New())
			r.Get("/v1/company/{symbol}/keystats", h.Keystats)

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    struct {
					Stats struct {
						MarketCap string `json:"market_cap"`
					} `json:"stats"`
					ClosureFinItemsResults []struct {
						KeystatsName   string `json:"keystats_name"`
						FinNameResults []struct {
							Fitem struct {
								Value string `json:"value"`
							} `json:"fitem"`
						} `json:"fin_name_results"`
					} `json:"closure_fin_items_results"`
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
			assert.Equal(t, tt.wantCap, env.Data.Stats.MarketCap)
			require.Len(t, env.Data.ClosureFinItemsResults, 1)
			assert.Equal(t, "Current Valuation", env.Data.ClosureFinItemsResults[0].KeystatsName)
			assert.Equal(t, "1,187.45", env.Data.ClosureFinItemsResults[0].FinNameResults[0].Fitem.Value)
		})
	}
}
