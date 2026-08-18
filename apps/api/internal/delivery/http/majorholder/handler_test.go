package majorholder

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/mocks"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

func TestMajorHolderHandlerMajorHolder(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		setup       func(uc *mocks.MockMajorHolderUsecase)
		wantStatus  int
		wantLen     int
		wantName    string
		wantErrCode string
	}{
		{
			name: "returns major holder movements",
			path: "/api/v1/insider/majorholder?symbols=DSSA&page=1&limit=20&action_type=ACTION_TYPE_BUY&source_type=SOURCE_TYPE_KSEI",
			setup: func(uc *mocks.MockMajorHolderUsecase) {
				uc.EXPECT().GetMajorHolder(gomock.Any(), "DSSA", "ACTION_TYPE_BUY", "SOURCE_TYPE_KSEI", 1, 20).Return(&domain.MajorHolderData{
					Movement: []domain.MajorHolderMovement{{Name: "DIAN SWASTATIKA SENTOSA", Symbol: "DSSA"}},
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
			wantName:   "DIAN SWASTATIKA SENTOSA",
		},
		{
			name:        "missing symbols returns 422",
			path:        "/api/v1/insider/majorholder?action_type=ACTION_TYPE_BUY",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid action_type returns 422",
			path:        "/api/v1/insider/majorholder?symbols=DSSA&action_type=BAD",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name: "usecase error returns 500",
			path: "/api/v1/insider/majorholder?symbols=DSSA",
			setup: func(uc *mocks.MockMajorHolderUsecase) {
				uc.EXPECT().GetMajorHolder(gomock.Any(), "DSSA", "", "", 0, 0).Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockMajorHolderUsecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			h := NewMajorHolderHandler(uc, validator.New())
			rec := httptest.NewRecorder()
			h.MajorHolder(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    struct {
					Movement []struct {
						Name   string `json:"name"`
						Symbol string `json:"symbol"`
					} `json:"movement"`
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
			require.Len(t, env.Data.Movement, tt.wantLen)
			assert.Equal(t, tt.wantName, env.Data.Movement[0].Name)
		})
	}
}
