package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptionsApply(t *testing.T) {
	cfg, err := pgxpool.ParseConfig("postgres://user:pass@localhost:5432/db?sslmode=disable")
	require.NoError(t, err)

	o := &options{}
	WithMaxConns(20)(o)
	WithMinConns(2)(o)
	WithMaxConnLifetime(time.Hour)(o)
	WithMaxConnIdleTime(10 * time.Minute)(o)
	o.apply(cfg)

	assert.Equal(t, int32(20), cfg.MaxConns)
	assert.Equal(t, int32(2), cfg.MinConns)
	assert.Equal(t, time.Hour, cfg.MaxConnLifetime)
	assert.Equal(t, 10*time.Minute, cfg.MaxConnIdleTime)
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		dsn     string
		wantErr bool
	}{
		{name: "valid dsn", dsn: "postgres://user:pass@localhost:5432/db?sslmode=disable"},
		{name: "malformed dsn", dsn: "://broken", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg, err := New(context.Background(), tt.dsn)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, pg)
			pg.Shutdown()
		})
	}
}

func TestPing(t *testing.T) {
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

			pg := NewWithPool(mock)
			err = pg.Ping(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestHealthCheck(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(m pgxmock.PgxPoolIface)
		wantErr bool
	}{
		{
			name: "health check succeeds",
			setup: func(m pgxmock.PgxPoolIface) {
				m.ExpectPing()
			},
		},
		{
			name: "health check fails",
			setup: func(m pgxmock.PgxPoolIface) {
				m.ExpectPing().WillReturnError(errors.New("database down"))
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

			pg := NewWithPool(mock)
			err = pg.HealthCheck(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestShutdown(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)

	pg := NewWithPool(mock)
	assert.NoError(t, pg.Shutdown())
}
