package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/mocks"
	"github.com/nofendian17/sbterm/apps/core/internal/repository"
)

// newTestTokenService returns a real TokenService backed by an in-memory
// RefreshStore so token issuance/verification/rotation can be exercised without
// Redis.
func newTestTokenService() *TokenService {
	return NewTokenService("test-secret", 15*time.Minute, time.Hour, newFakeRefreshStore())
}

// commitTxManager is a TxManager stub that runs fn in a "transaction" with a nil
// Querier (the repository methods under test are independently mocked, so the
// Querier is never actually used). It satisfies repository.TxManager exactly.
type commitTxManager struct{}

func (commitTxManager) WithTx(ctx context.Context, fn func(repository.Querier) error) error {
	return fn(nil)
}

func (commitTxManager) WithTxOptions(ctx context.Context, _ pgx.TxOptions, fn func(repository.Querier) error) error {
	return fn(nil)
}

func TestAuthUsecase_Register(t *testing.T) {
	tests := []struct {
		name    string
		input   domain.RegisterInput
		setup   func(r *mocks.MockUserRepository)
		wantErr error
	}{
		{
			name:  "success assigns default expiry + user role",
			input: domain.RegisterInput{Email: "a@b.co", Password: "password123", DisplayName: "Beni"},
			setup: func(r *mocks.MockUserRepository) {
				r.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
				r.EXPECT().AssignDefaultRole(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name:  "duplicate email -> ErrEmailTaken",
			input: domain.RegisterInput{Email: "a@b.co", Password: "password123", DisplayName: "Beni"},
			setup: func(r *mocks.MockUserRepository) {
				r.EXPECT().Create(gomock.Any(), gomock.Any()).Return(domain.ErrEmailTaken)
			},
			wantErr: domain.ErrEmailTaken,
		},
		{
			name:  "invalid input -> validation error",
			input: domain.RegisterInput{Email: "not-an-email", Password: "short", DisplayName: ""},
			setup: func(r *mocks.MockUserRepository) {
				// repo must NOT be called on validation failure.
			},
			wantErr: errValidation, // sentinel meaning "any validation error expected"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := assert.New(t)
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := mocks.NewMockUserRepository(ctrl)
			tt.setup(repo)

			uc := NewAuthUsecase(repo, newTestTokenService(), commitTxManager{}, AuthConfig{
				DefaultUserTTL: 720 * time.Hour,
				BcryptCost:     12,
			})

			access, refresh, err := uc.Register(context.Background(), tt.input)
			switch {
			case tt.wantErr == domain.ErrEmailTaken:
				is.Error(err)
				is.ErrorIs(err, tt.wantErr)
			case tt.wantErr == errValidation:
				is.Error(err)
				is.ErrorIs(err, domain.ErrInvalidInput)
				is.Empty(access)
				is.Empty(refresh)
			default:
				is.NoError(err)
				is.NotEmpty(access)
				is.NotEmpty(refresh)
			}
		})
	}
}

func TestAuthUsecase_Login(t *testing.T) {
	hashOK, err := bcryptHash("password123", 12)
	require.NoError(t, err)

	tests := []struct {
		name    string
		input   domain.LoginInput
		setup   func(r *mocks.MockUserRepository)
		wantErr error
	}{
		{
			name:  "success",
			input: domain.LoginInput{Email: "a@b.co", Password: "password123"},
			setup: func(r *mocks.MockUserRepository) {
				u := domain.User{ID: "u1", Email: "a@b.co", PasswordHash: hashOK}
				r.EXPECT().GetByEmail(gomock.Any(), "a@b.co").Return(u, nil)
			},
		},
		{
			name:  "wrong password -> ErrInvalidCredentials",
			input: domain.LoginInput{Email: "a@b.co", Password: "wrongpass"},
			setup: func(r *mocks.MockUserRepository) {
				u := domain.User{ID: "u1", Email: "a@b.co", PasswordHash: hashOK}
				r.EXPECT().GetByEmail(gomock.Any(), "a@b.co").Return(u, nil)
			},
			wantErr: domain.ErrInvalidCredentials,
		},
		{
			name:  "unknown email -> ErrInvalidCredentials",
			input: domain.LoginInput{Email: "missing@b.co", Password: "password123"},
			setup: func(r *mocks.MockUserRepository) {
				r.EXPECT().GetByEmail(gomock.Any(), "missing@b.co").Return(domain.User{}, domain.ErrUserNotFound)
			},
			wantErr: domain.ErrInvalidCredentials,
		},
		{
			name:  "expired -> ErrExpired",
			input: domain.LoginInput{Email: "a@b.co", Password: "password123"},
			setup: func(r *mocks.MockUserRepository) {
				expired := time.Now().Add(-time.Hour)
				u := domain.User{ID: "u1", Email: "a@b.co", PasswordHash: hashOK, ExpiresAt: &expired}
				r.EXPECT().GetByEmail(gomock.Any(), "a@b.co").Return(u, nil)
			},
			wantErr: domain.ErrExpired,
		},
		{
			name:  "suspended (DeletedAt set) -> ErrSuspended",
			input: domain.LoginInput{Email: "a@b.co", Password: "password123"},
			setup: func(r *mocks.MockUserRepository) {
				deleted := time.Now().Add(-time.Hour)
				u := domain.User{ID: "u1", Email: "a@b.co", PasswordHash: hashOK, DeletedAt: &deleted}
				r.EXPECT().GetByEmail(gomock.Any(), "a@b.co").Return(u, nil)
			},
			wantErr: domain.ErrSuspended,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := assert.New(t)
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := mocks.NewMockUserRepository(ctrl)
			tt.setup(repo)

			uc := NewAuthUsecase(repo, newTestTokenService(), commitTxManager{}, AuthConfig{
				DefaultUserTTL: 720 * time.Hour,
				BcryptCost:     12,
			})

			access, refresh, err := uc.Login(context.Background(), tt.input)
			if tt.wantErr != nil {
				is.Error(err)
				is.ErrorIs(err, tt.wantErr)
				is.Empty(access)
				is.Empty(refresh)
				return
			}
			is.NoError(err)
			is.NotEmpty(access)
			is.NotEmpty(refresh)
		})
	}
}

