// Package hotstate keeps the live order book state in Redis: the latest
// snapshot per symbol, refreshed on every frame so key presence tracks
// liveness while stored levels survive quiet-but-valid stretches. Keys carry
// a long TTL (default 24h); staleness decisions belong to readers via the
// embedded ts_receive, not to expiry.
package hotstate

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Side is one side of the current book.
type Side struct {
	Seq    int64
	Prices []float64
	Qtys   []int64
}

// BookUpdate carries a complete bid/ask pair for one symbol as known at
// ReceiveTS.
type BookUpdate struct {
	Symbol    string
	Board     string
	Bid, Ask  *Side
	ReceiveTS time.Time
}

// Store writes the hot-state keys under "<prefix>:...". It is safe for
// concurrent use; one writer goroutine per symbol is expected because Kafka
// partitions by stock code.
type Store struct {
	rdb    redis.Cmdable
	prefix string
	ttl    time.Duration
}

// NewStore builds a store over a redis client. A non-positive ttl falls back
// to 24h; an empty prefix defaults to "ob".
func NewStore(rdb redis.Cmdable, prefix string, ttl time.Duration) *Store {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	if strings.TrimSpace(prefix) == "" {
		prefix = "ob"
	}
	return &Store{rdb: rdb, prefix: prefix, ttl: ttl}
}

func (s *Store) bookKey(symbol string) string { return s.prefix + ":book:" + strings.ToUpper(symbol) }
func (s *Store) hashKey(symbol string) string { return s.prefix + ":hash:" + strings.ToUpper(symbol) }

// SetBook replaces the symbol's book hash with the update and applies the
// full TTL. Field names index levels from zero (bid_px_0 is the best bid).
func (s *Store) SetBook(ctx context.Context, u BookUpdate) error {
	fields := make(map[string]any, 8+
		len(u.Bid.Prices)*2+len(u.Ask.Prices)*2)
	fields["board"] = u.Board
	if !u.ReceiveTS.IsZero() {
		fields["ts_receive"] = u.ReceiveTS.UTC().Format(time.RFC3339Nano)
	}
	if u.Bid != nil {
		fields["bid_seq"] = u.Bid.Seq
		for i := range u.Bid.Prices {
			fields["bid_px_"+strconv.Itoa(i)] = strconv.FormatFloat(u.Bid.Prices[i], 'f', -1, 64)
			if i < len(u.Bid.Qtys) {
				fields["bid_qty_"+strconv.Itoa(i)] = u.Bid.Qtys[i]
			}
		}
	}
	if u.Ask != nil {
		fields["ask_seq"] = u.Ask.Seq
		for i := range u.Ask.Prices {
			fields["ask_px_"+strconv.Itoa(i)] = strconv.FormatFloat(u.Ask.Prices[i], 'f', -1, 64)
			if i < len(u.Ask.Qtys) {
				fields["ask_qty_"+strconv.Itoa(i)] = u.Ask.Qtys[i]
			}
		}
	}

	key := s.bookKey(u.Symbol)
	pipe := s.rdb.Pipeline()
	pipe.Del(ctx, key)
	pipe.HSet(ctx, key, fields)
	pipe.Expire(ctx, key, s.ttl)
	_, err := pipe.Exec(ctx)
	return err
}

// TouchBook refreshes liveness for a symbol without rewriting its levels:
// hash-skipped frames call this so TTL measures "time since last sign of
// life", never "time since the book changed".
func (s *Store) TouchBook(ctx context.Context, symbol string, receiveTS time.Time) error {
	key := s.bookKey(symbol)
	pipe := s.rdb.Pipeline()
	pipe.HSet(ctx, key, "ts_receive", receiveTS.UTC().Format(time.RFC3339Nano))
	pipe.Expire(ctx, key, s.ttl)
	_, err := pipe.Exec(ctx)
	return err
}

// SeenBefore reports whether this exact body hash was already processed for
// the symbol. Hash marks are written without TTL: they exist to keep
// deduplication stable across process restarts.
func (s *Store) SeenBefore(ctx context.Context, symbol, bodyHash string) (bool, error) {
	prev, err := s.rdb.Get(ctx, s.hashKey(symbol)).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return prev == bodyHash, nil
}

// MarkSeen records the body hash of the last processed frame.
func (s *Store) MarkSeen(ctx context.Context, symbol, bodyHash string) error {
	return s.rdb.Set(ctx, s.hashKey(symbol), bodyHash, 0).Err()
}
