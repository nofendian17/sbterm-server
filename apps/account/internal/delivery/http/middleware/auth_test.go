package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nofendian17/sbterm/apps/account/internal/domain"
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

func TestAuthMiddleware_NoHeader(t *testing.T) {
	verifier := &mockTokenVerifier{}
	loader := &mockUserLoader{}
	checker := &mockPermissionChecker{perms: map[string]bool{}}

	mw := AuthMiddleware(verifier, loader, checker)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	verifier := &mockTokenVerifier{err: assert.AnError}
	loader := &mockUserLoader{}
	checker := &mockPermissionChecker{perms: map[string]bool{}}

	mw := AuthMiddleware(verifier, loader, checker)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthMiddleware_UserNotFound(t *testing.T) {
	verifier := &mockTokenVerifier{userID: "u1"}
	loader := &mockUserLoader{err: domain.ErrUserNotFound}
	checker := &mockPermissionChecker{perms: map[string]bool{}}

	mw := AuthMiddleware(verifier, loader, checker)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthMiddleware_Expired(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	verifier := &mockTokenVerifier{userID: "u1"}
	loader := &mockUserLoader{user: domain.User{
		ID:        "u1",
		ExpiresAt: &past,
	}}
	checker := &mockPermissionChecker{perms: map[string]bool{}}

	mw := AuthMiddleware(verifier, loader, checker)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthMiddleware_Suspended(t *testing.T) {
	now := time.Now()
	verifier := &mockTokenVerifier{userID: "u1"}
	loader := &mockUserLoader{user: domain.User{
		ID:        "u1",
		DeletedAt: &now,
	}}
	checker := &mockPermissionChecker{perms: map[string]bool{}}

	mw := AuthMiddleware(verifier, loader, checker)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthMiddleware_Success(t *testing.T) {
	verifier := &mockTokenVerifier{userID: "u1"}
	loader := &mockUserLoader{user: domain.User{ID: "u1"}}
	checker := &mockPermissionChecker{perms: map[string]bool{
		"profile:read": true,
	}}

	mw := AuthMiddleware(verifier, loader, checker)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Assert context values
		userID, ok := UserIDFromContext(r.Context())
		require.True(t, ok)
		assert.Equal(t, "u1", userID)

		perms, ok := PermissionsFromContext(r.Context())
		require.True(t, ok)
		assert.Contains(t, perms, "profile:read")

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

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
