package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=admin.go -destination=../mocks/mock_admin_usecase.go -package=mocks -typed

// AdminUsecase manages admin operations: user management and RBAC.
type AdminUsecase interface {
	// ListUsers returns paginated users (excluding soft-deleted).
	ListUsers(ctx context.Context, page, limit int) ([]domain.User, int, error)
	// GetUser returns a single user by ID.
	GetUser(ctx context.Context, id string) (domain.User, error)
	// SuspendUser soft-deletes a user.
	SuspendUser(ctx context.Context, id string) error
	// DeleteUser soft-deletes a user (alias for suspend in M1).
	DeleteUser(ctx context.Context, id string) error
	// SetExpiry updates or extends a user's expires_at.
	SetExpiry(ctx context.Context, id string, expiresAt *time.Time) error
	// ExtendExpiry extends a user's expiry by N days from now.
	ExtendExpiry(ctx context.Context, id string, days int) error

	// AssignRoleToUser assigns a role to a user.
	AssignRoleToUser(ctx context.Context, userID, roleID string) error
	// RevokeRoleFromUser revokes a role from a user.
	RevokeRoleFromUser(ctx context.Context, userID, roleID string) error
	// ListUserRoles is not in the spec but useful; omitted for M1.

	// CreateRole creates a new role and returns it with the generated ID.
	CreateRole(ctx context.Context, role domain.Role) (domain.Role, error)
	// GetRole returns a role by ID.
	GetRole(ctx context.Context, id string) (domain.Role, error)
	// ListRoles returns all roles.
	ListRoles(ctx context.Context) ([]domain.Role, error)
	// DeleteRole removes a role.
	DeleteRole(ctx context.Context, id string) error
	// AssignPermissionToRole assigns a permission to a role.
	AssignPermissionToRole(ctx context.Context, roleID, permissionID string) error
	// RevokePermissionFromRole revokes a permission from a role.
	RevokePermissionFromRole(ctx context.Context, roleID, permissionID string) error
}

type adminUsecase struct {
	userRepo    repository.UserRepository
	rbacUsecase RBACUsecase
}

// NewAdminUsecase wires up the admin usecase.
func NewAdminUsecase(userRepo repository.UserRepository, rbac RBACUsecase) AdminUsecase {
	return &adminUsecase{userRepo: userRepo, rbacUsecase: rbac}
}

func (u *adminUsecase) ListUsers(ctx context.Context, page, limit int) ([]domain.User, int, error) {
	// For M1, we return all non-deleted users (pagination is a TODO).
	users, err := u.userRepo.ListAll(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("admin list users: %w", err)
	}
	return users, len(users), nil
}

func (u *adminUsecase) GetUser(ctx context.Context, id string) (domain.User, error) {
	return u.userRepo.GetByID(ctx, id)
}

func (u *adminUsecase) SuspendUser(ctx context.Context, id string) error {
	return u.userRepo.SoftDelete(ctx, id)
}

func (u *adminUsecase) DeleteUser(ctx context.Context, id string) error {
	return u.userRepo.SoftDelete(ctx, id)
}

func (u *adminUsecase) SetExpiry(ctx context.Context, id string, expiresAt *time.Time) error {
	return u.userRepo.SetExpiry(ctx, id, expiresAt)
}

func (u *adminUsecase) ExtendExpiry(ctx context.Context, id string, days int) error {
	user, err := u.userRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	var newExpiry time.Time
	if user.ExpiresAt != nil && user.ExpiresAt.After(time.Now()) {
		newExpiry = user.ExpiresAt.Add(time.Duration(days) * 24 * time.Hour)
	} else {
		newExpiry = time.Now().Add(time.Duration(days) * 24 * time.Hour)
	}
	return u.userRepo.SetExpiry(ctx, id, &newExpiry)
}

func (u *adminUsecase) AssignRoleToUser(ctx context.Context, userID, roleID string) error {
	return u.rbacUsecase.AssignRoleToUser(ctx, userID, roleID)
}

func (u *adminUsecase) RevokeRoleFromUser(ctx context.Context, userID, roleID string) error {
	return u.rbacUsecase.RevokeRoleFromUser(ctx, userID, roleID)
}

func (u *adminUsecase) CreateRole(ctx context.Context, role domain.Role) (domain.Role, error) {
	if role.ID == "" {
		role.ID = uuid.NewString()
	}
	if err := u.rbacUsecase.CreateRole(ctx, role); err != nil {
		return domain.Role{}, err
	}
	return role, nil
}

func (u *adminUsecase) GetRole(ctx context.Context, id string) (domain.Role, error) {
	return u.rbacUsecase.GetRole(ctx, id)
}

func (u *adminUsecase) ListRoles(ctx context.Context) ([]domain.Role, error) {
	return u.rbacUsecase.ListRoles(ctx)
}

func (u *adminUsecase) DeleteRole(ctx context.Context, id string) error {
	return u.rbacUsecase.DeleteRole(ctx, id)
}

func (u *adminUsecase) AssignPermissionToRole(ctx context.Context, roleID, permissionID string) error {
	return u.rbacUsecase.AssignPermissionToRole(ctx, roleID, permissionID)
}

func (u *adminUsecase) RevokePermissionFromRole(ctx context.Context, roleID, permissionID string) error {
	return u.rbacUsecase.RevokePermissionFromRole(ctx, roleID, permissionID)
}
