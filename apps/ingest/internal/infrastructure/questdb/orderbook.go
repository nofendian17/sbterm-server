package questdb

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	qdb "github.com/questdb/go-questdb-client/v4"

	consumerv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/consumer/entity/v1"
)

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
