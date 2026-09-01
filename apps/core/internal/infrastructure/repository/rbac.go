// Package repository implements the repository contracts using PostgreSQL/Redis backends.

package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/repository"
)

// rbacRepository is the pgx implementation of repository.RBACRepository.
type RBACRepository struct {
	q repository.Querier
}

// NewRBACRepository builds an RBACRepository backed by the given Querier.
func NewRBACRepository(q repository.Querier) *RBACRepository {
	return &RBACRepository{q: q}
}

// CreateRole inserts a new role. A conflicting name maps to a domain error.
func (r *RBACRepository) CreateRole(ctx context.Context, role domain.Role) error {
	const q = `INSERT INTO roles (id, name, description) VALUES ($1, $2, $3)`
	_, err := r.q.Exec(
		ctx,
		q,
		role.ID,
		role.Name,
		role.Description,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("role create: %w", domain.ErrRoleNameTaken)
		}
		return fmt.Errorf("role create: %w", err)
	}
	return nil
}

// GetRole returns the role with the given id.
func (r *RBACRepository) GetRole(ctx context.Context, id string) (domain.Role, error) {
	var role domain.Role
	err := r.q.QueryRow(ctx,
		`SELECT id, name, description FROM roles WHERE id = $1`, id,
	).Scan(&role.ID, &role.Name, &role.Description)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Role{}, fmt.Errorf("role get: %w", domain.ErrRoleNotFound)
		}
		return domain.Role{}, fmt.Errorf("role get: %w", err)
	}
	return role, nil
}

// ListRoles returns all roles.
func (r *RBACRepository) ListRoles(ctx context.Context) ([]domain.Role, error) {
	rows, err := r.q.Query(ctx, `SELECT id, name, description FROM roles ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("role list: %w", err)
	}
	defer rows.Close()

	roles := []domain.Role{}
	for rows.Next() {
		var role domain.Role
		if err := rows.Scan(&role.ID, &role.Name, &role.Description); err != nil {
			return nil, fmt.Errorf("role list scan: %w", err)
		}
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("role list rows: %w", err)
	}
	return roles, nil
}

// DeleteRole removes a role by id.
func (r *RBACRepository) DeleteRole(ctx context.Context, id string) error {
	const q = `DELETE FROM roles WHERE id = $1`
	tag, err := r.q.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("role delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("role delete: %w", domain.ErrRoleNotFound)
	}
	return nil
}

// AssignPermissionToRole links a permission to a role.
func (r *RBACRepository) AssignPermissionToRole(ctx context.Context, roleID, permissionID string) error {
	const q = `
		INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2)
		ON CONFLICT (role_id, permission_id) DO NOTHING
	`
	if _, err := r.q.Exec(
		ctx,
		q,
		roleID,
		permissionID,
	); err != nil {
		return fmt.Errorf("assign permission to role: %w", err)
	}
	return nil
}

// RevokePermissionFromRole unlinks a permission from a role.
func (r *RBACRepository) RevokePermissionFromRole(ctx context.Context, roleID, permissionID string) error {
	const q = `DELETE FROM role_permissions WHERE role_id = $1 AND permission_id = $2`
	if _, err := r.q.Exec(
		ctx,
		q,
		roleID,
		permissionID,
	); err != nil {
		return fmt.Errorf("revoke permission from role: %w", err)
	}
	return nil
}

// AssignRoleToUser links a role to a user.
func (r *RBACRepository) AssignRoleToUser(ctx context.Context, userID, roleID string) error {
	const q = `
		INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)
		ON CONFLICT (user_id, role_id) DO NOTHING
	`
	if _, err := r.q.Exec(
		ctx,
		q,
		userID,
		roleID,
	); err != nil {
		return fmt.Errorf("assign role to user: %w", err)
	}
	return nil
}

// RevokeRoleFromUser unlinks a role from a user.
func (r *RBACRepository) RevokeRoleFromUser(ctx context.Context, userID, roleID string) error {
	const q = `DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2`
	if _, err := r.q.Exec(
		ctx,
		q,
		userID,
		roleID,
	); err != nil {
		return fmt.Errorf("revoke role from user: %w", err)
	}
	return nil
}

// ListUserPermissions returns the deduplicated set of permission names for the
// user via the user_roles → role_permissions → permissions join.
func (r *RBACRepository) ListUserPermissions(ctx context.Context, userID string) ([]string, error) {
	const q = `
		SELECT DISTINCT p.name
		FROM permissions p
		JOIN role_permissions rp ON rp.permission_id = p.id
		JOIN user_roles ur ON ur.role_id = rp.role_id
		WHERE ur.user_id = $1
		ORDER BY p.name
	`
	rows, err := r.q.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list user permissions: %w", err)
	}
	defer rows.Close()

	perms := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("list user permissions scan: %w", err)
		}
		perms = append(perms, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list user permissions rows: %w", err)
	}
	return perms, nil
}
