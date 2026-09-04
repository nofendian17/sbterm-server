package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCachePinger struct {
	err error
}

func (f *fakeCachePinger) Ping(ctx context.Context) error {
	return f.err
}

func TestHealthRepositoryPingCache(t *testing.T) {
	tests := []struct {
		name    string
		pingErr error
		wantErr bool
	}{
		{name: "ping succeeds"},
		{name: "ping fails", pingErr: errors.New("cache down"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewHealthRepository(&fakeCachePinger{err: tt.pingErr})
			err := r.PingCache(context.Background())
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, r)
		})
	}
}
