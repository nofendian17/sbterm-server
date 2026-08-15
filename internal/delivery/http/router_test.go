package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm-server/internal/delivery/http/activity"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/brokertop"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/chart"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/companyprofile"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/corpaction"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/findata"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/foreigndomestic"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/fundachart"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/health"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/historicalsummary"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/index"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/indexsummary"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/keystats"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/majorholder"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/marketdetector"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/mover"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/network"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/orderbook"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/priceperformance"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/runningtrade"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/search"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/sectors"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/session"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/shareholding"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/stocks"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/stream"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/subsidiary"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/topstock"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/trending"
	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/mocks"
	"github.com/nofendian17/sbterm-server/pkg/log"
	"github.com/nofendian17/sbterm-server/pkg/validator"
)

func TestRouter(t *testing.T) {
	tests := []struct {
		name                   string
		method                 string
		path                   string
		setup                  func(uc *mocks.MockHealthUsecase)
		setupStocks            func(uc *mocks.MockStocksUsecase)
		setupCompanyProfile    func(uc *mocks.MockCompanyProfileUsecase)
		setupSubsidiary        func(uc *mocks.MockSubsidiaryUsecase)
		setupShareholding      func(uc *mocks.MockShareholdingCompositionUsecase)
		setupNetwork           func(uc *mocks.MockShareholdingNetworkUsecase)
		setupMajorHolder       func(uc *mocks.MockMajorHolderUsecase)
		setupMarketDetector    func(uc *mocks.MockMarketDetectorUsecase)
		setupTopStock          func(uc *mocks.MockTopStockUsecase)
		setupCorpAction        func(uc *mocks.MockCorpActionUsecase)
		setupKeystats          func(uc *mocks.MockKeystatsUsecase)
		setupPricePerformance  func(uc *mocks.MockPricePerformanceUsecase)
		setupChart             func(uc *mocks.MockChartbitUsecase)
		setupFundaChart        func(uc *mocks.MockFundaChartUsecase)
		setupFundaChartMetrics func(uc *mocks.MockFundaChartMetricsUsecase)
		setupFindataFinancial  func(uc *mocks.MockFindataFinancialUsecase)
		setupIndexSummary      func(uc *mocks.MockIndexSummaryUsecase)
		setupRunningTrade      func(uc *mocks.MockRunningTradeUsecase)
		setupOrderBook         func(uc *mocks.MockOrderBookUsecase)
		setupForeignDomestic   func(uc *mocks.MockForeignDomesticUsecase)
		setupHistoricalSummary func(uc *mocks.MockHistoricalSummaryUsecase)
		setupActivity          func(uc *mocks.MockActivityUsecase)
		setupBrokerTop         func(uc *mocks.MockBrokerTopUsecase)
		setupStream            func(uc *mocks.MockStreamUsecase)
		setupSearch            func(uc *mocks.MockSearchUsecase)
		wantStatus             int
	}{
		{
			name:   "get health returns 200",
			method: http.MethodGet,
			path:   "/health",
			setup: func(uc *mocks.MockHealthUsecase) {
				uc.EXPECT().GetHealth(gomock.Any()).Return(&domain.HealthStatus{Status: "ok", DBConnected: true, RedisConnected: true}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "unknown route returns 404",
			method:     http.MethodGet,
			path:       "/unknown",
			wantStatus: http.StatusNotFound,
		},
		{
			name:   "get stocks returns 200",
			method: http.MethodGet,
			path:   "/v1/stocks",
			setupStocks: func(uc *mocks.MockStocksUsecase) {
				uc.EXPECT().GetStocks(gomock.Any()).Return([]domain.Stock{{Symbol: "BBCA"}}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "get company profile returns 200",
			method: http.MethodGet,
			path:   "/v1/company/DSSA/profile",
			setupCompanyProfile: func(uc *mocks.MockCompanyProfileUsecase) {
				uc.EXPECT().GetProfile(gomock.Any(), "DSSA").Return(&domain.CompanyProfile{Background: "PT Dian Swastatika"}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "get company subsidiaries returns 200",
			method: http.MethodGet,
			path:   "/v1/company/DSSA/subsidiaries",
			setupSubsidiary: func(uc *mocks.MockSubsidiaryUsecase) {
				uc.EXPECT().GetSubsidiaries(gomock.Any(), "DSSA").Return(&domain.SubsidiaryData{Subsidiaries: []domain.Subsidiary{{CompanyName: "PT DSST Mas Gemilang"}}}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "get company shareholding composition returns 200",
			method: http.MethodGet,
			path:   "/v1/company/DSSA/shareholding-composition",
			setupShareholding: func(uc *mocks.MockShareholdingCompositionUsecase) {
				uc.EXPECT().GetShareholdingComposition(gomock.Any(), "DSSA", "", "").Return([]domain.ShareholdingCompositionPeriod{{ReportDate: "2026-07-31"}}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "get shareholding network returns 200",
			method: http.MethodGet,
			path:   "/v1/insider/shareholding-network?root_id=8824&root_type=SHAREHOLDING_NETWORK_NODE_TYPE_INVESTOR",
			setupNetwork: func(uc *mocks.MockShareholdingNetworkUsecase) {
				uc.EXPECT().GetShareholdingNetwork(gomock.Any(), "8824", "SHAREHOLDING_NETWORK_NODE_TYPE_INVESTOR", 0, 0).Return(&domain.ShareholdingNetwork{RootID: "investor:8824"}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "get major holder returns 200",
			method: http.MethodGet,
			path:   "/v1/insider/majorholder?symbols=DSSA",
			setupMajorHolder: func(uc *mocks.MockMajorHolderUsecase) {
				uc.EXPECT().GetMajorHolder(gomock.Any(), "DSSA", "", "", 0, 0).Return(&domain.MajorHolderData{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "get market detector returns 200",
			method: http.MethodGet,
			path:   "/v1/market-detector/BRPT?from=2026-08-03&to=2026-08-10",
			setupMarketDetector: func(uc *mocks.MockMarketDetectorUsecase) {
				uc.EXPECT().GetMarketDetector(gomock.Any(), "BRPT", "2026-08-03", "2026-08-10", "TRANSACTION_TYPE_NET", "MARKET_BOARD_REGULER", "INVESTOR_TYPE_ALL", 0).Return(&domain.MarketDetectorData{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "get top stock returns 200",
			method: http.MethodGet,
			path:   "/v1/top-stock?start=2026-08-09&end=2026-08-10",
			setupTopStock: func(uc *mocks.MockTopStockUsecase) {
				uc.EXPECT().GetTopStock(gomock.Any(), "2026-08-09", "2026-08-10", "INVESTOR_TYPE_ALL", "MARKET_TYPE_ALL", "VALUE_TYPE_NET", 0).Return(&domain.TopStockData{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "get corp actions returns 200",
			method: http.MethodGet,
			path:   "/v1/company/BUVA/corp-actions",
			setupCorpAction: func(uc *mocks.MockCorpActionUsecase) {
				uc.EXPECT().GetCorpActions(gomock.Any(), "BUVA", 0).Return([]domain.CompanyCorpAction{{ActionType: "rups"}}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "get keystats returns 200",
			method: http.MethodGet,
			path:   "/v1/company/BUVA/keystats",
			setupKeystats: func(uc *mocks.MockKeystatsUsecase) {
				uc.EXPECT().GetKeystats(gomock.Any(), "BUVA", 0).Return(&domain.Keystats{Stats: domain.KeystatsStats{MarketCap: "19,324 B"}}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "get price performance returns 200",
			method: http.MethodGet,
			path:   "/v1/company/BUVA/price-performance",
			setupPricePerformance: func(uc *mocks.MockPricePerformanceUsecase) {
				uc.EXPECT().GetPricePerformance(gomock.Any(), "BUVA").Return(&domain.PricePerformanceData{Prices: []domain.PricePerformance{{Timeframe: "1D"}}}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "get chart returns 200",
			method: http.MethodGet,
			path:   "/v1/company/BUVA/chart?timeframe=daily&from=2025-08-10&to=2026-08-10&limit=0",
			setupChart: func(uc *mocks.MockChartbitUsecase) {
				uc.EXPECT().GetChartPrice(gomock.Any(), "BUVA", "daily", "2025-08-10", "2026-08-10", 0).Return(&domain.ChartPriceData{Chartbit: []domain.ChartPrice{{Close: 985}}}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "get funda chart returns 200",
			method: http.MethodGet,
			path:   "/v1/company/BUVA/fundachart?item=12148",
			setupFundaChart: func(uc *mocks.MockFundaChartUsecase) {
				uc.EXPECT().GetFundaChart(gomock.Any(), "BUVA", "12148", "10y").Return([]domain.FundaChartCompany{{CompanyName: "BUVA"}}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "get funda chart metrics returns 200",
			method: http.MethodGet,
			path:   "/v1/fundachart/metrics?metric_name=fundachart",
			setupFundaChartMetrics: func(uc *mocks.MockFundaChartMetricsUsecase) {
				uc.EXPECT().GetFundaChartMetrics(gomock.Any(), "fundachart").Return([]domain.FundaChartMetric{{FitemID: 18, FitemName: "Size"}}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "get financial report returns 200",
			method: http.MethodGet,
			path:   "/v1/company/BUVA/financial?page=1&report_type=1&statement_type=1",
			setupFindataFinancial: func(uc *mocks.MockFindataFinancialUsecase) {
				uc.EXPECT().GetFindataFinancial(gomock.Any(), "BUVA", 0, 0, 1, 1, 1).Return(&domain.FindataFinancial{DefaultCurrency: "IDR"}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "get index summary returns 200",
			method: http.MethodGet,
			path:   "/v1/index/IHSG/summary?from=2026-08-10&to=2026-08-10&interval=INTERVAL_CHART_MINUTELY",
			setupIndexSummary: func(uc *mocks.MockIndexSummaryUsecase) {
				uc.EXPECT().GetIndexSummary(gomock.Any(), "IHSG", "2026-08-10", "2026-08-10", "INTERVAL_CHART_MINUTELY").Return(&domain.IndexSummaryData{Prices: []domain.IndexSummaryPrice{{Value: "6442.65"}}}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "get index chart returns 200",
			method: http.MethodGet,
			path:   "/v1/index/IHSG/chart?from=2026-08-10&to=2026-08-10&interval=INTERVAL_CHART_MINUTELY",
			setupIndexSummary: func(uc *mocks.MockIndexSummaryUsecase) {
				uc.EXPECT().GetIndexChart(gomock.Any(), "IHSG", "2026-08-10", "2026-08-10", "INTERVAL_CHART_MINUTELY").Return(&domain.IndexChartData{Chart: domain.ChartPriceData{Chartbit: []domain.ChartPrice{{Close: 6365.374}}}}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "get running trade returns 200",
			method: http.MethodGet,
			path:   "/v1/company/DSSA/running-trade-chart?broker_code=DR&from=2026-07-01&to=2026-08-10",
			setupRunningTrade: func(uc *mocks.MockRunningTradeUsecase) {
				uc.EXPECT().GetRunningTradeChart(gomock.Any(), "DSSA", []string{"DR"}, "2026-07-01", "2026-08-10", "INVESTOR_TYPE_ALL", "BOARD_TYPE_ALL", "").Return(&domain.RunningTradeData{From: "2026-07-01", DateSessionInfo: "10 Aug 2026"}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "get running trade feed returns 200",
			method: http.MethodGet,
			path:   "/v1/order-trade/running-trade?symbol=BBCA&sort=ASC&order_by=RUNNING_TRADE_ORDER_BY_TIME&date=2026-08-13&limit=80&trade_number=17796",
			setupRunningTrade: func(uc *mocks.MockRunningTradeUsecase) {
				uc.EXPECT().GetRunningTrade(gomock.Any(), "BBCA", "ASC", "RUNNING_TRADE_ORDER_BY_TIME", "2026-08-13", 80, int64(17796)).Return(&domain.RunningTradeFeed{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "get order book returns 200",
			method: http.MethodGet,
			path:   "/v1/company/VKTR/orderbook",
			setupOrderBook: func(uc *mocks.MockOrderBookUsecase) {
				uc.EXPECT().GetOrderBook(gomock.Any(), "VKTR").Return(&domain.OrderBookData{Symbol: "VKTR"}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "get foreign domestic historical returns 200",
			method: http.MethodGet,
			path:   "/v1/order-trade/foreign-domestic/historical?symbol=VKTR&market_type=MARKET_TYPE_ALL&period=TB_PERIOD_LAST_1_MONTH",
			setupForeignDomestic: func(uc *mocks.MockForeignDomesticUsecase) {
				uc.EXPECT().GetForeignDomesticHistorical(gomock.Any(), "VKTR", "MARKET_TYPE_ALL", "TB_PERIOD_LAST_1_MONTH", "", "").Return(&domain.ForeignDomesticData{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "get historical summary returns 200",
			method: http.MethodGet,
			path:   "/v1/company/DSSA/historical-summary?period=HS_PERIOD_WEEKLY&start_date=2025-08-11&end_date=2026-08-11",
			setupHistoricalSummary: func(uc *mocks.MockHistoricalSummaryUsecase) {
				uc.EXPECT().GetHistoricalSummary(gomock.Any(), "DSSA", "HS_PERIOD_WEEKLY", "2025-08-11", "2026-08-11", 50, 1).Return(&domain.HistoricalSummaryData{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "get activity chart returns 200",
			method: http.MethodGet,
			path:   "/v1/order-trade/broker/activity-chart?symbols=BBRI&brokers_code=DR&from=2026-07-01&to=2026-08-10",
			setupActivity: func(uc *mocks.MockActivityUsecase) {
				uc.EXPECT().GetActivityChart(gomock.Any(), []string{"BBRI"}, []string{"DR"}, "2026-07-01", "2026-08-10", "", "INVESTOR_TYPE_ALL", "BOARD_TYPE_ALL").Return(&domain.ActivityChartData{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "get activity returns 200",
			method: http.MethodGet,
			path:   "/v1/order-trade/broker/activity?broker_code=DR&transaction_type=TRANSACTION_TYPE_GROSS&from=2026-07-14&to=2026-07-31&limit=20",
			setupActivity: func(uc *mocks.MockActivityUsecase) {
				uc.EXPECT().GetActivity(gomock.Any(), []string{"DR"}, "TRANSACTION_TYPE_GROSS", "INVESTOR_TYPE_ALL", "MARKET_TYPE_REGULER", 20, 1, "2026-07-14", "2026-07-31", "NET_VAL_PERIOD_7D").Return(&domain.ActivityData{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "get activity historical returns 200",
			method: http.MethodGet,
			path:   "/v1/order-trade/broker/activity/historical?interval=INTERVAL_DAILY&date_from=2026-07-01&date_to=2026-08-31&broker_codes=ZP&broker_codes=BK&symbols=CUAN&market_board=BOARD_TYPE_REGULAR&investor_type=INVESTOR_TYPE_ALL&net_interval=INTERVAL_MONTHLY",
			setupActivity: func(uc *mocks.MockActivityUsecase) {
				uc.EXPECT().GetActivityHistorical(gomock.Any(), "INTERVAL_DAILY", "2026-07-01", "2026-08-31", []string{"ZP", "BK"}, []string{"CUAN"}, "BOARD_TYPE_REGULAR", "INVESTOR_TYPE_ALL", "INTERVAL_MONTHLY").Return(&domain.ActivityHistoricalData{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "get broker top returns 200",
			method: http.MethodGet,
			path:   "/v1/order-trade/broker/top?period=TB_PERIOD_LAST_1_DAY",
			setupBrokerTop: func(uc *mocks.MockBrokerTopUsecase) {
				uc.EXPECT().GetBrokerTop(gomock.Any(), "TB_SORT_BY_TOTAL_VALUE", "ORDER_BY_DESC", "TB_PERIOD_LAST_1_DAY", "MARKET_TYPE_ALL", true).Return(&domain.BrokerTopData{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "get user stream returns 200",
			method: http.MethodGet,
			path:   "/v1/user/StockbitReports/stream?category=STREAM_CATEGORY_MAIN_IDEAS&last_stream_id=34884782&limit=20",
			setupStream: func(uc *mocks.MockStreamUsecase) {
				uc.EXPECT().GetUserStream(gomock.Any(), "StockbitReports", "STREAM_CATEGORY_MAIN_IDEAS", int64(34884782), 20).Return(&domain.UserStreamData{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "get stream announcement returns 200",
			method: http.MethodGet,
			path:   "/v1/stream/announcement/f3e83a0aeb3c9c48800b7f3beafc8aba",
			setupStream: func(uc *mocks.MockStreamUsecase) {
				uc.EXPECT().GetStreamAnnouncement(gomock.Any(), "f3e83a0aeb3c9c48800b7f3beafc8aba").Return([]domain.StreamAnnouncement{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "get search returns 200",
			method: http.MethodGet,
			path:   "/v1/search?keyword=BBRI",
			setupSearch: func(uc *mocks.MockSearchUsecase) {
				uc.EXPECT().GetSearch(gomock.Any(), "BBRI", 1, "company").Return(&domain.SearchResult{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "wrong method returns 405",
			method:     http.MethodPost,
			path:       "/health",
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockHealthUsecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			logger := log.New(log.WithWriter(io.Discard))
			handler := health.NewHealthHandler(uc)
			trendingHandler := trending.NewTrendingHandler(mocks.NewMockTrendingUsecase(ctrl))
			moverHandler := mover.NewMarketMoverHandler(mocks.NewMockMarketMoverUsecase(ctrl), validator.New())
			sessionHandler := session.NewMarketSessionHandler(mocks.NewMockMarketSessionUsecase(ctrl))
			indexHandler := index.NewIndexHandler(mocks.NewMockIndexUsecase(ctrl))
			sectorsHandler := sectors.NewSectorsHandler(mocks.NewMockSectorsUsecase(ctrl))
			ucStocks := mocks.NewMockStocksUsecase(ctrl)
			if tt.setupStocks != nil {
				tt.setupStocks(ucStocks)
			}
			stocksHandler := stocks.NewStocksHandler(ucStocks)
			ucCompanyProfile := mocks.NewMockCompanyProfileUsecase(ctrl)
			if tt.setupCompanyProfile != nil {
				tt.setupCompanyProfile(ucCompanyProfile)
			}
			companyProfileHandler := companyprofile.NewCompanyProfileHandler(ucCompanyProfile, validator.New())
			ucSubsidiary := mocks.NewMockSubsidiaryUsecase(ctrl)
			if tt.setupSubsidiary != nil {
				tt.setupSubsidiary(ucSubsidiary)
			}
			subsidiaryHandler := subsidiary.NewSubsidiaryHandler(ucSubsidiary, validator.New())
			ucShareholding := mocks.NewMockShareholdingCompositionUsecase(ctrl)
			if tt.setupShareholding != nil {
				tt.setupShareholding(ucShareholding)
			}
			shareholdingHandler := shareholding.NewShareholdingHandler(ucShareholding, validator.New())
			ucNetwork := mocks.NewMockShareholdingNetworkUsecase(ctrl)
			if tt.setupNetwork != nil {
				tt.setupNetwork(ucNetwork)
			}
			networkHandler := network.NewShareholdingNetworkHandler(ucNetwork, validator.New())
			ucMajorHolder := mocks.NewMockMajorHolderUsecase(ctrl)
			if tt.setupMajorHolder != nil {
				tt.setupMajorHolder(ucMajorHolder)
			}
			majorHolderHandler := majorholder.NewMajorHolderHandler(ucMajorHolder, validator.New())
			ucMarketDetector := mocks.NewMockMarketDetectorUsecase(ctrl)
			if tt.setupMarketDetector != nil {
				tt.setupMarketDetector(ucMarketDetector)
			}
			marketDetectorHandler := marketdetector.NewMarketDetectorHandler(ucMarketDetector, validator.New())
			ucTopStock := mocks.NewMockTopStockUsecase(ctrl)
			if tt.setupTopStock != nil {
				tt.setupTopStock(ucTopStock)
			}
			topStockHandler := topstock.NewTopStockHandler(ucTopStock, validator.New())
			ucCorpAction := mocks.NewMockCorpActionUsecase(ctrl)
			if tt.setupCorpAction != nil {
				tt.setupCorpAction(ucCorpAction)
			}
			corpActionHandler := corpaction.NewCorpActionHandler(ucCorpAction, validator.New())
			ucKeystats := mocks.NewMockKeystatsUsecase(ctrl)
			if tt.setupKeystats != nil {
				tt.setupKeystats(ucKeystats)
			}
			keystatsHandler := keystats.NewKeystatsHandler(ucKeystats, validator.New())
			ucPricePerformance := mocks.NewMockPricePerformanceUsecase(ctrl)
			if tt.setupPricePerformance != nil {
				tt.setupPricePerformance(ucPricePerformance)
			}
			pricePerformanceHandler := priceperformance.NewPricePerformanceHandler(ucPricePerformance, validator.New())
			ucChart := mocks.NewMockChartbitUsecase(ctrl)
			if tt.setupChart != nil {
				tt.setupChart(ucChart)
			}
			chartHandler := chart.NewChartbitHandler(ucChart, validator.New())
			ucFundaChart := mocks.NewMockFundaChartUsecase(ctrl)
			if tt.setupFundaChart != nil {
				tt.setupFundaChart(ucFundaChart)
			}
			fundaChartHandler := fundachart.NewFundaChartHandler(ucFundaChart, validator.New())
			ucFundaChartMetrics := mocks.NewMockFundaChartMetricsUsecase(ctrl)
			if tt.setupFundaChartMetrics != nil {
				tt.setupFundaChartMetrics(ucFundaChartMetrics)
			}
			fundaChartMetricsHandler := fundachart.NewFundaChartMetricsHandler(ucFundaChartMetrics, validator.New())
			ucFindataFinancial := mocks.NewMockFindataFinancialUsecase(ctrl)
			if tt.setupFindataFinancial != nil {
				tt.setupFindataFinancial(ucFindataFinancial)
			}
			findataFinancialHandler := findata.NewFindataFinancialHandler(ucFindataFinancial, validator.New())
			ucIndexSummary := mocks.NewMockIndexSummaryUsecase(ctrl)
			if tt.setupIndexSummary != nil {
				tt.setupIndexSummary(ucIndexSummary)
			}
			indexSummaryHandler := indexsummary.NewIndexSummaryHandler(ucIndexSummary, validator.New())
			ucRunningTrade := mocks.NewMockRunningTradeUsecase(ctrl)
			if tt.setupRunningTrade != nil {
				tt.setupRunningTrade(ucRunningTrade)
			}
			runningTradeHandler := runningtrade.NewRunningTradeHandler(ucRunningTrade, validator.New())
			ucOrderBook := mocks.NewMockOrderBookUsecase(ctrl)
			if tt.setupOrderBook != nil {
				tt.setupOrderBook(ucOrderBook)
			}
			orderBookHandler := orderbook.NewOrderBookHandler(ucOrderBook, validator.New())
			ucForeignDomestic := mocks.NewMockForeignDomesticUsecase(ctrl)
			if tt.setupForeignDomestic != nil {
				tt.setupForeignDomestic(ucForeignDomestic)
			}
			foreignDomesticHandler := foreigndomestic.NewForeignDomesticHandler(ucForeignDomestic, validator.New())
			ucHistoricalSummary := mocks.NewMockHistoricalSummaryUsecase(ctrl)
			if tt.setupHistoricalSummary != nil {
				tt.setupHistoricalSummary(ucHistoricalSummary)
			}
			historicalSummaryHandler := historicalsummary.NewHistoricalSummaryHandler(ucHistoricalSummary, validator.New())
			ucActivity := mocks.NewMockActivityUsecase(ctrl)
			if tt.setupActivity != nil {
				tt.setupActivity(ucActivity)
			}
			activityHandler := activity.NewActivityHandler(ucActivity, validator.New())
			ucBrokerTop := mocks.NewMockBrokerTopUsecase(ctrl)
			if tt.setupBrokerTop != nil {
				tt.setupBrokerTop(ucBrokerTop)
			}
			brokerTopHandler := brokertop.NewBrokerTopHandler(ucBrokerTop, validator.New())
			ucStream := mocks.NewMockStreamUsecase(ctrl)
			if tt.setupStream != nil {
				tt.setupStream(ucStream)
			}
			streamHandler := stream.NewStreamHandler(ucStream, validator.New())
			ucSearch := mocks.NewMockSearchUsecase(ctrl)
			if tt.setupSearch != nil {
				tt.setupSearch(ucSearch)
			}
			searchHandler := search.NewSearchHandler(ucSearch, validator.New())
			router := NewRouter(handler, trendingHandler, moverHandler, sessionHandler, indexHandler, sectorsHandler, stocksHandler, companyProfileHandler, subsidiaryHandler, shareholdingHandler, networkHandler, majorHolderHandler, marketDetectorHandler, topStockHandler, corpActionHandler, keystatsHandler, pricePerformanceHandler, chartHandler, fundaChartHandler, fundaChartMetricsHandler, findataFinancialHandler, indexSummaryHandler, runningTradeHandler, orderBookHandler, foreignDomesticHandler, historicalSummaryHandler, activityHandler, brokerTopHandler, streamHandler, searchHandler, logger)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, nil)
			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

func TestRouterRateLimit(t *testing.T) {
	tests := []struct {
		name      string
		wantCodes []int
	}{
		{
			name:      "requests beyond burst are rejected",
			wantCodes: []int{http.StatusOK, http.StatusTooManyRequests},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockHealthUsecase(ctrl)
			uc.EXPECT().GetHealth(gomock.Any()).Return(&domain.HealthStatus{Status: "ok", DBConnected: true, RedisConnected: true}, nil).AnyTimes()

			logger := log.New(log.WithWriter(io.Discard))
			router := NewRouter(health.NewHealthHandler(uc), trending.NewTrendingHandler(mocks.NewMockTrendingUsecase(ctrl)), mover.NewMarketMoverHandler(mocks.NewMockMarketMoverUsecase(ctrl), validator.New()), session.NewMarketSessionHandler(mocks.NewMockMarketSessionUsecase(ctrl)), index.NewIndexHandler(mocks.NewMockIndexUsecase(ctrl)), sectors.NewSectorsHandler(mocks.NewMockSectorsUsecase(ctrl)), stocks.NewStocksHandler(mocks.NewMockStocksUsecase(ctrl)), companyprofile.NewCompanyProfileHandler(mocks.NewMockCompanyProfileUsecase(ctrl), validator.New()), subsidiary.NewSubsidiaryHandler(mocks.NewMockSubsidiaryUsecase(ctrl), validator.New()), shareholding.NewShareholdingHandler(mocks.NewMockShareholdingCompositionUsecase(ctrl), validator.New()), network.NewShareholdingNetworkHandler(mocks.NewMockShareholdingNetworkUsecase(ctrl), validator.New()), majorholder.NewMajorHolderHandler(mocks.NewMockMajorHolderUsecase(ctrl), validator.New()), marketdetector.NewMarketDetectorHandler(mocks.NewMockMarketDetectorUsecase(ctrl), validator.New()), topstock.NewTopStockHandler(mocks.NewMockTopStockUsecase(ctrl), validator.New()), corpaction.NewCorpActionHandler(mocks.NewMockCorpActionUsecase(ctrl), validator.New()), keystats.NewKeystatsHandler(mocks.NewMockKeystatsUsecase(ctrl), validator.New()), priceperformance.NewPricePerformanceHandler(mocks.NewMockPricePerformanceUsecase(ctrl), validator.New()), chart.NewChartbitHandler(mocks.NewMockChartbitUsecase(ctrl), validator.New()), fundachart.NewFundaChartHandler(mocks.NewMockFundaChartUsecase(ctrl), validator.New()), fundachart.NewFundaChartMetricsHandler(mocks.NewMockFundaChartMetricsUsecase(ctrl), validator.New()), findata.NewFindataFinancialHandler(mocks.NewMockFindataFinancialUsecase(ctrl), validator.New()), indexsummary.NewIndexSummaryHandler(mocks.NewMockIndexSummaryUsecase(ctrl), validator.New()), runningtrade.NewRunningTradeHandler(mocks.NewMockRunningTradeUsecase(ctrl), validator.New()), orderbook.NewOrderBookHandler(mocks.NewMockOrderBookUsecase(ctrl), validator.New()), foreigndomestic.NewForeignDomesticHandler(mocks.NewMockForeignDomesticUsecase(ctrl), validator.New()), historicalsummary.NewHistoricalSummaryHandler(mocks.NewMockHistoricalSummaryUsecase(ctrl), validator.New()), activity.NewActivityHandler(nil, validator.New()), brokertop.NewBrokerTopHandler(mocks.NewMockBrokerTopUsecase(ctrl), validator.New()), stream.NewStreamHandler(mocks.NewMockStreamUsecase(ctrl), validator.New()), search.NewSearchHandler(mocks.NewMockSearchUsecase(ctrl), validator.New()), logger, WithRateLimit(1, 1))

			for _, want := range tt.wantCodes {
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
				assert.Equal(t, want, rec.Code)
				if want == http.StatusTooManyRequests {
					assert.Equal(t, "1", rec.Header().Get("Retry-After"))
				}
			}
		})
	}
}
