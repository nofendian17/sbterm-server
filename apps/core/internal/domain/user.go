package domain

import "time"

type User struct {
	ID           string
	Email        string
	PasswordHash string
	DisplayName  string
	ExpiresAt    *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

type RegisterInput struct {
	Email       string `validate:"required,email"`
	Password    string `validate:"required,min=8"`
	DisplayName string `validate:"required"`
}

type LoginInput struct {
	Email    string `validate:"required,email"`
	Password string `validate:"required"`
}
