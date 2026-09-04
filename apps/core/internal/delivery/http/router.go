// Package http provides HTTP handlers for the core service API.

package http

import (
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	slogchi "github.com/samber/slog-chi"

	"github.com/nofendian17/sbterm/apps/core/internal/delivery/http/admin"
	"github.com/nofendian17/sbterm/apps/core/internal/delivery/http/auth"
	"github.com/nofendian17/sbterm/apps/core/internal/delivery/http/health"
	appmw "github.com/nofendian17/sbterm/apps/core/internal/delivery/http/middleware"
	"github.com/nofendian17/sbterm/apps/core/internal/delivery/http/user"
	"github.com/nofendian17/sbterm/apps/core/internal/delivery/http/watchlist"
	"github.com/nofendian17/sbterm/libs/pkg/log"
)

type RouterOption func(*routerOptions)

type routerOptions struct {
	rateLimitRate  int
	rateLimitBurst int
}

func WithRateLimit(rate, burst int) RouterOption {
	return func(o *routerOptions) {
		o.rateLimitRate = rate
		o.rateLimitBurst = burst
	}
}

// Handlers groups every REST handler the router needs, one field per domain.
type Handlers struct {
	Health    *health.HealthHandler
	Auth      *auth.AuthHandler
	User      *user.UserHandler
	Watchlist *watchlist.WatchlistHandler
	Admin     *admin.AdminHandler
}

// AuthDeps are the dependencies the auth middleware needs.
type AuthDeps struct {
	Verifier appmw.TokenVerifier
	Loader   appmw.UserLoader
	Checker  appmw.PermissionChecker
}

func NewRouter(hs Handlers, authDeps AuthDeps, logger log.Logger, opts ...RouterOption) chi.Router {
	o := &routerOptions{}
	for _, opt := range opts {
		opt(o)
	}

	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(slogchi.NewWithConfig(logger.Slog(), slogchi.Config{
		DefaultLevel:     slog.LevelInfo,
		ClientErrorLevel: slog.LevelWarn,
		ServerErrorLevel: slog.LevelError,
		WithRequestID:    true,
		WithClientIP:     true,
	}))
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))
	if o.rateLimitRate > 0 {
		r.Use(appmw.NewRateLimit(
			appmw.WithRatePerSecond(o.rateLimitRate),
			appmw.WithBurst(o.rateLimitBurst),
			appmw.WithCleanupInterval(time.Minute),
		))
	}

	// Public routes
	r.Get("/healthz", hs.Health.Health)

	r.Route("/api", func(r chi.Router) {
		r.Route("/v1", func(r chi.Router) {
			// Auth (public)
			r.Post("/auth/register", hs.Auth.Register)
			r.Post("/auth/login", hs.Auth.Login)

			// Auth (authenticated)
			r.Group(func(r chi.Router) {
				r.Use(appmw.AuthMiddleware(authDeps.Verifier, authDeps.Loader, authDeps.Checker))
				r.Post("/auth/refresh", hs.Auth.Refresh)
				r.Post("/auth/logout", hs.Auth.Logout)

				// Users
				r.Get("/users/me", hs.User.GetMe)
				r.Put("/users/me", hs.User.UpdateMe)

				// Watchlists
				r.Get("/watchlists", hs.Watchlist.List)
				r.Post("/watchlists", hs.Watchlist.Add)
				r.Delete("/watchlists/{symbol}", hs.Watchlist.Remove)

				// Admin (requires specific permissions)
				r.Route("/admin", func(r chi.Router) {
					// Role management
					r.Group(func(r chi.Router) {
						r.Use(appmw.RequirePermission("admin:roles:read"))
						r.Get("/roles", hs.Admin.ListRoles)
						r.Get("/roles/{id}", hs.Admin.GetRole)
					})
					r.Group(func(r chi.Router) {
						r.Use(appmw.RequirePermission("admin:roles:write"))
						r.Post("/roles", hs.Admin.CreateRole)
						r.Delete("/roles/{id}", hs.Admin.DeleteRole)
					})
					r.Group(func(r chi.Router) {
						r.Use(appmw.RequirePermission("admin:rbac:assign"))
						r.Post("/roles/{id}/permissions", hs.Admin.AssignPermissionToRole)
						r.Delete("/roles/{id}/permissions/{permId}", hs.Admin.RevokePermissionFromRole)
						r.Post("/users/{id}/roles", hs.Admin.AssignRoleToUser)
						r.Delete("/users/{id}/roles/{roleId}", hs.Admin.RevokeRoleFromUser)
					})

					// User management
					r.Group(func(r chi.Router) {
						r.Use(appmw.RequirePermission("admin:users:read"))
						r.Get("/users", hs.Admin.ListUsers)
						r.Get("/users/{id}", hs.Admin.GetUser)
						r.Get("/users/{id}/watchlists", hs.Watchlist.ListByAdmin) // admin view of user watchlists
					})
					r.Group(func(r chi.Router) {
						r.Use(appmw.RequirePermission("admin:users:manage"))
						r.Post("/users/{id}/suspend", hs.Admin.SuspendUser)
						r.Patch("/users/{id}/expiry", hs.Admin.SetExpiry)
						r.Delete("/users/{id}", hs.Admin.DeleteUser)
					})
				})
			})
		})
	})

	return r
}
