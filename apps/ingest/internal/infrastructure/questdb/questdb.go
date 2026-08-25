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
	"strings"
	"sync"
	"time"

	qdb "github.com/questdb/go-questdb-client/v4"

	"github.com/nofendian17/sbterm/libs/pkg/log"
)

// DefaultTable is the running trade table created when config does not name one.
const DefaultTable = "running_trades"

// qwpAutoFlushRows batches buffered rows into QWP transactions of meaningful
// size (QuestDB recommends > 100 rows per transaction); the sender's default
// auto-flush interval (1s) still bounds latency on slow streams.
const qwpAutoFlushRows = 100

// schemaRetryInterval throttles CREATE TABLE attempts while the server is down.
const schemaRetryInterval = 5 * time.Second

// errSchemaPending reports that the schema retry is throttled and the batch
// should be skipped this round.
var errSchemaPending = errors.New("questdb: running trade table schema pending")

// Client is a handle to one QuestDB deployment. It is safe for concurrent use;
// the underlying facade owns pools of senders and query sessions.
type Client struct {
	db     *qdb.QuestDB
	table  string
	logger log.Logger

	orderBookTable string
	bookTTLDays    int

	mu           sync.Mutex
	schemaOK     map[string]bool
	lastSchemaAt map[string]time.Time
}

// New dials QuestDB over QWP and prepares the running trade table. The dial is
// lazy (lazy_connect), so a down server does not fail construction; the schema
// is created on first successful contact and re-attempted until it lands. An
// empty conf falls back to the local default endpoint; an empty table uses
// DefaultTable.
func New(ctx context.Context, conf, table string, logger log.Logger) (*Client, error) {
	if conf == "" {
		conf = "ws::addr=localhost:9000;"
	}
	if !strings.Contains(conf, "auto_flush_rows") {
		conf = fmt.Sprintf("%s;auto_flush_rows=%d", strings.TrimRight(conf, ";"), qwpAutoFlushRows)
	}
	if table == "" {
		table = DefaultTable
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
		db:           db,
		table:        table,
		logger:       logger,
		schemaOK:     make(map[string]bool),
		lastSchemaAt: make(map[string]time.Time),
	}
	if err := c.ensureSchema(ctx, c.table, errSchemaPending, c.createTradeTable); err != nil && !errors.Is(err, errSchemaPending) {
		logger.Warn("questdb: create running trade table failed; will retry", "error", err)
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
func (c *Client) ensureSchema(ctx context.Context, table string, pending error, create func(context.Context) error) error {
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

	err := create(ctx)
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

	// SYMBOL columns carry a lookup cache so filters on stock/market_board stay
	// fast as the table grows. The designated timestamp column must be part of
	// the dedup keys; (ts, stock, trade_number) make a replayed trade collapse
	// onto itself when the QWP sender reconnects and replays an unacked frame.
	ddl := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
  ts TIMESTAMP,
  websocket_ts TIMESTAMP,
  stock SYMBOL capacity 2048 CACHE INDEX,
  price DOUBLE,
  volume LONG,
  value DOUBLE,
  change_value DOUBLE,
  change_percentage DOUBLE,
  trade_number LONG,
  market_board SYMBOL capacity 10 CACHE,
  is_global BOOLEAN,
  action SYMBOL
) TIMESTAMP(ts) PARTITION BY DAY WAL DEDUP UPSERT KEYS (ts, stock, trade_number)`, c.table)
	if _, err := q.Exec(ctx, ddl); err != nil {
		return fmt.Errorf("questdb: create table %s: %w", c.table, err)
	}
	return nil
}

// createBookTable creates the order book snapshot table: one row per accepted
// snapshot with both sides' depth as 1-D arrays (article-compatible layout:
// bid_px[1] is the best bid). DEDUP keys collapse reconnect replays.
func (c *Client) createBookTable(ctx context.Context) error {
	q, err := c.db.BorrowQuery(ctx)
	if err != nil {
		return fmt.Errorf("questdb: borrow query: %w", err)
	}
	defer q.Close()

	ttl := c.bookTTLDays
	if ttl <= 0 {
		ttl = 30
	}
	ddl := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
  ts TIMESTAMP,
  receive_ts TIMESTAMP,
  symbol SYMBOL,
  board SYMBOL,
  bid_seq LONG,
  ask_seq LONG,
  bid_px DOUBLE[],
  bid_qty LONG[],
  ask_px DOUBLE[],
  ask_qty LONG[]
) TIMESTAMP(ts) PARTITION BY DAY WAL DEDUP UPSERT KEYS(ts, symbol, bid_seq, ask_seq) TTL %d DAYS`, c.orderBookTable, ttl)
	if _, err := q.Exec(ctx, ddl); err != nil {
		return fmt.Errorf("questdb: create table %s: %w", c.orderBookTable, err)
	}
	return nil
}
