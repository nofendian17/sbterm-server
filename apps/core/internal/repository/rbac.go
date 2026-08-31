package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=rbac.go -destination=../mocks/mock_rbac_repository.go -package=mocks -typed

// RBACRepository manages roles, permissions, and their assignments. The contract
// lives in the repository layer so usecases depend on it without importing
// infrastructure (pgx). All implementations run queries through a repository.Querier.
type RBACRepository interface {
	// CreateRole inserts a new role. A conflicting name (unique violation 23505)
	// is mapped to a specific error.
	CreateRole(ctx context.Context, role domain.Role) error
	// GetRole returns the role with the given id, or domain.ErrRoleNotFound.
	GetRole(ctx context.Context, id string) (domain.Role, error)
	// ListRoles returns all roles.
	ListRoles(ctx context.Context) ([]domain.Role, error)
	// DeleteRole removes a role by id. Cascading deletes handle role_permissions.
	DeleteRole(ctx context.Context, id string) error

	// AssignPermissionToRole links a permission to a role. Duplicate assignments
	// are silently ignored (ON CONFLICT DO NOTHING).
	AssignPermissionToRole(ctx context.Context, roleID, permissionID string) error
	// RevokePermissionFromRole unlinks a permission from a role.
	RevokePermissionFromRole(ctx context.Context, roleID, permissionID string) error

	// AssignRoleToUser links a role to a user. Duplicate assignments are silently ignored.
	AssignRoleToUser(ctx context.Context, userID, roleID string) error
	// RevokeRoleFromUser unlinks a role from a user.
	RevokeRoleFromUser(ctx context.Context, userID, roleID string) error

	// ListUserPermissions returns the deduplicated set of permission names
	// (e.g. "profile:read") granted to the user via all their assigned roles.
	ListUserPermissions(ctx context.Context, userID string) ([]string, error)
}
