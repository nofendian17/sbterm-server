// Package questdb wraps the QuestDB Go client facade (QWP WebSocket protocol)
// for ingesting Stockbit datafeed frames into time-series tables.
//
// The client connects lazily: a QuestDB that is down at startup does not block
// the process. The running trade table schema is created on first contact and
// re-attempted (throttled) until it succeeds, so a server that comes up later
// is adopted before any writes are sent.
package questdb

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	qdb "github.com/questdb/go-questdb-client/v4"

	"github.com/nofendian17/sbterm/libs/pkg/log"
)

// DefaultTable is the running trade table created when config does not name one.
const DefaultTable = "running_trades"

// DefaultOrderBookTable is the order book table created when config does not
// name one.
const DefaultOrderBookTable = "order_books"

// schemaRetryInterval throttles CREATE TABLE attempts while the server is down.
const schemaRetryInterval = 5 * time.Second

// errSchemaPending reports that the schema retry is throttled and the batch
// should be skipped this round.
var errSchemaPending = errors.New("questdb: running trade table schema pending")

// errOrderBookSchemaPending reports that the order book schema retry is
// throttled and the frame should be skipped this round.
var errOrderBookSchemaPending = errors.New("questdb: order book table schema pending")

// errMalformedOrderBook reports an order book body that is not a #O snapshot.
var errMalformedOrderBook = errors.New("questdb: malformed order book body")

// Option configures a Client.
type Option func(*options)

type options struct {
	orderBookTable string
}

// WithOrderBookTable names the table used for order book level rows. An empty
// name keeps DefaultOrderBookTable.
func WithOrderBookTable(name string) Option {
	return func(o *options) { o.orderBookTable = name }
}

// Client is a handle to one QuestDB deployment. It is safe for concurrent use;
// the underlying facade owns pools of senders and query sessions.
type Client struct {
	db             *qdb.QuestDB
	table          string
	orderBookTable string
	logger         log.Logger

	mu           sync.Mutex
	schemaOK     map[string]bool
	lastSchemaAt map[string]time.Time
}

