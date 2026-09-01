package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	appvalidator "github.com/nofendian17/sbterm/libs/pkg/validator"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=auth_usecase.go -destination=../mocks/mock_auth_usecase.go -package=mocks -typed

// AuthUsecase bundles the account registration / authentication flows. It is
// consumed by the HTTP handler layer (a later task) and depends only on the
// repository/usecase contracts, never on infrastructure.
type AuthUsecase interface {
	// Register validates the input, hashes the password, persists the user with
	// a default per-user expiry and the seeded "user" role, then returns a fresh
	// access/refresh token pair.
	Register(ctx context.Context, input domain.RegisterInput) (access, refresh string, err error)
	// Login authenticates by email+password, rejecting expired or soft-deleted
	// accounts, and returns a fresh token pair.
	Login(ctx context.Context, input domain.LoginInput) (access, refresh string, err error)
	// Refresh verifies a refresh token, rotates it (single-use: the old jti is
	// consumed and a new pair is issued), and returns the new pair.
	Refresh(ctx context.Context, refreshToken string) (access, refresh string, err error)
	// Logout deletes the refresh token's jti so it can no longer be used.
	Logout(ctx context.Context, refreshToken string) error
}

// AuthConfig is the subset of configuration the auth usecase needs. It is
// copied here (rather than importing infrastructure/config) to keep the
// usecase layer free of viper.
type AuthConfig struct {
	DefaultUserTTL time.Duration
	BcryptCost     int
}

// authUsecase is the default AuthUsecase implementation.
type authUsecase struct {
	repo   repository.UserRepository
	tokens *TokenService
	txm    repository.TxManager
	cfg    AuthConfig
}

// NewAuthUsecase wires up the auth usecase.
func NewAuthUsecase(
	repo repository.UserRepository,
	tokens *TokenService,
	txm repository.TxManager,
	cfg AuthConfig,
) AuthUsecase {
	return &authUsecase{
		repo:   repo,
		tokens: tokens,
		txm:    txm,
		cfg:    cfg,
	}
}

// Register implements AuthUsecase.
func (u *authUsecase) Register(ctx context.Context, input domain.RegisterInput) (string, string, error) {
	v := appvalidator.New()
	if err := v.Validate(input); err != nil {
		return "", "", fmt.Errorf("%w: %w", domain.ErrInvalidInput, err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), u.bcryptCost())
	if err != nil {
		return "", "", fmt.Errorf("auth register: hash password: %w", err)
	}

	now := time.Now()
	user := domain.User{
		ID:           uuid.NewString(),
		Email:        input.Email,
		PasswordHash: string(hash),
		DisplayName:  input.DisplayName,
		ExpiresAt:    ptrTime(now.Add(u.cfg.DefaultUserTTL)),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Create the user and assign the default role atomically.
	if err := u.txm.WithTx(ctx, func(_ repository.Querier) error {
		if err := u.repo.Create(ctx, user); err != nil {
			return err
		}
		return u.repo.AssignDefaultRole(ctx, user.ID)
	}); err != nil {
		return "", "", err
	}

	access, refresh, err := u.tokens.GenerateTokenPair(ctx, user.ID, user.ExpiresAt)
	if err != nil {
		return "", "", fmt.Errorf("auth register: issue tokens: %w", err)
	}
	return access, refresh, nil
}

// Login implements AuthUsecase.
func (u *authUsecase) Login(ctx context.Context, input domain.LoginInput) (string, string, error) {
	v := appvalidator.New()
	if err := v.Validate(input); err != nil {
		return "", "", fmt.Errorf("%w: %w", domain.ErrInvalidInput, err)
	}

	user, err := u.repo.GetByEmail(ctx, input.Email)
	if err != nil {
		// Same generic error whether the user is missing or the password wrong,
		// to avoid account enumeration.
		if errors.Is(err, domain.ErrUserNotFound) {
			return "", "", domain.ErrInvalidCredentials
		}
		return "", "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return "", "", domain.ErrInvalidCredentials
	}

	// Per-user expiry guard.
	if user.ExpiresAt != nil && user.ExpiresAt.Before(time.Now()) {
		return "", "", domain.ErrExpired
	}

	// Soft-delete guard.
	if user.DeletedAt != nil {
		return "", "", domain.ErrSuspended
	}

	access, refresh, err := u.tokens.GenerateTokenPair(ctx, user.ID, user.ExpiresAt)
	if err != nil {
		return "", "", fmt.Errorf("auth login: issue tokens: %w", err)
	}
	return access, refresh, nil
}

// Refresh implements AuthUsecase.
func (u *authUsecase) Refresh(ctx context.Context, refreshToken string) (string, string, error) {
	userID, jti, err := u.tokens.VerifyRefresh(refreshToken)
	if err != nil {
		return "", "", domain.ErrInvalidCredentials
	}

	// Single-use: the jti must exist (it was stored when the pair was issued).
	if _, ok := u.tokens.ConsumeRefresh(ctx, jti); !ok {
		return "", "", domain.ErrInvalidCredentials
	}
	// Delete the old jti so it cannot be replayed (rotation).
	if err := u.tokens.DeleteRefresh(ctx, jti); err != nil {
		return "", "", fmt.Errorf("auth refresh: delete old jti: %w", err)
	}

	// Verify the user still exists and is not suspended/expired.
	user, err := u.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return "", "", domain.ErrInvalidCredentials
		}
		return "", "", err
	}
	if user.DeletedAt != nil {
		return "", "", domain.ErrSuspended
	}
	if user.ExpiresAt != nil && user.ExpiresAt.Before(time.Now()) {
		return "", "", domain.ErrExpired
	}

	access, refresh, err := u.tokens.GenerateTokenPair(ctx, userID, nil)
	if err != nil {
		return "", "", fmt.Errorf("auth refresh: issue tokens: %w", err)
	}
	return access, refresh, nil
}

// Logout implements AuthUsecase.
func (u *authUsecase) Logout(ctx context.Context, refreshToken string) error {
	_, jti, err := u.tokens.VerifyRefresh(refreshToken)
	if err != nil {
		return domain.ErrInvalidCredentials
	}
	if err := u.tokens.DeleteRefresh(ctx, jti); err != nil {
		return fmt.Errorf("auth logout: delete jti: %w", err)
	}
	return nil
}

// bcryptCost returns the configured bcrypt cost, falling back to the secure
// default of 12 when unset.
func (u *authUsecase) bcryptCost() int {
	if u.cfg.BcryptCost <= 0 {
		return 12
	}
	return u.cfg.BcryptCost
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
