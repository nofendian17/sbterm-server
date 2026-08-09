package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm-server/internal/delivery/http/companyprofile"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/health"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/index"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/majorholder"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/mover"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/network"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/sectors"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/session"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/shareholding"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/stocks"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/subsidiary"
	"github.com/nofendian17/sbterm-server/internal/delivery/http/trending"
	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/mocks"
	"github.com/nofendian17/sbterm-server/pkg/log"
	"github.com/nofendian17/sbterm-server/pkg/validator"
)

func TestRouter(t *testing.T) {
	tests := []struct {
		name                string
		method              string
		path                string
		setup               func(uc *mocks.MockHealthUsecase)
		setupStocks         func(uc *mocks.MockStocksUsecase)
		setupCompanyProfile func(uc *mocks.MockCompanyProfileUsecase)
		setupSubsidiary     func(uc *mocks.MockSubsidiaryUsecase)
		setupShareholding   func(uc *mocks.MockShareholdingCompositionUsecase)
		setupNetwork        func(uc *mocks.MockShareholdingNetworkUsecase)
		setupMajorHolder    func(uc *mocks.MockMajorHolderUsecase)
		wantStatus          int
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
			router := NewRouter(handler, trendingHandler, moverHandler, sessionHandler, indexHandler, sectorsHandler, stocksHandler, companyProfileHandler, subsidiaryHandler, shareholdingHandler, networkHandler, majorHolderHandler, logger)

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
			router := NewRouter(health.NewHealthHandler(uc), trending.NewTrendingHandler(mocks.NewMockTrendingUsecase(ctrl)), mover.NewMarketMoverHandler(mocks.NewMockMarketMoverUsecase(ctrl), validator.New()), session.NewMarketSessionHandler(mocks.NewMockMarketSessionUsecase(ctrl)), index.NewIndexHandler(mocks.NewMockIndexUsecase(ctrl)), sectors.NewSectorsHandler(mocks.NewMockSectorsUsecase(ctrl)), stocks.NewStocksHandler(mocks.NewMockStocksUsecase(ctrl)), companyprofile.NewCompanyProfileHandler(mocks.NewMockCompanyProfileUsecase(ctrl), validator.New()), subsidiary.NewSubsidiaryHandler(mocks.NewMockSubsidiaryUsecase(ctrl), validator.New()), shareholding.NewShareholdingHandler(mocks.NewMockShareholdingCompositionUsecase(ctrl), validator.New()), network.NewShareholdingNetworkHandler(mocks.NewMockShareholdingNetworkUsecase(ctrl), validator.New()), majorholder.NewMajorHolderHandler(mocks.NewMockMajorHolderUsecase(ctrl), validator.New()), logger, WithRateLimit(1, 1))

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