// New dials QuestDB over QWP and prepares the running trade and order book
// tables. The dial is lazy (lazy_connect), so a down server does not fail
// construction; each schema is created on first successful contact and
// re-attempted until it lands. An empty conf falls back to the local default
// endpoint; an empty table uses DefaultTable, and an empty order book table
// (or no WithOrderBookTable) uses DefaultOrderBookTable.
func New(ctx context.Context, conf, table string, logger log.Logger, opts ...Option) (*Client, error) {
	if conf == "" {
		conf = "ws::addr=localhost:9000;"
	}
	if table == "" {
		table = DefaultTable
	}
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	if o.orderBookTable == "" {
		o.orderBookTable = DefaultOrderBookTable
	}

	db, err := qdb.NewQuestDB(ctx, conf,
		qdb.WithLazyConnect(true),
		qdb.WithQuestDBErrorHandler(func(e *qdb.SenderError) {
			logger.Warn("questdb: server rejected ingest batch",
				"category", e.Category,
				"table", e.TableName,
				"message", e.ServerMessage,
			)
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("questdb: connect: %w", err)
	}

	c := &Client{
		db:             db,
		table:          table,
		orderBookTable: o.orderBookTable,
		logger:         logger,
		schemaOK:       make(map[string]bool),
		lastSchemaAt:   make(map[string]time.Time),
	}
	if err := c.ensureSchema(ctx, c.table, errSchemaPending); err != nil && !errors.Is(err, errSchemaPending) {
		logger.Warn("questdb: create running trade table failed; will retry", "error", err)
	}
	if err := c.ensureSchema(ctx, c.orderBookTable, errOrderBookSchemaPending); err != nil && !errors.Is(err, errOrderBookSchemaPending) {
		logger.Warn("questdb: create order book table failed; will retry", "error", err)
	}
	return c, nil
}

// NewRunningTradeBatchSink leases a QWP sender for one writer goroutine. The
// sink is not safe for concurrent use; callers must Close it to flush and
// return the sender to the pool.
func (c *Client) NewRunningTradeBatchSink(ctx context.Context) (*runningTradeBatchSink, error) {
	sender, err := c.db.BorrowSender(ctx)
	if err != nil {
		return nil, fmt.Errorf("questdb: borrow sender: %w", err)
	}
	qs, ok := sender.(qdb.QwpSender)
	if !ok {
		_ = sender.Close(ctx)
		return nil, errors.New("questdb: sender is not a QWP sender")
	}
	return &runningTradeBatchSink{client: c, sender: qs}, nil
}

// NewOrderBookSink leases a QWP sender for one writer goroutine. The sink is
// not safe for concurrent use; callers must Close it to flush and return the
// sender to the pool.
func (c *Client) NewOrderBookSink(ctx context.Context) (*orderBookSink, error) {
	sender, err := c.db.BorrowSender(ctx)
	if err != nil {
		return nil, fmt.Errorf("questdb: borrow sender: %w", err)
	}
	qs, ok := sender.(qdb.QwpSender)
	if !ok {
		_ = sender.Close(ctx)
		return nil, errors.New("questdb: sender is not a QWP sender")
	}
	return &orderBookSink{client: c, sender: qs}, nil
}

// Ping verifies the server answers a trivial query. With lazy_connect the pool
// connects on first use, so this both connects and checks the wire.
func (c *Client) Ping(ctx context.Context) error {
	q, err := c.db.BorrowQuery(ctx)
	if err != nil {
		return err
	}
	defer q.Close()
	cursor := q.Query(ctx, "SELECT 1")
	defer cursor.Close()
	for batch, err := range cursor.Batches() {
		if err != nil {
			return err
		}
		_ = batch
	}
	return nil
}

// HealthCheck implements the samber/do health check hook.
func (c *Client) HealthCheck(ctx context.Context) error {
	return c.Ping(ctx)
}

// Shutdown closes the handle and every pooled connection.
func (c *Client) Shutdown() error {
	return c.db.Close(context.Background())
}

// ensureSchema creates a table once the server is reachable. Concurrent
// callers share the result; failed attempts are throttled to at most one per
// schemaRetryInterval. pending is returned while the retry is throttled.
func (c *Client) ensureSchema(ctx context.Context, table string, pending error) error {
	c.mu.Lock()
	if c.schemaOK[table] {
		c.mu.Unlock()
		return nil
	}
	if time.Since(c.lastSchemaAt[table]) < schemaRetryInterval {
		c.mu.Unlock()
		return pending
	}
	c.lastSchemaAt[table] = time.Now()
	c.mu.Unlock()

	var err error
	switch table {
	case c.orderBookTable:
		err = c.createOrderBookTable(ctx)
	default:
		err = c.createTradeTable(ctx)
	}
	c.mu.Lock()
	if err == nil {
		c.schemaOK[table] = true
	}
	c.mu.Unlock()
	return err
}

func (c *Client) createTradeTable(ctx context.Context) error {
	q, err := c.db.BorrowQuery(ctx)
	if err != nil {
		return fmt.Errorf("questdb: borrow query: %w", err)
	}
	defer q.Close()

	// The designated timestamp column must be part of the dedup keys; the
	// keys (ts, symbol, trade_number) make a replayed trade collapse onto
	// itself when the QWP sender reconnects and replays an unacked frame.
	ddl := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
  ts TIMESTAMP,
  symbol SYMBOL,
  price DOUBLE,
  volume DOUBLE,
  action SYMBOL,
  is_global BOOLEAN,
  change_value DOUBLE,
  change_percentage DOUBLE,
  trade_number INT,
  market_board SYMBOL,
  value DOUBLE,
  websocket_time TIMESTAMP
) TIMESTAMP(ts) PARTITION BY DAY WAL DEDUP UPSERT KEYS (ts, symbol, trade_number)`, c.table)
	if _, err := q.Exec(ctx, ddl); err != nil {
		return fmt.Errorf("questdb: create table %s: %w", c.table, err)
	}
	return nil
}

// createOrderBookTable creates the order book level table. Each frame is a
// full side snapshot; every level becomes a row keyed by the frame timestamp,
// symbol, side, price, and sequence number, so a replayed frame collapses onto
// itself while distinct frames keep their full history.
func (c *Client) createOrderBookTable(ctx context.Context) error {
	q, err := c.db.BorrowQuery(ctx)
	if err != nil {
		return fmt.Errorf("questdb: borrow query: %w", err)
	}
	defer q.Close()

	ddl := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
  ts TIMESTAMP,
  symbol SYMBOL,
  side SYMBOL,
  price LONG,
  frequency LONG,
  shares LONG,
  sequence_number LONG,
  order_book_id LONG,
  receive_ts TIMESTAMP,
  board SYMBOL
) TIMESTAMP(ts) PARTITION BY DAY WAL DEDUP UPSERT KEYS (ts, symbol, side, price, sequence_number)`, c.orderBookTable)
	if _, err := q.Exec(ctx, ddl); err != nil {
		return fmt.Errorf("questdb: create table %s: %w", c.orderBookTable, err)
	}
	return nil
}
