// Package usecase implements the business logic for the core domain.

package usecase

import (
	"context"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=rbac.go -destination=../mocks/mock_rbac_usecase.go -package=mocks -typed

// RBACUsecase manages roles, permissions, and their assignments. Authorization
// is checked by permission, not by a static role string.
type RBACUsecase interface {
	// CreateRole creates a new role.
	CreateRole(ctx context.Context, role domain.Role) error
	// GetRole returns the role with the given id.
	GetRole(ctx context.Context, id string) (domain.Role, error)
	// ListRoles returns all roles.
	ListRoles(ctx context.Context) ([]domain.Role, error)
	// DeleteRole removes a role by id.
	DeleteRole(ctx context.Context, id string) error

	// AssignPermissionToRole links a permission to a role and invalidates
	// affected user permission caches.
	AssignPermissionToRole(ctx context.Context, roleID, permissionID string) error
	// RevokePermissionFromRole unlinks a permission from a role and invalidates
	// affected user permission caches.
	RevokePermissionFromRole(ctx context.Context, roleID, permissionID string) error

	// AssignRoleToUser links a role to a user and invalidates their permission cache.
	AssignRoleToUser(ctx context.Context, userID, roleID string) error
	// RevokeRoleFromUser unlinks a role from a user and invalidates their permission cache.
	RevokeRoleFromUser(ctx context.Context, userID, roleID string) error

	// HasPermission checks whether the given user holds the specified permission.
	// Uses the permission cache to avoid repeated DB joins.
	HasPermission(ctx context.Context, userID string, perm string) (bool, error)
}
