package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/mocks"
)

// mockTokenVerifier implements the narrow TokenVerifier interface for tests.
type mockTokenVerifier struct {
	userID string
	err    error
}

func (m *mockTokenVerifier) VerifyAccess(token string) (string, error) {
	return m.userID, m.err
}

func TestAuthMiddleware(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		verifier *mockTokenVerifier
		loader   func(ctrl *gomock.Controller) *mocks.MockUserRepository
		checker  func(ctrl *gomock.Controller) *mocks.MockRBACUsecase
		wantCode int
		checkCtx func(t *testing.T, r *http.Request)
	}{
		{
			name:     "no header",
			verifier: &mockTokenVerifier{},
			loader:   func(ctrl *gomock.Controller) *mocks.MockUserRepository { return mocks.NewMockUserRepository(ctrl) },
			checker:  func(ctrl *gomock.Controller) *mocks.MockRBACUsecase { return mocks.NewMockRBACUsecase(ctrl) },
			wantCode: http.StatusUnauthorized,
		},
		{
			name:     "invalid token",
			token:    "invalid-token",
			verifier: &mockTokenVerifier{err: assert.AnError},
			loader:   func(ctrl *gomock.Controller) *mocks.MockUserRepository { return mocks.NewMockUserRepository(ctrl) },
			checker:  func(ctrl *gomock.Controller) *mocks.MockRBACUsecase { return mocks.NewMockRBACUsecase(ctrl) },
			wantCode: http.StatusUnauthorized,
		},
		{
			name:     "user not found",
			token:    "valid-token",
			verifier: &mockTokenVerifier{userID: "u1"},
			loader: func(ctrl *gomock.Controller) *mocks.MockUserRepository {
				m := mocks.NewMockUserRepository(ctrl)
				m.EXPECT().GetByID(gomock.Any(), "u1").Return(domain.User{}, domain.ErrUserNotFound)
				return m
			},
			checker:  func(ctrl *gomock.Controller) *mocks.MockRBACUsecase { return mocks.NewMockRBACUsecase(ctrl) },
			wantCode: http.StatusUnauthorized,
		},
		{
			name:     "expired account",
			token:    "valid-token",
			verifier: &mockTokenVerifier{userID: "u1"},
			loader: func(ctrl *gomock.Controller) *mocks.MockUserRepository {
				m := mocks.NewMockUserRepository(ctrl)
				m.EXPECT().GetByID(gomock.Any(), "u1").Return(domain.User{
					ID:        "u1",
					ExpiresAt: ptrTime(time.Now().Add(-time.Hour)),
				}, nil)
				return m
			},
			checker:  func(ctrl *gomock.Controller) *mocks.MockRBACUsecase { return mocks.NewMockRBACUsecase(ctrl) },
			wantCode: http.StatusUnauthorized,
		},
		{
			name:     "suspended account",
			token:    "valid-token",
			verifier: &mockTokenVerifier{userID: "u1"},
			loader: func(ctrl *gomock.Controller) *mocks.MockUserRepository {
				m := mocks.NewMockUserRepository(ctrl)
				m.EXPECT().GetByID(gomock.Any(), "u1").Return(domain.User{
					ID:        "u1",
					DeletedAt: ptrTime(time.Now()),
				}, nil)
				return m
			},
			checker:  func(ctrl *gomock.Controller) *mocks.MockRBACUsecase { return mocks.NewMockRBACUsecase(ctrl) },
			wantCode: http.StatusUnauthorized,
		},
		{
			name:     "success",
			token:    "valid-token",
			verifier: &mockTokenVerifier{userID: "u1"},
			loader: func(ctrl *gomock.Controller) *mocks.MockUserRepository {
				m := mocks.NewMockUserRepository(ctrl)
				m.EXPECT().GetByID(gomock.Any(), "u1").Return(domain.User{ID: "u1"}, nil)
				return m
			},
			checker: func(ctrl *gomock.Controller) *mocks.MockRBACUsecase {
				m := mocks.NewMockRBACUsecase(ctrl)
				m.EXPECT().ListPermissions(gomock.Any(), "u1").Return([]string{"profile:read"}, nil)
				return m
			},
			wantCode: http.StatusOK,
			checkCtx: func(t *testing.T, r *http.Request) {
				t.Helper()
				userID, ok := UserIDFromContext(r.Context())
				require.True(t, ok)
				assert.Equal(t, "u1", userID)

				perms, ok := PermissionsFromContext(r.Context())
				require.True(t, ok)
				assert.Contains(t, perms, "profile:read")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			loader := tt.loader(ctrl)
			checker := tt.checker(ctrl)

			mw := AuthMiddleware(AuthDeps{
				Verifier: tt.verifier,
				Loader:   loader,
				Checker:  checker,
			})
			handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.checkCtx != nil {
					tt.checkCtx(t, r)
				}
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantCode, rec.Code)
		})
	}
}

func ptrTime(t time.Time) *time.Time { return &t }

func TestRequirePermission_Present(t *testing.T) {
	ctx := context.WithValue(context.Background(), CtxPermissions, []string{"profile:read", "watchlist:read"})
	req := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)

	mw := RequirePermission("profile:read")
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequirePermission_Missing(t *testing.T) {
	ctx := context.WithValue(context.Background(), CtxPermissions, []string{"profile:read"})
	req := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)

	mw := RequirePermission("admin:users:manage")
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRequirePermission_NoPermsInContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	mw := RequirePermission("profile:read")
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}
