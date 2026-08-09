package sectors

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/mocks"
)

func TestSectorsHandlerSectors(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(uc *mocks.MockSectorsUsecase)
		wantStatus  int
		wantLen     int
		wantSymbol  string
		wantCompany string
		wantErrCode string
	}{
		{
			name: "returns sectors with nested companies",
			setup: func(uc *mocks.MockSectorsUsecase) {
				uc.EXPECT().GetSectors(gomock.Any()).Return([]domain.Sector{
					{Symbol: "IDXCYCLIC", ID: "1000003293", Last: 959.874, Change: "-6.62", Percent: -0.69,
						Companies: []domain.SubsectorCompany{{Symbol: "MNCN", Name: "Media Nusantara Citra Tbk.", Last: "3160"}}},
				}, nil)
			},
			wantStatus:  http.StatusOK,
			wantLen:     1,
			wantSymbol:  "IDXCYCLIC",
			wantCompany: "MNCN",
		},
		{
			name: "usecase error returns 500",
			setup: func(uc *mocks.MockSectorsUsecase) {
				uc.EXPECT().GetSectors(gomock.Any()).Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockSectorsUsecase(ctrl)
			tt.setup(uc)

			h := NewSectorsHandler(uc)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/v1/sectors", nil)
			h.Sectors(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    []struct {
					Symbol    string `json:"symbol"`
					Companies []struct {
						Symbol string `json:"symbol"`
					} `json:"companies"`
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
			assert.Equal(t, tt.wantSymbol, env.Data[0].Symbol)
			require.Len(t, env.Data[0].Companies, 1)
			assert.Equal(t, tt.wantCompany, env.Data[0].Companies[0].Symbol)
		})
	}
}
