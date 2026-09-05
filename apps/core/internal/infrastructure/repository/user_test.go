package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/require"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
)

// pgErrUniqueViolation builds a pgconn.PgError carrying the Postgres unique
// violation code (23505), as returned by a real driver on a duplicate key.
func pgErrUniqueViolation() error {
	return &pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint"}
}

func TestUserRepository_Create(t *testing.T) {
	tests := []struct {
		name    string
		user    domain.User
		setup   func(mock pgxmock.PgxPoolIface)
		wantErr error
	}{
		{
			name: "success",
			user: domain.User{Email: "a@b.co", PasswordHash: "hash", DisplayName: "Beni"},
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`INSERT INTO users`).
					WithArgs(pgxmock.AnyArg(), "a@b.co", "hash", "Beni", pgxmock.AnyArg()).
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
			},
		},
		{
			name: "duplicate email",
			user: domain.User{Email: "a@b.co", PasswordHash: "hash", DisplayName: "Beni"},
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`INSERT INTO users`).
					WithArgs(pgxmock.AnyArg(), "a@b.co", "hash", "Beni", pgxmock.AnyArg()).
					WillReturnError(pgErrUniqueViolation())
			},
			wantErr: domain.ErrEmailTaken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, _ := pgxmock.NewPool()
			defer mock.Close()
			tt.setup(mock)

			repo := NewUserRepository(AdaptQuerier(mock))
			err := repo.Create(context.Background(), tt.user)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_GetByEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		setup   func(mock pgxmock.PgxPoolIface)
		wantID  string
		wantErr error
	}{
		{
			name:  "found",
			email: "a@b.co",
			setup: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "email", "password_hash", "display_name", "expires_at", "created_at", "updated_at", "deleted_at",
				}).AddRow("u1", "a@b.co", "hash", "Beni", nil, now, now, nil)
				mock.ExpectQuery(`SELECT id, email, password_hash, display_name, expires_at, created_at, updated_at, deleted_at FROM users`).
					WithArgs("a@b.co").WillReturnRows(rows)
			},
			wantID: "u1",
		},
		{
			name:  "not found",
			email: "x@y.co",
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(`(?s)SELECT .* FROM users`).WithArgs("x@y.co").WillReturnError(sql.ErrNoRows)
			},
			wantErr: domain.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, _ := pgxmock.NewPool()
			defer mock.Close()
			tt.setup(mock)

			repo := NewUserRepository(AdaptQuerier(mock))
			got, err := repo.GetByEmail(context.Background(), tt.email)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantID, got.ID)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_GetByID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		setup   func(mock pgxmock.PgxPoolIface)
		wantID  string
		wantErr error
	}{
		{
			name: "found",
			id:   "u1",
			setup: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "email", "password_hash", "display_name", "expires_at", "created_at", "updated_at", "deleted_at",
				}).AddRow("u1", "a@b.co", "hash", "Beni", nil, now, now, nil)
				mock.ExpectQuery(`SELECT id, email, password_hash, display_name, expires_at, created_at, updated_at, deleted_at FROM users`).
					WithArgs("u1").WillReturnRows(rows)
			},
			wantID: "u1",
		},
		{
			name: "not found",
			id:   "missing",
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(`(?s)SELECT .* FROM users`).WithArgs("missing").WillReturnError(sql.ErrNoRows)
			},
			wantErr: domain.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, _ := pgxmock.NewPool()
			defer mock.Close()
			tt.setup(mock)

			repo := NewUserRepository(AdaptQuerier(mock))
			got, err := repo.GetByID(context.Background(), tt.id)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantID, got.ID)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_Update(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	expiry := time.Now().Add(24 * time.Hour)
	mock.ExpectExec(`UPDATE users SET display_name = \$2, expires_at = \$3`).
		WithArgs("u1", "NewName", expiry).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewUserRepository(AdaptQuerier(mock))
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

	repo := NewUserRepository(AdaptQuerier(mock))
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

	repo := NewUserRepository(AdaptQuerier(mock))
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

	repo := NewUserRepository(AdaptQuerier(mock))
	err := repo.AssignDefaultRole(context.Background(), "u1")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_ListUsersPage(t *testing.T) {
	tests := []struct {
		name      string
		page      int
		limit     int
		setup     func(mock pgxmock.PgxPoolIface)
		wantTotal int
		wantLen   int
		wantErr   bool
	}{
		{
			name:  "first page returns rows",
			page:  1,
			limit: 2,
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users`).
					WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(5))
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "email", "password_hash", "display_name",
					"expires_at", "created_at", "updated_at", "deleted_at",
				}).
					AddRow("u1", "a@b.co", "h", "A", nil, now, now, nil).
					AddRow("u2", "c@d.co", "h", "B", nil, now, now, nil)
				mock.ExpectQuery(`SELECT id, email.* FROM users`).WithArgs(2, 0).WillReturnRows(rows)
			},
			wantTotal: 5,
			wantLen:   2,
		},
		{
			name:  "empty page beyond end",
			page:  10,
			limit: 10,
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users`).
					WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(3))
				mock.ExpectQuery(`SELECT id, email.* FROM users`).WithArgs(10, 90).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "email", "password_hash", "display_name",
						"expires_at", "created_at", "updated_at", "deleted_at",
					}))
			},
			wantTotal: 3,
			wantLen:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, _ := pgxmock.NewPool()
			defer mock.Close()
			tt.setup(mock)

			repo := NewUserRepository(AdaptQuerier(mock))
			users, total, err := repo.ListUsersPage(context.Background(), tt.page, tt.limit)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantTotal, total)
				require.Len(t, users, tt.wantLen)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
