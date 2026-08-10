package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm-server/internal/delivery/http/chart"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/companyprofile"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/corpaction"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/findata"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/fundachart"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/health"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/index"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/indexsummary"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/keystats"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/majorholder"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/marketdetector"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/mover"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/network"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/priceperformance"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/sectors"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/session"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/shareholding"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/stocks"
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
			router := NewRouter(handler, trendingHandler, moverHandler, sessionHandler, indexHandler, sectorsHandler, stocksHandler, companyProfileHandler, subsidiaryHandler, shareholdingHandler, networkHandler, majorHolderHandler, marketDetectorHandler, topStockHandler, corpActionHandler, keystatsHandler, pricePerformanceHandler, chartHandler, fundaChartHandler, fundaChartMetricsHandler, findataFinancialHandler, indexSummaryHandler, logger)

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
			router := NewRouter(health.NewHealthHandler(uc), trending.NewTrendingHandler(mocks.NewMockTrendingUsecase(ctrl)), mover.NewMarketMoverHandler(mocks.NewMockMarketMoverUsecase(ctrl), validator.New()), session.NewMarketSessionHandler(mocks.NewMockMarketSessionUsecase(ctrl)), index.NewIndexHandler(mocks.NewMockIndexUsecase(ctrl)), sectors.NewSectorsHandler(mocks.NewMockSectorsUsecase(ctrl)), stocks.NewStocksHandler(mocks.NewMockStocksUsecase(ctrl)), companyprofile.NewCompanyProfileHandler(mocks.NewMockCompanyProfileUsecase(ctrl), validator.New()), subsidiary.NewSubsidiaryHandler(mocks.NewMockSubsidiaryUsecase(ctrl), validator.New()), shareholding.NewShareholdingHandler(mocks.NewMockShareholdingCompositionUsecase(ctrl), validator.New()), network.NewShareholdingNetworkHandler(mocks.NewMockShareholdingNetworkUsecase(ctrl), validator.New()), majorholder.NewMajorHolderHandler(mocks.NewMockMajorHolderUsecase(ctrl), validator.New()), marketdetector.NewMarketDetectorHandler(mocks.NewMockMarketDetectorUsecase(ctrl), validator.New()), topstock.NewTopStockHandler(mocks.NewMockTopStockUsecase(ctrl), validator.New()), corpaction.NewCorpActionHandler(mocks.NewMockCorpActionUsecase(ctrl), validator.New()), keystats.NewKeystatsHandler(mocks.NewMockKeystatsUsecase(ctrl), validator.New()), priceperformance.NewPricePerformanceHandler(mocks.NewMockPricePerformanceUsecase(ctrl), validator.New()), chart.NewChartbitHandler(mocks.NewMockChartbitUsecase(ctrl), validator.New()), fundachart.NewFundaChartHandler(mocks.NewMockFundaChartUsecase(ctrl), validator.New()), fundachart.NewFundaChartMetricsHandler(mocks.NewMockFundaChartMetricsUsecase(ctrl), validator.New()), findata.NewFindataFinancialHandler(mocks.NewMockFindataFinancialUsecase(ctrl), validator.New()), indexsummary.NewIndexSummaryHandler(mocks.NewMockIndexSummaryUsecase(ctrl), validator.New()), logger, WithRateLimit(1, 1))

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
