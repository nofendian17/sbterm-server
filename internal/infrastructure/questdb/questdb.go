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
	"strconv"
	"strings"
	"sync"
	"time"

	qdb "github.com/questdb/go-questdb-client/v4"

	consumerv1 "github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit/proto/securities/transactional/datafeed/consumer/entity/v1"
	datafeedv1 "github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit/proto/securities/transactional/datafeed/v1"
	"github.com/nofendian17/sbterm-server/pkg/log"
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

// RunningTradeBatchSink persists running trade batches on one writer goroutine.
// Implementations are not safe for concurrent use; Close flushes buffered rows
// and releases the underlying connection.
type RunningTradeBatchSink interface {
	Store(ctx context.Context, batch *datafeedv1.RunningTradeBatch) error
	Close(ctx context.Context) error
}

// RunningTradeBatchStore leases per-goroutine running trade batch sinks.
type RunningTradeBatchStore interface {
	NewRunningTradeBatchSink(ctx context.Context) (RunningTradeBatchSink, error)
}

// OrderBookSink persists one order book side snapshot on one writer goroutine.
// Implementations are not safe for concurrent use; Close flushes buffered rows
// and releases the underlying connection.
type OrderBookSink interface {
	Store(ctx context.Context, ob *consumerv1.Orderbook) error
	Close(ctx context.Context) error
}

