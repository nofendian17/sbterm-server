package dto

import (
	"time"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
)

type UserResponse struct {
	ID          string     `json:"id"`
	Email       string     `json:"email"`
	DisplayName string     `json:"display_name"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func ToUserResponse(u domain.User) UserResponse {
	return UserResponse{
		ID:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		ExpiresAt:   u.ExpiresAt,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}
