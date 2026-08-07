package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nofendian17/sbterm-server/internal/repository"
)

func TestTxManagerWithTx(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(m pgxmock.PgxPoolIface)
		fn      func() error
		wantErr bool
	}{
		{
			name: "commits when fn succeeds",
			setup: func(m pgxmock.PgxPoolIface) {
				m.ExpectBegin()
				m.ExpectCommit()
			},
			fn: func() error {
				return nil
			},
		},
		{
			name: "rolls back when fn fails",
			setup: func(m pgxmock.PgxPoolIface) {
				m.ExpectBegin()
				m.ExpectRollback()
			},
			fn: func() error {
				return errors.New("business error")
			},
			wantErr: true,
		},
		{
			name: "returns error when commit fails",
			setup: func(m pgxmock.PgxPoolIface) {
				m.ExpectBegin()
				m.ExpectCommit().WillReturnError(errors.New("commit failed"))
				m.ExpectRollback()
			},
			fn: func() error {
				return nil
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

			tm := NewTxManager(mock)
			err = tm.WithTx(context.Background(), func(tx repository.Querier) error {
				return tt.fn()
			})

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestTxManagerWithTxBeginError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()
	mock.ExpectBegin().WillReturnError(errors.New("connection refused"))

	tm := NewTxManager(mock)
	err = tm.WithTx(context.Background(), func(tx repository.Querier) error {
		t.Fatal("fn must not run when Begin fails")
		return nil
	})
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTxManagerExecInTx(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE accounts SET balance = balance - \$1 WHERE id = \$2`).
		WithArgs(int64(100), int64(1)).
		WillReturnResult(pgconn.NewCommandTag("UPDATE 1"))
	mock.ExpectCommit()

	tm := NewTxManager(mock)
	err = tm.WithTx(context.Background(), func(tx repository.Querier) error {
		_, err := tx.Exec(context.Background(), `UPDATE accounts SET balance = balance - $1 WHERE id = $2`, int64(100), int64(1))
		return err
	})
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTxManagerWithTxPanicRecovery(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()
	mock.ExpectBegin()
	mock.ExpectRollback()

	tm := NewTxManager(mock)

	assert.PanicsWithValue(t, "unexpected panic", func() {
		_ = tm.WithTx(context.Background(), func(tx repository.Querier) error {
			panic("unexpected panic")
		})
	})
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTxManagerWithTxOptions(t *testing.T) {
	tests := []struct {
		name    string
		opts    pgx.TxOptions
		setup   func(m pgxmock.PgxPoolIface)
		fn      func() error
		wantErr bool
	}{
		{
			name: "commits with serializable isolation",
			opts: pgx.TxOptions{IsoLevel: pgx.Serializable},
			setup: func(m pgxmock.PgxPoolIface) {
				m.ExpectBeginTx(pgx.TxOptions{IsoLevel: pgx.Serializable})
				m.ExpectCommit()
			},
			fn: func() error {
				return nil
			},
		},
		{
			name: "rolls back with repeatable read isolation",
			opts: pgx.TxOptions{IsoLevel: pgx.RepeatableRead},
			setup: func(m pgxmock.PgxPoolIface) {
				m.ExpectBeginTx(pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
				m.ExpectRollback()
			},
			fn: func() error {
				return errors.New("business error")
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

			tm := NewTxManager(mock)
			err = tm.WithTxOptions(context.Background(), tt.opts, func(tx repository.Querier) error {
				return tt.fn()
			})

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestTxManagerWithTxOptionsBeginError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()
	mock.ExpectBeginTx(pgx.TxOptions{IsoLevel: pgx.Serializable}).
		WillReturnError(errors.New("connection refused"))

	tm := NewTxManager(mock)
	err = tm.WithTxOptions(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable}, func(tx repository.Querier) error {
		t.Fatal("fn must not run when BeginTx fails")
		return nil
	})
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
