package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/mocks"
)

// newAdminUsecase is a small test helper that builds the admin usecase with
// fresh mocks. It returns all four so individual tests can attach expectations.
func newAdminUsecase(ctrl *gomock.Controller) (*adminUsecase, *mocks.MockUserRepository, *mocks.MockRBACUsecase, *mocks.MockPermissionCache) {
	userRepo := mocks.NewMockUserRepository(ctrl)
	rbac := mocks.NewMockRBACUsecase(ctrl)
	cache := mocks.NewMockPermissionCache(ctrl)
	return &adminUsecase{userRepo: userRepo, rbacUsecase: rbac, cache: cache, log: testLogger()}, userRepo, rbac, cache
}

func TestAdminUsecase_ListUsers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uc, userRepo, rbac, _ := newAdminUsecase(ctrl)

	userRepo.EXPECT().ListUsersPage(gomock.Any(), 1, 10).Return([]domain.User{
		{ID: "u1", Email: "a@b.co"},
		{ID: "u2", Email: "c@d.co"},
	}, 2, nil)

	users, total, err := uc.ListUsers(context.Background(), 1, 10)
	require.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Equal(t, 2, total)
	_ = rbac
}

func TestAdminUsecase_ListUsers_AppliesDefaults(t *testing.T) {
	tests := []struct {
		name        string
		page, limit int
		wantPage    int
		wantLimit   int
	}{
		{name: "page zero defaults to 1", page: 0, limit: 10, wantPage: 1, wantLimit: 10},
		{name: "negative page defaults to 1", page: -3, limit: 10, wantPage: 1, wantLimit: 10},
		{name: "limit zero defaults to 20", page: 2, limit: 0, wantPage: 2, wantLimit: 20},
		{name: "negative limit defaults to 20", page: 2, limit: -1, wantPage: 2, wantLimit: 20},
		{name: "oversized limit capped at 100", page: 1, limit: 500, wantPage: 1, wantLimit: 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			uc, userRepo, _, _ := newAdminUsecase(ctrl)

			userRepo.EXPECT().ListUsersPage(gomock.Any(), tt.wantPage, tt.wantLimit).
				Return([]domain.User{}, 0, nil)

			_, _, err := uc.ListUsers(context.Background(), tt.page, tt.limit)
			require.NoError(t, err)
		})
	}
}

func TestAdminUsecase_GetUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uc, userRepo, _, _ := newAdminUsecase(ctrl)

	userRepo.EXPECT().GetByID(gomock.Any(), "u1").Return(domain.User{ID: "u1", Email: "a@b.co"}, nil)

	user, err := uc.GetUser(context.Background(), "u1")
	require.NoError(t, err)
	assert.Equal(t, "u1", user.ID)
}

func TestAdminUsecase_DeleteUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uc, userRepo, _, cache := newAdminUsecase(ctrl)

	userRepo.EXPECT().SoftDelete(gomock.Any(), "u1").Return(nil)
	cache.EXPECT().Invalidate(gomock.Any(), "u1").Return(nil)

	require.NoError(t, uc.DeleteUser(context.Background(), "u1"))
}

func TestAdminUsecase_SetExpiry(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uc, userRepo, _, _ := newAdminUsecase(ctrl)

	expiry := time.Now().Add(24 * time.Hour)
	userRepo.EXPECT().SetExpiry(gomock.Any(), "u1", &expiry).Return(nil)

	require.NoError(t, uc.SetExpiry(context.Background(), "u1", &expiry))
}

func TestAdminUsecase_ExtendExpiry(t *testing.T) {
	tests := []struct {
		name    string
		days    int
		user    domain.User
		wantErr bool
	}{
		{
			name: "extends from future expiry",
			days: 7,
			user: domain.User{
				ID:        "u1",
				ExpiresAt: ptrTime(time.Now().Add(24 * time.Hour)),
			},
		},
		{
			name: "extends from expired",
			days: 7,
			user: domain.User{
				ID:        "u1",
				ExpiresAt: ptrTime(time.Now().Add(-time.Hour)),
			},
		},
		{
			name: "extends from nil expiry",
			days: 7,
			user: domain.User{
				ID:        "u1",
				ExpiresAt: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			uc, userRepo, _, _ := newAdminUsecase(ctrl)

			userRepo.EXPECT().GetByID(gomock.Any(), "u1").Return(tt.user, nil)
			userRepo.EXPECT().SetExpiry(gomock.Any(), "u1", gomock.Any()).Return(nil)

			err := uc.ExtendExpiry(context.Background(), "u1", tt.days)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAdminUsecase_AssignRoleToUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uc, _, rbac, _ := newAdminUsecase(ctrl)

	rbac.EXPECT().AssignRoleToUser(gomock.Any(), "u1", "r1").Return(nil)

	require.NoError(t, uc.AssignRoleToUser(context.Background(), "u1", "r1"))
}

func TestAdminUsecase_CreateRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uc, _, rbac, _ := newAdminUsecase(ctrl)

	rbac.EXPECT().CreateRole(gomock.Any(), gomock.Any()).Return(nil)

	role := domain.Role{Name: "moderator", Description: "Moderator"}
	created, err := uc.CreateRole(context.Background(), role)
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, "moderator", created.Name)
}

func TestAdminUsecase_DeleteRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uc, _, rbac, _ := newAdminUsecase(ctrl)

	rbac.EXPECT().DeleteRole(gomock.Any(), "r1").Return(nil)

	require.NoError(t, uc.DeleteRole(context.Background(), "r1"))
}

func TestAdminUsecase_CreateRole_GeneratesID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uc, _, rbac, _ := newAdminUsecase(ctrl)

	rbac.EXPECT().CreateRole(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, role domain.Role) error {
		require.NotEmpty(t, role.ID, "should generate UUID")
		return nil
	})

	created, err := uc.CreateRole(context.Background(), domain.Role{Name: "mod"})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)
}
