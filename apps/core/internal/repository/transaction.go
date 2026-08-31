package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Querier is the data-access seam satisfied by both *pgxpool.Pool (outside a
// transaction) and pgx.Tx (inside one), so a single repository method runs in
// either context without duplication. The pgx types here are the documented
// sqlc/DBTX pattern; usecases only pass Querier through, they never run SQL.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// TxManager gives usecases an ACID boundary: WithTx runs fn inside a single
// Postgres transaction, committing on success and rolling back on any error.
// WithTxOptions allows callers to specify custom transaction options such as
// isolation level (e.g. pgx.Serializable for financial operations).
//
//go:generate go run go.uber.org/mock/mockgen -source=transaction.go -destination=../mocks/mock_tx_manager.go -package=mocks -typed
type TxManager interface {
	WithTx(ctx context.Context, fn func(tx Querier) error) error
	WithTxOptions(ctx context.Context, txOptions pgx.TxOptions, fn func(tx Querier) error) error
}
