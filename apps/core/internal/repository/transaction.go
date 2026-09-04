package repository

import (
	"context"
)

// Row is a single-row result. Satisfied by pgx.Row without importing pgx.
type Row interface {
	Scan(dest ...any) error
}

// Rows is a multi-row result. Satisfied by pgx.Rows without importing pgx.
type Rows interface {
	Close()
	Err() error
	Next() bool
	Scan(dest ...any) error
}

// CommandResult is the outcome of an Exec. Satisfied by pgconn.CommandTag
// without importing pgx.
type CommandResult interface {
	RowsAffected() int64
}

// Querier is the data-access seam satisfied (via an infrastructure adapter)
// by both *pgxpool.Pool (outside a transaction) and pgx.Tx (inside one), so
// a single repository method runs in either context without duplication.
// Usecases only pass Querier through, they never run SQL.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (CommandResult, error)
	Query(ctx context.Context, sql string, args ...any) (Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) Row
}

// IsolationLevel selects the transaction isolation level.
type IsolationLevel string

const (
	IsolationDefault         IsolationLevel = ""
	IsolationReadCommitted   IsolationLevel = "read committed"
	IsolationRepeatableRead  IsolationLevel = "repeatable read"
	IsolationSerializable    IsolationLevel = "serializable"
	IsolationReadUncommitted IsolationLevel = "read uncommitted"
)

// TxOptions carries transaction options without importing a driver.
type TxOptions struct {
	Isolation IsolationLevel
}

// TxManager gives usecases an ACID boundary: WithTx runs fn inside a single
// transaction, committing on success and rolling back on any error.
// WithTxOptions allows callers to specify custom transaction options such as
// isolation level (e.g. IsolationSerializable for financial operations).
//
//go:generate go run go.uber.org/mock/mockgen -source=transaction.go -destination=../mocks/mock_tx_manager.go -package=mocks -typed
type TxManager interface {
	WithTx(ctx context.Context, fn func(tx Querier) error) error
	WithTxOptions(ctx context.Context, txOptions TxOptions, fn func(tx Querier) error) error
}
