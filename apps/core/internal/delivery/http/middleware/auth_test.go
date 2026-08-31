package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
)

// mockTokenVerifier implements TokenVerifier for tests.
type mockTokenVerifier struct {
	userID string
	err    error
}

func (m *mockTokenVerifier) VerifyAccess(token string) (string, error) {
	return m.userID, m.err
}

// mockUserLoader implements UserLoader for tests.
type mockUserLoader struct {
	user domain.User
	err  error
}

func (m *mockUserLoader) GetByID(_ context.Context, _ string) (domain.User, error) {
	return m.user, m.err
}

// mockPermissionChecker implements PermissionChecker for tests.
type mockPermissionChecker struct {
	perms map[string]bool
	err   error
}

func (m *mockPermissionChecker) HasPermission(_ context.Context, _ string, perm string) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.perms[perm], nil
}

func TestAuthMiddleware(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		verifier *mockTokenVerifier
		loader   *mockUserLoader
		checker  *mockPermissionChecker
		wantCode int
		checkCtx func(t *testing.T, r *http.Request)
	}{
		{
			name:     "no header",
			verifier: &mockTokenVerifier{},
			loader:   &mockUserLoader{},
			checker:  &mockPermissionChecker{perms: map[string]bool{}},
			wantCode: http.StatusUnauthorized,
		},
		{
			name:     "invalid token",
			token:    "invalid-token",
			verifier: &mockTokenVerifier{err: assert.AnError},
			loader:   &mockUserLoader{},
			checker:  &mockPermissionChecker{perms: map[string]bool{}},
			wantCode: http.StatusUnauthorized,
		},
		{
			name:     "user not found",
			token:    "valid-token",
			verifier: &mockTokenVerifier{userID: "u1"},
			loader:   &mockUserLoader{err: domain.ErrUserNotFound},
			checker:  &mockPermissionChecker{perms: map[string]bool{}},
			wantCode: http.StatusUnauthorized,
		},
		{
			name:     "expired account",
			token:    "valid-token",
			verifier: &mockTokenVerifier{userID: "u1"},
			loader: &mockUserLoader{user: domain.User{
				ID:        "u1",
				ExpiresAt: ptrTime(time.Now().Add(-time.Hour)),
			}},
			checker:  &mockPermissionChecker{perms: map[string]bool{}},
			wantCode: http.StatusUnauthorized,
		},
		{
			name:     "suspended account",
			token:    "valid-token",
			verifier: &mockTokenVerifier{userID: "u1"},
			loader: &mockUserLoader{user: domain.User{
				ID:        "u1",
				DeletedAt: ptrTime(time.Now()),
			}},
			checker:  &mockPermissionChecker{perms: map[string]bool{}},
			wantCode: http.StatusUnauthorized,
		},
		{
			name:     "success",
			token:    "valid-token",
			verifier: &mockTokenVerifier{userID: "u1"},
			loader:   &mockUserLoader{user: domain.User{ID: "u1"}},
			checker:  &mockPermissionChecker{perms: map[string]bool{"profile:read": true}},
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
			mw := AuthMiddleware(tt.verifier, tt.loader, tt.checker)
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
