package ws

import "context"

// Topics names the Kafka topics used by the datafeed pipeline.
type Topics struct {
	RunningTradeBatch string
	OrderBook         string
}

// Publisher sends one protobuf frame to a topic. Implementations must be safe
// for concurrent use.
type Publisher interface {
	Publish(ctx context.Context, topic string, key string, value []byte) error
	Close()
}
