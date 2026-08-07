package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/nofendian17/sbterm-server/internal/repository"
)

// TxBeginner is satisfied by *database.Postgres in production and by
// *pgxmock.PgxPool in tests.
type TxBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

// TxManagerImpl implements repository.TxManager on top of a pgx transaction.
type TxManagerImpl struct {
	pool TxBeginner
}

func NewTxManager(pool TxBeginner) *TxManagerImpl {
	return &TxManagerImpl{pool: pool}
}

// WithTx runs fn inside one Postgres transaction using default options.
func (m *TxManagerImpl) WithTx(ctx context.Context, fn func(tx repository.Querier) error) error {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("tx: begin: %w", err)
	}
	return m.execTx(ctx, tx, fn)
}

// WithTxOptions runs fn inside one Postgres transaction with custom options
// such as isolation level (e.g. pgx.TxOptions{IsoLevel: pgx.Serializable}).
func (m *TxManagerImpl) WithTxOptions(ctx context.Context, txOptions pgx.TxOptions, fn func(tx repository.Querier) error) error {
	tx, err := m.pool.BeginTx(ctx, txOptions)
	if err != nil {
		return fmt.Errorf("tx: begin: %w", err)
	}
	return m.execTx(ctx, tx, fn)
}

// execTx is the shared commit/rollback/panic-recovery logic for both WithTx
// and WithTxOptions. It guarantees the transaction is always finalised:
//   - on success: commit
//   - on error from fn: rollback, return fn's error
//   - on panic from fn: rollback, then re-panic
func (m *TxManagerImpl) execTx(ctx context.Context, tx pgx.Tx, fn func(tx repository.Querier) error) (retErr error) {
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p) // re-panic after ensuring rollback
		}
		if retErr != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

var _ repository.TxManager = (*TxManagerImpl)(nil)

