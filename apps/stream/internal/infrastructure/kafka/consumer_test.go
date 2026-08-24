package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kgo"
)

func TestConsumerReceivesProducedRecords(t *testing.T) {
	cluster, err := kfake.NewCluster(kfake.SeedTopics(1, "datafeed.running_trade_batch"))
	require.NoError(t, err)
	defer cluster.Close()

	consumer, err := NewConsumer(cluster.ListenAddrs(), "sbterm-stream-test", []string{"datafeed.running_trade_batch"})
	require.NoError(t, err)
	defer consumer.Close()

	want := &kgo.Record{Topic: "datafeed.running_trade_batch", Key: []byte("BBCA"), Value: []byte("payload")}
	require.NoError(t, produce(t, cluster.ListenAddrs(), want))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var got *kgo.Record
	for got == nil {
		fetches := consumer.PollFetches(ctx)
		if fetches.IsClientClosed() || ctx.Err() != nil {
			t.Fatal("consumer returned no records before deadline")
		}
		if errs := fetches.Errors(); len(errs) > 0 {
			t.Fatalf("fetch errors: %v", errs)
		}
		for _, rec := range fetches.Records() {
			got = rec
			break
		}
	}

	assert.Equal(t, want.Topic, got.Topic)
	assert.Equal(t, string(want.Key), string(got.Key))
	assert.Equal(t, want.Value, got.Value)
}

func TestNewConsumerErrorsWithoutBrokers(t *testing.T) {
	_, err := NewConsumer(nil, "sbterm-stream-test", []string{"datafeed.running_trade_batch"})
	require.Error(t, err)
}

// produce writes one record through a throwaway franz-go client.
func produce(t *testing.T, brokers []string, rec *kgo.Record) error {
	t.Helper()
	client, err := kgo.NewClient(kgo.SeedBrokers(brokers...))
	require.NoError(t, err)
	defer client.Close()

	return client.ProduceSync(context.Background(), rec).FirstErr()
}
