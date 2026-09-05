// Package usecase implements the business logic for the core domain.

package usecase

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/repository"
	"github.com/nofendian17/sbterm/libs/pkg/log"
)

const permCacheTTL = 5 * time.Minute

// rbacUsecase is the default RBACUsecase implementation.
type rbacUsecase struct {
	repo  repository.RBACRepository
	cache repository.PermissionCache
	log   log.Logger
}

// NewRBACUsecase wires up the RBAC usecase.
func NewRBACUsecase(repo repository.RBACRepository, cache repository.PermissionCache, logger log.Logger) RBACUsecase {
	return &rbacUsecase{repo: repo, cache: cache, log: logger}
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
	// Capture affected users before the delete; FK cascade on user_roles removes
	// the linking rows, so the query must run first.
	userIDs, err := u.repo.ListUserIDsByRole(ctx, id)
	if err != nil {
		u.log.Warn("rbac: failed to list users before role delete", "role_id", id, "error", err)
	}

	if err := u.repo.DeleteRole(ctx, id); err != nil {
		return fmt.Errorf("rbac delete role: %w", err)
	}

	if len(userIDs) > 0 {
		if err := u.cache.Invalidate(ctx, userIDs...); err != nil {
			u.log.Warn("rbac: failed to invalidate permission cache after role delete", "role_id", id, "error", err)
		}
	}
	return nil
}

func (u *rbacUsecase) AssignPermissionToRole(ctx context.Context, roleID, permissionID string) error {
	if err := u.repo.AssignPermissionToRole(ctx, roleID, permissionID); err != nil {
		return fmt.Errorf("rbac assign permission: %w", err)
	}
	userIDs, err := u.repo.ListUserIDsByRole(ctx, roleID)
	if err != nil {
		u.log.Warn("rbac: failed to list users for role invalidation", "role_id", roleID, "error", err)
		return nil
	}
	if len(userIDs) > 0 {
		if err := u.cache.Invalidate(ctx, userIDs...); err != nil {
			u.log.Warn("rbac: failed to invalidate permission cache", "role_id", roleID, "error", err)
		}
	}
	return nil
}

func (u *rbacUsecase) RevokePermissionFromRole(ctx context.Context, roleID, permissionID string) error {
	if err := u.repo.RevokePermissionFromRole(ctx, roleID, permissionID); err != nil {
		return fmt.Errorf("rbac revoke permission: %w", err)
	}
	userIDs, err := u.repo.ListUserIDsByRole(ctx, roleID)
	if err != nil {
		u.log.Warn("rbac: failed to list users for role invalidation", "role_id", roleID, "error", err)
		return nil
	}
	if len(userIDs) > 0 {
		if err := u.cache.Invalidate(ctx, userIDs...); err != nil {
			u.log.Warn("rbac: failed to invalidate permission cache", "role_id", roleID, "error", err)
		}
	}
	return nil
}

func (u *rbacUsecase) AssignRoleToUser(ctx context.Context, userID, roleID string) error {
	if err := u.repo.AssignRoleToUser(ctx, userID, roleID); err != nil {
		return fmt.Errorf("rbac assign role: %w", err)
	}
	if err := u.cache.Invalidate(ctx, userID); err != nil {
		u.log.Warn("rbac: failed to invalidate permission cache", "user_id", userID, "error", err)
	}
	return nil
}

func (u *rbacUsecase) RevokeRoleFromUser(ctx context.Context, userID, roleID string) error {
	if err := u.repo.RevokeRoleFromUser(ctx, userID, roleID); err != nil {
		return fmt.Errorf("rbac revoke role: %w", err)
	}
	if err := u.cache.Invalidate(ctx, userID); err != nil {
		u.log.Warn("rbac: failed to invalidate permission cache", "user_id", userID, "error", err)
	}
	return nil
}

// HasPermission checks whether the given user holds the specified permission.
// It uses the permission cache; on miss, it resolves from DB and caches.
func (u *rbacUsecase) HasPermission(ctx context.Context, userID string, perm string) (bool, error) {
	perms, err := u.ListPermissions(ctx, userID)
	if err != nil {
		return false, err
	}
	return slices.Contains(perms, perm), nil
}

// ListPermissions returns the full set of permission names for the given user.
// It uses the permission cache; on miss, it resolves from DB and caches.
func (u *rbacUsecase) ListPermissions(ctx context.Context, userID string) ([]string, error) {
	if perms, ok := u.cache.Get(ctx, userID); ok {
		return perms, nil
	}

	perms, err := u.repo.ListUserPermissions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("rbac list permissions: %w", err)
	}

	_ = u.cache.Set(ctx, userID, perms, permCacheTTL)
	return perms, nil
}
