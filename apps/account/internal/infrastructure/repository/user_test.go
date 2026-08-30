package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/require"

	"github.com/nofendian17/sbterm/apps/account/internal/domain"
)

// pgErrUniqueViolation builds a pgconn.PgError carrying the Postgres unique
// violation code (23505), as returned by a real driver on a duplicate key.
func pgErrUniqueViolation() error {
	return &pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint"}
}

func TestUserRepository_Create(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectExec(`INSERT INTO users`).
		WithArgs(pgxmock.AnyArg(), "a@b.co", "hash", "Beni", pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	repo := NewUserRepository(mock)
	err := repo.Create(context.Background(), domain.User{
		Email:        "a@b.co",
		PasswordHash: "hash",
		DisplayName:  "Beni",
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Create_DuplicateEmail(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectExec(`INSERT INTO users`).
		WithArgs(pgxmock.AnyArg(), "a@b.co", "hash", "Beni", pgxmock.AnyArg()).
		WillReturnError(pgErrUniqueViolation())

	repo := NewUserRepository(mock)
	err := repo.Create(context.Background(), domain.User{
		Email:        "a@b.co",
		PasswordHash: "hash",
		DisplayName:  "Beni",
	})
	require.ErrorIs(t, err, domain.ErrEmailTaken)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetByEmail(t *testing.T) {
	now := time.Now()
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery(`SELECT id, email, password_hash, display_name, expires_at, created_at, updated_at, deleted_at FROM users`).
		WithArgs("a@b.co").
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "email", "password_hash", "display_name", "expires_at", "created_at", "updated_at", "deleted_at",
		}).AddRow("u1", "a@b.co", "hash", "Beni", nil, now, now, nil))

	repo := NewUserRepository(mock)
	got, err := repo.GetByEmail(context.Background(), "a@b.co")
	require.NoError(t, err)
	require.Equal(t, "u1", got.ID)
	require.Equal(t, "a@b.co", got.Email)
	require.Equal(t, "Beni", got.DisplayName)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetByEmail_NotFound(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery(`(?s)SELECT .* FROM users`).WithArgs("x@y.co").WillReturnError(sql.ErrNoRows)

	repo := NewUserRepository(mock)
	_, err := repo.GetByEmail(context.Background(), "x@y.co")
	require.ErrorIs(t, err, domain.ErrUserNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetByID(t *testing.T) {
	now := time.Now()
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery(`SELECT id, email, password_hash, display_name, expires_at, created_at, updated_at, deleted_at FROM users`).
		WithArgs("u1").
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "email", "password_hash", "display_name", "expires_at", "created_at", "updated_at", "deleted_at",
		}).AddRow("u1", "a@b.co", "hash", "Beni", nil, now, now, nil))

	repo := NewUserRepository(mock)
	got, err := repo.GetByID(context.Background(), "u1")
	require.NoError(t, err)
	require.Equal(t, "u1", got.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetByID_NotFound(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery(`(?s)SELECT .* FROM users`).WithArgs("missing").WillReturnError(sql.ErrNoRows)

	repo := NewUserRepository(mock)
	_, err := repo.GetByID(context.Background(), "missing")
	require.ErrorIs(t, err, domain.ErrUserNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Update(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	expiry := time.Now().Add(24 * time.Hour)
	mock.ExpectExec(`UPDATE users SET display_name = \$2, expires_at = \$3`).
		WithArgs("u1", "NewName", expiry).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewUserRepository(mock)
	err := repo.Update(context.Background(), "u1", "NewName", &expiry)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_SoftDelete(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectExec(`UPDATE users SET deleted_at`).
		WithArgs("u1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewUserRepository(mock)
	err := repo.SoftDelete(context.Background(), "u1")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_SetExpiry(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	expiry := time.Now().Add(48 * time.Hour)
	mock.ExpectExec(`UPDATE users SET expires_at`).
		WithArgs(expiry, "u1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewUserRepository(mock)
	err := repo.SetExpiry(context.Background(), "u1", &expiry)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_AssignDefaultRole(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectExec(`INSERT INTO user_roles`).
		WithArgs("u1").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	repo := NewUserRepository(mock)
	err := repo.AssignDefaultRole(context.Background(), "u1")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
