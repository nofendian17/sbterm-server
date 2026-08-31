// postgres_test.go
package database

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/require"
)

func TestNewPostgres_Ping(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()
	mock.ExpectPing()

	db := &Postgres{pool: mock}
	require.NoError(t, db.Ping(context.Background()))
	require.NoError(t, mock.ExpectationsWereMet())
}
