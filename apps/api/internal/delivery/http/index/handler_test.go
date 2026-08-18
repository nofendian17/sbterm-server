package index

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
)

func TestIndexHandlerIndex(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(uc *mocks.MockIndexUsecase)
		wantStatus  int
		wantMainLen int
		wantSymbol  string
		wantErrCode string
	}{
		{
			name: "returns indexes",
			setup: func(uc *mocks.MockIndexUsecase) {
				uc.EXPECT().GetIndexes(gomock.Any()).Return(&domain.Indexes{
					Main: []domain.Index{{Symbol: "IDX30", Name: "IDX30", Last: "359.40"}},
					All:  []domain.Index{{Symbol: "ABX", Name: "Papan Akselerasi"}},
				}, nil)
			},
			wantStatus:  http.StatusOK,
			wantMainLen: 1,
			wantSymbol:  "IDX30",
		},
		{
			name: "usecase error returns 500",
			setup: func(uc *mocks.MockIndexUsecase) {
				uc.EXPECT().GetIndexes(gomock.Any()).Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockIndexUsecase(ctrl)
			tt.setup(uc)

			h := NewIndexHandler(uc)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/indexes", nil)
			h.Index(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    struct {
					Main []struct {
						Symbol string `json:"symbol"`
					} `json:"main"`
					All []struct {
						Symbol string `json:"symbol"`
					} `json:"all"`
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
			require.Len(t, env.Data.Main, tt.wantMainLen)
			assert.Equal(t, tt.wantSymbol, env.Data.Main[0].Symbol)
			require.Len(t, env.Data.All, 1)
			assert.Equal(t, "ABX", env.Data.All[0].Symbol)
		})
	}
}
