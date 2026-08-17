package domain

// OrderQueueData is the order queue for a symbol at a price.
type OrderQueueData struct {
	Orders       []OrderQueueOrder
	IsOpenMarket bool
	Pagination   OrderQueuePagination
}

// OrderQueuePagination tells whether more pages are available after the
// requested limit.
type OrderQueuePagination struct {
	HasNextPage bool
}

// OrderQueueOrder is a single order sitting in the queue at a price level.
// QueueNumber is a string because upstream returns it as such for orders
// beyond nine digits.
type OrderQueueOrder struct {
	ID                  string
	QueueNumber         string
	StockCode           string
	Time                string
	ActionType          string
	Price               int64
	Status              string
	Open                int64
	Lot                 int64
	BoardType           string
	BrokerCode          string
	ExchangeOrderNumber OrderQueueExchangeOrderNumber
	QueueLot            int64
	BrokerGroup         string
	OrderNumber         string
}

// OrderQueueExchangeOrderNumber identifies an order at the exchange: Full is
// the complete order number, Formatted is its display form.
type OrderQueueExchangeOrderNumber struct {
	Full      string
	Formatted string
}
