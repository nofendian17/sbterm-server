// Package marketdata holds the canonical projections shared by the pipeline
// apps, so every consumer of one datafeed frame sees the identical shape.
//
// Trade mirrors exactly what apps/ingest persists into the QuestDB
// running_trades table (see its questdb sink): enum fields as their string
// names, computed change_value/change_percentage, and the same timestamp
// fallback chain. The WebSocket fan-out (apps/stream) serializes this struct,
// so a client envelope matches a database row by construction — the two
// mappers must never drift because both call NewTrade.
package marketdata

import (
	"time"

	datafeedv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/v1"
)

// Trade is the canonical projection of one RunningTrade. JSON field names
// match the QuestDB column names 1:1.
type Trade struct {
	Stock            string     `json:"stock"`
	Action           string     `json:"action"`
	MarketBoard      string     `json:"market_board"`
	Price            float64    `json:"price"`
	Volume           int64      `json:"volume"`
	IsGlobal         bool       `json:"is_global"`
	ChangeValue      float64    `json:"change_value"`
	ChangePercentage float64    `json:"change_percentage"`
	TradeNumber      int64      `json:"trade_number"`
	Value            float64    `json:"value"`
	WebsocketTS      *time.Time `json:"websocket_ts,omitempty"` // nil = frame carried no websocket_time
	TS               time.Time  `json:"ts"`
}

// NewTrade projects one proto trade onto Trade. TS follows the sink's
// designated-timestamp rule: trade time, else websocket receive time, else now
// (so degenerate frames still land on a sane timeline).
func NewTrade(t *datafeedv1.RunningTrade) Trade {
	out := Trade{
		Stock:            t.GetStock(),
		Action:           t.GetAction().String(),
		MarketBoard:      t.GetMarketBoard().String(),
		Price:            t.GetPrice(),
		Volume:           int64(t.GetVolume()),
		IsGlobal:         t.GetIsGlobal(),
		ChangeValue:      changeValue(t),
		ChangePercentage: changePercentage(t),
		TradeNumber:      int64(t.GetTradeNumber()),
		Value:            t.GetValue(),
	}
	if ws := t.GetWebsocketTime(); ws != nil && ws.IsValid() {
		ts := ws.AsTime()
		out.WebsocketTS = &ts
	}
	out.TS = tradeTimestamp(t)
	return out
}

// NewTrades projects a whole batch.
func NewTrades(batch []*datafeedv1.RunningTrade) []Trade {
	out := make([]Trade, 0, len(batch))
	for _, t := range batch {
		out = append(out, NewTrade(t))
	}
	return out
}

// tradeTimestamp picks the designated timestamp: the trade time when present,
// else the websocket receive time, else now.
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