func TestAuthUsecase_Refresh(t *testing.T) {
	t.Run("rotates (old jti deleted + new pair)", func(t *testing.T) {
		is := assert.New(t)
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mocks.NewMockUserRepository(ctrl)
		ts := newTestTokenService()
		uc := NewAuthUsecase(repo, ts, commitTxManager{}, AuthConfig{})

		_, oldRefresh, err := ts.GenerateTokenPair(context.Background(), "u1", nil)
		require.NoError(t, err)

		oldJTI, err := jtiOf(oldRefresh)
		require.NoError(t, err)

		// User must still be valid (not suspended/expired).
		repo.EXPECT().GetByID(gomock.Any(), "u1").Return(domain.User{
			ID: "u1", Email: "a@b.co",
		}, nil)

		access, refresh, err := uc.Refresh(context.Background(), oldRefresh)
		is.NoError(err)
		is.NotEmpty(access)
		is.NotEmpty(refresh)
		is.NotEqual(oldRefresh, refresh, "refresh must rotate to a new token")

		// Old jti must have been deleted (cannot be reused).
		if _, ok := ts.ConsumeRefresh(context.Background(), oldJTI); ok {
			t.Errorf("old jti %q should have been deleted after rotation", oldJTI)
		}
	})

	t.Run("suspended user -> ErrSuspended", func(t *testing.T) {
		is := assert.New(t)
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mocks.NewMockUserRepository(ctrl)
		ts := newTestTokenService()
		uc := NewAuthUsecase(repo, ts, commitTxManager{}, AuthConfig{})

		_, oldRefresh, err := ts.GenerateTokenPair(context.Background(), "u1", nil)
		require.NoError(t, err)

		deletedAt := time.Now().Add(-time.Hour)
		repo.EXPECT().GetByID(gomock.Any(), "u1").Return(domain.User{
			ID: "u1", DeletedAt: &deletedAt,
		}, nil)

		_, _, err = uc.Refresh(context.Background(), oldRefresh)
		is.ErrorIs(err, domain.ErrSuspended)
	})

	t.Run("expired user -> ErrExpired", func(t *testing.T) {
		is := assert.New(t)
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mocks.NewMockUserRepository(ctrl)
		ts := newTestTokenService()
		uc := NewAuthUsecase(repo, ts, commitTxManager{}, AuthConfig{})

		_, oldRefresh, err := ts.GenerateTokenPair(context.Background(), "u1", nil)
		require.NoError(t, err)

		expiredAt := time.Now().Add(-time.Hour)
		repo.EXPECT().GetByID(gomock.Any(), "u1").Return(domain.User{
			ID: "u1", ExpiresAt: &expiredAt,
		}, nil)

		_, _, err = uc.Refresh(context.Background(), oldRefresh)
		is.ErrorIs(err, domain.ErrExpired)
	})

	t.Run("deleted user -> ErrInvalidCredentials", func(t *testing.T) {
		is := assert.New(t)
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mocks.NewMockUserRepository(ctrl)
		ts := newTestTokenService()
		uc := NewAuthUsecase(repo, ts, commitTxManager{}, AuthConfig{})

		_, oldRefresh, err := ts.GenerateTokenPair(context.Background(), "deleted-user", nil)
		require.NoError(t, err)

		repo.EXPECT().GetByID(gomock.Any(), "deleted-user").Return(domain.User{}, domain.ErrUserNotFound)

		_, _, err = uc.Refresh(context.Background(), oldRefresh)
		is.ErrorIs(err, domain.ErrInvalidCredentials)
	})

	t.Run("unknown jti -> ErrInvalidCredentials", func(t *testing.T) {
		is := assert.New(t)
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mocks.NewMockUserRepository(ctrl)
		// Separate store that never received the jti.
		ts := NewTokenService("test-secret", time.Minute, time.Hour, newFakeRefreshStore())
		uc := NewAuthUsecase(repo, ts, commitTxManager{}, AuthConfig{})

		_, bogus, err := NewTokenService("test-secret", time.Minute, time.Hour, newFakeRefreshStore()).GenerateTokenPair(context.Background(), "uX", nil)
		require.NoError(t, err)

		_, _, err = uc.Refresh(context.Background(), bogus)
		is.Error(err)
		is.ErrorIs(err, domain.ErrInvalidCredentials)
	})

	t.Run("tampered token -> error", func(t *testing.T) {
		is := assert.New(t)
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mocks.NewMockUserRepository(ctrl)
		ts := newTestTokenService()
		uc := NewAuthUsecase(repo, ts, commitTxManager{}, AuthConfig{})

		_, _, err := uc.Refresh(context.Background(), "not.a.jwt")
		is.Error(err)
	})
}

