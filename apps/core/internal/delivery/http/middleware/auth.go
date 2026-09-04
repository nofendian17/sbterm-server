// Package middleware provides HTTP middleware for authentication, authorization, and rate limiting.

package middleware

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/repository"
	"github.com/nofendian17/sbterm/apps/core/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/response"
)

type contextKey string

const (
	// CtxUserID is the context key for the authenticated user's ID.
	CtxUserID contextKey = "user_id"
	// CtxPermissions is the context key for the user's resolved permission set.
	CtxPermissions contextKey = "permissions"
)

// KnownPermissions is the list of all permissions resolved by AuthMiddleware.
// To add a new permission, add it here and to the RBAC seed migration.
var KnownPermissions = []string{
	"auth:login",
	"profile:read",
	"profile:write",
	"watchlist:read",
	"watchlist:write",
	"admin:roles:read",
	"admin:roles:write",
	"admin:users:read",
	"admin:users:manage",
	"admin:rbac:assign",
}

// TokenVerifier verifies an access token and returns the user ID.
// Implemented by *token.JWTTokenService.
type TokenVerifier interface {
	VerifyAccess(token string) (userID string, err error)
}

// AuthDeps holds the dependencies the auth middleware needs.
type AuthDeps struct {
	Verifier TokenVerifier             // interface — narrow, only what middleware needs
	Loader   repository.UserRepository // full interface from repository port
	Checker  usecase.RBACUsecase       // full interface from usecase layer
}

// AuthMiddleware validates the Bearer token, loads the user, enforces
// account expiry and suspension, and injects the user ID and permission set
// into the request context.
func AuthMiddleware(deps AuthDeps) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token == "" {
				response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing or invalid token")
				return
			}

			userID, err := deps.Verifier.VerifyAccess(token)
			if err != nil {
				response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "invalid or expired token")
				return
			}

			user, err := deps.Loader.GetByID(r.Context(), userID)
			if err != nil {
				if errors.Is(err, domain.ErrUserNotFound) {
					response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "user not found")
					return
				}
				response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
				return
			}

			// Enforce suspension
			if user.DeletedAt != nil {
				response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "account suspended")
				return
			}

			// Enforce expiry (server-side, never from JWT claim)
			if user.ExpiresAt != nil && user.ExpiresAt.Before(time.Now()) {
				response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "account expired")
				return
			}

			// Resolve permissions (uses cache under the hood)
			perms, err := resolvePermissions(deps.Checker, r.Context(), userID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
				return
			}

			// Inject into context
			ctx := context.WithValue(r.Context(), CtxUserID, userID)
			ctx = context.WithValue(ctx, CtxPermissions, perms)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequirePermission returns a middleware that checks whether the authenticated
// user holds the given permission. Must be used after AuthMiddleware.
func RequirePermission(perm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			perms, ok := r.Context().Value(CtxPermissions).([]string)
			if !ok {
				response.Error(w, http.StatusForbidden, response.CodeForbidden, "forbidden")
				return
			}
			if slices.Contains(perms, perm) {
				next.ServeHTTP(w, r)
				return
			}
			response.Error(w, http.StatusForbidden, response.CodeForbidden, "forbidden")
		})
	}
}

// UserIDFromContext extracts the user ID from the context.
func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(CtxUserID).(string)
	return id, ok
}

// PermissionsFromContext extracts the permission set from the context.
func PermissionsFromContext(ctx context.Context) ([]string, bool) {
	perms, ok := ctx.Value(CtxPermissions).([]string)
	return perms, ok
}

// extractBearerToken extracts the token from the Authorization header.
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(auth, "Bearer ")
}

// resolvePermissions resolves the user's permission set using the checker.
func resolvePermissions(checker usecase.RBACUsecase, ctx context.Context, userID string) ([]string, error) {
	perms := make([]string, 0, len(KnownPermissions))
	for _, p := range KnownPermissions {
		ok, err := checker.HasPermission(ctx, userID, p)
		if err != nil {
			return nil, err
		}
		if ok {
			perms = append(perms, p)
		}
	}
	return perms, nil
}
