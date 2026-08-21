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
	"github.com/nofendian17/sbterm/apps/api/internal/delivery/http/notification"
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

// Handlers groups every REST handler the router needs, one field per domain.
type Handlers struct {
	Health            *health.HealthHandler
	Trending          *trending.TrendingHandler
	Notification      *notification.NotificationHandler
	MarketMover       *mover.MarketMoverHandler
	MarketSession     *session.MarketSessionHandler
	Index             *index.IndexHandler
	Sectors           *sectors.SectorsHandler
	Stocks            *stocks.StocksHandler
	CompanyProfile    *companyprofile.CompanyProfileHandler
	Subsidiary        *subsidiary.SubsidiaryHandler
	Shareholding      *shareholding.ShareholdingHandler
	Network           *network.ShareholdingNetworkHandler
	MajorHolder       *majorholder.MajorHolderHandler
	MarketDetector    *marketdetector.MarketDetectorHandler
	TopStock          *topstock.TopStockHandler
	CorpAction        *corpaction.CorpActionHandler
	Keystats          *keystats.KeystatsHandler
	PricePerformance  *priceperformance.PricePerformanceHandler
	Chart             *chart.ChartbitHandler
	FundaChart        *fundachart.FundaChartHandler
	FundaChartMetrics *fundachart.FundaChartMetricsHandler
	Financial         *findata.FindataFinancialHandler
	IndexSummary      *indexsummary.IndexSummaryHandler
	RunningTrade      *runningtrade.RunningTradeHandler
	OrderBook         *orderbook.OrderBookHandler
	ForeignDomestic   *foreigndomestic.ForeignDomesticHandler
	HistoricalSummary *historicalsummary.HistoricalSummaryHandler
	Activity          *activity.ActivityHandler
	BrokerTop         *brokertop.BrokerTopHandler
	Stream            *stream.StreamHandler
	Search            *search.SearchHandler
	OrderQueue        *orderqueue.OrderQueueHandler
}

func NewRouter(hs Handlers, logger log.Logger, opts ...RouterOption) chi.Router {
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

	r.Get("/health", hs.Health.Health)

	r.Route("/api", func(r chi.Router) {
		r.Route("/v1", func(r chi.Router) {
			r.Get("/trending", hs.Trending.Trending)
			r.Get("/notifications", hs.Notification.GetNotifications)
			r.Get("/market-mover", hs.MarketMover.MarketMover)
			r.Get("/market-session", hs.MarketSession.MarketSession)
			r.Get("/indexes", hs.Index.Index)
			r.Get("/sectors", hs.Sectors.Sectors)
			r.Get("/stocks", hs.Stocks.Stocks)
			r.Get("/company/{symbol}/profile", hs.CompanyProfile.CompanyProfile)
			r.Get("/company/{symbol}/subsidiaries", hs.Subsidiary.Subsidiaries)
			r.Get("/company/{symbol}/shareholding-composition", hs.Shareholding.ShareholdingComposition)
			r.Get("/insider/shareholding-network", hs.Network.ShareholdingNetwork)
			r.Get("/insider/majorholder", hs.MajorHolder.MajorHolder)
			r.Get("/market-detector/{symbol}", hs.MarketDetector.MarketDetector)
			r.Get("/top-stock", hs.TopStock.TopStock)
			r.Get("/company/{symbol}/corp-actions", hs.CorpAction.CorpActions)
			r.Get("/company/{symbol}/keystats", hs.Keystats.Keystats)
			r.Get("/company/{symbol}/price-performance", hs.PricePerformance.PricePerformance)
			r.Get("/company/{symbol}/chart", hs.Chart.ChartPrice)
			r.Get("/company/{symbol}/fundachart", hs.FundaChart.FundaChart)
			r.Get("/fundachart/metrics", hs.FundaChartMetrics.Metrics)
			r.Get("/company/{symbol}/financial", hs.Financial.Financial)
			r.Get("/index/{symbol}/summary", hs.IndexSummary.IndexSummary)
			r.Get("/index/{symbol}/chart", hs.IndexSummary.IndexChart)
			r.Get("/company/{symbol}/running-trade-chart", hs.RunningTrade.RunningTradeChart)
			r.Get("/order-trade/running-trade", hs.RunningTrade.RunningTrade)
			r.Get("/company/{symbol}/orderbook", hs.OrderBook.OrderBook)
			r.Get("/order-trade/foreign-domestic/historical", hs.ForeignDomestic.ForeignDomesticHistorical)
			r.Get("/company/{symbol}/historical-summary", hs.HistoricalSummary.HistoricalSummary)
			r.Get("/order-trade/broker/activity-chart", hs.Activity.ActivityChart)
			r.Get("/order-trade/broker/activity", hs.Activity.Activity)
			r.Get("/order-trade/broker/activity/historical", hs.Activity.ActivityHistorical)
			r.Get("/order-trade/broker/top", hs.BrokerTop.BrokerTop)
			r.Get("/order-trade/order-queue", hs.OrderQueue.OrderQueue)
			r.Get("/user/{username}/stream", hs.Stream.UserStream)
			r.Get("/search", hs.Search.Search)
			r.Get("/stream/announcement/{stream_id}", hs.Stream.StreamAnnouncement)
			r.Get("/stream/conversation/{stream_id}", hs.Stream.StreamConversation)
		})
	})

	return r
}