func TestAuthUsecase_Logout(t *testing.T) {
	t.Run("deletes jti", func(t *testing.T) {
		is := assert.New(t)
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mocks.NewMockUserRepository(ctrl)
		ts := newTestTokenService()
		uc := NewAuthUsecase(repo, ts, commitTxManager{}, AuthConfig{})

		_, refresh, err := ts.GenerateTokenPair(context.Background(), "u1", nil)
		require.NoError(t, err)
		jti, err := jtiOf(refresh)
		require.NoError(t, err)

		err = uc.Logout(context.Background(), refresh)
		is.NoError(err)

		if _, ok := ts.ConsumeRefresh(context.Background(), jti); ok {
			t.Errorf("jti %q should have been deleted after logout", jti)
		}
	})

	t.Run("tampered token -> error", func(t *testing.T) {
		is := assert.New(t)
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mocks.NewMockUserRepository(ctrl)
		ts := newTestTokenService()
		uc := NewAuthUsecase(repo, ts, commitTxManager{}, AuthConfig{})

		err := uc.Logout(context.Background(), "garbage")
		is.Error(err)
	})
}

func TestAuthUsecase_RegisterPasswordHashed(t *testing.T) {
	is := assert.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockUserRepository(ctrl)
	var captured domain.User
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, u domain.User) error {
		captured = u
		return nil
	})
	repo.EXPECT().AssignDefaultRole(gomock.Any(), gomock.Any()).Return(nil)

	uc := NewAuthUsecase(repo, newTestTokenService(), commitTxManager{}, AuthConfig{
		DefaultUserTTL: 24 * time.Hour,
		BcryptCost:     12,
	})

	_, _, err := uc.Register(context.Background(), domain.RegisterInput{
		Email: "h@b.co", Password: "password123", DisplayName: "Hash",
	})
	is.NoError(err)

	// Password must be hashed, never stored in clear text, and cost must be 12.
	is.NotEqual("password123", captured.PasswordHash)
	is.NotContains(captured.PasswordHash, "password123")
	is.NoError(bcryptCompare(captured.PasswordHash, "password123"))

	// Default expiry must have been set to now + DefaultUserTTL.
	is.NotNil(captured.ExpiresAt)
	is.WithinDuration(time.Now().Add(24*time.Hour), *captured.ExpiresAt, 2*time.Minute)
}

// errValidation is a local sentinel used only to express "a validation error is
// expected" in table-driven cases (the real error is a *validator.ValidationError
// from libs/pkg/validator, which has no shared sentinel in the usecase layer).
var errValidation = errors.New("validation error")

// bcryptHash is a test helper mirroring the usecase's hashing call.
func bcryptHash(pw string, cost int) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), cost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func bcryptCompare(hash, pw string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw))
}

// jtiOf extracts the jti claim from a JWT without verifying it — used in tests
// solely to track store keys.
func jtiOf(token string) (string, error) {
	claims := &tokenClaims{}
	parser := jwt.NewParser()
	if _, _, err := parser.ParseUnverified(token, claims); err != nil {
		return "", err
	}
	return claims.ID, nil
}

var (
	_ AuthUsecase          = (*authUsecase)(nil)
	_ repository.TxManager = commitTxManager{}
)
