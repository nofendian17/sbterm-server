// Package repository implements the repository contracts using PostgreSQL/Redis backends.

package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/nofendian17/sbterm/apps/core/internal/repository"
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

// WithTxOptions runs fn inside one Postgres transaction with custom options.
func (m *TxManagerImpl) WithTxOptions(ctx context.Context, txOptions repository.TxOptions, fn func(tx repository.Querier) error) error {
	tx, err := m.pool.BeginTx(ctx, toPgxTxOptions(txOptions))
	if err != nil {
		return fmt.Errorf("tx: begin: %w", err)
	}
	return m.execTx(ctx, tx, fn)
}

// toPgxTxOptions maps the driver-free repository.TxOptions onto pgx.
func toPgxTxOptions(o repository.TxOptions) pgx.TxOptions {
	return pgx.TxOptions{IsoLevel: toPgxIsoLevel(o.Isolation)}
}

func toPgxIsoLevel(l repository.IsolationLevel) pgx.TxIsoLevel {
	switch l {
	case repository.IsolationReadUncommitted:
		return pgx.ReadUncommitted
	case repository.IsolationReadCommitted:
		return pgx.ReadCommitted
	case repository.IsolationRepeatableRead:
		return pgx.RepeatableRead
	case repository.IsolationSerializable:
		return pgx.Serializable
	default:
		return ""
	}
}

func (m *TxManagerImpl) execTx(ctx context.Context, tx pgx.Tx, fn func(tx repository.Querier) error) (retErr error) {
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		if retErr != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	if err := fn(AdaptQuerier(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

var _ repository.TxManager = (*TxManagerImpl)(nil)
