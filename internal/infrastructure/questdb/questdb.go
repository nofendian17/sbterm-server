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

	datafeedv1 "github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit/proto/securities/transactional/datafeed/v1"
	"github.com/nofendian17/sbterm-server/pkg/log"
)

// DefaultTable is the running trade table created when config does not name one.
const DefaultTable = "running_trades"

// schemaRetryInterval throttles CREATE TABLE attempts while the server is down.
const schemaRetryInterval = 5 * time.Second

// errSchemaPending reports that the schema retry is throttled and the batch
// should be skipped this round.
var errSchemaPending = errors.New("questdb: running trade table schema pending")

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

// Client is a handle to one QuestDB deployment. It is safe for concurrent use;
// the underlying facade owns pools of senders and query sessions.
type Client struct {
	db     *qdb.QuestDB
	table  string
	logger log.Logger

	mu         sync.Mutex
	schemaOK   bool
	lastSchema time.Time
}

// New dials QuestDB over QWP and prepares the running trade table. The dial is
// lazy (lazy_connect), so a down server does not fail construction; the schema
// is created on first successful contact and re-attempted until it lands.
// An empty conf falls back to the local default endpoint; an empty table uses
// DefaultTable.
func New(ctx context.Context, conf, table string, logger log.Logger) (*Client, error) {
	if conf == "" {
		conf = "ws::addr=localhost:9000;"
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

	c := &Client{db: db, table: table, logger: logger}
	if err := c.ensureSchema(ctx); err != nil && !errors.Is(err, errSchemaPending) {
		logger.Warn("questdb: create running trade table failed; will retry", "error", err)
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

// ensureSchema creates the running trade table once the server is reachable.
// Concurrent callers share the result; failed attempts are throttled to at most
// one per schemaRetryInterval.
func (c *Client) ensureSchema(ctx context.Context) error {
	c.mu.Lock()
	if c.schemaOK {
		c.mu.Unlock()
		return nil
	}
	if time.Since(c.lastSchema) < schemaRetryInterval {
		c.mu.Unlock()
		return errSchemaPending
	}
	c.lastSchema = time.Now()
	c.mu.Unlock()

	err := c.createTable(ctx)
	c.mu.Lock()
	if err == nil {
		c.schemaOK = true
	}
	c.mu.Unlock()
	return err
}

func (c *Client) createTable(ctx context.Context) error {
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
	if err := s.client.ensureSchema(ctx); err != nil {
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
