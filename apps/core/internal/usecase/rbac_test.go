package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/mocks"
)

func TestRBACUsecase_HasPermission(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		perm    string
		setup   func(cache *mocks.MockPermissionCache, repo *mocks.MockRBACRepository)
		wantOK  bool
		wantErr bool
	}{
		{
			name:   "cache hit with permission present",
			userID: "u1",
			perm:   "profile:read",
			setup: func(cache *mocks.MockPermissionCache, repo *mocks.MockRBACRepository) {
				cache.EXPECT().Get(gomock.Any(), "u1").Return([]string{"profile:read", "watchlist:read"}, true)
			},
			wantOK: true,
		},
		{
			name:   "cache hit but permission missing",
			userID: "u1",
			perm:   "admin:users:manage",
			setup: func(cache *mocks.MockPermissionCache, repo *mocks.MockRBACRepository) {
				cache.EXPECT().Get(gomock.Any(), "u1").Return([]string{"profile:read"}, true)
			},
			wantOK: false,
		},
		{
			name:   "cache miss resolves from DB",
			userID: "u1",
			perm:   "watchlist:write",
			setup: func(cache *mocks.MockPermissionCache, repo *mocks.MockRBACRepository) {
				cache.EXPECT().Get(gomock.Any(), "u1").Return(nil, false)
				repo.EXPECT().ListUserPermissions(gomock.Any(), "u1").Return([]string{"profile:read", "watchlist:write"}, nil)
				cache.EXPECT().Set(gomock.Any(), "u1", []string{"profile:read", "watchlist:write"}, 5*time.Minute).Return(nil)
			},
			wantOK: true,
		},
		{
			name:   "cache miss with empty permissions",
			userID: "u1",
			perm:   "profile:read",
			setup: func(cache *mocks.MockPermissionCache, repo *mocks.MockRBACRepository) {
				cache.EXPECT().Get(gomock.Any(), "u1").Return(nil, false)
				repo.EXPECT().ListUserPermissions(gomock.Any(), "u1").Return(nil, nil)
				cache.EXPECT().Set(gomock.Any(), "u1", ([]string)(nil), 5*time.Minute).Return(nil)
			},
			wantOK: false,
		},
		{
			name:   "DB error propagates",
			userID: "u1",
			perm:   "profile:read",
			setup: func(cache *mocks.MockPermissionCache, repo *mocks.MockRBACRepository) {
				cache.EXPECT().Get(gomock.Any(), "u1").Return(nil, false)
				repo.EXPECT().ListUserPermissions(gomock.Any(), "u1").Return(nil, errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			cache := mocks.NewMockPermissionCache(ctrl)
			repo := mocks.NewMockRBACRepository(ctrl)
			uc := NewRBACUsecase(repo, cache)
			tt.setup(cache, repo)

			ok, err := uc.HasPermission(context.Background(), tt.userID, tt.perm)
			if tt.wantErr {
				assert.Error(t, err)
				assert.False(t, ok)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantOK, ok)
			}
		})
	}
}

func TestRBACUsecase_AssignmentOps(t *testing.T) {
	tests := []struct {
		name  string
		setup func(cache *mocks.MockPermissionCache, repo *mocks.MockRBACRepository)
		call  func(uc RBACUsecase) error
	}{
		{
			name: "assign role invalidates cache",
			setup: func(cache *mocks.MockPermissionCache, repo *mocks.MockRBACRepository) {
				repo.EXPECT().AssignRoleToUser(gomock.Any(), "u1", "r1").Return(nil)
				cache.EXPECT().Invalidate(gomock.Any(), "u1").Return(nil)
			},
			call: func(uc RBACUsecase) error {
				return uc.AssignRoleToUser(context.Background(), "u1", "r1")
			},
		},
		{
			name: "revoke role invalidates cache",
			setup: func(cache *mocks.MockPermissionCache, repo *mocks.MockRBACRepository) {
				repo.EXPECT().RevokeRoleFromUser(gomock.Any(), "u1", "r1").Return(nil)
				cache.EXPECT().Invalidate(gomock.Any(), "u1").Return(nil)
			},
			call: func(uc RBACUsecase) error {
				return uc.RevokeRoleFromUser(context.Background(), "u1", "r1")
			},
		},
		{
			name: "create role delegates to repo",
			setup: func(cache *mocks.MockPermissionCache, repo *mocks.MockRBACRepository) {
				role := domain.Role{ID: "r1", Name: "moderator", Description: "Moderator"}
				repo.EXPECT().CreateRole(gomock.Any(), role).Return(nil)
			},
			call: func(uc RBACUsecase) error {
				return uc.CreateRole(context.Background(), domain.Role{ID: "r1", Name: "moderator", Description: "Moderator"})
			},
		},
		{
			name: "list roles returns from repo",
			setup: func(cache *mocks.MockPermissionCache, repo *mocks.MockRBACRepository) {
				roles := []domain.Role{{ID: "r1", Name: "admin"}, {ID: "r2", Name: "user"}}
				repo.EXPECT().ListRoles(gomock.Any()).Return(roles, nil)
			},
			call: func(uc RBACUsecase) error {
				_, err := uc.ListRoles(context.Background())
				return err
			},
		},
		{
			name: "delete role delegates to repo",
			setup: func(cache *mocks.MockPermissionCache, repo *mocks.MockRBACRepository) {
				repo.EXPECT().DeleteRole(gomock.Any(), "r1").Return(nil)
			},
			call: func(uc RBACUsecase) error {
				return uc.DeleteRole(context.Background(), "r1")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			cache := mocks.NewMockPermissionCache(ctrl)
			repo := mocks.NewMockRBACRepository(ctrl)
			uc := NewRBACUsecase(repo, cache)
			tt.setup(cache, repo)

			err := tt.call(uc)
			require.NoError(t, err)
		})
	}
}
