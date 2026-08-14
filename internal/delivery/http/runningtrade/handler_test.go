package runningtrade

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

func TestRunningTradeHandlerRunningTrade(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		setup          func(uc *mocks.MockRunningTradeUsecase)
		wantStatus     int
		wantErrCode    string
		wantErrDetails map[string]string
		wantFrom       string
	}{
		{
			name: "returns running trade chart",
			path: "/v1/company/DSSA/running-trade-chart?broker_code=DR&broker_code=AK&from=2026-07-01&to=2026-08-10",
			setup: func(uc *mocks.MockRunningTradeUsecase) {
				uc.EXPECT().GetRunningTradeChart(gomock.Any(), "DSSA", []string{"DR", "AK"}, "2026-07-01", "2026-08-10", "INVESTOR_TYPE_ALL", "BOARD_TYPE_ALL", "").Return(&domain.RunningTradeData{
					From:            "2026-07-01",
					To:              "2026-08-10",
					DateSessionInfo: "10 Aug 2026",
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantFrom:   "2026-07-01",
		},
		{
			name: "defaults period when from/to omitted",
			path: "/v1/company/DSSA/running-trade-chart?broker_code=DR",
			setup: func(uc *mocks.MockRunningTradeUsecase) {
				uc.EXPECT().GetRunningTradeChart(gomock.Any(), "DSSA", []string{"DR"}, "", "", "INVESTOR_TYPE_ALL", "BOARD_TYPE_ALL", "RT_PERIOD_LAST_1_DAY").Return(&domain.RunningTradeData{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "missing symbol returns 422",
			path:        "/v1/company//running-trade-chart",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid investor type returns 422",
			path:        "/v1/company/DSSA/running-trade-chart?investor_type=BOGUS",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid market board returns 422",
			path:        "/v1/company/DSSA/running-trade-chart?market_board=BOGUS",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid period returns 422",
			path:        "/v1/company/DSSA/running-trade-chart?period=BOGUS",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "from without to returns 422",
			path:        "/v1/company/DSSA/running-trade-chart?from=2026-07-01",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
			wantErrDetails: map[string]string{
				"to": "from and to must both be provided or both omitted",
			},
		},
		{
			name:        "reversed range returns 422",
			path:        "/v1/company/DSSA/running-trade-chart?from=2026-08-10&to=2026-07-01",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
			wantErrDetails: map[string]string{
				"from": "must be earlier than or equal to to",
			},
		},
		{
			name: "usecase error returns 500",
			path: "/v1/company/DSSA/running-trade-chart?broker_code=DR",
			setup: func(uc *mocks.MockRunningTradeUsecase) {
				uc.EXPECT().GetRunningTradeChart(gomock.Any(), "DSSA", []string{"DR"}, "", "", "INVESTOR_TYPE_ALL", "BOARD_TYPE_ALL", "RT_PERIOD_LAST_1_DAY").Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
		{
			name: "upstream 400 for date with no session data returns 422",
			path: "/v1/company/DSSA/running-trade-chart?broker_code=DR&from=2026-08-11&to=2026-08-11",
			setup: func(uc *mocks.MockRunningTradeUsecase) {
				uc.EXPECT().GetRunningTradeChart(gomock.Any(), "DSSA", []string{"DR"}, "2026-08-11", "2026-08-11", "INVESTOR_TYPE_ALL", "BOARD_TYPE_ALL", "").Return(nil, &domain.UpstreamError{Status: http.StatusBadRequest, Msg: "Please check your request"})
			},
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockRunningTradeUsecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			r := chi.NewRouter()
			h := NewRunningTradeHandler(uc, validator.New())
			r.Get("/v1/company/{symbol}/running-trade-chart", h.RunningTradeChart)

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    struct {
					From            string `json:"from"`
					DateSessionInfo string `json:"date_session_info"`
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
				if tt.wantErrDetails != nil {
					assert.Equal(t, tt.wantErrDetails, env.Error.Details)
				}
				return
			}
			if tt.wantFrom != "" {
				assert.Equal(t, tt.wantFrom, env.Data.From)
			}
		})
	}
}

func TestRunningTradeHandlerRunningTradeFeed(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		setup       func(uc *mocks.MockRunningTradeUsecase)
		wantStatus  int
		wantErrCode string
		wantCode    string
	}{
		{
			name: "returns running trade feed with all params",
			path: "/v1/order-trade/running-trade?symbol=BBCA&sort=ASC&order_by=RUNNING_TRADE_ORDER_BY_TIME&date=2026-08-13&limit=80&trade_number=17796",
			setup: func(uc *mocks.MockRunningTradeUsecase) {
				uc.EXPECT().GetRunningTrade(gomock.Any(), "BBCA", "ASC", "RUNNING_TRADE_ORDER_BY_TIME", "2026-08-13", 80, int64(17796)).Return(&domain.RunningTradeFeed{
					IsOpenMarket: false,
					RunningTrade: []domain.RunningTradeFeedItem{{ID: "4760187264", Time: "08:58:00", Action: "buy", Code: "BBCA", TradeNumber: "17797"}},
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantCode:   "BBCA",
		},
		{
			name: "defaults sort order_by and limit when omitted and omits date",
			path: "/v1/order-trade/running-trade?symbol=BBCA",
			setup: func(uc *mocks.MockRunningTradeUsecase) {
				uc.EXPECT().GetRunningTrade(gomock.Any(), "BBCA", "ASC", "RUNNING_TRADE_ORDER_BY_TIME", "", 80, int64(0)).Return(&domain.RunningTradeFeed{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "missing symbol returns 422",
			path:        "/v1/order-trade/running-trade",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid sort returns 422",
			path:        "/v1/order-trade/running-trade?symbol=BBCA&sort=BOGUS",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid order_by returns 422",
			path:        "/v1/order-trade/running-trade?symbol=BBCA&order_by=BOGUS",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid date returns 422",
			path:        "/v1/order-trade/running-trade?symbol=BBCA&date=bogus",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid limit returns 422",
			path:        "/v1/order-trade/running-trade?symbol=BBCA&limit=abc",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "zero limit returns 422",
			path:        "/v1/order-trade/running-trade?symbol=BBCA&limit=0",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name: "upstream 400 returns 422",
			path: "/v1/order-trade/running-trade?symbol=BBCA",
			setup: func(uc *mocks.MockRunningTradeUsecase) {
				uc.EXPECT().GetRunningTrade(gomock.Any(), "BBCA", "ASC", "RUNNING_TRADE_ORDER_BY_TIME", "", 80, int64(0)).Return(nil, &domain.UpstreamError{Status: http.StatusBadRequest, Msg: "invalid"})
			},
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name: "usecase error returns 500",
			path: "/v1/order-trade/running-trade?symbol=BBCA",
			setup: func(uc *mocks.MockRunningTradeUsecase) {
				uc.EXPECT().GetRunningTrade(gomock.Any(), "BBCA", "ASC", "RUNNING_TRADE_ORDER_BY_TIME", "", 80, int64(0)).Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockRunningTradeUsecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			r := chi.NewRouter()
			h := NewRunningTradeHandler(uc, validator.New())
			r.Get("/v1/order-trade/running-trade", h.RunningTrade)

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    struct {
					IsOpenMarket bool `json:"is_open_market"`
					RunningTrade []struct {
						Code string `json:"code"`
					} `json:"running_trade"`
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
			if tt.wantCode != "" {
				require.Len(t, env.Data.RunningTrade, 1)
				assert.Equal(t, tt.wantCode, env.Data.RunningTrade[0].Code)
			}
		})
	}
}

func TestRunningTradeRangeRequirements(t *testing.T) {
	tests := []struct {
		name    string
		req     runningTradeRequest
		wantNil bool
		want    map[string]string
	}{
		{
			name:    "from/to both present is valid",
			req:     runningTradeRequest{From: "2026-07-01", To: "2026-08-10"},
			wantNil: true,
		},
		{
			name:    "both omitted is valid (period default applies)",
			req:     runningTradeRequest{},
			wantNil: true,
		},
		{
			name: "from without to",
			req:  runningTradeRequest{From: "2026-07-01"},
			want: map[string]string{"to": "from and to must both be provided or both omitted"},
		},
		{
			name: "to without from",
			req:  runningTradeRequest{To: "2026-08-10"},
			want: map[string]string{"from": "from and to must both be provided or both omitted"},
		},
		{
			name: "reversed range",
			req:  runningTradeRequest{From: "2026-08-10", To: "2026-07-01"},
			want: map[string]string{"from": "must be earlier than or equal to to"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runningTradeRangeRequirements(tt.req)
			if tt.wantNil {
				assert.Nil(t, got)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
