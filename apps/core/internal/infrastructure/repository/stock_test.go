package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
)

func newTestStockRepo(t *testing.T) (*StockRepository, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(mock.Close)
	return NewStockRepository(AdaptQuerier(mock)), mock
}

func TestStockRepository_GetBySymbol_NotFound(t *testing.T) {
	repo, mock := newTestStockRepo(t)
	mock.ExpectQuery(`s.symbol = \$1 AND s.deleted_at IS NULL`).
		WithArgs("BBCA").
		WillReturnError(sql.ErrNoRows)

	_, err := repo.GetBySymbol(context.Background(), "BBCA")
	assert.ErrorIs(t, err, domain.ErrStockNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestStockRepository_GetBySymbol_Found(t *testing.T) {
	repo, mock := newTestStockRepo(t)
	sectorID := "s1"
	rows := pgxmock.NewRows([]string{
		"symbol", "name", "sector_id", "sector_name", "exchange",
		"icon_url", "is_active", "synced_at", "created_at", "updated_at",
	}).AddRow("BBCA", "Bank Central Asia", sectorID, "Financials", "IDX",
		"https://x/BBCA.png", true, nil, time.Now(), time.Now())
	mock.ExpectQuery(`s.symbol = \$1 AND s.deleted_at IS NULL`).
		WithArgs("BBCA").WillReturnRows(rows)

	got, err := repo.GetBySymbol(context.Background(), "BBCA")
	require.NoError(t, err)
	assert.Equal(t, "BBCA", got.Symbol)
	require.NotNil(t, got.Sector)
	assert.Equal(t, "Financials", got.Sector.Name)
	assert.True(t, got.IsActive)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestStockRepository_Create_SymbolTaken(t *testing.T) {
	repo, mock := newTestStockRepo(t)
	mock.ExpectExec(`INSERT INTO stocks`).
		WithArgs("BBCA", "Bank Central Asia", pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), true).
		WillReturnError(pgUniqueViolation())

	err := repo.Create(context.Background(), domain.Stock{Symbol: "BBCA", Name: "Bank Central Asia", IsActive: true})
	assert.ErrorIs(t, err, domain.ErrStockSymbolTaken)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestStockRepository_SoftDelete_NotFound(t *testing.T) {
	repo, mock := newTestStockRepo(t)
	mock.ExpectExec(`UPDATE stocks SET deleted_at`).
		WithArgs("BBCA").
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs("BBCA").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))

	err := repo.SoftDelete(context.Background(), "BBCA")
	assert.ErrorIs(t, err, domain.ErrStockNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestStockRepository_SoftDelete_HasWatchlists(t *testing.T) {
	repo, mock := newTestStockRepo(t)
	mock.ExpectExec(`UPDATE stocks SET deleted_at`).
		WithArgs("BBCA").
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs("BBCA").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))

	err := repo.SoftDelete(context.Background(), "BBCA")
	assert.ErrorIs(t, err, domain.ErrStockHasWatchlists)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestStockRepository_Upsert(t *testing.T) {
	repo, mock := newTestStockRepo(t)
	mock.ExpectQuery(`RETURNING \(xmax = 0\) AS created`).
		WithArgs("BBCA", "Bank Central Asia", pgxmock.AnyArg(), true).
		WillReturnRows(pgxmock.NewRows([]string{"created"}).AddRow(true))

	created, err := repo.Upsert(context.Background(), domain.Stock{
		Symbol: "BBCA", Name: "Bank Central Asia", IsActive: true,
	})
	require.NoError(t, err)
	assert.True(t, created)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestStockRepository_Upsert_Unchanged(t *testing.T) {
	repo, mock := newTestStockRepo(t)
	// No row returned => ON CONFLICT ... WHERE matched nothing.
	mock.ExpectQuery(`RETURNING \(xmax = 0\) AS created`).
		WithArgs("BBCA", "Bank Central Asia", pgxmock.AnyArg(), true).
		WillReturnError(sql.ErrNoRows)

	created, err := repo.Upsert(context.Background(), domain.Stock{
		Symbol: "BBCA", Name: "Bank Central Asia", IsActive: true,
	})
	require.NoError(t, err)
	assert.False(t, created)
	require.NoError(t, mock.ExpectationsWereMet())
}
