package topstock

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

func TestTopStockHandlerTopStock(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		setup         func(uc *mocks.MockTopStockUsecase)
		wantStatus    int
		wantCode      string
		wantValueType string
		wantErrCode   string
		wantErrField  string
		wantErrMsg    string
	}{
		{
			name: "returns top stock data with defaults",
			path: "/api/v1/top-stock?start=2026-08-09&end=2026-08-10",
			setup: func(uc *mocks.MockTopStockUsecase) {
				uc.EXPECT().GetTopStock(gomock.Any(), "2026-08-09", "2026-08-10", "INVESTOR_TYPE_ALL", "MARKET_TYPE_ALL", "VALUE_TYPE_NET", 0).Return(&domain.TopStockData{
					TopBuy:       []domain.TopStockItem{{Rank: 1, Code: "DSSA", Value: domain.RawFormatted{Raw: "1297165000000", Formatted: "1,297.2B"}}},
					ResponseInfo: domain.TopStockResponseInfo{ValueType: "VALUE_TYPE_NET"},
				}, nil)
			},
			wantStatus:    http.StatusOK,
			wantCode:      "DSSA",
			wantValueType: "VALUE_TYPE_NET",
		},
		{
			name: "passes explicit filter params",
			path: "/api/v1/top-stock?start=2026-08-09&end=2026-08-10&investor_type=INVESTOR_TYPE_FOREIGN&market_type=MARKET_TYPE_NEGO&value_type=VALUE_TYPE_GROSS&page=2",
			setup: func(uc *mocks.MockTopStockUsecase) {
				uc.EXPECT().GetTopStock(gomock.Any(), "2026-08-09", "2026-08-10", "INVESTOR_TYPE_FOREIGN", "MARKET_TYPE_NEGO", "VALUE_TYPE_GROSS", 2).Return(&domain.TopStockData{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:         "missing start returns 422",
			path:         "/api/v1/top-stock?end=2026-08-10",
			wantStatus:   http.StatusUnprocessableEntity,
			wantErrCode:  "VALIDATION_ERROR",
			wantErrField: "start",
			wantErrMsg:   "is required",
		},
		{
			name:         "missing end returns 422",
			path:         "/api/v1/top-stock?start=2026-08-09",
			wantStatus:   http.StatusUnprocessableEntity,
			wantErrCode:  "VALIDATION_ERROR",
			wantErrField: "end",
			wantErrMsg:   "is required",
		},
		{
			name:         "invalid investor_type returns 422",
			path:         "/api/v1/top-stock?start=2026-08-09&end=2026-08-10&investor_type=BAD",
			wantStatus:   http.StatusUnprocessableEntity,
			wantErrCode:  "VALIDATION_ERROR",
			wantErrField: "investor_type",
			wantErrMsg:   "must be one of: INVESTOR_TYPE_ALL INVESTOR_TYPE_FOREIGN INVESTOR_TYPE_DOMESTIC",
		},
		{
			name:         "invalid market_type returns 422",
			path:         "/api/v1/top-stock?start=2026-08-09&end=2026-08-10&market_type=BAD",
			wantStatus:   http.StatusUnprocessableEntity,
			wantErrCode:  "VALIDATION_ERROR",
			wantErrField: "market_type",
			wantErrMsg:   "must be one of: MARKET_TYPE_ALL MARKET_TYPE_REGULER MARKET_TYPE_TUNAI MARKET_TYPE_NEGO",
		},
		{
			name:         "invalid value_type returns 422",
			path:         "/api/v1/top-stock?start=2026-08-09&end=2026-08-10&value_type=BAD",
			wantStatus:   http.StatusUnprocessableEntity,
			wantErrCode:  "VALIDATION_ERROR",
			wantErrField: "value_type",
			wantErrMsg:   "must be one of: VALUE_TYPE_NET VALUE_TYPE_GROSS VALUE_TYPE_TOTAL",
		},
		{
			name:         "invalid start date format returns 422",
			path:         "/api/v1/top-stock?start=09-08-2026&end=2026-08-10",
			wantStatus:   http.StatusUnprocessableEntity,
			wantErrCode:  "VALIDATION_ERROR",
			wantErrField: "start",
			wantErrMsg:   "must match datetime format 2006-01-02",
		},
		{
			name:         "invalid end date format returns 422",
			path:         "/api/v1/top-stock?start=2026-08-09&end=2026/08/10",
			wantStatus:   http.StatusUnprocessableEntity,
			wantErrCode:  "VALIDATION_ERROR",
			wantErrField: "end",
			wantErrMsg:   "must match datetime format 2006-01-02",
		},
		{
			name:         "impossible calendar date returns 422",
			path:         "/api/v1/top-stock?start=2026-13-40&end=2026-08-10",
			wantStatus:   http.StatusUnprocessableEntity,
			wantErrCode:  "VALIDATION_ERROR",
			wantErrField: "start",
			wantErrMsg:   "must match datetime format 2006-01-02",
		},
		{
			name: "usecase error returns 500",
			path: "/api/v1/top-stock?start=2026-08-09&end=2026-08-10",
			setup: func(uc *mocks.MockTopStockUsecase) {
				uc.EXPECT().GetTopStock(gomock.Any(), "2026-08-09", "2026-08-10", "INVESTOR_TYPE_ALL", "MARKET_TYPE_ALL", "VALUE_TYPE_NET", 0).Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockTopStockUsecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			h := NewTopStockHandler(uc, validator.New())
			rec := httptest.NewRecorder()
			h.TopStock(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    struct {
					TopBuy []struct {
						Code string `json:"code"`
					} `json:"top_buy"`
					ResponseInfo struct {
						ValueType string `json:"value_type"`
					} `json:"response_info"`
				} `json:"data"`
				Error *struct {
					Code    string            `json:"code"`
					Details map[string]string `json:"details"`
				} `json:"error"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))

			if tt.wantErrCode != "" {
				require.NotNil(t, env.Error)
				assert.Equal(t, tt.wantErrCode, env.Error.Code)
				if tt.wantErrField != "" {
					require.Contains(t, env.Error.Details, tt.wantErrField)
					assert.Equal(t, tt.wantErrMsg, env.Error.Details[tt.wantErrField])
				}
				return
			}
			if tt.wantCode != "" {
				require.Len(t, env.Data.TopBuy, 1)
				assert.Equal(t, tt.wantCode, env.Data.TopBuy[0].Code)
			}
			if tt.wantValueType != "" {
				assert.Equal(t, tt.wantValueType, env.Data.ResponseInfo.ValueType)
			}
		})
	}
}
