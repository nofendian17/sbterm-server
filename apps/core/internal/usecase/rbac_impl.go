package usecase

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/repository"
)

const permCacheTTL = 5 * time.Minute

// rbacUsecase is the default RBACUsecase implementation.
type rbacUsecase struct {
	repo  repository.RBACRepository
	cache repository.PermissionCache
}

// NewRBACUsecase wires up the RBAC usecase.
func NewRBACUsecase(repo repository.RBACRepository, cache repository.PermissionCache) RBACUsecase {
	return &rbacUsecase{repo: repo, cache: cache}
}

func (u *rbacUsecase) CreateRole(ctx context.Context, role domain.Role) error {
	if err := u.repo.CreateRole(ctx, role); err != nil {
		return fmt.Errorf("rbac create role: %w", err)
	}
	return nil
}

func (u *rbacUsecase) GetRole(ctx context.Context, id string) (domain.Role, error) {
	role, err := u.repo.GetRole(ctx, id)
	if err != nil {
		return domain.Role{}, fmt.Errorf("rbac get role: %w", err)
	}
	return role, nil
}

func (u *rbacUsecase) ListRoles(ctx context.Context) ([]domain.Role, error) {
	roles, err := u.repo.ListRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("rbac list roles: %w", err)
	}
	return roles, nil
}

func (u *rbacUsecase) DeleteRole(ctx context.Context, id string) error {
	if err := u.repo.DeleteRole(ctx, id); err != nil {
		return fmt.Errorf("rbac delete role: %w", err)
	}
	return nil
}

func (u *rbacUsecase) AssignPermissionToRole(ctx context.Context, roleID, permissionID string) error {
	if err := u.repo.AssignPermissionToRole(ctx, roleID, permissionID); err != nil {
		return fmt.Errorf("rbac assign permission: %w", err)
	}
	// Cache invalidation for affected users would require knowing which users
	// have this role — for M1 simplicity, we rely on TTL expiry. A production
	// system would query user_roles for the role and invalidate each.
	return nil
}

func (u *rbacUsecase) RevokePermissionFromRole(ctx context.Context, roleID, permissionID string) error {
	if err := u.repo.RevokePermissionFromRole(ctx, roleID, permissionID); err != nil {
		return fmt.Errorf("rbac revoke permission: %w", err)
	}
	return nil
}

func (u *rbacUsecase) AssignRoleToUser(ctx context.Context, userID, roleID string) error {
	if err := u.repo.AssignRoleToUser(ctx, userID, roleID); err != nil {
		return fmt.Errorf("rbac assign role: %w", err)
	}
	// Invalidate the user's permission cache so the new role takes effect
	// immediately.
	_ = u.cache.Invalidate(ctx, userID)
	return nil
}

func (u *rbacUsecase) RevokeRoleFromUser(ctx context.Context, userID, roleID string) error {
	if err := u.repo.RevokeRoleFromUser(ctx, userID, roleID); err != nil {
		return fmt.Errorf("rbac revoke role: %w", err)
	}
	_ = u.cache.Invalidate(ctx, userID)
	return nil
}

// HasPermission checks whether the given user holds the specified permission.
// It uses the permission cache; on miss, it resolves from DB and caches.
func (u *rbacUsecase) HasPermission(ctx context.Context, userID string, perm string) (bool, error) {
	// Try cache first
	if perms, ok := u.cache.Get(ctx, userID); ok {
		return slices.Contains(perms, perm), nil
	}

	// Cache miss: resolve from DB
	perms, err := u.repo.ListUserPermissions(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("rbac has permission: %w", err)
	}

	// Cache the result (even if empty — avoids repeated lookups for users with
	// no permissions).
	_ = u.cache.Set(ctx, userID, perms, permCacheTTL)

	return slices.Contains(perms, perm), nil
}
