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

// errMalformedOrderBook reports an order book body that is not an #O snapshot.
var errMalformedOrderBook = errors.New("questdb: malformed order book body")

// Side identifies one side of the order book.
type Side uint8

const (
	SideUnknown Side = iota
	SideBid
	SideAsk
)

// sideOf normalizes the side token embedded in the #O body. Upstream has been
// observed with terse and verbose spellings; unknown tokens surface to the
// caller instead of being guessed.
func sideOf(token string) Side {
	switch strings.ToUpper(strings.TrimSpace(token)) {
	case "B", "BID", "BUY":
		return SideBid
	case "A", "ASK", "SELL", "OFFER":
		return SideAsk
	default:
		return SideUnknown
	}
}

// bookLevel is one price level of an order book side snapshot.
type bookLevel struct {
	Price     int64
	Frequency int64
	Shares    int64
}

// splitBody parses a body of the form
// "#O|<symbol>|<side>|<price>;<frequency>;<shares>|..." into its symbol, raw
// side token, and levels. Malformed level triplets are skipped; a body that is
// not an order book snapshot errors.
func splitBody(body string) (string, string, []bookLevel, error) {
	parts := strings.Split(body, "|")
	if len(parts) < 3 || parts[0] != "#O" || parts[2] == "" {
		return "", "", nil, errMalformedOrderBook
	}
	levels := make([]bookLevel, 0, len(parts)-3)
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
		levels = append(levels, bookLevel{Price: price, Frequency: frequency, Shares: shares})
	}
	return parts[1], parts[2], levels, nil
}

// BookHalf is one stored side of a symbol's book: capped top levels in book
// order (best first).
type BookHalf struct {
	Seq    int64
	Prices []float64
	Qtys   []int64
}

// BookPair pairs both sides of one symbol's book as known at ExchangeTS.
type BookPair struct {
	Symbol     string
	Board      consumerv1.BoardType
	Bid, Ask   *BookHalf
	ExchangeTS time.Time
	ReceiveTS  time.Time
}

type symbolState struct {
	board            consumerv1.BoardType
	bid, ask         *BookHalf
	bidTS, askTS     time.Time
	bidRecv, askRecv time.Time
}

// Combiner keeps the latest half per side per symbol so every incoming frame
// can be persisted as a complete bid/ask pair. It is not safe for concurrent
// use; one combiner serves one consumer goroutine.
type Combiner struct {
	maxLevels int
	states    map[string]*symbolState
}

// NewCombiner caps how many price levels are retained per side.
func NewCombiner(maxLevels int) *Combiner {
	if maxLevels <= 0 {
		maxLevels = 25
	}
	return &Combiner{maxLevels: maxLevels, states: make(map[string]*symbolState)}
}

// Observe ingests one upstream order book frame. It reports a pair when both
// sides of the symbol are known: the arriving side fresh, the other riding
// along from its last snapshot. Stale replays (sequence going backwards) are
// dropped.
func (c *Combiner) Observe(ob *consumerv1.Orderbook, receiveTS time.Time) (*BookPair, bool) {
	symbol, rawSide, levels, err := splitBody(ob.GetBody())
	if err != nil || len(levels) == 0 {
		return nil, false
	}
	side := sideOf(rawSide)
	if side == SideUnknown {
		return nil, false
	}
	pair := c.apply(symbol, side, ob.GetBoard(), ob.GetSequenceNumber(),
		orderBookTimestamp(ob, receiveTS), receiveTS, levels)
	return pair, pair != nil
}

func (c *Combiner) apply(symbol string, side Side, board consumerv1.BoardType,
	seq int64, exchangeTS, receiveTS time.Time, levels []bookLevel) *BookPair {

	half := &BookHalf{
		Seq:    seq,
		Prices: make([]float64, 0, len(levels)),
		Qtys:   make([]int64, 0, len(levels)),
	}
	for i, lvl := range levels {
		if i >= c.maxLevels {
			break
		}
		half.Prices = append(half.Prices, float64(lvl.Price))
		half.Qtys = append(half.Qtys, lvl.Shares)
	}

	state := c.states[symbol]
	if state == nil {
		state = &symbolState{}
		c.states[symbol] = state
	}
	state.board = board

	switch side {
	case SideBid:
		if state.bid != nil && seq <= state.bid.Seq {
			return nil // stale replay
		}
		state.bid, state.bidTS, state.bidRecv = half, exchangeTS, receiveTS
	case SideAsk:
		if state.ask != nil && seq <= state.ask.Seq {
			return nil
		}
		state.ask, state.askTS, state.askRecv = half, exchangeTS, receiveTS
	default:
		return nil
	}

	if state.bid == nil || state.ask == nil {
		return nil
	}
	return &BookPair{
		Symbol:     symbol,
		Board:      state.board,
		Bid:        state.bid,
		Ask:        state.ask,
		ExchangeTS: later(state.bidTS, state.askTS),
		ReceiveTS:  later(state.bidRecv, state.askRecv),
	}
}

