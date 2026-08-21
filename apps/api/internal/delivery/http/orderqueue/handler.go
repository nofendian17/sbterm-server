package orderqueue

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/response"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

// Default values applied when the corresponding query param is omitted.
const (
	defaultActionType    = "ACTION_TYPE_ALL"
	defaultBoardType     = "BOARD_TYPE_REGULAR"
	defaultOrderStatus   = "ORDER_STATUS_OPEN"
	defaultSortBy        = "SORT_BY_QUEUE"
	defaultSortDirection = "SORT_DIRECTION_ASC"
	defaultLimit         = 100
)

type OrderQueueHandler struct {
	uc usecase.OrderQueueUsecase
	v  validator.Validator
}

func NewOrderQueueHandler(uc usecase.OrderQueueUsecase, v validator.Validator) *OrderQueueHandler {
	return &OrderQueueHandler{uc: uc, v: v}
}

type orderQueueRequest struct {
	StockCode     string `json:"stock_code" validate:"required"`
	ActionType    string `json:"action_type" validate:"omitempty,oneof=ACTION_TYPE_BUY ACTION_TYPE_SELL ACTION_TYPE_ALL"`
	BoardType     string `json:"board_type" validate:"omitempty,oneof=BOARD_TYPE_REGULAR BOARD_TYPE_NEGOTIATION BOARD_TYPE_CASH BOARD_TYPE_ALL"`
	OrderStatus   string `json:"order_status" validate:"omitempty,oneof=ORDER_STATUS_OPEN ORDER_STATUS_FULL_MATCH ORDER_STATUS_WITHDRAWN ORDER_STATUS_PARTIAL_MATCH ORDER_STATUS_AMEND ORDER_STATUS_ALL"`
	SortBy        string `json:"sort_by" validate:"omitempty,oneof=SORT_BY_QUEUE"`
	SortDirection string `json:"sort_direction" validate:"omitempty,oneof=SORT_DIRECTION_ASC SORT_DIRECTION_DESC"`
	Price         int64
	Limit         int `validate:"min=1"`
}

type orderQueueResponse struct {
	IsOpenMarket bool                 `json:"is_open_market"`
	Orders       []orderQueueItemResp `json:"orders"`
	Pagination   orderQueuePageResp   `json:"pagination"`
}

type orderQueueItemResp struct {
	ID                  string                    `json:"id"`
	QueueNumber         string                    `json:"queue_number"`
	StockCode           string                    `json:"stock_code"`
	Time                string                    `json:"time"`
	ActionType          string                    `json:"action_type"`
	Price               int64                     `json:"price"`
	Status              string                    `json:"status"`
	Open                int64                     `json:"open"`
	Lot                 int64                     `json:"lot"`
	BoardType           string                    `json:"board_type"`
	BrokerCode          string                    `json:"broker_code"`
	ExchangeOrderNumber orderQueueOrderNumberResp `json:"exchange_order_number"`
	QueueLot            int64                     `json:"queue_lot"`
	BrokerGroup         string                    `json:"broker_group"`
	OrderNumber         string                    `json:"order_number"`
}

type orderQueueOrderNumberResp struct {
	Full      string `json:"full"`
	Formatted string `json:"formatted"`
}

type orderQueuePageResp struct {
	HasNextPage bool `json:"has_next_page"`
}

func (h *OrderQueueHandler) OrderQueue(w http.ResponseWriter, r *http.Request) {
	req := orderQueueRequest{
		StockCode:     r.URL.Query().Get("stock_code"),
		ActionType:    r.URL.Query().Get("action_type"),
		BoardType:     r.URL.Query().Get("board_type"),
		OrderStatus:   r.URL.Query().Get("order_status"),
		SortBy:        r.URL.Query().Get("sort_by"),
		SortDirection: r.URL.Query().Get("sort_direction"),
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		if r.URL.Query().Get("limit") != "" {
			response.ValidationError(w, "validation failed", map[string]string{"limit": "must be a valid integer"})
			return
		}
		limit = defaultLimit
	}
	req.Limit = limit

	price, err := strconv.ParseInt(r.URL.Query().Get("price"), 10, 64)
	if err != nil {
		if r.URL.Query().Get("price") != "" {
			response.ValidationError(w, "validation failed", map[string]string{"price": "must be a valid integer"})
			return
		}
		price = 0
	}
	req.Price = price

	if req.ActionType == "" {
		req.ActionType = defaultActionType
	}
	if req.BoardType == "" {
		req.BoardType = defaultBoardType
	}
	if req.OrderStatus == "" {
		req.OrderStatus = defaultOrderStatus
	}
	if req.SortBy == "" {
		req.SortBy = defaultSortBy
	}
	if req.SortDirection == "" {
		req.SortDirection = defaultSortDirection
	}

	if err := h.v.Validate(req); err != nil {
		if verr, ok := validator.AsValidationError(err); ok {
			response.ValidationError(w, "validation failed", verr.Fields)
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to validate order queue params")
		return
	}

	data, err := h.uc.GetOrderQueue(r.Context(), req.StockCode, req.ActionType, req.BoardType, req.OrderStatus, req.SortBy, req.SortDirection, req.Limit, req.Price)
	if err != nil {
		var upErr *domain.UpstreamError
		if errors.As(err, &upErr) && response.Upstream(w, upErr.Status, upErr.RetryAfter, "no order queue data for the requested parameters", "failed to get order queue") {
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get order queue")
		return
	}
	response.OK(w, toOrderQueueResponse(data))
}

func toOrderQueueResponse(d *domain.OrderQueueData) orderQueueResponse {
	out := orderQueueResponse{
		IsOpenMarket: d.IsOpenMarket,
		Orders:       make([]orderQueueItemResp, 0, len(d.Orders)),
		Pagination:   orderQueuePageResp{HasNextPage: d.Pagination.HasNextPage},
	}
	for _, o := range d.Orders {
		out.Orders = append(out.Orders, orderQueueItemResp{
			ID:          o.ID,
			QueueNumber: o.QueueNumber,
			StockCode:   o.StockCode,
			Time:        o.Time,
			ActionType:  o.ActionType,
			Price:       o.Price,
			Status:      o.Status,
			Open:        o.Open,
			Lot:         o.Lot,
			BoardType:   o.BoardType,
			BrokerCode:  o.BrokerCode,
			ExchangeOrderNumber: orderQueueOrderNumberResp{
				Full:      o.ExchangeOrderNumber.Full,
				Formatted: o.ExchangeOrderNumber.Formatted,
			},
			QueueLot:    o.QueueLot,
			BrokerGroup: o.BrokerGroup,
			OrderNumber: o.OrderNumber,
		})
	}
	return out
}
