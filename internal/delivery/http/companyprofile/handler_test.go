package companyprofile

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

func TestCompanyProfileHandlerCompanyProfile(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		setup       func(uc *mocks.MockCompanyProfileUsecase)
		wantStatus  int
		wantSymbol  string
		wantErrCode string
	}{
		{
			name: "returns company profile",
			path: "/v1/company/DSSA/profile",
			setup: func(uc *mocks.MockCompanyProfileUsecase) {
				uc.EXPECT().GetProfile(gomock.Any(), "DSSA").Return(&domain.CompanyProfile{
					Background:         "PT Dian Swastatika",
					History:            &domain.ProfileHistory{Date: "10 Dec 2009"},
					Shareholder:        []domain.ProfileShareholder{{Percentage: "59.9%", Name: "PT SINAR MAS TUNGGAL", Value: "115.39 B", Badges: []string{"pengendali"}}},
					ShareholderNumbers: []domain.ProfileShareholderNumber{{ShareholderDate: "30 Jun 2026", TotalShare: "86,926", Change: 48123, ChangeFormatted: "(+48,123)"}},
					ShareholderOnePercent: []domain.ProfileShareholder{{ID: "1000004641", Percentage: "59.90%", Name: "SINAR MAS TUNGGAL", Value: "115,388,080,000", Type: "CP", Location: "Local", Nationality: "-", Domicile: "INDONESIA", Scripless: "0", Scrip: "115,388,080,000", ValueFormatted: "115.39 B", Classification: "Corporate"}},
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantSymbol: "DSSA",
		},
		{
			name:        "missing path param returns 422",
			path:        "/v1/company//profile",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name: "usecase error returns 500",
			path: "/v1/company/DSSA/profile",
			setup: func(uc *mocks.MockCompanyProfileUsecase) {
				uc.EXPECT().GetProfile(gomock.Any(), "DSSA").Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockCompanyProfileUsecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			r := chi.NewRouter()
			h := NewCompanyProfileHandler(uc, validator.New())
			r.Get("/v1/company/{symbol}/profile", h.CompanyProfile)

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			assert.Equal(t, tt.wantStatus, rec.Code)

var env struct {
				Success bool `json:"success"`
				Data    struct {
					Background string `json:"background"`
					Shareholder []struct {
						Name   string   `json:"name"`
						Badges []string `json:"badges"`
					} `json:"shareholder"`
					ShareholderOnePercent []struct {
						ID             string `json:"id"`
						Name           string `json:"name"`
						Type           string `json:"type"`
						Location       string `json:"location"`
						Nationality    string `json:"nationality"`
						Domicile       string `json:"domicile"`
						ValueFormatted string `json:"value_formatted"`
						Classification string `json:"classification"`
					} `json:"shareholder_one_percent"`
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
			assert.Equal(t, "PT Dian Swastatika", env.Data.Background)
			require.Len(t, env.Data.Shareholder, 1)
			assert.Equal(t, "PT SINAR MAS TUNGGAL", env.Data.Shareholder[0].Name)
			assert.Equal(t, "pengendali", env.Data.Shareholder[0].Badges[0])
			require.Len(t, env.Data.ShareholderOnePercent, 1)
			assert.Equal(t, "1000004641", env.Data.ShareholderOnePercent[0].ID)
			assert.Equal(t, "CP", env.Data.ShareholderOnePercent[0].Type)
			assert.Equal(t, "INDONESIA", env.Data.ShareholderOnePercent[0].Domicile)
			assert.Equal(t, "115.39 B", env.Data.ShareholderOnePercent[0].ValueFormatted)
			assert.Equal(t, "Corporate", env.Data.ShareholderOnePercent[0].Classification)
		})
	}
}