// OrderBookStore leases per-goroutine order book sinks.
type OrderBookStore interface {
	NewOrderBookSink(ctx context.Context) (OrderBookSink, error)
}

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
func (c *Client) NewRunningTradeBatchSink(ctx context.Context) (RunningTradeBatchSink, error) {
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
func (c *Client) NewOrderBookSink(ctx context.Context) (OrderBookSink, error) {
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

// runningTradeBatchSink maps one RunningTradeBatch frame to rows on a leased
// sender. Each Store call flushes the batch so trades reach QuestDB promptly;
// the facade's auto-flush remains as a backstop for rows buffered in between.
type runningTradeBatchSink struct {
	client *Client
	sender qdb.QwpSender
}

// Store writes every trade in the batch as a row. While the table schema is
// pending the batch is skipped (logged at debug) instead of buffered, so the
// first data written always lands in the partitioned, deduplicated table.
func (s *runningTradeBatchSink) Store(ctx context.Context, batch *datafeedv1.RunningTradeBatch) error {
	if err := s.client.ensureSchema(ctx, s.client.table, errSchemaPending); err != nil {
		if errors.Is(err, errSchemaPending) {
			s.client.logger.Debug("questdb: running trade table schema pending; skipping batch")
			return nil
		}
		return err
	}
	for _, t := range batch.GetBatch() {
		if err := s.write(ctx, t); err != nil {
			return err
		}
	}
	// Publish the buffered rows now. The facade's auto-flush is used as a
	// backstop but observed to lag well past its documented interval, and
	// trade data is latency-sensitive.
	return s.sender.Flush(ctx)
}

// Close flushes buffered rows and returns the sender to the pool.
func (s *runningTradeBatchSink) Close(ctx context.Context) error {
	return s.sender.Close(ctx)
}

// write appends one trade row. Column order follows the QWP constraint: Table,
// then Symbols, then columns, then the row finalizer (At).
func (s *runningTradeBatchSink) write(ctx context.Context, t *datafeedv1.RunningTrade) error {
	s.sender.Table(s.client.table)
	s.sender.Symbol("symbol", t.GetStock())
	s.sender.Symbol("action", t.GetAction().String())
	s.sender.Symbol("market_board", t.GetMarketBoard().String())
	s.sender.Float64Column("price", t.GetPrice())
	s.sender.Float64Column("volume", t.GetVolume())
	s.sender.BoolColumn("is_global", t.GetIsGlobal())
	s.sender.Float64Column("change_value", changeValue(t))
	s.sender.Float64Column("change_percentage", changePercentage(t))
	s.sender.Int32Column("trade_number", t.GetTradeNumber())
	s.sender.Float64Column("value", t.GetValue())
	if ws := t.GetWebsocketTime(); ws != nil && ws.IsValid() {
		s.sender.TimestampColumn("websocket_time", ws.AsTime())
	}
	return s.sender.At(ctx, tradeTimestamp(t))
}

// tradeTimestamp picks the row's designated timestamp: the trade time when
// present, else the websocket receive time, else now.
func tradeTimestamp(t *datafeedv1.RunningTrade) time.Time {
	if ts := t.GetTime(); ts != nil && ts.IsValid() {
		return ts.AsTime()
	}
	if ws := t.GetWebsocketTime(); ws != nil && ws.IsValid() {
		return ws.AsTime()
	}
	return time.Now()
}

func changeValue(t *datafeedv1.RunningTrade) float64 {
	if c := t.GetChange(); c != nil {
		return c.GetValue()
	}
	return 0
}

func changePercentage(t *datafeedv1.RunningTrade) float64 {
	if c := t.GetChange(); c != nil {
		return c.GetPercentage()
	}
	return 0
}

// orderBookLevel is one price level of an order book side snapshot.
type orderBookLevel struct {
	Price     int64
	Frequency int64
	Shares    int64
}

// parseOrderBookBody decodes a datafeed order book body of the form
// "#O|<symbol>|<side>|<price>;<frequency>;<shares>|...". A trailing token
// after the levels (exchange sequence & timestamp) does not match the
// three-integer level shape and is skipped. Malformed levels are skipped;
// a body that is not an order book snapshot is an error.
func parseOrderBookBody(body string) (string, []orderBookLevel, error) {
	parts := strings.Split(body, "|")
	if len(parts) < 3 || parts[0] != "#O" || parts[2] == "" {
		return "", nil, errMalformedOrderBook
	}
	side := parts[2]
	var levels []orderBookLevel
	for _, part := range parts[3:] {
		fields := strings.Split(part, ";")
		if len(fields) != 3 {
			continue
		}
		price, err := strconv.ParseInt(strings.TrimSpace(fields[0]), 10, 64)
		if err != nil {
			continue
		}
		frequency, err := strconv.ParseInt(strings.TrimSpace(fields[1]), 10, 64)
		if err != nil {
			continue
		}
		shares, err := strconv.ParseInt(strings.TrimSpace(fields[2]), 10, 64)
		if err != nil {
			continue
		}
		levels = append(levels, orderBookLevel{Price: price, Frequency: frequency, Shares: shares})
	}
	return side, levels, nil
}

// orderBookTimestamp picks the order book snapshot timestamp: the exchange
// datetime when present, else the itch incoming time, else now.
func orderBookTimestamp(ob *consumerv1.Orderbook) time.Time {
	for _, raw := range []string{ob.GetDatetime(), ob.GetItchIncomingTime()} {
		if ts, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			return ts
		}
	}
	return time.Now()
}

// orderBookReceiveTime returns the receive-side timestamp of the frame when
// present.
func orderBookReceiveTime(ob *consumerv1.Orderbook) (time.Time, bool) {
	if ts, err := time.Parse(time.RFC3339Nano, ob.GetItchIncomingTime()); err == nil {
		return ts, true
	}
	return time.Time{}, false
}

// orderBookSink maps one order book side snapshot to per-level rows on a
// leased sender. Each Store call flushes the frame so the book reaches QuestDB
// promptly; the facade's auto-flush remains as a backstop.
type orderBookSink struct {
	client *Client
	sender qdb.QwpSender
}

// Store writes every price level in the snapshot as a row. While the table
// schema is pending the frame is skipped (logged at debug) instead of
// buffered, so the first data written always lands in the partitioned,
// deduplicated table.
func (s *orderBookSink) Store(ctx context.Context, ob *consumerv1.Orderbook) error {
	if err := s.client.ensureSchema(ctx, s.client.orderBookTable, errOrderBookSchemaPending); err != nil {
		if errors.Is(err, errOrderBookSchemaPending) {
			s.client.logger.Debug("questdb: order book table schema pending; skipping frame")
			return nil
		}
		return err
	}
	side, levels, err := parseOrderBookBody(ob.GetBody())
	if err != nil {
		return fmt.Errorf("questdb: %w: %q", err, ob.GetBody())
	}
	ts := orderBookTimestamp(ob)
	for _, lvl := range levels {
		if err := s.write(ctx, ob, side, lvl, ts); err != nil {
			return err
		}
	}
	return s.sender.Flush(ctx)
}

// Close flushes buffered rows and returns the sender to the pool.
func (s *orderBookSink) Close(ctx context.Context) error {
	return s.sender.Close(ctx)
}

// write appends one order book level row. Column order follows the QWP
// constraint: Table, then Symbols, then columns, then the row finalizer (At).
func (s *orderBookSink) write(ctx context.Context, ob *consumerv1.Orderbook, side string, lvl orderBookLevel, ts time.Time) error {
	s.sender.Table(s.client.orderBookTable)
	s.sender.Symbol("symbol", ob.GetStockCode())
	s.sender.Symbol("side", side)
	s.sender.Symbol("board", ob.GetBoard().String())
	s.sender.Int64Column("price", lvl.Price)
	s.sender.Int64Column("frequency", lvl.Frequency)
	s.sender.Int64Column("shares", lvl.Shares)
	s.sender.Int64Column("sequence_number", ob.GetSequenceNumber())
	s.sender.Int64Column("order_book_id", ob.GetOrderBookId())
	if receiveTS, ok := orderBookReceiveTime(ob); ok {
		s.sender.TimestampColumn("receive_ts", receiveTS)
	}
	return s.sender.At(ctx, ts)
}
