package usecase

import (
	"context"
	"time"

	"github.com/nofendian17/sbterm/apps/account/internal/domain"
	"github.com/nofendian17/sbterm/apps/account/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=user.go -destination=../mocks/mock_user_usecase.go -package=mocks -typed

// UserUsecase manages user profile operations.
type UserUsecase interface {
	// GetMe returns the authenticated user's profile.
	GetMe(ctx context.Context, userID string) (domain.User, error)
	// UpdateMe updates the authenticated user's display name.
	UpdateMe(ctx context.Context, userID, displayName string) error
}

type userUsecase struct {
	repo repository.UserRepository
}

// NewUserUsecase wires up the user usecase.
func NewUserUsecase(repo repository.UserRepository) UserUsecase {
	return &userUsecase{repo: repo}
}

func (u *userUsecase) GetMe(ctx context.Context, userID string) (domain.User, error) {
	user, err := u.repo.GetByID(ctx, userID)
	if err != nil {
		return domain.User{}, err
	}
	return user, nil
}

func (u *userUsecase) UpdateMe(ctx context.Context, userID, displayName string) error {
	// Load existing user to preserve other fields
	user, err := u.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	return u.repo.Update(ctx, userID, displayName, user.ExpiresAt)
}

// UserResponse is the API response for a user profile.
type UserResponse struct {
	ID          string     `json:"id"`
	Email       string     `json:"email"`
	DisplayName string     `json:"display_name"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ToResponse converts a domain.User to an API response.
func ToResponse(u domain.User) UserResponse {
	return UserResponse{
		ID:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		ExpiresAt:   u.ExpiresAt,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}
