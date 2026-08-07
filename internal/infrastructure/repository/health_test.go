package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

			r := NewHealthRepository(mock)
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
