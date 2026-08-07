package http

import (
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	slogchi "github.com/samber/slog-chi"

	mw "github.com/nofendian17/sbterm-server/internal/delivery/http/middleware"
	"github.com/nofendian17/sbterm-server/pkg/log"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/health"
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

func NewRouter(handler *health.HealthHandler, logger log.Logger, opts ...RouterOption) chi.Router {
	o := &routerOptions{}
	for _, opt := range opts {
		opt(o)
	}

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(slogchi.NewWithConfig(logger.Slog(), slogchi.Config{
		DefaultLevel:     slog.LevelInfo,
		ClientErrorLevel: slog.LevelWarn,
		ServerErrorLevel: slog.LevelError,
		WithRequestID:    true,
		WithClientIP:     true,
	}))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	if o.rateLimitRate > 0 {
		r.Use(mw.NewRateLimit(
			mw.WithRatePerSecond(o.rateLimitRate),
			mw.WithBurst(o.rateLimitBurst),
			mw.WithCleanupInterval(time.Minute),
		))
	}

	r.Get("/health", handler.Health)

	return r
}
