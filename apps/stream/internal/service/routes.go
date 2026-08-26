package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/nofendian17/sbterm/libs/pkg/log"
	"google.golang.org/protobuf/proto"

	consumerv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/consumer/entity/v1"
)

// RouteFn handles one Kafka record value for its topic. Returning an error
// makes the poll loop redeliver, so only durable failures should escape:
// malformed frames are logged and dropped inside the route.
type RouteFn func(ctx context.Context, value []byte) error

// Routes maps topic names onto their fan-out handlers.
type Routes map[string]RouteFn

// orderBookEnvelope is the client shape for one #O half-snapshot.
type orderBookEnvelope struct {
	Type   string `json:"type"`
	Symbol string `json:"symbol"`
	Data   struct {
		Side     string      `json:"side"`
		Seq      int64       `json:"seq"`
		Datetime string      `json:"datetime"`
		Levels   [][]float64 `json:"levels"` // [price, frequency, qty]
	} `json:"data"`
}

// alertEnvelope wraps the detector's JSON alert for subscribers.
type alertEnvelope struct {
	Type   string          `json:"type"`
	Symbol string          `json:"symbol"`
	Data   json.RawMessage `json:"data"`
}

// OrderBookRoute fans each order book half-snapshot out to the symbol's
// subscribers. Malformed frames are dropped after logging: one poisoned frame
// must not wedge redelivery for the whole topic.
func OrderBookRoute(hub *Hub, logger log.Logger) RouteFn {
	return func(ctx context.Context, value []byte) error {
		ob := &consumerv1.Orderbook{}
		if err := proto.Unmarshal(value, ob); err != nil {
			logger.Warn("stream: decode order book frame", "error", err)
			return nil
		}
		side, levels := parseOBLevels(ob.GetBody())
		if side == "" {
			logger.Warn("stream: unparsable order book body", "symbol", ob.GetStockCode())
			return nil
		}

		env := orderBookEnvelope{Type: string(ChannelOrderBook), Symbol: ob.GetStockCode()}
		env.Data.Side = side
		env.Data.Seq = ob.GetSequenceNumber()
		env.Data.Datetime = ob.GetDatetime()
		env.Data.Levels = levels

		payload, err := json.Marshal(env)
		if err != nil {
			return fmt.Errorf("stream: marshal order book envelope: %w", err)
		}
		hub.Broadcast(ChannelOrderBook, ob.GetStockCode(), payload)
		return nil
	}
}

// AlertsRoute passes detector alerts through to their symbol's subscribers.
func AlertsRoute(hub *Hub, logger log.Logger) RouteFn {
	return func(ctx context.Context, value []byte) error {
		var probe struct {
			Symbol string `json:"symbol"`
		}
		if err := json.Unmarshal(value, &probe); err != nil || probe.Symbol == "" {
			logger.Warn("stream: undecodable alert payload", "error", err)
			return nil
		}
		payload, err := json.Marshal(alertEnvelope{
			Type:   string(ChannelAlerts),
			Symbol: probe.Symbol,
			Data:   value,
		})
		if err != nil {
			return fmt.Errorf("stream: marshal alert envelope: %w", err)
		}
		hub.Broadcast(ChannelAlerts, probe.Symbol, payload)
		return nil
	}
}

// parseOBLevels mirrors the ingest parser's tolerance: malformed level
// triplets are skipped, and an unknown body shape yields an empty side.
func parseOBLevels(body string) (string, [][]float64) {
	parts := strings.Split(body, "|")
	if len(parts) < 3 || parts[0] != "#O" {
		return "", nil
	}
	levels := make([][]float64, 0, len(parts)-3)
	for _, part := range parts[3:] {
		fields := strings.Split(part, ";")
		if len(fields) != 3 {
			continue
		}
		price, err1 := strconv.ParseFloat(strings.TrimSpace(fields[0]), 64)
		freq, err2 := strconv.ParseFloat(strings.TrimSpace(fields[1]), 64)
		qty, err3 := strconv.ParseFloat(strings.TrimSpace(fields[2]), 64)
		if err1 != nil || err2 != nil || err3 != nil {
			continue
		}
		levels = append(levels, []float64{price, freq, qty})
	}
	if len(levels) == 0 {
		return "", nil
	}
	return strings.ToUpper(strings.TrimSpace(parts[2])), levels
}
