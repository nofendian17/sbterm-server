package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// orderQueueBody mirrors the real upstream response captured from
// /order-trade/order-queue for SLIS (see /tmp/opencode/oq_slis.json),
// trimmed to three orders, with pagination and market state.
const orderQueueBody = `{"message":"Successfully get list order queue","data":{"is_open_market":false,"orders":[{"id":"3495619555","queue_number":"1","stock_code":"SLIS","time":"2026-08-14T14:09:19.282073Z","action_type":"ACTION_TYPE_BUY","price":101,"status":"ORDER_STATUS_PARTIAL_MATCH","open":39,"lot":50,"board_type":"BOARD_TYPE_REGULAR","broker_code":"YP","exchange_order_number":{"full":"202608140003318022","formatted":"3318022"},"queue_lot":0,"broker_group":"BROKER_GROUP_FOREIGN","order_number":"202608140003318022"},{"id":"3495632066","queue_number":"2","stock_code":"SLIS","time":"2026-08-14T14:09:54.736694Z","action_type":"ACTION_TYPE_BUY","price":101,"status":"ORDER_STATUS_OPEN","open":20,"lot":20,"board_type":"BOARD_TYPE_REGULAR","broker_code":"","exchange_order_number":{"full":"202608140003326502","formatted":"3326502"},"queue_lot":39,"broker_group":"BROKER_GROUP_UNSPECIFIED","order_number":"202608140003326502"},{"id":"3495689852","queue_number":"3","stock_code":"SLIS","time":"2026-08-14T14:12:39.894109Z","action_type":"ACTION_TYPE_BUY","price":101,"status":"ORDER_STATUS_OPEN","open":88,"lot":88,"board_type":"BOARD_TYPE_REGULAR","broker_code":"","exchange_order_number":{"full":"202608140003364209","formatted":"3364209"},"queue_lot":59,"broker_group":"BROKER_GROUP_UNSPECIFIED","order_number":"202608140003364209"}],"pagination":{"has_next_page":true}}}`

func TestGetOrderQueue(t *testing.T) {
	tests := []struct {
		name   string
		params OrderQueueParams
		opts   []Option
		check  func(t *testing.T, r *http.Request)
		verify func(t *testing.T, resp *OrderQueueResponse)
	}{
		{
			name: "returns order queue with all params",
			params: OrderQueueParams{
				StockCode:     "SLIS",
				ActionType:    "ACTION_TYPE_ALL",
				BoardType:     "BOARD_TYPE_REGULAR",
				Limit:         100,
				OrderStatus:   "ORDER_STATUS_OPEN",
				Price:         101,
				SortBy:        "SORT_BY_QUEUE",
				SortDirection: "SORT_DIRECTION_ASC",
			},
			check: func(t *testing.T, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/order-trade/order-queue", r.URL.Path)
				q := r.URL.Query()
				assert.Equal(t, "SLIS", q.Get("stock_code"))
				assert.Equal(t, "ACTION_TYPE_ALL", q.Get("action_type"))
				assert.Equal(t, "BOARD_TYPE_REGULAR", q.Get("board_type"))
				assert.Equal(t, "100", q.Get("limit"))
				assert.Equal(t, "ORDER_STATUS_OPEN", q.Get("order_status"))
				assert.Equal(t, "101", q.Get("price"))
				assert.Equal(t, "SORT_BY_QUEUE", q.Get("sort_by"))
				assert.Equal(t, "SORT_DIRECTION_ASC", q.Get("sort_direction"))
			},
			verify: func(t *testing.T, resp *OrderQueueResponse) {
				d := resp.Data
				assert.False(t, d.IsOpenMarket)
				assert.True(t, d.Pagination.HasNextPage)
				require.Len(t, d.Orders, 3)
				o := d.Orders[0]
				assert.Equal(t, "3495619555", o.ID)
				assert.Equal(t, "1", o.QueueNumber)
				assert.Equal(t, "SLIS", o.StockCode)
				assert.Equal(t, "2026-08-14T14:09:19.282073Z", o.Time)
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
				assert.Equal(t, "", d.Orders[1].BrokerCode)
			},
		},
		{
			name:   "omits empty params",
			params: OrderQueueParams{StockCode: "SLIS"},
			check: func(t *testing.T, r *http.Request) {
				q := r.URL.Query()
				assert.Equal(t, "SLIS", q.Get("stock_code"))
				assert.Equal(t, "", q.Get("action_type"))
				assert.Equal(t, "", q.Get("board_type"))
				assert.Equal(t, "", q.Get("limit"))
				assert.Equal(t, "", q.Get("order_status"))
				assert.Equal(t, "", q.Get("price"))
				assert.Equal(t, "", q.Get("sort_by"))
				assert.Equal(t, "", q.Get("sort_direction"))
			},
		},
		{
			name: "uses access token",
			params: OrderQueueParams{
				StockCode:  "SLIS",
				ActionType: "ACTION_TYPE_ALL",
				Price:      101,
			},
			opts: []Option{WithAuthenticator(&stubAuth{token: "at-ok"})},
			check: func(t *testing.T, r *http.Request) {
				assert.Equal(t, "Bearer at-ok", r.Header.Get("Authorization"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.check != nil {
					tt.check(t, r)
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(orderQueueBody))
			}))
			defer srv.Close()

			opts := append([]Option{WithBaseURL(srv.URL)}, tt.opts...)
			resp, err := New(opts...).GetOrderQueue(context.Background(), tt.params)
			require.NoError(t, err)
			if tt.verify != nil {
				tt.verify(t, resp)
			}
		})
	}
}
