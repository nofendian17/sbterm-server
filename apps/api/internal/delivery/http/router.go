package http

import (
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	slogchi "github.com/samber/slog-chi"

	"github.com/nofendian17/sbterm/apps/api/internal/delivery/http/activity"
	"github.com/nofendian17/sbterm/apps/api/internal/delivery/http/brokertop"
	"github.com/nofendian17/sbterm/apps/api/internal/delivery/http/chart"
	"github.com/nofendian17/sbterm/apps/api/internal/delivery/http/companyprofile"
	"github.com/nofendian17/sbterm/apps/api/internal/delivery/http/corpaction"
	"github.com/nofendian17/sbterm/apps/api/internal/delivery/http/findata"
	"github.com/nofendian17/sbterm/apps/api/internal/delivery/http/foreigndomestic"
	"github.com/nofendian17/sbterm/apps/api/internal/delivery/http/fundachart"
	"github.com/nofendian17/sbterm/apps/api/internal/delivery/http/health"
	"github.com/nofendian17/sbterm/apps/api/internal/delivery/http/historicalsummary"
	"github.com/nofendian17/sbterm/apps/api/internal/delivery/http/index"
	"github.com/nofendian17/sbterm/apps/api/internal/delivery/http/indexsummary"
	"github.com/nofendian17/sbterm/apps/api/internal/delivery/http/keystats"
	"github.com/nofendian17/sbterm/apps/api/internal/delivery/http/majorholder"
	"github.com/nofendian17/sbterm/apps/api/internal/delivery/http/marketdetector"
	mw "github.com/nofendian17/sbterm/apps/api/internal/delivery/http/middleware"
	"github.com/nofendian17/sbterm/apps/api/internal/delivery/http/mover"
	"github.com/nofendian17/sbterm/apps/api/internal/delivery/http/network"
	"github.com/nofendian17/sbterm/apps/api/internal/delivery/http/orderbook"
	"github.com/nofendian17/sbterm/apps/api/internal/delivery/http/orderqueue"
	"github.com/nofendian17/sbterm/apps/api/internal/delivery/http/priceperformance"
	"github.com/nofendian17/sbterm/apps/api/internal/delivery/http/runningtrade"
	"github.com/nofendian17/sbterm/apps/api/internal/delivery/http/search"
	"github.com/nofendian17/sbterm/apps/api/internal/delivery/http/sectors"
	"github.com/nofendian17/sbterm/apps/api/internal/delivery/http/session"
	"github.com/nofendian17/sbterm/apps/api/internal/delivery/http/shareholding"
	"github.com/nofendian17/sbterm/apps/api/internal/delivery/http/stocks"
	"github.com/nofendian17/sbterm/apps/api/internal/delivery/http/stream"
	"github.com/nofendian17/sbterm/apps/api/internal/delivery/http/subsidiary"
	"github.com/nofendian17/sbterm/apps/api/internal/delivery/http/topstock"
	"github.com/nofendian17/sbterm/apps/api/internal/delivery/http/trending"
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

func NewRouter(handler *health.HealthHandler, trendingHandler *trending.TrendingHandler, moverHandler *mover.MarketMoverHandler, sessionHandler *session.MarketSessionHandler, indexHandler *index.IndexHandler, sectorsHandler *sectors.SectorsHandler, stocksHandler *stocks.StocksHandler, companyProfileHandler *companyprofile.CompanyProfileHandler, subsidiaryHandler *subsidiary.SubsidiaryHandler, shareholdingHandler *shareholding.ShareholdingHandler, networkHandler *network.ShareholdingNetworkHandler, majorHolderHandler *majorholder.MajorHolderHandler, marketDetectorHandler *marketdetector.MarketDetectorHandler, topStockHandler *topstock.TopStockHandler, corpActionHandler *corpaction.CorpActionHandler, keystatsHandler *keystats.KeystatsHandler, pricePerformanceHandler *priceperformance.PricePerformanceHandler, chartHandler *chart.ChartbitHandler, fundaChartHandler *fundachart.FundaChartHandler, fundaChartMetricsHandler *fundachart.FundaChartMetricsHandler, findataFinancialHandler *findata.FindataFinancialHandler, indexSummaryHandler *indexsummary.IndexSummaryHandler, runningTradeHandler *runningtrade.RunningTradeHandler, orderBookHandler *orderbook.OrderBookHandler, foreignDomesticHandler *foreigndomestic.ForeignDomesticHandler, historicalSummaryHandler *historicalsummary.HistoricalSummaryHandler, activityHandler *activity.ActivityHandler, brokerTopHandler *brokertop.BrokerTopHandler, streamHandler *stream.StreamHandler, searchHandler *search.SearchHandler, orderQueueHandler *orderqueue.OrderQueueHandler, logger log.Logger, opts ...RouterOption) chi.Router {
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

	r.Route("/api", func(r chi.Router) {
		r.Route("/v1", func(r chi.Router) {
			r.Get("/trending", trendingHandler.Trending)
			r.Get("/market-mover", moverHandler.MarketMover)
			r.Get("/market-session", sessionHandler.MarketSession)
			r.Get("/indexes", indexHandler.Index)
			r.Get("/sectors", sectorsHandler.Sectors)
			r.Get("/stocks", stocksHandler.Stocks)
			r.Get("/company/{symbol}/profile", companyProfileHandler.CompanyProfile)
			r.Get("/company/{symbol}/subsidiaries", subsidiaryHandler.Subsidiaries)
			r.Get("/company/{symbol}/shareholding-composition", shareholdingHandler.ShareholdingComposition)
			r.Get("/insider/shareholding-network", networkHandler.ShareholdingNetwork)
			r.Get("/insider/majorholder", majorHolderHandler.MajorHolder)
			r.Get("/market-detector/{symbol}", marketDetectorHandler.MarketDetector)
			r.Get("/top-stock", topStockHandler.TopStock)
			r.Get("/company/{symbol}/corp-actions", corpActionHandler.CorpActions)
			r.Get("/company/{symbol}/keystats", keystatsHandler.Keystats)
			r.Get("/company/{symbol}/price-performance", pricePerformanceHandler.PricePerformance)
			r.Get("/company/{symbol}/chart", chartHandler.ChartPrice)
			r.Get("/company/{symbol}/fundachart", fundaChartHandler.FundaChart)
			r.Get("/fundachart/metrics", fundaChartMetricsHandler.Metrics)
			r.Get("/company/{symbol}/financial", findataFinancialHandler.Financial)
			r.Get("/index/{symbol}/summary", indexSummaryHandler.IndexSummary)
			r.Get("/index/{symbol}/chart", indexSummaryHandler.IndexChart)
			r.Get("/company/{symbol}/running-trade-chart", runningTradeHandler.RunningTradeChart)
			r.Get("/order-trade/running-trade", runningTradeHandler.RunningTrade)
			r.Get("/company/{symbol}/orderbook", orderBookHandler.OrderBook)
			r.Get("/order-trade/foreign-domestic/historical", foreignDomesticHandler.ForeignDomesticHistorical)
			r.Get("/company/{symbol}/historical-summary", historicalSummaryHandler.HistoricalSummary)
			r.Get("/order-trade/broker/activity-chart", activityHandler.ActivityChart)
			r.Get("/order-trade/broker/activity", activityHandler.Activity)
			r.Get("/order-trade/broker/activity/historical", activityHandler.ActivityHistorical)
			r.Get("/order-trade/broker/top", brokerTopHandler.BrokerTop)
			r.Get("/order-trade/order-queue", orderQueueHandler.OrderQueue)
			r.Get("/user/{username}/stream", streamHandler.UserStream)
			r.Get("/search", searchHandler.Search)
			r.Get("/stream/announcement/{stream_id}", streamHandler.StreamAnnouncement)
		})
	})

	return r
}
