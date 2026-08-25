package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
)

// The spec forbids offset commits ("TANPA commit offset"). franz-go
// auto-commits every 5s by default once a consumer group is set, so the
// consumer must explicitly disable it. This test would catch a regression:
// with auto-commit enabled, the group's committed offset becomes visible
// after the promote poll + interval (verified against kfake's commit store).
func TestConsumerNeverCommitsOffsets(t *testing.T) {
	const topic = "datafeed.running_trade_batch"
	const group = "sbterm-stream-nocommit"
	cluster, err := kfake.NewCluster(kfake.SeedTopics(1, topic))
	require.NoError(t, err)
	defer cluster.Close()

	consumer, err := NewConsumer(cluster.ListenAddrs(), group, []string{topic})
	require.NoError(t, err)
	defer consumer.Close()

	// Join the group and consume one record, giving a default client
	// something to autocommit.
	rec := &kgo.Record{Topic: topic, Key: []byte("k"), Value: []byte("v")}
	require.NoError(t, produce(t, cluster.ListenAddrs(), rec))

	sawRecord := false
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline) && !sawRecord; {
		fetches := consumer.PollFetches(context.Background())
		if len(fetches.Records()) > 0 {
			sawRecord = true
		}
	}
	require.True(t, sawRecord, "sanity: record consumed")

	// Past the default 5s autocommit interval...
	time.Sleep(6 * time.Second)

	// ...one more poll: under default auto-commit, dirty offsets promote to
	// head at the start of the next poll, which is what gets committed...
	pollCtx, cancelPoll := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancelPoll()
	consumer.PollFetches(pollCtx)

	// ...and one more full interval for the autocommit ticker to fire after
	// the promotion.
	time.Sleep(6 * time.Second)

	fetchOffset := func(t *testing.T) int64 {
		t.Helper()
		req := kmsg.NewPtrOffsetFetchRequest()
		req.Group = group
		req.Version = 7 // classic path; kfake maps it to the group form
		req.Topics = []kmsg.OffsetFetchRequestTopic{{Topic: topic, Partitions: []int32{0}}}
		resp, err := consumer.client.Request(context.Background(), req)
		require.NoError(t, err)
		offResp := resp.(*kmsg.OffsetFetchResponse)
		require.Len(t, offResp.Groups, 1)
		require.Zero(t, offResp.Groups[0].ErrorCode)
		require.Len(t, offResp.Groups[0].Topics, 1)
		require.Len(t, offResp.Groups[0].Topics[0].Partitions, 1)
		return offResp.Groups[0].Topics[0].Partitions[0].Offset
	}

	got := fetchOffset(t)
	assert.Equal(t, int64(-1), got,
		"no offset may be committed: spec says fan-out never commits")

	// Control: prove this harness would observe a real commit.
	consumer.client.CommitOffsetsSync(context.Background(),
		map[string]map[int32]kgo.EpochOffset{topic: {0: {Offset: 42}}},
		func(_ *kgo.Client, _ *kmsg.OffsetCommitRequest, resp *kmsg.OffsetCommitResponse, err error) {
			require.NoError(t, err)
		})
	time.Sleep(500 * time.Millisecond)
	assert.Equal(t, int64(42), fetchOffset(t),
		"harness sanity: an actual commit must be visible through this probe")
}
