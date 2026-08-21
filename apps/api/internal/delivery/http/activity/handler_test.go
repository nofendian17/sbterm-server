package activity

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

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/mocks"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

func TestActivityHandlerActivity(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		setup       func(uc *mocks.MockActivityUsecase)
		wantStatus  int
		wantErrCode string
		wantFrom    string
	}{
		{
			name: "returns activity with all params",
			path: "/api/v1/order-trade/broker/activity?broker_code=AK&broker_code=ZP&transaction_type=TRANSACTION_TYPE_GROSS&investor_type=INVESTOR_TYPE_ALL&limit=20&market_board=MARKET_TYPE_REGULER&page=1&from=2026-07-14&to=2026-07-31&net_val_period=NET_VAL_PERIOD_7D",
			setup: func(uc *mocks.MockActivityUsecase) {
				uc.EXPECT().GetActivity(gomock.Any(), []string{"AK", "ZP"}, "TRANSACTION_TYPE_GROSS", "INVESTOR_TYPE_ALL", "MARKET_TYPE_REGULER", 20, 1, "2026-07-14", "2026-07-31", "NET_VAL_PERIOD_7D").Return(&domain.ActivityData{
					From:       "2026-07-14",
					To:         "2026-07-31",
					BrokerCode: "AK, ZP",
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantFrom:   "2026-07-14",
		},
		{
			name: "defaults enums and pagination when omitted",
			path: "/api/v1/order-trade/broker/activity",
			setup: func(uc *mocks.MockActivityUsecase) {
				uc.EXPECT().GetActivity(gomock.Any(), nil, "TRANSACTION_TYPE_GROSS", "INVESTOR_TYPE_ALL", "MARKET_TYPE_REGULER", 20, 1, "", "", "NET_VAL_PERIOD_7D").Return(&domain.ActivityData{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "invalid transaction_type returns 422",
			path:        "/api/v1/order-trade/broker/activity?transaction_type=BUY",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid market_board returns 422",
			path:        "/api/v1/order-trade/broker/activity?market_board=BOARD_TYPE_REGULAR",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid net_val_period returns 422",
			path:        "/api/v1/order-trade/broker/activity?net_val_period=BOGUS",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid limit returns 422",
			path:        "/api/v1/order-trade/broker/activity?limit=abc",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name: "upstream 400 returns 422",
			path: "/api/v1/order-trade/broker/activity?from=2026-07-14&to=2026-07-31",
			setup: func(uc *mocks.MockActivityUsecase) {
				uc.EXPECT().GetActivity(gomock.Any(), nil, "TRANSACTION_TYPE_GROSS", "INVESTOR_TYPE_ALL", "MARKET_TYPE_REGULER", 20, 1, "2026-07-14", "2026-07-31", "NET_VAL_PERIOD_7D").Return(nil, &domain.UpstreamError{Status: http.StatusBadRequest, Msg: "invalid"})
			},
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name: "usecase error returns 500",
			path: "/api/v1/order-trade/broker/activity",
			setup: func(uc *mocks.MockActivityUsecase) {
				uc.EXPECT().GetActivity(gomock.Any(), nil, "TRANSACTION_TYPE_GROSS", "INVESTOR_TYPE_ALL", "MARKET_TYPE_REGULER", 20, 1, "", "", "NET_VAL_PERIOD_7D").Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockActivityUsecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			r := chi.NewRouter()
			h := NewActivityHandler(uc, validator.New())
			r.Get("/api/v1/order-trade/broker/activity", h.Activity)

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    struct {
					From string `json:"from"`
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
			if tt.wantFrom != "" {
				assert.Equal(t, tt.wantFrom, env.Data.From)
			}
		})
	}
}

func TestActivityHandlerActivityChart(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		setup       func(uc *mocks.MockActivityUsecase)
		wantStatus  int
		wantErrCode string
		wantFrom    string
	}{
		{
			name: "returns activity chart with symbols, brokers and range",
			path: "/api/v1/order-trade/broker/activity-chart?symbols=BUMI&symbols=DSSA&brokers_code=XL&brokers_code=ZP&from=2025-08-11&to=2026-08-11",
			setup: func(uc *mocks.MockActivityUsecase) {
				uc.EXPECT().GetActivityChart(gomock.Any(), []string{"BUMI", "DSSA"}, []string{"XL", "ZP"}, "2025-08-11", "2026-08-11", "", "INVESTOR_TYPE_ALL", "BOARD_TYPE_ALL").Return(&domain.ActivityChartData{
					From:       "2025-08-11",
					To:         "2026-08-11",
					BrokerCode: []string{"XL", "ZP"},
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantFrom:   "2025-08-11",
		},
		{
			name: "sends period when from/to omitted and defaults period when all omitted",
			path: "/api/v1/order-trade/broker/activity-chart",
			setup: func(uc *mocks.MockActivityUsecase) {
				uc.EXPECT().GetActivityChart(gomock.Any(), nil, nil, "", "", "RT_PERIOD_LAST_1_DAY", "INVESTOR_TYPE_ALL", "BOARD_TYPE_ALL").Return(&domain.ActivityChartData{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "invalid period returns 422",
			path:        "/api/v1/order-trade/broker/activity-chart?period=BOGUS",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid investor type returns 422",
			path:        "/api/v1/order-trade/broker/activity-chart?investor_type=BOGUS",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name: "upstream 400 for date with no data returns 422",
			path: "/api/v1/order-trade/broker/activity-chart?from=2026-08-11&to=2026-08-11",
			setup: func(uc *mocks.MockActivityUsecase) {
				uc.EXPECT().GetActivityChart(gomock.Any(), nil, nil, "2026-08-11", "2026-08-11", "", "INVESTOR_TYPE_ALL", "BOARD_TYPE_ALL").Return(nil, &domain.UpstreamError{Status: http.StatusBadRequest, Msg: "invalid"})
			},
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name: "usecase error returns 500",
			path: "/api/v1/order-trade/broker/activity-chart",
			setup: func(uc *mocks.MockActivityUsecase) {
				uc.EXPECT().GetActivityChart(gomock.Any(), nil, nil, "", "", "RT_PERIOD_LAST_1_DAY", "INVESTOR_TYPE_ALL", "BOARD_TYPE_ALL").Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockActivityUsecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			r := chi.NewRouter()
			h := NewActivityHandler(uc, validator.New())
			r.Get("/api/v1/order-trade/broker/activity-chart", h.ActivityChart)

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    struct {
					From string `json:"from"`
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
			if tt.wantFrom != "" {
				assert.Equal(t, tt.wantFrom, env.Data.From)
			}
		})
	}
}

func TestActivityHandlerActivityHistorical(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		setup       func(uc *mocks.MockActivityUsecase)
		wantStatus  int
		wantErrCode string
		wantFrom    string
	}{
		{
			name: "returns activity historical with all params",
			path: "/api/v1/order-trade/broker/activity/historical?interval=INTERVAL_DAILY&date_from=2026-07-01&date_to=2026-08-31&broker_codes=ZP&broker_codes=BK&symbols=CUAN&market_board=BOARD_TYPE_REGULAR&investor_type=INVESTOR_TYPE_ALL&net_interval=INTERVAL_MONTHLY",
			setup: func(uc *mocks.MockActivityUsecase) {
				uc.EXPECT().GetActivityHistorical(gomock.Any(), "INTERVAL_DAILY", "2026-07-01", "2026-08-31", []string{"ZP", "BK"}, []string{"CUAN"}, "BOARD_TYPE_REGULAR", "INVESTOR_TYPE_ALL", "INTERVAL_MONTHLY").Return(&domain.ActivityHistoricalData{
					DateFrom: "2026-07-01",
					DateTo:   "2026-08-12",
					Summary:  domain.ActivityHistoricalSummary{GroupType: "INTERVAL_TYPE_MONTHLY"},
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantFrom:   "2026-07-01",
		},
		{
			name: "defaults enums when omitted",
			path: "/api/v1/order-trade/broker/activity/historical",
			setup: func(uc *mocks.MockActivityUsecase) {
				uc.EXPECT().GetActivityHistorical(gomock.Any(), "INTERVAL_DAILY", "", "", nil, nil, "BOARD_TYPE_ALL", "INVESTOR_TYPE_ALL", "INTERVAL_MONTHLY").Return(&domain.ActivityHistoricalData{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "invalid interval returns 422",
			path:        "/api/v1/order-trade/broker/activity/historical?interval=BOGUS",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid net_interval returns 422",
			path:        "/api/v1/order-trade/broker/activity/historical?net_interval=BOGUS",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid market_board returns 422",
			path:        "/api/v1/order-trade/broker/activity/historical?market_board=BOARD_TYPE_FOO",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid date_from returns 422",
			path:        "/api/v1/order-trade/broker/activity/historical?date_from=not-a-date",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name: "upstream 400 returns 422",
			path: "/api/v1/order-trade/broker/activity/historical?date_from=2026-07-01&date_to=2026-08-31",
			setup: func(uc *mocks.MockActivityUsecase) {
				uc.EXPECT().GetActivityHistorical(gomock.Any(), "INTERVAL_DAILY", "2026-07-01", "2026-08-31", nil, nil, "BOARD_TYPE_ALL", "INVESTOR_TYPE_ALL", "INTERVAL_MONTHLY").Return(nil, &domain.UpstreamError{Status: http.StatusBadRequest, Msg: "invalid"})
			},
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name: "usecase error returns 500",
			path: "/api/v1/order-trade/broker/activity/historical",
			setup: func(uc *mocks.MockActivityUsecase) {
				uc.EXPECT().GetActivityHistorical(gomock.Any(), "INTERVAL_DAILY", "", "", nil, nil, "BOARD_TYPE_ALL", "INVESTOR_TYPE_ALL", "INTERVAL_MONTHLY").Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockActivityUsecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			r := chi.NewRouter()
			h := NewActivityHandler(uc, validator.New())
			r.Get("/api/v1/order-trade/broker/activity/historical", h.ActivityHistorical)

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    struct {
					From string `json:"date_from"`
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
			if tt.wantFrom != "" {
				assert.Equal(t, tt.wantFrom, env.Data.From)
			}
		})
	}
}

func TestActivityHandlerBrokerDistribution(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		setup       func(uc *mocks.MockActivityUsecase)
		wantStatus  int
		wantErrCode string
	}{
		{
			name:  "returns distribution with defaults",
			query: "?symbol=BUMI",
			setup: func(uc *mocks.MockActivityUsecase) {
				uc.EXPECT().GetBrokerDistribution(gomock.Any(), "BUMI", "INVESTOR_TYPE_ALL", "MARKET_TYPE_REGULER", "BROKER_DISTRIBUTION_DATA_TYPE_VALUE", "", "", "").Return(&domain.BrokerDistributionData{
					DateInfo: "2026-08-19",
					ByValue: domain.BrokerDistributionSides{
						TopBrokerBuy: []domain.BrokerDistributionEntry{{
							Detail:       domain.BrokerDistributionCounterparty{Code: "MG", Type: "Lokal", Amount: 683315577200},
							DistributeTo: []domain.BrokerDistributionCounterparty{{Code: "XL", Type: "Asing", Amount: 96841421100}},
						}},
					},
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:  "passes enums and range through",
			query: "?symbol=BUMI&investor_type=INVESTOR_TYPE_FOREIGN&market_board=MARKET_TYPE_ALL&data_type=BROKER_DISTRIBUTION_DATA_TYPE_VOLUME&from=2026-08-02&to=2026-08-19",
			setup: func(uc *mocks.MockActivityUsecase) {
				uc.EXPECT().GetBrokerDistribution(gomock.Any(), "BUMI", "INVESTOR_TYPE_FOREIGN", "MARKET_TYPE_ALL", "BROKER_DISTRIBUTION_DATA_TYPE_VOLUME", "", "2026-08-02", "2026-08-19").Return(&domain.BrokerDistributionData{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{name: "missing symbol returns 422", query: "?investor_type=INVESTOR_TYPE_ALL", wantStatus: http.StatusUnprocessableEntity, wantErrCode: "VALIDATION_ERROR"},
		{name: "single bound returns 422", query: "?symbol=BUMI&from=2026-08-02", wantStatus: http.StatusUnprocessableEntity, wantErrCode: "VALIDATION_ERROR"},
		{name: "reversed range returns 422", query: "?symbol=BUMI&from=2026-08-19&to=2026-08-02", wantStatus: http.StatusUnprocessableEntity, wantErrCode: "VALIDATION_ERROR"},
		{name: "bad enum returns 422", query: "?symbol=BUMI&data_type=BOGUS", wantStatus: http.StatusUnprocessableEntity, wantErrCode: "VALIDATION_ERROR"},
		{
			name:  "upstream 400 returns 422",
			query: "?symbol=BUMI&from=2026-08-19&to=2026-08-19",
			setup: func(uc *mocks.MockActivityUsecase) {
				uc.EXPECT().GetBrokerDistribution(gomock.Any(), "BUMI", "INVESTOR_TYPE_ALL", "MARKET_TYPE_REGULER", "BROKER_DISTRIBUTION_DATA_TYPE_VALUE", "", "2026-08-19", "2026-08-19").Return(nil, &domain.UpstreamError{Status: http.StatusBadRequest, Msg: "Invalid date"})
			},
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:  "usecase error returns 500",
			query: "?symbol=BUMI",
			setup: func(uc *mocks.MockActivityUsecase) {
				uc.EXPECT().GetBrokerDistribution(gomock.Any(), "BUMI", "INVESTOR_TYPE_ALL", "MARKET_TYPE_REGULER", "BROKER_DISTRIBUTION_DATA_TYPE_VALUE", "", "", "").Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockActivityUsecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			h := NewActivityHandler(uc, validator.New())
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/order-trade/broker/distribution"+tt.query, nil)
			h.BrokerDistribution(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    *struct {
					DateInfo string `json:"date_info"`
					ByValue  struct {
						TopBrokerBuy []struct {
							Detail struct {
								Code   string `json:"code"`
								Amount int64  `json:"amount"`
							} `json:"detail"`
						} `json:"top_broker_buy"`
					} `json:"by_value"`
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
			require.NotNil(t, env.Data)
		})
	}
}
