package kafka

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/nofendian17/sbterm/libs/pkg/log"
)

func TestProducerPublish(t *testing.T) {
	cluster, err := kfake.NewCluster(kfake.SeedTopics(1, "datafeed.running_trade_batch"))
	require.NoError(t, err)
	defer cluster.Close()

	tests := []struct {
		name  string
		topic string
		key   string
		value []byte
	}{
		{
			name:  "running trade batch keyed by symbol",
			topic: "datafeed.running_trade_batch",
			key:   "BBRI",
			value: []byte(`{"batch":[]}`),
		},
		{
			name:  "running trade keyed by stock code",
			topic: "datafeed.running_trade_batch",
			key:   "BBCA",
			value: []byte(`{"batch":[]}`),
		},
		{
			name:  "empty key and value",
			topic: "datafeed.running_trade_batch",
			key:   "",
			value: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewProducer(cluster.ListenAddrs(), log.New(log.WithWriter(io.Discard)))
			require.NoError(t, err)
			defer p.Close()

			want := &kgo.Record{Topic: tt.topic, Key: []byte(tt.key), Value: tt.value}
			require.NoError(t, p.Publish(context.Background(), tt.topic, tt.key, tt.value))

			got := consumeMatching(t, cluster.ListenAddrs(), want)
			assert.Equal(t, tt.topic, got.Topic)
			assert.Equal(t, tt.key, string(got.Key))
			assert.Equal(t, tt.value, got.Value)
		})
	}
}

func TestNewProducerErrors(t *testing.T) {
	tests := []struct {
		name    string
		brokers []string
	}{
		{name: "no brokers", brokers: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewProducer(tt.brokers, log.New(log.WithWriter(io.Discard)))
			require.Error(t, err)
		})
	}
}

func TestProducerPublishUnreachable(t *testing.T) {
	p, err := NewProducer([]string{"127.0.0.1:1"}, log.New(log.WithWriter(io.Discard)))
	require.NoError(t, err)
	defer p.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.Error(t, p.Publish(ctx, "datafeed.running_trade_batch", "BBCA", []byte("v")))
}

// consumeMatching reads records from the fake cluster until one equals want,
// or fails after the timeout.
func consumeMatching(t *testing.T, addrs []string, want *kgo.Record) *kgo.Record {
	t.Helper()
	client, err := kgo.NewClient(kgo.SeedBrokers(addrs...), kgo.ConsumeTopics(want.Topic))
	require.NoError(t, err)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for ctx.Err() == nil {
		fetches := client.PollFetches(ctx)
		for _, rec := range fetches.Records() {
			if rec.Topic == want.Topic && string(rec.Key) == string(want.Key) && bytes.Equal(rec.Value, want.Value) {
				return rec
			}
		}
	}
	t.Fatalf("record %s/%q not found on topic %s", want.Topic, want.Key, want.Topic)
	return nil
}
