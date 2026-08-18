package orderqueue

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

func TestOrderQueueHandlerOrderQueue(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		setup          func(uc *mocks.MockOrderQueueUsecase)
		wantStatus     int
		wantErrCode    string
		wantErrDetails map[string]string
		wantCode       string
	}{
		{
			name: "returns order queue with all params",
			path: "/api/v1/order-trade/order-queue?stock_code=SLIS&action_type=ACTION_TYPE_ALL&board_type=BOARD_TYPE_REGULAR&order_status=ORDER_STATUS_OPEN&sort_by=SORT_BY_QUEUE&sort_direction=SORT_DIRECTION_ASC&price=101&limit=100",
			setup: func(uc *mocks.MockOrderQueueUsecase) {
				uc.EXPECT().GetOrderQueue(gomock.Any(), "SLIS", "ACTION_TYPE_ALL", "BOARD_TYPE_REGULAR", "ORDER_STATUS_OPEN", "SORT_BY_QUEUE", "SORT_DIRECTION_ASC", 100, int64(101)).Return(&domain.OrderQueueData{
					IsOpenMarket: false,
					Orders: []domain.OrderQueueOrder{{
						ID:          "3495619555",
						QueueNumber: "1",
						StockCode:   "SLIS",
						ActionType:  "ACTION_TYPE_BUY",
						Price:       101,
						Status:      "ORDER_STATUS_PARTIAL_MATCH",
						Open:        39,
						Lot:         50,
						BoardType:   "BOARD_TYPE_REGULAR",
						BrokerCode:  "YP",
						ExchangeOrderNumber: domain.OrderQueueExchangeOrderNumber{
							Full:      "202608140003318022",
							Formatted: "3318022",
						},
						QueueLot:    0,
						BrokerGroup: "BROKER_GROUP_FOREIGN",
						OrderNumber: "202608140003318022",
					}},
					Pagination: domain.OrderQueuePagination{HasNextPage: true},
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantCode:   "SLIS",
		},
		{
			name: "defaults enums and limit when omitted",
			path: "/api/v1/order-trade/order-queue?stock_code=SLIS",
			setup: func(uc *mocks.MockOrderQueueUsecase) {
				uc.EXPECT().GetOrderQueue(gomock.Any(), "SLIS", "ACTION_TYPE_ALL", "BOARD_TYPE_REGULAR", "ORDER_STATUS_OPEN", "SORT_BY_QUEUE", "SORT_DIRECTION_ASC", 100, int64(0)).Return(&domain.OrderQueueData{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "missing stock code returns 422",
			path:        "/api/v1/order-trade/order-queue",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid action type returns 422",
			path:        "/api/v1/order-trade/order-queue?stock_code=SLIS&action_type=BOGUS",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid board type returns 422",
			path:        "/api/v1/order-trade/order-queue?stock_code=SLIS&board_type=BOGUS",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid order status returns 422",
			path:        "/api/v1/order-trade/order-queue?stock_code=SLIS&order_status=BOGUS",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid sort by returns 422",
			path:        "/api/v1/order-trade/order-queue?stock_code=SLIS&sort_by=BOGUS",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid sort direction returns 422",
			path:        "/api/v1/order-trade/order-queue?stock_code=SLIS&sort_direction=BOGUS",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid limit returns 422",
			path:        "/api/v1/order-trade/order-queue?stock_code=SLIS&limit=abc",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
			wantErrDetails: map[string]string{
				"limit": "must be a valid integer",
			},
		},
		{
			name:        "invalid price returns 422",
			path:        "/api/v1/order-trade/order-queue?stock_code=SLIS&price=abc",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
			wantErrDetails: map[string]string{
				"price": "must be a valid integer",
			},
		},
		{
			name: "usecase error returns 500",
			path: "/api/v1/order-trade/order-queue?stock_code=SLIS",
			setup: func(uc *mocks.MockOrderQueueUsecase) {
				uc.EXPECT().GetOrderQueue(gomock.Any(), "SLIS", "ACTION_TYPE_ALL", "BOARD_TYPE_REGULAR", "ORDER_STATUS_OPEN", "SORT_BY_QUEUE", "SORT_DIRECTION_ASC", 100, int64(0)).Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
		{
			name: "upstream 400 returns 422",
			path: "/api/v1/order-trade/order-queue?stock_code=SLIS",
			setup: func(uc *mocks.MockOrderQueueUsecase) {
				uc.EXPECT().GetOrderQueue(gomock.Any(), "SLIS", "ACTION_TYPE_ALL", "BOARD_TYPE_REGULAR", "ORDER_STATUS_OPEN", "SORT_BY_QUEUE", "SORT_DIRECTION_ASC", 100, int64(0)).Return(nil, &domain.UpstreamError{Status: http.StatusBadRequest, Msg: "invalid"})
			},
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockOrderQueueUsecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			r := chi.NewRouter()
			h := NewOrderQueueHandler(uc, validator.New())
			r.Get("/api/v1/order-trade/order-queue", h.OrderQueue)

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    struct {
					IsOpenMarket bool                 `json:"is_open_market"`
					Orders       []orderQueueItemResp `json:"orders"`
					Pagination   orderQueuePageResp   `json:"pagination"`
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
			if tt.wantCode != "" {
				require.Len(t, env.Data.Orders, 1)
				assert.Equal(t, tt.wantCode, env.Data.Orders[0].StockCode)
				assert.Equal(t, "202608140003318022", env.Data.Orders[0].ExchangeOrderNumber.Full)
				assert.True(t, env.Data.Pagination.HasNextPage)
			}
		})
	}
}
