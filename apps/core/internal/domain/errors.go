package domain

import "errors"

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrExpired            = errors.New("account expired")
	ErrSuspended          = errors.New("account suspended")
	ErrPermissionDenied   = errors.New("permission denied")
	ErrDuplicateWatchlist = errors.New("symbol already in watchlist")
	ErrRoleNotFound       = errors.New("role not found")
	ErrRoleNameTaken      = errors.New("role name already exists")
	ErrPermissionNotFound = errors.New("permission not found")
	ErrInvalidInput       = errors.New("invalid input")
)
