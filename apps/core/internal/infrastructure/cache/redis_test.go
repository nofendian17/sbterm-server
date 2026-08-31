// redis_test.go
package cache

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/require"
)

func TestNewRedis_Ping(t *testing.T) {
	srv, err := miniredis.Run()
	require.NoError(t, err)
	defer srv.Close()

	r, err := New(context.Background(), "redis://"+srv.Addr(),
		WithDialTimeout(0), WithReadTimeout(0), WithWriteTimeout(0))
	require.NoError(t, err)
	require.NoError(t, r.Ping(context.Background()))
}
