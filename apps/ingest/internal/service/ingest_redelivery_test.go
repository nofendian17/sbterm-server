package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/nofendian17/sbterm/libs/pkg/log"
	datafeedv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/v1"
)

// fakeCommitter records the service asked to commit and replays a
// canned sequence of fetches.
type fakeCommitter struct {
	// fetches is drained one batch per processFetches call.
	fetches []kgo.Fetches
	// committed holds, per call, the records passed to MarkCommit.
	committed [][]*kgo.Record
}

func rec(topic string, partition int32, offset int64) *kgo.Record {
	return &kgo.Record{Topic: topic, Partition: partition, Offset: offset, LeaderEpoch: 5}
}

func (f *fakeCommitter) PollFetches(context.Context) kgo.Fetches {
	if len(f.fetches) == 0 {
		return kgo.Fetches{}
	}
	fs := f.fetches[0]
	f.fetches = f.fetches[1:]
	return fs
}

func (f *fakeCommitter) MarkCommit(_ context.Context, processed []*kgo.Record) error {
	f.committed = append(f.committed, processed)
	return nil
}

func (f *fakeCommitter) AllowRebalance() {}

// verifyCommitted asserts the offsets committed in each call match want.
func verifyCommitted(t *testing.T, c *fakeCommitter, want []int64) {
	t.Helper()
	var got []int64
	for _, batch := range c.committed {
		for _, r := range batch {
			got = append(got, r.Offset)
		}
	}
	assert.Equal(t, want, got)
}

func TestProcessFetchesCommitsEachRecordOnSuccess(t *testing.T) {
	logger := log.New(log.WithWriter(io.Discard))
	committer := &fakeCommitter{
		fetches: []kgo.Fetches{
			{{
				Topics: []kgo.FetchTopic{{ //nolint:exhaustruct
					Topic: "t",
					Partitions: []kgo.FetchPartition{{ //nolint:exhaustruct
						Partition: 0,
						Records:   []*kgo.Record{rec("t", 0, 10), rec("t", 0, 11), rec("t", 0, 12)},
					}},
				}},
			}},
		},
	}
	h := NewFrameHandler(&fakeRunningSink{}, Topics{RunningTradeBatch: "t"}, logger)
	s := &Service{consumer: committer, handler: h, logger: logger}

	s.processFetches(committer.fetches[0])
	verifyCommitted(t, committer, []int64{10, 11, 12})
}

func TestProcessFetchesStopsAndRedeliversOnError(t *testing.T) {
	logger := log.New(log.WithWriter(io.Discard))
	committer := &fakeCommitter{
		fetches: []kgo.Fetches{
			{{
				Topics: []kgo.FetchTopic{{ //nolint:exhaustruct
					Topic: "t",
					Partitions: []kgo.FetchPartition{{ //nolint:exhaustruct
						Partition: 0,
						Records:   []*kgo.Record{rec("t", 0, 10), rec("t", 0, 11), rec("t", 0, 12)},
					}},
				}},
			}},
		},
	}
	// Fail only the record at offset 11; 10 must still be committed, 11 and 12
	// must be redelivered (the loop halts at the failure).
	failing := &offsetFailingSink{}
	h := NewFrameHandler(failing, Topics{RunningTradeBatch: "t"}, logger)
	s := &Service{consumer: committer, handler: h, logger: logger}

	s.processFetches(committer.fetches[0])
	// Only offset 10 is committed; 11 and 12 are redelivered.
	verifyCommitted(t, committer, []int64{10})
	assert.Equal(t, uint64(2), failing.calls, "the failure record should be attempted but not beyond")
}

// TestTrimToPartitionKeepsOtherTopics verifies that trimming on failure keys
// on (topic, partition), not partition alone: another topic's record that
// shares the partition number must survive the trim.
func TestTrimToPartitionKeepsOtherTopics(t *testing.T) {
	committed := []*kgo.Record{rec("t2", 0, 20), rec("t1", 0, 10)}
	out := trimToPartition(committed, rec("t1", 0, 11))

	var got []string
	for _, r := range out {
		got = append(got, fmt.Sprintf("%s/%d@%d", r.Topic, r.Partition, r.Offset))
	}
	assert.Equal(t, []string{"t2/0@20", "t1/0@10"}, got)
}

// offsetFailingSink fails on its second Store call. Records are processed in
// offset order (10, 11, 12), so the second call corresponds to offset 11.
type offsetFailingSink struct {
	calls uint64
}

func (o *offsetFailingSink) Store(_ context.Context, _ *datafeedv1.RunningTradeBatch) error {
	o.calls++
	if o.calls == 2 {
		return errors.New("downstream unavailable")
	}
	return nil
}

func (o *offsetFailingSink) Close(context.Context) error { return nil }
