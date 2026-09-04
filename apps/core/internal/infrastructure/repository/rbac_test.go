package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
)

// pgErrNoRows simulates a Postgres "no rows" result.
func pgErrNoRows() error {
	return sql.ErrNoRows
}

// pgUniqueViolation simulates a Postgres unique constraint violation.
func pgUniqueViolation() error {
	return &pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint"}
}

func TestRBACRepository_CreateRole(t *testing.T) {
	tests := []struct {
		name    string
		role    domain.Role
		setup   func(mock pgxmock.PgxPoolIface)
		wantErr bool
	}{
		{
			name: "success",
			role: domain.Role{ID: "r1", Name: "moderator", Description: "Moderator"},
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`INSERT INTO roles`).
					WithArgs("r1", "moderator", "Moderator").
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
			},
		},
		{
			name: "duplicate name",
			role: domain.Role{ID: "r2", Name: "user", Description: "Duplicate"},
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`INSERT INTO roles`).
					WithArgs("r2", "user", "Duplicate").
					WillReturnError(pgUniqueViolation())
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, _ := pgxmock.NewPool()
			defer mock.Close()
			tt.setup(mock)

			repo := NewRBACRepository(AdaptQuerier(mock))
			err := repo.CreateRole(context.Background(), tt.role)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRBACRepository_GetRole(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		setup   func(mock pgxmock.PgxPoolIface)
		want    domain.Role
		wantErr error
	}{
		{
			name: "found",
			id:   "r1",
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "name", "description"}).
					AddRow("r1", "admin", "Full admin")
				mock.ExpectQuery(`SELECT id, name, description FROM roles`).WithArgs("r1").WillReturnRows(rows)
			},
			want: domain.Role{ID: "r1", Name: "admin", Description: "Full admin"},
		},
		{
			name: "not found",
			id:   "missing",
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(`SELECT id, name, description FROM roles`).
					WithArgs("missing").
					WillReturnError(pgErrNoRows())
			},
			wantErr: domain.ErrRoleNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, _ := pgxmock.NewPool()
			defer mock.Close()
			tt.setup(mock)

			repo := NewRBACRepository(AdaptQuerier(mock))
			got, err := repo.GetRole(context.Background(), tt.id)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRBACRepository_ListRoles(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"id", "name", "description"}).
		AddRow("r1", "admin", "Admin").
		AddRow("r2", "user", "User")
	mock.ExpectQuery(`SELECT id, name, description FROM roles`).WillReturnRows(rows)

	repo := NewRBACRepository(AdaptQuerier(mock))
	roles, err := repo.ListRoles(context.Background())
	require.NoError(t, err)
	assert.Len(t, roles, 2)
	assert.Equal(t, "admin", roles[0].Name)
	assert.Equal(t, "user", roles[1].Name)
}

func TestRBACRepository_DeleteRole(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		setup   func(mock pgxmock.PgxPoolIface)
		wantErr error
	}{
		{
			name: "success",
			id:   "r1",
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`DELETE FROM roles`).WithArgs("r1").
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
		},
		{
			name: "not found",
			id:   "missing",
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`DELETE FROM roles`).WithArgs("missing").
					WillReturnResult(pgxmock.NewResult("DELETE", 0))
			},
			wantErr: domain.ErrRoleNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, _ := pgxmock.NewPool()
			defer mock.Close()
			tt.setup(mock)

			repo := NewRBACRepository(AdaptQuerier(mock))
			err := repo.DeleteRole(context.Background(), tt.id)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRBACRepository_AssignPermissionToRole(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectExec(`INSERT INTO role_permissions`).
		WithArgs("r1", "p1").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	repo := NewRBACRepository(AdaptQuerier(mock))
	assert.NoError(t, repo.AssignPermissionToRole(context.Background(), "r1", "p1"))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRBACRepository_RevokePermissionFromRole(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectExec(`DELETE FROM role_permissions`).
		WithArgs("r1", "p1").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	repo := NewRBACRepository(AdaptQuerier(mock))
	assert.NoError(t, repo.RevokePermissionFromRole(context.Background(), "r1", "p1"))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRBACRepository_AssignRoleToUser(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectExec(`INSERT INTO user_roles`).
		WithArgs("u1", "r1").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	repo := NewRBACRepository(AdaptQuerier(mock))
	assert.NoError(t, repo.AssignRoleToUser(context.Background(), "u1", "r1"))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRBACRepository_RevokeRoleFromUser(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectExec(`DELETE FROM user_roles`).
		WithArgs("u1", "r1").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	repo := NewRBACRepository(AdaptQuerier(mock))
	assert.NoError(t, repo.RevokeRoleFromUser(context.Background(), "u1", "r1"))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRBACRepository_ListUserPermissions(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		setup   func(mock pgxmock.PgxPoolIface)
		want    []string
		wantErr bool
	}{
		{
			name:   "returns permissions",
			userID: "u1",
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"name"}).
					AddRow("profile:read").
					AddRow("profile:write").
					AddRow("watchlist:read")
				mock.ExpectQuery(`SELECT DISTINCT p.name`).WithArgs("u1").WillReturnRows(rows)
			},
			want: []string{"profile:read", "profile:write", "watchlist:read"},
		},
		{
			name:   "no permissions",
			userID: "u2",
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"name"})
				mock.ExpectQuery(`SELECT DISTINCT p.name`).WithArgs("u2").WillReturnRows(rows)
			},
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, _ := pgxmock.NewPool()
			defer mock.Close()
			tt.setup(mock)

			repo := NewRBACRepository(AdaptQuerier(mock))
			got, err := repo.ListUserPermissions(context.Background(), tt.userID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
