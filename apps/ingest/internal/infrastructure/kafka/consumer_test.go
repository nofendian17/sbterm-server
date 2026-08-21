package kafka

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kgo"
)

func TestConsumerPollFetches(t *testing.T) {
	cluster, err := kfake.NewCluster(kfake.SeedTopics(1, "t-single", "t-x", "t-y", "t-empty"))
	require.NoError(t, err)
	defer cluster.Close()

	tests := []struct {
		name   string
		topics []string
		seed   map[string][]string
		want   []string
	}{
		{
			name:   "single topic",
			topics: []string{"t-single"},
			seed:   map[string][]string{"t-single": {"a", "b", "c"}},
			want:   []string{"a", "b", "c"},
		},
		{
			name:   "two topics interleaved",
			topics: []string{"t-x", "t-y"},
			seed:   map[string][]string{"t-x": {"1"}, "t-y": {"2", "3"}},
			want:   []string{"1", "2", "3"},
		},
		{
			name:   "topic with no records",
			topics: []string{"t-empty"},
			seed:   map[string][]string{"t-empty": {}},
			want:   nil,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for topic, values := range tt.seed {
				seedRecords(t, cluster.ListenAddrs(), topic, values)
			}

			group := fmt.Sprintf("grp-%d-%s", i, strings.ReplaceAll(tt.name, " ", "-"))
			consumer, err := NewConsumer(cluster.ListenAddrs(), group, tt.topics)
			require.NoError(t, err)
			defer consumer.Close()

			recs := pollN(t, consumer, len(tt.want), 5*time.Second)
			var got []string
			for _, r := range recs {
				got = append(got, string(r.Value))
			}
			assert.ElementsMatch(t, tt.want, got)
		})
	}
}

func TestConsumerShutdown(t *testing.T) {
	cluster, err := kfake.NewCluster()
	require.NoError(t, err)
	defer cluster.Close()

	consumer, err := NewConsumer(cluster.ListenAddrs(), "grp-shutdown", []string{"t-x"})
	require.NoError(t, err)
	require.NoError(t, consumer.Shutdown())
	// Double shutdown is safe.
	require.NoError(t, consumer.Shutdown())
}

// seedRecords produces one record per value into topic via a throwaway client.
func seedRecords(t *testing.T, addrs []string, topic string, values []string) {
	t.Helper()
	if len(values) == 0 {
		return
	}
	client, err := kgo.NewClient(kgo.SeedBrokers(addrs...), kgo.AllowAutoTopicCreation())
	require.NoError(t, err)
	defer client.Close()
	for _, v := range values {
		res := client.ProduceSync(context.Background(), &kgo.Record{Topic: topic, Value: []byte(v)})
		require.NoError(t, res.FirstErr())
	}
}

// pollN drains fetches until n records arrive or the timeout elapses.
func pollN(t *testing.T, c *Consumer, n int, timeout time.Duration) []*kgo.Record {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var out []*kgo.Record
	for len(out) < n && ctx.Err() == nil {
		fetches := c.PollFetches(ctx)
		if errs := fetches.Errors(); len(errs) > 0 {
			for _, e := range errs {
				t.Logf("fetch error: %v", e.Err)
			}
		}
		out = append(out, fetches.Records()...)
	}
	return out
}

func TestConsumerMarkCommit(t *testing.T) {
	cluster, err := kfake.NewCluster(kfake.SeedTopics(1, "t-commit"))
	require.NoError(t, err)
	defer cluster.Close()

	seedRecords(t, cluster.ListenAddrs(), "t-commit", []string{"a", "b", "c", "d"})

	consumer, err := NewConsumer(cluster.ListenAddrs(), "grp-commit", []string{"t-commit"})
	require.NoError(t, err)

	recs := pollN(t, consumer, 4, 5*time.Second)
	require.Len(t, recs, 4)

	// Committing an empty slice is a no-op and must not error.
	require.NoError(t, consumer.MarkCommit(context.Background(), nil))

	// Commit the first two records. Then fully leave the group so a fresh
	// consumer in the same group resumes at offset 2 ("c", "d").
	require.NoError(t, consumer.MarkCommit(context.Background(), recs[:2]))
	consumer.Close()

	// Give kfake time to apply the commit and process the leave-group so the
	// second consumer sees the committed offset rather than the topic head.
	time.Sleep(500 * time.Millisecond)

	consumer2, err := NewConsumer(cluster.ListenAddrs(), "grp-commit", []string{"t-commit"})
	require.NoError(t, err)
	defer consumer2.Close()
	rest := pollN(t, consumer2, 2, 5*time.Second)
	require.Len(t, rest, 2, "redelivery should skip already-committed records")
	assert.Equal(t, "c", string(rest[0].Value))
	assert.Equal(t, "d", string(rest[1].Value))
}
