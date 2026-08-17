package stockbit

import (
	"context"
	"net/url"
	"strconv"
)

const orderQueuePath = "/order-trade/order-queue"

// OrderQueueParams selects the orders returned by GetOrderQueue. Enum fields
// (ActionType, BoardType, OrderStatus, SortBy, SortDirection) take the
// uppercase enum values of the upstream API, e.g. ACTION_TYPE_ALL,
// BOARD_TYPE_REGULAR.
type OrderQueueParams struct {
	StockCode     string
	ActionType    string
	BoardType     string
	Limit         int
	OrderStatus   string
	Price         int64
	SortBy        string
	SortDirection string
}

// OrderQueueResponse is the order queue response for a symbol.
type OrderQueueResponse struct {
	Message string         `json:"message"`
	Data    OrderQueueData `json:"data"`
}

type OrderQueueData struct {
	IsOpenMarket bool                 `json:"is_open_market"`
	Orders       []OrderQueueOrder    `json:"orders"`
	Pagination   OrderQueuePagination `json:"pagination"`
}

// OrderQueuePagination tells whether more pages are available after the
// requested limit.
type OrderQueuePagination struct {
	HasNextPage bool `json:"has_next_page"`
}

// OrderQueueOrder is a single order sitting in the queue at a price level.
// queue_number is a string upstream for orders beyond nine digits.
type OrderQueueOrder struct {
	ID                  string                `json:"id"`
	QueueNumber         string                `json:"queue_number"`
	StockCode           string                `json:"stock_code"`
	Time                string                `json:"time"`
	ActionType          string                `json:"action_type"`
	Price               int64                 `json:"price"`
	Status              string                `json:"status"`
	Open                int64                 `json:"open"`
	Lot                 int64                 `json:"lot"`
	BoardType           string                `json:"board_type"`
	BrokerCode          string                `json:"broker_code"`
	ExchangeOrderNumber OrderQueueOrderNumber `json:"exchange_order_number"`
	QueueLot            int64                 `json:"queue_lot"`
	BrokerGroup         string                `json:"broker_group"`
	OrderNumber         string                `json:"order_number"`
}

// OrderQueueOrderNumber identifies an order at the exchange: full is the
// complete order number, formatted is its display form.
type OrderQueueOrderNumber struct {
	Full      string `json:"full"`
	Formatted string `json:"formatted"`
}

// GetOrderQueue returns the order queue for a symbol at a price. The access
// token is attached automatically. params carries the filter/order enums;
// empty values are omitted so upstream applies its defaults.
func (c *Client) GetOrderQueue(ctx context.Context, params OrderQueueParams) (*OrderQueueResponse, error) {
	q := url.Values{}
	q.Set("stock_code", params.StockCode)
	if params.ActionType != "" {
		q.Set("action_type", params.ActionType)
	}
	if params.BoardType != "" {
		q.Set("board_type", params.BoardType)
	}
	if params.Limit > 0 {
		q.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.OrderStatus != "" {
		q.Set("order_status", params.OrderStatus)
	}
	if params.Price > 0 {
		q.Set("price", strconv.FormatInt(params.Price, 10))
	}
	if params.SortBy != "" {
		q.Set("sort_by", params.SortBy)
	}
	if params.SortDirection != "" {
		q.Set("sort_direction", params.SortDirection)
	}
	var out OrderQueueResponse
	if err := c.Get(ctx, orderQueuePath, q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
