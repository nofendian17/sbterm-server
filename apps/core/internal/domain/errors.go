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
	// ErrStockNotFound is returned when a watchlist add references a symbol
	// that has no row in the stocks master table (FK violation). It is also
	// the sentinel the stocks feature surfaces for missing catalog entries.
	ErrStockNotFound            = errors.New("stock not found")
	ErrStockSymbolTaken         = errors.New("stock symbol already exists")
	ErrStockHasWatchlists       = errors.New("stock has active watchlists")
	ErrStockSyncFailed          = errors.New("stock sync failed")
	ErrSectorNotFound           = errors.New("sector not found")
	ErrSectorNameTaken          = errors.New("sector name already exists")
	ErrSectorHasStocks          = errors.New("sector has stocks")
	ErrCompanyProfileNotFound   = errors.New("company profile not found")
	ErrCompanyProfileSyncFailed = errors.New("company profile sync failed")
	ErrRoleNotFound             = errors.New("role not found")
	ErrRoleNameTaken            = errors.New("role name already exists")
	ErrPermissionNotFound       = errors.New("permission not found")
	ErrInvalidInput             = errors.New("invalid input")
)
