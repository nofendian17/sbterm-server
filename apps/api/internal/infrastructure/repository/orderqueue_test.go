package repository

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/libs/stockbit"
)

func TestOrderQueueRepositoryGetOrderQueue(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		wantErr  bool
		wantUp   bool
		wantCode int
	}{
		{
			name:   "returns mapped order queue",
			status: http.StatusOK,
			body:   orderQueueRepoBody,
		},
		{
			name:    "propagates upstream error",
			status:  http.StatusInternalServerError,
			body:    `{"message":"boom"}`,
			wantErr: true,
		},
		{
			name:     "translates upstream 400 into domain error",
			status:   http.StatusBadRequest,
			body:     `{"message":"Please check your request"}`,
			wantErr:  true,
			wantUp:   true,
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/order-trade/order-queue", r.URL.Path)
				assert.Equal(t, "SLIS", r.URL.Query().Get("stock_code"))
				assert.Equal(t, "ACTION_TYPE_ALL", r.URL.Query().Get("action_type"))
				assert.Equal(t, "BOARD_TYPE_REGULAR", r.URL.Query().Get("board_type"))
				assert.Equal(t, "ORDER_STATUS_OPEN", r.URL.Query().Get("order_status"))
				assert.Equal(t, "SORT_BY_QUEUE", r.URL.Query().Get("sort_by"))
				assert.Equal(t, "SORT_DIRECTION_ASC", r.URL.Query().Get("sort_direction"))
				assert.Equal(t, "100", r.URL.Query().Get("limit"))
				assert.Equal(t, "101", r.URL.Query().Get("price"))
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewOrderQueueRepository(client)

			got, err := repo.GetOrderQueue(context.Background(), "SLIS", "ACTION_TYPE_ALL", "BOARD_TYPE_REGULAR", "ORDER_STATUS_OPEN", "SORT_BY_QUEUE", "SORT_DIRECTION_ASC", 100, 101)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantUp {
					var up *domain.UpstreamError
					require.ErrorAs(t, err, &up)
					assert.Equal(t, tt.wantCode, up.Status)
				}
				return
			}
			require.NoError(t, err)
			assert.False(t, got.IsOpenMarket)
			assert.True(t, got.Pagination.HasNextPage)
			require.Len(t, got.Orders, 1)
			o := got.Orders[0]
			assert.Equal(t, "3495619555", o.ID)
			assert.Equal(t, "1", o.QueueNumber)
			assert.Equal(t, "SLIS", o.StockCode)
			assert.Equal(t, "ACTION_TYPE_BUY", o.ActionType)
			assert.Equal(t, int64(101), o.Price)
			assert.Equal(t, "ORDER_STATUS_PARTIAL_MATCH", o.Status)
			assert.Equal(t, int64(39), o.Open)
			assert.Equal(t, int64(50), o.Lot)
			assert.Equal(t, "BOARD_TYPE_REGULAR", o.BoardType)
			assert.Equal(t, "YP", o.BrokerCode)
			assert.Equal(t, "202608140003318022", o.ExchangeOrderNumber.Full)
			assert.Equal(t, "3318022", o.ExchangeOrderNumber.Formatted)
			assert.Equal(t, int64(0), o.QueueLot)
			assert.Equal(t, "BROKER_GROUP_FOREIGN", o.BrokerGroup)
			assert.Equal(t, "202608140003318022", o.OrderNumber)
		})
	}
}

const orderQueueRepoBody = `{"message":"Successfully get list order queue","data":{"is_open_market":false,"orders":[{"id":"3495619555","queue_number":"1","stock_code":"SLIS","time":"2026-08-14T14:09:19.282073Z","action_type":"ACTION_TYPE_BUY","price":101,"status":"ORDER_STATUS_PARTIAL_MATCH","open":39,"lot":50,"board_type":"BOARD_TYPE_REGULAR","broker_code":"YP","exchange_order_number":{"full":"202608140003318022","formatted":"3318022"},"queue_lot":0,"broker_group":"BROKER_GROUP_FOREIGN","order_number":"202608140003318022"}],"pagination":{"has_next_page":true}}}`
