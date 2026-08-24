package container

import (
	"io"
	"net/http"
	"testing"

	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nofendian17/sbterm/apps/stream/internal/infrastructure/config"
	"github.com/nofendian17/sbterm/apps/stream/internal/infrastructure/kafka"
	"github.com/nofendian17/sbterm/apps/stream/internal/service"
	"github.com/nofendian17/sbterm/libs/pkg/log"
)

func testConfig() *config.Config {
	return &config.Config{
		Port: ":0",
		Kafka: config.KafkaConfig{
			Brokers:                []string{"localhost:29092"},
			Group:                  "sbterm-stream-test",
			RunningTradeBatchTopic: "datafeed.running_trade_batch",
		},
		Log: config.LogConfig{Level: "info", Format: "text"},
	}
}

// TestNewWiresEveryComponent builds the whole graph without a reachable
// broker: franz-go connects lazily, so construction must not dial.
func TestNewWiresEveryComponent(t *testing.T) {
	injector := New(testConfig(), log.New(log.WithWriter(io.Discard)))
	defer injector.Shutdown()

	consumer, err := do.Invoke[*kafka.Consumer](injector)
	require.NoError(t, err)
	assert.NotNil(t, consumer)

	hub, err := do.Invoke[*service.Hub](injector)
	require.NoError(t, err)
	assert.NotNil(t, hub)

	svc, err := do.Invoke[*service.Service](injector)
	require.NoError(t, err)
	assert.NotNil(t, svc)

	srv, err := do.Invoke[*http.Server](injector)
	require.NoError(t, err)
	assert.Equal(t, ":0", srv.Addr)
	assert.NotNil(t, srv.Handler)
}
