package questdb

import (
	"context"
	"errors"
	"time"

	qdb "github.com/questdb/go-questdb-client/v4"

	datafeedv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/v1"
)

// runningTradeBatchSink maps one RunningTradeBatch frame to rows on a leased
// sender. Each Store call flushes the batch so trades reach QuestDB promptly;
// the facade's auto-flush remains as a backstop for rows buffered in between.
type runningTradeBatchSink struct {
	client *Client
	sender qdb.QwpSender
}

// Store writes every trade in the batch as a row. While the table schema is
// pending the batch is skipped (logged at debug) instead of buffered, so the
// first data written always lands in the partitioned, deduplicated table. Rows
// are left buffered for the sender's auto-flush (qwpAutoFlushRows) so each QWP
// transaction carries a meaningful number of rows; Close flushes the tail.
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
	return nil
}

// Close flushes buffered rows and returns the sender to the pool.
func (s *runningTradeBatchSink) Close(ctx context.Context) error {
	return s.sender.Close(ctx)
}

// write appends one trade row. Column order follows the QWP constraint: Table,
// then Symbols, then columns, then the row finalizer (At).
func (s *runningTradeBatchSink) write(ctx context.Context, t *datafeedv1.RunningTrade) error {
	s.sender.Table(s.client.table)
	s.sender.Symbol("stock", t.GetStock())
	s.sender.Symbol("action", t.GetAction().String())
	s.sender.Symbol("market_board", t.GetMarketBoard().String())
	s.sender.Float64Column("price", t.GetPrice())
	s.sender.Int64Column("volume", int64(t.GetVolume()))
	s.sender.BoolColumn("is_global", t.GetIsGlobal())
	s.sender.Float64Column("change_value", changeValue(t))
	s.sender.Float64Column("change_percentage", changePercentage(t))
	s.sender.Int64Column("trade_number", int64(t.GetTradeNumber()))
	s.sender.Float64Column("value", t.GetValue())
	if ws := t.GetWebsocketTime(); ws != nil && ws.IsValid() {
		s.sender.TimestampColumn("websocket_ts", ws.AsTime())
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
