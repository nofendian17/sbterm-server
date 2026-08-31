package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm/apps/account/internal/domain"
	"github.com/nofendian17/sbterm/apps/account/internal/mocks"
)

func TestRBACUsecase_HasPermission_CacheHit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cache := mocks.NewMockPermissionCache(ctrl)
	repo := mocks.NewMockRBACRepository(ctrl)

	uc := NewRBACUsecase(repo, cache)

	// Cache hit with the permission present
	cache.EXPECT().Get(gomock.Any(), "u1").Return([]string{"profile:read", "watchlist:read"}, true)

	ok, err := uc.HasPermission(context.Background(), "u1", "profile:read")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestRBACUsecase_HasPermission_CacheHit_Missing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cache := mocks.NewMockPermissionCache(ctrl)
	repo := mocks.NewMockRBACRepository(ctrl)

	uc := NewRBACUsecase(repo, cache)

	// Cache hit but permission not present
	cache.EXPECT().Get(gomock.Any(), "u1").Return([]string{"profile:read"}, true)

	ok, err := uc.HasPermission(context.Background(), "u1", "admin:users:manage")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestRBACUsecase_HasPermission_CacheMiss(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cache := mocks.NewMockPermissionCache(ctrl)
	repo := mocks.NewMockRBACRepository(ctrl)

	uc := NewRBACUsecase(repo, cache)

	// Cache miss
	cache.EXPECT().Get(gomock.Any(), "u1").Return(nil, false)
	// DB lookup
	repo.EXPECT().ListUserPermissions(gomock.Any(), "u1").Return([]string{"profile:read", "watchlist:write"}, nil)
	// Cache the result
	cache.EXPECT().Set(gomock.Any(), "u1", []string{"profile:read", "watchlist:write"}, 5*time.Minute).Return(nil)

	ok, err := uc.HasPermission(context.Background(), "u1", "watchlist:write")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestRBACUsecase_HasPermission_CacheMiss_Empty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cache := mocks.NewMockPermissionCache(ctrl)
	repo := mocks.NewMockRBACRepository(ctrl)

	uc := NewRBACUsecase(repo, cache)

	cache.EXPECT().Get(gomock.Any(), "u1").Return(nil, false)
	repo.EXPECT().ListUserPermissions(gomock.Any(), "u1").Return(nil, nil)
	cache.EXPECT().Set(gomock.Any(), "u1", ([]string)(nil), 5*time.Minute).Return(nil)

	ok, err := uc.HasPermission(context.Background(), "u1", "profile:read")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestRBACUsecase_HasPermission_DBError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cache := mocks.NewMockPermissionCache(ctrl)
	repo := mocks.NewMockRBACRepository(ctrl)

	uc := NewRBACUsecase(repo, cache)

	cache.EXPECT().Get(gomock.Any(), "u1").Return(nil, false)
	repo.EXPECT().ListUserPermissions(gomock.Any(), "u1").Return(nil, errors.New("db error"))

	ok, err := uc.HasPermission(context.Background(), "u1", "profile:read")
	assert.Error(t, err)
	assert.False(t, ok)
}

func TestRBACUsecase_AssignRoleToUser_InvalidatesCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cache := mocks.NewMockPermissionCache(ctrl)
	repo := mocks.NewMockRBACRepository(ctrl)

	uc := NewRBACUsecase(repo, cache)

	repo.EXPECT().AssignRoleToUser(gomock.Any(), "u1", "r1").Return(nil)
	cache.EXPECT().Invalidate(gomock.Any(), "u1").Return(nil)

	require.NoError(t, uc.AssignRoleToUser(context.Background(), "u1", "r1"))
}

func TestRBACUsecase_RevokeRoleFromUser_InvalidatesCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cache := mocks.NewMockPermissionCache(ctrl)
	repo := mocks.NewMockRBACRepository(ctrl)

	uc := NewRBACUsecase(repo, cache)

	repo.EXPECT().RevokeRoleFromUser(gomock.Any(), "u1", "r1").Return(nil)
	cache.EXPECT().Invalidate(gomock.Any(), "u1").Return(nil)

	require.NoError(t, uc.RevokeRoleFromUser(context.Background(), "u1", "r1"))
}

func TestRBACUsecase_CreateRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cache := mocks.NewMockPermissionCache(ctrl)
	repo := mocks.NewMockRBACRepository(ctrl)

	uc := NewRBACUsecase(repo, cache)

	role := domain.Role{ID: "r1", Name: "moderator", Description: "Moderator"}
	repo.EXPECT().CreateRole(gomock.Any(), role).Return(nil)

	require.NoError(t, uc.CreateRole(context.Background(), role))
}

func TestRBACUsecase_ListRoles(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cache := mocks.NewMockPermissionCache(ctrl)
	repo := mocks.NewMockRBACRepository(ctrl)

	uc := NewRBACUsecase(repo, cache)

	roles := []domain.Role{
		{ID: "r1", Name: "admin"},
		{ID: "r2", Name: "user"},
	}
	repo.EXPECT().ListRoles(gomock.Any()).Return(roles, nil)

	got, err := uc.ListRoles(context.Background())
	require.NoError(t, err)
	assert.Equal(t, roles, got)
}

func TestRBACUsecase_DeleteRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cache := mocks.NewMockPermissionCache(ctrl)
	repo := mocks.NewMockRBACRepository(ctrl)

	uc := NewRBACUsecase(repo, cache)

	repo.EXPECT().DeleteRole(gomock.Any(), "r1").Return(nil)

	require.NoError(t, uc.DeleteRole(context.Background(), "r1"))
}
