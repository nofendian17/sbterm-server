package service

import (
	"fmt"
	"strings"
	"time"

	stockbitws "github.com/nofendian17/sbterm/apps/ws/internal/infrastructure/stockbit"
	datafeedv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/v1"
)

// wib is the market-calendar timezone. Indonesia has no daylight saving, so a
// fixed offset is exact and avoids tzdata availability concerns in scratch
// containers.
var wib = time.FixedZone("WIB", 7*3600)

// microstructureBuilders maps the dynamic_channels config names onto the
// channel builders the upstream order-book bundle requires. The order_book
// channel only streams when liveprice, iepiev, and best_bid_offer ride along
// on one connection (verified upstream), so deployments should list all four.
var microstructureBuilders = map[string]func(symbols ...string) *datafeedv1.WebsocketChannel{
	"order_book":     stockbitws.WSChannelOrderBook,
	"liveprice":      stockbitws.WSChannelLiveprice,
	"iepiev":         stockbitws.WSChannelIepiev,
	"best_bid_offer": stockbitws.WSChannelBestBidOffer,
}

// BuildMicrostructureChannel composes the requested microstructure channels,
// each carrying the full symbol array, into one subscription channel.
func BuildMicrostructureChannel(channels []string, symbols []string) (*datafeedv1.WebsocketChannel, error) {
	built := make([]*datafeedv1.WebsocketChannel, 0, len(channels))
	for _, name := range channels {
		key := strings.ToLower(strings.TrimSpace(name))
		build, ok := microstructureBuilders[key]
		if !ok {
			return nil, fmt.Errorf("ws: unknown dynamic channel %q (supported: order_book, liveprice, iepiev, best_bid_offer)", name)
		}
		built = append(built, build(symbols...))
	}
	return stockbitws.MergeWSChannels(built...), nil
}

// NextRefreshAt returns the next occurrence of hh:mm in loc strictly after
// now, so a scheduler firing at exactly the slot rolls to tomorrow instead of
// spinning.
func NextRefreshAt(now time.Time, hhmm string, loc *time.Location) (time.Time, error) {
	hour, minute, err := parseHHMM(hhmm)
	if err != nil {
		return time.Time{}, fmt.Errorf("ws: parse refresh_time %q: %w", hhmm, err)
	}
	local := now.In(loc)
	target := time.Date(local.Year(), local.Month(), local.Day(), hour, minute, 0, 0, loc)
	if !target.After(now) {
		target = target.AddDate(0, 0, 1)
	}
	return target, nil
}

func parseHHMM(hhmm string) (int, int, error) {
	parts := strings.Split(strings.TrimSpace(hhmm), ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected HH:MM")
	}
	var hour, minute int
	if _, err := fmt.Sscanf(parts[0], "%d", &hour); err != nil || hour < 0 || hour > 23 {
		return 0, 0, fmt.Errorf("invalid hour %q", parts[0])
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &minute); err != nil || minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("invalid minute %q", parts[1])
	}
	return hour, minute, nil
}
