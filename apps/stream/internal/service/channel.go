// Package service fans Kafka datafeed records out to connected WebSocket
// clients according to their per-channel symbol subscriptions.
package service

// Channel identifies one subscribable data stream. Its string value doubles
// as the envelope "type" clients see, so streaming another data kind means
// adding one entry here plus a poll loop for its Kafka topic — the hub,
// client, and delivery layers never change.
type Channel string

// ChannelRunningTrade streams running trade batches from
// datafeed.running_trade_batch.
const ChannelRunningTrade Channel = "running_trade"

// channels lists every channel this instance serves; delivery validates
// inbound subscriptions against it.
var channels = map[Channel]bool{
	ChannelRunningTrade: true,
}

// KnownChannel reports whether the channel is served by this instance.
func KnownChannel(ch Channel) bool {
	return channels[ch]
}
