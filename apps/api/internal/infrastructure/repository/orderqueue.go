package repository

import (
	"context"
	"errors"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
	"github.com/nofendian17/sbterm/libs/stockbit"
)

// OrderQueueRepository fetches order queues from the Stockbit API.
type OrderQueueRepository struct {
	client *stockbit.Client
}

func NewOrderQueueRepository(client *stockbit.Client) *OrderQueueRepository {
	return &OrderQueueRepository{client: client}
}

func (r *OrderQueueRepository) GetOrderQueue(ctx context.Context, stockCode, actionType, boardType, orderStatus, sortBy, sortDirection string, limit int, price int64) (*domain.OrderQueueData, error) {
	resp, err := r.client.GetOrderQueue(ctx, stockbit.OrderQueueParams{
		StockCode:     stockCode,
		ActionType:    actionType,
		BoardType:     boardType,
		Limit:         limit,
		OrderStatus:   orderStatus,
		Price:         price,
		SortBy:        sortBy,
		SortDirection: sortDirection,
	})
	if err != nil {
		var se *stockbit.StatusError
		if errors.As(err, &se) {
			return nil, &domain.UpstreamError{Status: se.Status, Msg: se.Msg}
		}
		return nil, err
	}
	d := resp.Data
	out := &domain.OrderQueueData{
		IsOpenMarket: d.IsOpenMarket,
		Orders:       make([]domain.OrderQueueOrder, 0, len(d.Orders)),
		Pagination:   domain.OrderQueuePagination{HasNextPage: d.Pagination.HasNextPage},
	}
	for _, o := range d.Orders {
		out.Orders = append(out.Orders, domain.OrderQueueOrder{
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
			ExchangeOrderNumber: domain.OrderQueueExchangeOrderNumber{
				Full:      o.ExchangeOrderNumber.Full,
				Formatted: o.ExchangeOrderNumber.Formatted,
			},
			QueueLot:    o.QueueLot,
			BrokerGroup: o.BrokerGroup,
			OrderNumber: o.OrderNumber,
		})
	}
	return out, nil
}

var _ repository.OrderQueueRepository = (*OrderQueueRepository)(nil)
