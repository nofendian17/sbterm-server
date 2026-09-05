package repository

import (
	"context"
	"time"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=user.go -destination=../mocks/mock_user_repository.go -package=mocks -typed

// UserRepository persists and retrieves domain.User records. The contract lives
// in the repository layer so usecases depend on it without importing
// infrastructure (pgx). All implementations run queries through a repository.Querier.
type UserRepository interface {
	// Create inserts a new user row. A conflicting email (unique violation
	// 23505) is mapped to domain.ErrEmailTaken.
	Create(ctx context.Context, user domain.User) error
	// GetByEmail returns the user with the given email, or domain.ErrUserNotFound.
	GetByEmail(ctx context.Context, email string) (domain.User, error)
	// GetByID returns the user with the given id, or domain.ErrUserNotFound.
	GetByID(ctx context.Context, id string) (domain.User, error)
	// Update changes the display_name and optionally expires_at for the user id.
	Update(ctx context.Context, id, displayName string, expiresAt *time.Time) error
	// SoftDelete sets deleted_at rather than removing the row.
	SoftDelete(ctx context.Context, id string) error
	// SetExpiry updates the expires_at column for the user id.
	SetExpiry(ctx context.Context, id string, expiresAt *time.Time) error
	// AssignDefaultRole links the user to the seeded "user" role by inserting a
	// row into user_roles. The role id is resolved from the roles table by name.
	AssignDefaultRole(ctx context.Context, userID string) error
	// ListUsersPage returns a page of non-deleted users ordered by created_at, id
	// (the id tiebreaker keeps the page stable when timestamps tie). The second
	// return value is the total number of non-deleted users, used by callers to
	// compute the total page count.
	ListUsersPage(ctx context.Context, page, limit int) ([]domain.User, int, error)
}
