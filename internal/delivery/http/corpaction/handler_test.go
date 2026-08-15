package corpaction

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

func TestCorpActionHandlerCorpActions(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		setup       func(uc *mocks.MockCorpActionUsecase)
		wantStatus  int
		wantLen     int
		wantType    string
		wantErrCode string
	}{
		{
			name: "returns corp actions",
			path: "/api/v1/company/BUVA/corp-actions?limit=30",
			setup: func(uc *mocks.MockCorpActionUsecase) {
				uc.EXPECT().GetCorpActions(gomock.Any(), "BUVA", 30).Return([]domain.CompanyCorpAction{
					{ActionType: "rups", Rups: &domain.RupsInfo{RupsID: "1460868", RupsDate: "2026-06-11"}},
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
			wantType:   "rups",
		},
		{
			name:        "missing path param returns 422",
			path:        "/api/v1/company//corp-actions",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name: "usecase error returns 500",
			path: "/api/v1/company/BUVA/corp-actions",
			setup: func(uc *mocks.MockCorpActionUsecase) {
				uc.EXPECT().GetCorpActions(gomock.Any(), "BUVA", 0).Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockCorpActionUsecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			r := chi.NewRouter()
			h := NewCorpActionHandler(uc, validator.New())
			r.Get("/api/v1/company/{symbol}/corp-actions", h.CorpActions)

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    []struct {
					ActionType string `json:"action_type"`
					ActionInfo struct {
						Rups *struct {
							RupsID   string `json:"rups_id"`
							RupsDate string `json:"rups_date"`
						} `json:"rups"`
					} `json:"action_info"`
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
			assert.Equal(t, tt.wantType, env.Data[0].ActionType)
			require.NotNil(t, env.Data[0].ActionInfo.Rups)
			assert.Equal(t, "1460868", env.Data[0].ActionInfo.Rups.RupsID)
			assert.Equal(t, "2026-06-11", env.Data[0].ActionInfo.Rups.RupsDate)
		})
	}
}