// orderBookTimestamp picks the snapshot timestamp: the exchange datetime when
// present, else the itch incoming time, else the receive-side fallback.
func orderBookTimestamp(ob *consumerv1.Orderbook, fallback time.Time) time.Time {
	for _, raw := range []string{ob.GetDatetime(), ob.GetItchIncomingTime()} {
		if ts, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			return ts
		}
	}
	return fallback
}

func later(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

// errBookSchemaPending reports that the order book table schema retry is
// throttled and the pair should be skipped this round.
var errBookSchemaPending = errors.New("questdb: order book table schema pending")

// UseOrderBookTable enables the ob_book sink on this client: table name and
// retention days come from config. Safe to call before any Store.
func (c *Client) UseOrderBookTable(table string, ttlDays int) *Client {
	if strings.TrimSpace(table) == "" {
		table = "ob_book"
	}
	if ttlDays <= 0 {
		ttlDays = 30
	}
	c.mu.Lock()
	c.orderBookTable = table
	c.bookTTLDays = ttlDays
	delete(c.schemaOK, table)
	c.mu.Unlock()
	return c
}

// NewOrderBookSink leases a QWP sender for one writer goroutine. The sink is
// not safe for concurrent use; callers must Close it to flush and return the
// sender to the pool.
func (c *Client) NewOrderBookSink(ctx context.Context) (*bookPairSink, error) {
	sender, err := c.db.BorrowSender(ctx)
	if err != nil {
		return nil, fmt.Errorf("questdb: borrow sender: %w", err)
	}
	qs, ok := sender.(qdb.QwpSender)
	if !ok {
		_ = sender.Close(ctx)
		return nil, errors.New("questdb: sender is not a QWP sender")
	}
	return &bookPairSink{client: c, sender: qs}, nil
}

// bookPairSink persists BookPair rows — one row per accepted snapshot, depth
// as 1-D arrays per side.
type bookPairSink struct {
	client *Client
	sender qdb.QwpSender
}

// Store writes the pair. While the table schema is pending the pair is
// skipped (logged at debug) instead of buffered, matching the running trade
// sink contract.
func (s *bookPairSink) Store(ctx context.Context, pair *BookPair) error {
	c := s.client
	if err := c.ensureSchema(ctx, c.orderBookTable, errBookSchemaPending, c.createBookTable); err != nil {
		if errors.Is(err, errBookSchemaPending) {
			c.logger.Debug("questdb: order book table schema pending; skipping pair")
			return nil
		}
		return err
	}

	snd := s.sender
	snd.Table(c.orderBookTable)
	snd.Symbol("symbol", pair.Symbol)
	snd.Symbol("board", pair.Board.String())
	snd.Int64Column("bid_seq", pair.Bid.Seq)
	snd.Int64Column("ask_seq", pair.Ask.Seq)
	// QWP column order: symbols first, then typed columns, then the row
	// finalizer (At).
	snd.Float64Array1DColumn("bid_px", pair.Bid.Prices)
	snd.Float64Array1DColumn("bid_qty", toFloat64(pair.Bid.Qtys))
	snd.Float64Array1DColumn("ask_px", pair.Ask.Prices)
	snd.Float64Array1DColumn("ask_qty", toFloat64(pair.Ask.Qtys))
	if !pair.ReceiveTS.IsZero() {
		snd.TimestampColumn("receive_ts", pair.ReceiveTS)
	}
	ts := pair.ExchangeTS
	if ts.IsZero() {
		ts = pair.ReceiveTS
	}
	return snd.At(ctx, ts)
}

// Close flushes buffered rows and returns the sender to the pool.
func (s *bookPairSink) Close(ctx context.Context) error {
	return s.sender.Close(ctx)
}

// toFloat64 widens lot counts for QuestDB arrays: the server's ARRAY type
// does not accept LONG elements, and lot counts below 2^53 are exact in
// float64.
func toFloat64(in []int64) []float64 {
	out := make([]float64, len(in))
	for i, v := range in {
		out[i] = float64(v)
	}
	return out
}
