package http

import (
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	slogchi "github.com/samber/slog-chi"

	"github.com/nofendian17/sbterm-server/internal/delivery/http/companyprofile"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/health"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/index"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/majorholder"
	mw "github.com/nofendian17/sbterm-server/internal/delivery/http/middleware"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/mover"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/network"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/sectors"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/session"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/shareholding"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/stocks"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/subsidiary"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/trending"
	"github.com/nofendian17/sbterm-server/pkg/log"
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

func NewRouter(handler *health.HealthHandler, trendingHandler *trending.TrendingHandler, moverHandler *mover.MarketMoverHandler, sessionHandler *session.MarketSessionHandler, indexHandler *index.IndexHandler, sectorsHandler *sectors.SectorsHandler, stocksHandler *stocks.StocksHandler, companyProfileHandler *companyprofile.CompanyProfileHandler, subsidiaryHandler *subsidiary.SubsidiaryHandler, shareholdingHandler *shareholding.ShareholdingHandler, networkHandler *network.ShareholdingNetworkHandler, majorHolderHandler *majorholder.MajorHolderHandler, logger log.Logger, opts ...RouterOption) chi.Router {
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
	r.Get("/v1/trending", trendingHandler.Trending)
	r.Get("/v1/market-mover", moverHandler.MarketMover)
	r.Get("/v1/market-session", sessionHandler.MarketSession)
	r.Get("/v1/indexes", indexHandler.Index)
	r.Get("/v1/sectors", sectorsHandler.Sectors)
	r.Get("/v1/stocks", stocksHandler.Stocks)
	r.Get("/v1/company/{symbol}/profile", companyProfileHandler.CompanyProfile)
	r.Get("/v1/company/{symbol}/subsidiaries", subsidiaryHandler.Subsidiaries)
	r.Get("/v1/company/{symbol}/shareholding-composition", shareholdingHandler.ShareholdingComposition)
	r.Get("/v1/insider/shareholding-network", networkHandler.ShareholdingNetwork)
	r.Get("/v1/insider/majorholder", majorHolderHandler.MajorHolder)

	return r
}
