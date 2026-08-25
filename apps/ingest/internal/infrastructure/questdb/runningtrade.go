package questdb

import (
	"context"
	"errors"
	"fmt"

	qdb "github.com/questdb/go-questdb-client/v4"

	"github.com/nofendian17/sbterm/libs/marketdata"
	datafeedv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/v1"
)

// runningTradeBatchSink maps one RunningTradeBatch frame to rows on a leased
// sender. Each Store call flushes the batch so trades reach QuestDB promptly;
// the facade's auto-flush remains as a backstop for rows buffered in between.
type runningTradeBatchSink struct {
	client *Client
	sender qdb.QwpSender
}

// Store writes every trade in the batch as a row and flushes the sender so the
// rows are pushed to QuestDB before the call returns. The returned error (from
// either a schema wait or a flush) signals that the batch was not durably
// persisted; the ingest loop uses that to withhold the Kafka offset and force
// redelivery. While the table schema is pending the batch is skipped (logged at
// debug) instead of buffered, so the first data written always lands in the
// partitioned, deduplicated table.
func (s *runningTradeBatchSink) Store(ctx context.Context, batch *datafeedv1.RunningTradeBatch) error {
	if err := s.client.ensureSchema(ctx, s.client.table, errSchemaPending, s.client.createTradeTable); err != nil {
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
	if err := s.sender.Flush(ctx); err != nil {
		return fmt.Errorf("questdb: flush running trade batch: %w", err)
	}
	return nil
}

// Close flushes buffered rows and returns the sender to the pool.
func (s *runningTradeBatchSink) Close(ctx context.Context) error {
	return s.sender.Close(ctx)
}

// write appends one trade row. Column order follows the QWP constraint: Table,
// then Symbols, then columns, then the row finalizer (At). The projection
// comes from marketdata.NewTrade — the same mapper apps/stream serializes into
// WebSocket envelopes, so a row and a client frame can never drift.
func (s *runningTradeBatchSink) write(ctx context.Context, t *datafeedv1.RunningTrade) error {
	row := marketdata.NewTrade(t)
	s.sender.Table(s.client.table)
	s.sender.Symbol("stock", row.Stock)
	s.sender.Symbol("action", row.Action)
	s.sender.Symbol("market_board", row.MarketBoard)
	s.sender.Float64Column("price", row.Price)
	s.sender.Int64Column("volume", row.Volume)
	s.sender.BoolColumn("is_global", row.IsGlobal)
	s.sender.Float64Column("change_value", row.ChangeValue)
	s.sender.Float64Column("change_percentage", row.ChangePercentage)
	s.sender.Int64Column("trade_number", row.TradeNumber)
	s.sender.Float64Column("value", row.Value)
	if row.WebsocketTS != nil {
		s.sender.TimestampColumn("websocket_ts", *row.WebsocketTS)
	}
	return s.sender.At(ctx, row.TS)
}
