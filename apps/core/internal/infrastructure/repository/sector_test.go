package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
)

func newTestSectorRepo(t *testing.T) (*SectorRepository, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(mock.Close)
	return NewSectorRepository(AdaptQuerier(mock)), mock
}

func TestSectorRepository_Create_NameTaken(t *testing.T) {
	repo, mock := newTestSectorRepo(t)
	mock.ExpectQuery(`INSERT INTO sectors`).
		WithArgs("Financials").
		WillReturnError(pgUniqueViolation())

	_, err := repo.Create(context.Background(), "Financials")
	assert.ErrorIs(t, err, domain.ErrSectorNameTaken)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSectorRepository_SoftDelete_HasStocks(t *testing.T) {
	repo, mock := newTestSectorRepo(t)
	mock.ExpectExec(`UPDATE sectors SET deleted_at`).
		WithArgs("s1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs("s1").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))

	err := repo.SoftDelete(context.Background(), "s1")
	assert.ErrorIs(t, err, domain.ErrSectorHasStocks)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSectorRepository_GetByID_NotFound(t *testing.T) {
	repo, mock := newTestSectorRepo(t)
	mock.ExpectQuery(`FROM sectors WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs("s1").
		WillReturnError(sql.ErrNoRows)

	_, err := repo.GetByID(context.Background(), "s1")
	assert.ErrorIs(t, err, domain.ErrSectorNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}
