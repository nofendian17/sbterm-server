package search

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

func TestSearchHandler(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		setup       func(uc *mocks.MockSearchUsecase)
		wantStatus  int
		wantErrCode string
		wantSymbol  string
		wantItems   int
	}{
		{
			name: "returns search results with all params",
			path: "/v1/search?keyword=BBRI&page=1&type=company",
			setup: func(uc *mocks.MockSearchUsecase) {
				uc.EXPECT().GetSearch(gomock.Any(), "BBRI", 1, "company").Return(&domain.SearchResult{
					Company: []domain.SearchCompany{{ID: "59", Name: "BBRI", Desc: "Bank Rakyat Indonesia (Persero) Tbk.", Exchange: "IDX", IsTradeable: true, Type: "Saham", URL: "symbol/BBRI"}},
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantSymbol: "BBRI",
			wantItems:  1,
		},
		{
			name: "defaults type to company when omitted",
			path: "/v1/search?keyword=BBRI",
			setup: func(uc *mocks.MockSearchUsecase) {
				uc.EXPECT().GetSearch(gomock.Any(), "BBRI", 1, "company").Return(&domain.SearchResult{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "missing keyword returns 422",
			path:        "/v1/search",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid page returns 422",
			path:        "/v1/search?keyword=BBRI&page=0",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "non-numeric page returns 422",
			path:        "/v1/search?keyword=BBRI&page=abc",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid type returns 422",
			path:        "/v1/search?keyword=BBRI&type=BOGUS",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name: "upstream 400 returns 422",
			path: "/v1/search?keyword=BBRI",
			setup: func(uc *mocks.MockSearchUsecase) {
				uc.EXPECT().GetSearch(gomock.Any(), "BBRI", 1, "company").Return(nil, &domain.UpstreamError{Status: http.StatusBadRequest, Msg: "boom"})
			},
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name: "usecase error returns 500",
			path: "/v1/search?keyword=BBRI",
			setup: func(uc *mocks.MockSearchUsecase) {
				uc.EXPECT().GetSearch(gomock.Any(), "BBRI", 1, "company").Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockSearchUsecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			r := chi.NewRouter()
			h := NewSearchHandler(uc, validator.New())
			r.Get("/v1/search", h.Search)

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    struct {
					Company []map[string]any `json:"company"`
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
			if tt.wantSymbol != "" {
				require.Len(t, env.Data.Company, tt.wantItems)
				assert.Equal(t, tt.wantSymbol, env.Data.Company[0]["name"])
			}
		})
	}
}
