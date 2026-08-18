package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRedisPinger struct {
	err error
}

func (f *fakeRedisPinger) Ping(ctx context.Context) error {
	return f.err
}

func TestHealthRepositoryPing(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(m pgxmock.PgxPoolIface)
		wantErr bool
	}{
		{
			name: "ping succeeds",
			setup: func(m pgxmock.PgxPoolIface) {
				m.ExpectPing()
			},
		},
		{
			name: "ping fails",
			setup: func(m pgxmock.PgxPoolIface) {
				m.ExpectPing().WillReturnError(errors.New("connection refused"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()
			tt.setup(mock)

			r := NewHealthRepository(mock, &fakeRedisPinger{})
			err = r.Ping(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestHealthRepositoryPingRedis(t *testing.T) {
	tests := []struct {
		name    string
		pingErr error
		wantErr bool
	}{
		{name: "ping succeeds"},
		{name: "ping fails", pingErr: errors.New("redis down"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			r := NewHealthRepository(mock, &fakeRedisPinger{err: tt.pingErr})
			err = r.PingRedis(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
