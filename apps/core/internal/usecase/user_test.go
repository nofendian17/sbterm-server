package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/mocks"
)

func TestUserUsecase_GetMe(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		setup   func(repo *mocks.MockUserRepository)
		wantErr bool
	}{
		{
			name:   "success",
			userID: "u1",
			setup: func(repo *mocks.MockUserRepository) {
				repo.EXPECT().GetByID(gomock.Any(), "u1").Return(domain.User{ID: "u1", Email: "a@b.co"}, nil)
			},
		},
		{
			name:   "not found",
			userID: "missing",
			setup: func(repo *mocks.MockUserRepository) {
				repo.EXPECT().GetByID(gomock.Any(), "missing").Return(domain.User{}, domain.ErrUserNotFound)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			repo := mocks.NewMockUserRepository(ctrl)
			tt.setup(repo)

			uc := NewUserUsecase(repo)
			got, err := uc.GetMe(context.Background(), tt.userID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "u1", got.ID)
			}
		})
	}
}

func TestUserUsecase_UpdateMe(t *testing.T) {
	tests := []struct {
		name        string
		userID      string
		displayName string
		setup       func(repo *mocks.MockUserRepository)
		wantErr     bool
	}{
		{
			name:        "success",
			userID:      "u1",
			displayName: "New Name",
			setup: func(repo *mocks.MockUserRepository) {
				repo.EXPECT().GetByID(gomock.Any(), "u1").Return(domain.User{ID: "u1", ExpiresAt: nil}, nil)
				repo.EXPECT().Update(gomock.Any(), "u1", "New Name", (*time.Time)(nil)).Return(nil)
			},
		},
		{
			name:   "user not found",
			userID: "missing",
			setup: func(repo *mocks.MockUserRepository) {
				repo.EXPECT().GetByID(gomock.Any(), "missing").Return(domain.User{}, domain.ErrUserNotFound)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			repo := mocks.NewMockUserRepository(ctrl)
			tt.setup(repo)

			uc := NewUserUsecase(repo)
			err := uc.UpdateMe(context.Background(), tt.userID, tt.displayName)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
