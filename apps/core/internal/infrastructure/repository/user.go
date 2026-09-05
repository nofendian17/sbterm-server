// Package repository implements the repository contracts using PostgreSQL/Redis backends.

package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/repository"
)

// uniqueViolationCode is the Postgres SQLSTATE for a unique constraint breach.
const uniqueViolationCode = "23505"

// userRepository is the pgx implementation of repository.UserRepository. It
// runs every query through a repository.Querier so the same code works outside
// (pool) and inside (tx) a transaction.
type UserRepository struct {
	q repository.Querier
}

// NewUserRepository builds a UserRepository backed by the given Querier (a
// *pgxpool.Pool or a pgx.Tx both satisfy it). pgxmock pools satisfy it too,
// which is what the tests use.
func NewUserRepository(q repository.Querier) *UserRepository {
	return &UserRepository{q: q}
}

// Create inserts a new user. email, password_hash and display_name are
// provided; id/created_at/updated_at default in Postgres, expires_at is null.
func (r *UserRepository) Create(ctx context.Context, user domain.User) error {
	const q = `
		INSERT INTO users (id, email, password_hash, display_name, expires_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.q.Exec(ctx, q,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.DisplayName,
		user.ExpiresAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("user create: %w", domain.ErrEmailTaken)
		}
		return fmt.Errorf("user create: %w", err)
	}
	return nil
}

// GetByEmail returns the user matching email, or domain.ErrUserNotFound.
// Soft-deleted users are treated as not found.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	return r.scanUser(ctx,
		`SELECT id, email, password_hash, display_name, expires_at, created_at, updated_at, deleted_at
		 FROM users WHERE email = $1 AND deleted_at IS NULL`,
		email,
	)
}

// GetByID returns the user matching id, or domain.ErrUserNotFound.
// Soft-deleted users are treated as not found.
func (r *UserRepository) GetByID(ctx context.Context, id string) (domain.User, error) {
	return r.scanUser(ctx,
		`SELECT id, email, password_hash, display_name, expires_at, created_at, updated_at, deleted_at
		 FROM users WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)
}

// scanUser runs a single-row query and maps the result to domain.User,
// translating sql.ErrNoRows into domain.ErrUserNotFound.
func (r *UserRepository) scanUser(ctx context.Context, query string, args ...any) (domain.User, error) {
	var u domain.User
	err := r.q.QueryRow(ctx, query, args...).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.DisplayName,
		&u.ExpiresAt,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, fmt.Errorf("user get: %w", domain.ErrUserNotFound)
		}
		return domain.User{}, fmt.Errorf("user get: %w", err)
	}
	return u, nil
}

// Update sets display_name and expires_at for the given id.
func (r *UserRepository) Update(ctx context.Context, id, displayName string, expiresAt *time.Time) error {
	const q = `UPDATE users SET display_name = $2, expires_at = $3, updated_at = now() WHERE id = $1`
	if _, err := r.q.Exec(
		ctx,
		q,
		id,
		displayName,
		derefTime(expiresAt),
	); err != nil {
		return fmt.Errorf("user update: %w", err)
	}
	return nil
}

// SoftDelete sets deleted_at instead of deleting the row.
func (r *UserRepository) SoftDelete(ctx context.Context, id string) error {
	const q = `UPDATE users SET deleted_at = now() WHERE id = $1`
	if _, err := r.q.Exec(ctx, q, id); err != nil {
		return fmt.Errorf("user soft delete: %w", err)
	}
	return nil
}

// SetExpiry updates the expires_at column for the given id.
func (r *UserRepository) SetExpiry(ctx context.Context, id string, expiresAt *time.Time) error {
	const q = `UPDATE users SET expires_at = $1, updated_at = now() WHERE id = $2`
	if _, err := r.q.Exec(
		ctx,
		q,
		derefTime(expiresAt),
		id,
	); err != nil {
		return fmt.Errorf("user set expiry: %w", err)
	}
	return nil
}

// AssignDefaultRole links the user to the seeded "user" role by inserting a row
// into user_roles. The role id is resolved from roles by name in SQL. Soft-
// deleted roles are excluded.
func (r *UserRepository) AssignDefaultRole(ctx context.Context, userID string) error {
	const q = `
		INSERT INTO user_roles (user_id, role_id)
		SELECT $1, id FROM roles WHERE name = 'user' AND deleted_at IS NULL
		ON CONFLICT (user_id, role_id) DO UPDATE
		SET deleted_at = NULL, updated_at = now()
	`
	if _, err := r.q.Exec(ctx, q, userID); err != nil {
		return fmt.Errorf("user assign default role: %w", err)
	}
	return nil
}

// ListUsersPage returns a page of non-deleted users ordered by created_at, id
// and the total count of non-deleted users.
func (r *UserRepository) ListUsersPage(ctx context.Context, page, limit int) ([]domain.User, int, error) {
	// Total count first.
	var total int
	if err := r.q.QueryRow(ctx,
		`SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("user list count: %w", err)
	}

	offset := (page - 1) * limit
	rows, err := r.q.Query(ctx,
		`SELECT id, email, password_hash, display_name, expires_at, created_at, updated_at, deleted_at
		 FROM users WHERE deleted_at IS NULL ORDER BY created_at, id LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("user list page: %w", err)
	}
	defer rows.Close()

	users := []domain.User{}
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(
			&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName,
			&u.ExpiresAt, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("user list page scan: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("user list page rows: %w", err)
	}
	return users, total, nil
}

// isUniqueViolation reports whether err is (or wraps) a Postgres unique
// violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == uniqueViolationCode
	}
	return false
}

// derefTime returns the pointed-to time, or nil when the pointer is nil, so a
// *time.Time maps cleanly onto a nullable TIMESTAMPTZ column argument.
func derefTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}
