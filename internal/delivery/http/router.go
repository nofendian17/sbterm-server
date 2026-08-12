package http

import (
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	slogchi "github.com/samber/slog-chi"

	"github.com/nofendian17/sbterm-server/internal/delivery/http/activity"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/brokertop"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/chart"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/companyprofile"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/corpaction"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/findata"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/fundachart"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/health"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/historicalsummary"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/index"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/indexsummary"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/keystats"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/majorholder"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/marketdetector"
	mw "github.com/nofendian17/sbterm-server/internal/delivery/http/middleware"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/mover"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/network"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/priceperformance"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/runningtrade"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/sectors"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/session"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/shareholding"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/stocks"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/subsidiary"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/topstock"
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

func NewRouter(handler *health.HealthHandler, trendingHandler *trending.TrendingHandler, moverHandler *mover.MarketMoverHandler, sessionHandler *session.MarketSessionHandler, indexHandler *index.IndexHandler, sectorsHandler *sectors.SectorsHandler, stocksHandler *stocks.StocksHandler, companyProfileHandler *companyprofile.CompanyProfileHandler, subsidiaryHandler *subsidiary.SubsidiaryHandler, shareholdingHandler *shareholding.ShareholdingHandler, networkHandler *network.ShareholdingNetworkHandler, majorHolderHandler *majorholder.MajorHolderHandler, marketDetectorHandler *marketdetector.MarketDetectorHandler, topStockHandler *topstock.TopStockHandler, corpActionHandler *corpaction.CorpActionHandler, keystatsHandler *keystats.KeystatsHandler, pricePerformanceHandler *priceperformance.PricePerformanceHandler, chartHandler *chart.ChartbitHandler, fundaChartHandler *fundachart.FundaChartHandler, fundaChartMetricsHandler *fundachart.FundaChartMetricsHandler, findataFinancialHandler *findata.FindataFinancialHandler, indexSummaryHandler *indexsummary.IndexSummaryHandler, runningTradeHandler *runningtrade.RunningTradeHandler, historicalSummaryHandler *historicalsummary.HistoricalSummaryHandler, activityHandler *activity.ActivityHandler, brokerTopHandler *brokertop.BrokerTopHandler, logger log.Logger, opts ...RouterOption) chi.Router {
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
	r.Get("/v1/market-detector/{symbol}", marketDetectorHandler.MarketDetector)
	r.Get("/v1/top-stock", topStockHandler.TopStock)
	r.Get("/v1/company/{symbol}/corp-actions", corpActionHandler.CorpActions)
	r.Get("/v1/company/{symbol}/keystats", keystatsHandler.Keystats)
	r.Get("/v1/company/{symbol}/price-performance", pricePerformanceHandler.PricePerformance)
	r.Get("/v1/company/{symbol}/chart", chartHandler.ChartPrice)
	r.Get("/v1/company/{symbol}/fundachart", fundaChartHandler.FundaChart)
	r.Get("/v1/fundachart/metrics", fundaChartMetricsHandler.Metrics)
	r.Get("/v1/company/{symbol}/financial", findataFinancialHandler.Financial)
	r.Get("/v1/index/{symbol}/summary", indexSummaryHandler.IndexSummary)
	r.Get("/v1/index/{symbol}/chart", indexSummaryHandler.IndexChart)
	r.Get("/v1/company/{symbol}/running-trade-chart", runningTradeHandler.RunningTradeChart)
	r.Get("/v1/company/{symbol}/historical-summary", historicalSummaryHandler.HistoricalSummary)
	r.Get("/v1/order-trade/broker/activity-chart", activityHandler.ActivityChart)
	r.Get("/v1/order-trade/broker/activity", activityHandler.Activity)
	r.Get("/v1/order-trade/broker/activity/historical", activityHandler.ActivityHistorical)
	r.Get("/v1/order-trade/broker/top", brokerTopHandler.BrokerTop)

	return r
}
