package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisRefreshStore(t *testing.T) {
	tests := []struct {
		name     string
		actions  func(t *testing.T, store *RedisRefreshStore)
		wantJTI  string
		wantOK   bool
		wantUser string
	}{
		{
			name: "store then consume returns user",
			actions: func(t *testing.T, store *RedisRefreshStore) {
				require.NoError(t, store.StoreRefresh("abc123", "u1", 10*time.Minute))
			},
			wantJTI:  "abc123",
			wantOK:   true,
			wantUser: "u1",
		},
		{
			name: "delete then consume returns false",
			actions: func(t *testing.T, store *RedisRefreshStore) {
				require.NoError(t, store.StoreRefresh("def456", "u2", 10*time.Minute))
				require.NoError(t, store.DeleteRefresh("def456"))
			},
			wantJTI: "def456",
			wantOK:  false,
		},
		{
			name:    "consume missing jti returns false",
			actions: func(t *testing.T, store *RedisRefreshStore) {},
			wantJTI: "nope",
			wantOK:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestRedisClient(t)
			store := NewRedisRefreshStore(client)
			tt.actions(t, store)

			got, ok := store.ConsumeRefresh(tt.wantJTI)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.wantUser, got)
			}
		})
	}
}
