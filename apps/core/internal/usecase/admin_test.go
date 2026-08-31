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

func TestAdminUsecase_ListUsers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	userRepo := mocks.NewMockUserRepository(ctrl)
	rbac := mocks.NewMockRBACUsecase(ctrl)

	userRepo.EXPECT().ListAll(gomock.Any()).Return([]domain.User{
		{ID: "u1", Email: "a@b.co"},
		{ID: "u2", Email: "c@d.co"},
	}, nil)

	uc := NewAdminUsecase(userRepo, rbac)
	users, total, err := uc.ListUsers(context.Background(), 1, 10)
	require.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Equal(t, 2, total)
}

func TestAdminUsecase_GetUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	userRepo := mocks.NewMockUserRepository(ctrl)
	rbac := mocks.NewMockRBACUsecase(ctrl)

	userRepo.EXPECT().GetByID(gomock.Any(), "u1").Return(domain.User{ID: "u1", Email: "a@b.co"}, nil)

	uc := NewAdminUsecase(userRepo, rbac)
	user, err := uc.GetUser(context.Background(), "u1")
	require.NoError(t, err)
	assert.Equal(t, "u1", user.ID)
}

func TestAdminUsecase_SuspendUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	userRepo := mocks.NewMockUserRepository(ctrl)
	rbac := mocks.NewMockRBACUsecase(ctrl)

	userRepo.EXPECT().SoftDelete(gomock.Any(), "u1").Return(nil)

	uc := NewAdminUsecase(userRepo, rbac)
	require.NoError(t, uc.SuspendUser(context.Background(), "u1"))
}

func TestAdminUsecase_DeleteUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	userRepo := mocks.NewMockUserRepository(ctrl)
	rbac := mocks.NewMockRBACUsecase(ctrl)

	userRepo.EXPECT().SoftDelete(gomock.Any(), "u1").Return(nil)

	uc := NewAdminUsecase(userRepo, rbac)
	require.NoError(t, uc.DeleteUser(context.Background(), "u1"))
}

func TestAdminUsecase_SetExpiry(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	userRepo := mocks.NewMockUserRepository(ctrl)
	rbac := mocks.NewMockRBACUsecase(ctrl)

	expiry := time.Now().Add(24 * time.Hour)
	userRepo.EXPECT().SetExpiry(gomock.Any(), "u1", &expiry).Return(nil)

	uc := NewAdminUsecase(userRepo, rbac)
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
			userRepo := mocks.NewMockUserRepository(ctrl)
			rbac := mocks.NewMockRBACUsecase(ctrl)

			userRepo.EXPECT().GetByID(gomock.Any(), "u1").Return(tt.user, nil)
			userRepo.EXPECT().SetExpiry(gomock.Any(), "u1", gomock.Any()).Return(nil)

			uc := NewAdminUsecase(userRepo, rbac)
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
	userRepo := mocks.NewMockUserRepository(ctrl)
	rbac := mocks.NewMockRBACUsecase(ctrl)

	rbac.EXPECT().AssignRoleToUser(gomock.Any(), "u1", "r1").Return(nil)

	uc := NewAdminUsecase(userRepo, rbac)
	require.NoError(t, uc.AssignRoleToUser(context.Background(), "u1", "r1"))
}

func TestAdminUsecase_CreateRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	userRepo := mocks.NewMockUserRepository(ctrl)
	rbac := mocks.NewMockRBACUsecase(ctrl)

	rbac.EXPECT().CreateRole(gomock.Any(), gomock.Any()).Return(nil)

	uc := NewAdminUsecase(userRepo, rbac)
	role := domain.Role{Name: "moderator", Description: "Moderator"}
	require.NoError(t, uc.CreateRole(context.Background(), role))
}

func TestAdminUsecase_DeleteRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	userRepo := mocks.NewMockUserRepository(ctrl)
	rbac := mocks.NewMockRBACUsecase(ctrl)

	rbac.EXPECT().DeleteRole(gomock.Any(), "r1").Return(nil)

	uc := NewAdminUsecase(userRepo, rbac)
	require.NoError(t, uc.DeleteRole(context.Background(), "r1"))
}

func TestAdminUsecase_CreateRole_GeneratesID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	userRepo := mocks.NewMockUserRepository(ctrl)
	rbac := mocks.NewMockRBACUsecase(ctrl)

	rbac.EXPECT().CreateRole(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, role domain.Role) error {
		require.NotEmpty(t, role.ID, "should generate UUID")
		return nil
	})

	uc := NewAdminUsecase(userRepo, rbac)
	require.NoError(t, uc.CreateRole(context.Background(), domain.Role{Name: "mod"}))
}
