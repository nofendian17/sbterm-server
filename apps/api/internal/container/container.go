package container

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/samber/do/v2"
	"golang.org/x/sync/errgroup"

	deliveryhttp "github.com/nofendian17/sbterm/apps/api/internal/delivery/http"
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
	"github.com/nofendian17/sbterm/apps/api/internal/infrastructure/cache"
	"github.com/nofendian17/sbterm/apps/api/internal/infrastructure/config"
	"github.com/nofendian17/sbterm/apps/api/internal/infrastructure/database"
	infraRepo "github.com/nofendian17/sbterm/apps/api/internal/infrastructure/repository"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
	"github.com/nofendian17/sbterm/apps/api/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/log"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
	"github.com/nofendian17/sbterm/libs/stockbit"
)

func New(cfg *config.Config, logger log.Logger) *do.RootScope {
	injector := do.New()

	do.ProvideValue(injector, cfg)
	do.ProvideValue(injector, logger)

	provideCommon(injector)
	provideInfrastructure(injector)
	provideStockbit(injector)
	provideRepositories(injector)
	provideUsecases(injector)
	provideHandlers(injector)

	return injector
}

// provideCommon registers shared cross-cutting providers such as the request
// validator used by HTTP handlers.
func provideCommon(injector *do.RootScope) {
	do.Provide(injector, func(i do.Injector) (validator.Validator, error) {
		return validator.New(), nil
	})
}

func provideInfrastructure(injector *do.RootScope) {
	do.Provide(injector, func(i do.Injector) (*database.Postgres, error) {
		cfg := do.MustInvoke[*config.Config](i)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return database.New(ctx, cfg.Database.URL,
			database.WithMaxConns(cfg.Database.MaxConns),
			database.WithMinConns(cfg.Database.MinConns),
			database.WithMaxConnLifetime(cfg.Database.MaxConnLifetime),
			database.WithMaxConnIdleTime(cfg.Database.MaxConnIdleTime),
		)
	})

	do.Provide(injector, func(i do.Injector) (*cache.Redis, error) {
		cfg := do.MustInvoke[*config.Config](i)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return cache.New(ctx, cfg.Redis.URL,
			cache.WithMaxRetries(cfg.Redis.MaxRetries),
			cache.WithPoolSize(cfg.Redis.PoolSize),
			cache.WithMinIdleConns(cfg.Redis.MinIdleConns),
			cache.WithDialTimeout(cfg.Redis.DialTimeout),
			cache.WithReadTimeout(cfg.Redis.ReadTimeout),
			cache.WithWriteTimeout(cfg.Redis.WriteTimeout),
		)
	})
}

func provideRepositories(injector *do.RootScope) {
	do.Provide(injector, func(i do.Injector) (*infraRepo.TxManagerImpl, error) {
		return infraRepo.NewTxManager(do.MustInvoke[*database.Postgres](i)), nil
	})
	do.MustAs[*infraRepo.TxManagerImpl, repository.TxManager](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.HealthRepository, error) {
		return infraRepo.NewHealthRepository(
			do.MustInvoke[*database.Postgres](i),
			do.MustInvoke[*cache.Redis](i),
		), nil
	})
	do.MustAs[*infraRepo.HealthRepository, repository.HealthRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.TrendingRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewTrendingRepository(client), nil
	})
	do.MustAs[*infraRepo.TrendingRepository, repository.TrendingRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.NotificationRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewNotificationRepository(client), nil
	})
	do.MustAs[*infraRepo.NotificationRepository, repository.NotificationRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.MarketMoverRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewMarketMoverRepository(client), nil
	})
	do.MustAs[*infraRepo.MarketMoverRepository, repository.MarketMoverRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.MarketSessionRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewMarketSessionRepository(client), nil
	})
	do.MustAs[*infraRepo.MarketSessionRepository, repository.MarketSessionRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.IndexRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewIndexRepository(client), nil
	})
	do.MustAs[*infraRepo.IndexRepository, repository.IndexRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.SectorsRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewSectorsRepository(client), nil
	})
	do.MustAs[*infraRepo.SectorsRepository, repository.SectorsRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.SubsectorRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewSubsectorRepository(client), nil
	})
	do.MustAs[*infraRepo.SubsectorRepository, repository.SubsectorRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.StocksRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewStocksRepository(client), nil
	})
	do.MustAs[*infraRepo.StocksRepository, repository.StocksRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.CompanyProfileRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewCompanyProfileRepository(client), nil
	})
	do.MustAs[*infraRepo.CompanyProfileRepository, repository.CompanyProfileRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.SubsidiaryRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewSubsidiaryRepository(client), nil
	})
	do.MustAs[*infraRepo.SubsidiaryRepository, repository.SubsidiaryRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.ShareholdingCompositionRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewShareholdingCompositionRepository(client), nil
	})
	do.MustAs[*infraRepo.ShareholdingCompositionRepository, repository.ShareholdingCompositionRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.ShareholdingNetworkRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewShareholdingNetworkRepository(client), nil
	})
	do.MustAs[*infraRepo.ShareholdingNetworkRepository, repository.ShareholdingNetworkRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.MarketDetectorRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewMarketDetectorRepository(client), nil
	})
	do.MustAs[*infraRepo.MarketDetectorRepository, repository.MarketDetectorRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.TopStockRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewTopStockRepository(client), nil
	})
	do.MustAs[*infraRepo.TopStockRepository, repository.TopStockRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.MajorHolderRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewMajorHolderRepository(client), nil
	})
	do.MustAs[*infraRepo.MajorHolderRepository, repository.MajorHolderRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.CorpActionRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewCorpActionRepository(client), nil
	})
	do.MustAs[*infraRepo.CorpActionRepository, repository.CorpActionRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.KeystatsRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewKeystatsRepository(client), nil
	})
	do.MustAs[*infraRepo.KeystatsRepository, repository.KeystatsRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.PricePerformanceRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewPricePerformanceRepository(client), nil
	})
	do.MustAs[*infraRepo.PricePerformanceRepository, repository.PricePerformanceRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.ChartbitRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewChartbitRepository(client), nil
	})
	do.MustAs[*infraRepo.ChartbitRepository, repository.ChartbitRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.FundaChartRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewFundaChartRepository(client), nil
	})
	do.MustAs[*infraRepo.FundaChartRepository, repository.FundaChartRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.FundaChartMetricsRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewFundaChartMetricsRepository(client), nil
	})
	do.MustAs[*infraRepo.FundaChartMetricsRepository, repository.FundaChartMetricsRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.FindataFinancialRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewFindataFinancialRepository(client), nil
	})
	do.MustAs[*infraRepo.FindataFinancialRepository, repository.FindataFinancialRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.IndexSummaryRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewIndexSummaryRepository(client), nil
	})
	do.MustAs[*infraRepo.IndexSummaryRepository, repository.IndexSummaryRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.RunningTradeRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewRunningTradeRepository(client), nil
	})
	do.MustAs[*infraRepo.RunningTradeRepository, repository.RunningTradeRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.OrderBookRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewOrderBookRepository(client), nil
	})
	do.MustAs[*infraRepo.OrderBookRepository, repository.OrderBookRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.ForeignDomesticRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewForeignDomesticRepository(client), nil
	})
	do.MustAs[*infraRepo.ForeignDomesticRepository, repository.ForeignDomesticRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.HistoricalSummaryRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewHistoricalSummaryRepository(client), nil
	})
	do.MustAs[*infraRepo.HistoricalSummaryRepository, repository.HistoricalSummaryRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.ActivityRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewActivityRepository(client), nil
	})
	do.MustAs[*infraRepo.ActivityRepository, repository.ActivityRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.BrokerTopRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewBrokerTopRepository(client), nil
	})
	do.MustAs[*infraRepo.BrokerTopRepository, repository.BrokerTopRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.OrderQueueRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewOrderQueueRepository(client), nil
	})
	do.MustAs[*infraRepo.OrderQueueRepository, repository.OrderQueueRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.StreamRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewStreamRepository(client), nil
	})
	do.MustAs[*infraRepo.StreamRepository, repository.StreamRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.SearchRepository, error) {
		client, err := do.Invoke[*stockbit.Client](i)
		if err != nil {
			return nil, err
		}
		return infraRepo.NewSearchRepository(client), nil
	})
	do.MustAs[*infraRepo.SearchRepository, repository.SearchRepository](injector)
}

func provideStockbit(injector *do.RootScope) {
	do.Provide(injector, func(i do.Injector) (*stockbit.Refresher, error) {
		cfg := do.MustInvoke[*config.Config](i)
		logger := do.MustInvoke[log.Logger](i)

		opts := []stockbit.Option{
			stockbit.WithTimeout(cfg.Stockbit.Timeout),
			stockbit.WithRetryCount(cfg.Stockbit.RetryCount),
			stockbit.WithLogger(logger),
		}
		if cfg.Stockbit.BaseURL != "" {
			opts = append(opts, stockbit.WithBaseURL(cfg.Stockbit.BaseURL))
		}
		client := stockbit.New(opts...)

		cmd := do.MustInvoke[*cache.Redis](i).Cmdable()
		if cmd == nil {
			return nil, fmt.Errorf("container: redis client unavailable for token store")
		}
		store := stockbit.NewRedisTokenStore(cmd)
		refresher := stockbit.NewRefresher(client, store, stockbit.Credentials{
			PlayerID: cfg.Stockbit.PlayerID,
			Username: cfg.Stockbit.Username,
			Password: cfg.Stockbit.Password,
		}, logger)
		client.SetAuthenticator(refresher)
		return refresher, nil
	})

	do.Provide(injector, func(i do.Injector) (*stockbit.Client, error) {
		refresher, err := do.Invoke[*stockbit.Refresher](i)
		if err != nil {
			return nil, err
		}
		return refresher.Client(), nil
	})
}

func provideUsecases(injector *do.RootScope) {
	do.Provide(injector, func(i do.Injector) (usecase.HealthUsecase, error) {
		return usecase.NewHealthUsecase(do.MustInvoke[repository.HealthRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.TrendingUsecase, error) {
		return usecase.NewTrendingUsecase(do.MustInvoke[repository.TrendingRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.NotificationUsecase, error) {
		return usecase.NewNotificationUsecase(do.MustInvoke[repository.NotificationRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.MarketMoverUsecase, error) {
		return usecase.NewMarketMoverUsecase(do.MustInvoke[repository.MarketMoverRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.MarketSessionUsecase, error) {
		return usecase.NewMarketSessionUsecase(do.MustInvoke[repository.MarketSessionRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.IndexUsecase, error) {
		return usecase.NewIndexUsecase(do.MustInvoke[repository.IndexRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.SectorsUsecase, error) {
		return usecase.NewSectorsUsecase(do.MustInvoke[repository.SectorsRepository](i), do.MustInvoke[repository.SubsectorRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.StocksUsecase, error) {
		return usecase.NewStocksUsecase(do.MustInvoke[repository.StocksRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.CompanyProfileUsecase, error) {
		return usecase.NewCompanyProfileUsecase(do.MustInvoke[repository.CompanyProfileRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.SubsidiaryUsecase, error) {
		return usecase.NewSubsidiaryUsecase(do.MustInvoke[repository.SubsidiaryRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.ShareholdingCompositionUsecase, error) {
		return usecase.NewShareholdingCompositionUsecase(do.MustInvoke[repository.ShareholdingCompositionRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.ShareholdingNetworkUsecase, error) {
		return usecase.NewShareholdingNetworkUsecase(do.MustInvoke[repository.ShareholdingNetworkRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.MajorHolderUsecase, error) {
		return usecase.NewMajorHolderUsecase(do.MustInvoke[repository.MajorHolderRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.MarketDetectorUsecase, error) {
		return usecase.NewMarketDetectorUsecase(do.MustInvoke[repository.MarketDetectorRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.TopStockUsecase, error) {
		return usecase.NewTopStockUsecase(do.MustInvoke[repository.TopStockRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.CorpActionUsecase, error) {
		return usecase.NewCorpActionUsecase(do.MustInvoke[repository.CorpActionRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.KeystatsUsecase, error) {
		return usecase.NewKeystatsUsecase(do.MustInvoke[repository.KeystatsRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.PricePerformanceUsecase, error) {
		return usecase.NewPricePerformanceUsecase(do.MustInvoke[repository.PricePerformanceRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.ChartbitUsecase, error) {
		return usecase.NewChartbitUsecase(do.MustInvoke[repository.ChartbitRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.FundaChartUsecase, error) {
		return usecase.NewFundaChartUsecase(do.MustInvoke[repository.FundaChartRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.FundaChartMetricsUsecase, error) {
		return usecase.NewFundaChartMetricsUsecase(do.MustInvoke[repository.FundaChartMetricsRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.FindataFinancialUsecase, error) {
		return usecase.NewFindataFinancialUsecase(do.MustInvoke[repository.FindataFinancialRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.IndexSummaryUsecase, error) {
		return usecase.NewIndexSummaryUsecase(do.MustInvoke[repository.IndexSummaryRepository](i), do.MustInvoke[repository.ChartbitRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.RunningTradeUsecase, error) {
		return usecase.NewRunningTradeUsecase(do.MustInvoke[repository.RunningTradeRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.OrderBookUsecase, error) {
		return usecase.NewOrderBookUsecase(do.MustInvoke[repository.OrderBookRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.ForeignDomesticUsecase, error) {
		return usecase.NewForeignDomesticUsecase(do.MustInvoke[repository.ForeignDomesticRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.HistoricalSummaryUsecase, error) {
		return usecase.NewHistoricalSummaryUsecase(do.MustInvoke[repository.HistoricalSummaryRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.ActivityUsecase, error) {
		return usecase.NewActivityUsecase(do.MustInvoke[repository.ActivityRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.BrokerTopUsecase, error) {
		return usecase.NewBrokerTopUsecase(do.MustInvoke[repository.BrokerTopRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.OrderQueueUsecase, error) {
		return usecase.NewOrderQueueUsecase(do.MustInvoke[repository.OrderQueueRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.StreamUsecase, error) {
		return usecase.NewStreamUsecase(do.MustInvoke[repository.StreamRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.SearchUsecase, error) {
		return usecase.NewSearchUsecase(do.MustInvoke[repository.SearchRepository](i)), nil
	})
}

func provideHandlers(injector *do.RootScope) {
	do.Provide(injector, func(i do.Injector) (*health.HealthHandler, error) {
		return health.NewHealthHandler(do.MustInvoke[usecase.HealthUsecase](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*trending.TrendingHandler, error) {
		return trending.NewTrendingHandler(do.MustInvoke[usecase.TrendingUsecase](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*notification.NotificationHandler, error) {
		return notification.NewNotificationHandler(do.MustInvoke[usecase.NotificationUsecase](i), do.MustInvoke[validator.Validator](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*mover.MarketMoverHandler, error) {
		return mover.NewMarketMoverHandler(do.MustInvoke[usecase.MarketMoverUsecase](i), do.MustInvoke[validator.Validator](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*session.MarketSessionHandler, error) {
		return session.NewMarketSessionHandler(do.MustInvoke[usecase.MarketSessionUsecase](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*index.IndexHandler, error) {
		return index.NewIndexHandler(do.MustInvoke[usecase.IndexUsecase](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*sectors.SectorsHandler, error) {
		return sectors.NewSectorsHandler(do.MustInvoke[usecase.SectorsUsecase](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*stocks.StocksHandler, error) {
		return stocks.NewStocksHandler(do.MustInvoke[usecase.StocksUsecase](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*companyprofile.CompanyProfileHandler, error) {
		return companyprofile.NewCompanyProfileHandler(do.MustInvoke[usecase.CompanyProfileUsecase](i), do.MustInvoke[validator.Validator](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*subsidiary.SubsidiaryHandler, error) {
		return subsidiary.NewSubsidiaryHandler(do.MustInvoke[usecase.SubsidiaryUsecase](i), do.MustInvoke[validator.Validator](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*shareholding.ShareholdingHandler, error) {
		return shareholding.NewShareholdingHandler(do.MustInvoke[usecase.ShareholdingCompositionUsecase](i), do.MustInvoke[validator.Validator](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*network.ShareholdingNetworkHandler, error) {
		return network.NewShareholdingNetworkHandler(do.MustInvoke[usecase.ShareholdingNetworkUsecase](i), do.MustInvoke[validator.Validator](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*majorholder.MajorHolderHandler, error) {
		return majorholder.NewMajorHolderHandler(do.MustInvoke[usecase.MajorHolderUsecase](i), do.MustInvoke[validator.Validator](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*marketdetector.MarketDetectorHandler, error) {
		return marketdetector.NewMarketDetectorHandler(do.MustInvoke[usecase.MarketDetectorUsecase](i), do.MustInvoke[validator.Validator](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*topstock.TopStockHandler, error) {
		return topstock.NewTopStockHandler(do.MustInvoke[usecase.TopStockUsecase](i), do.MustInvoke[validator.Validator](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*corpaction.CorpActionHandler, error) {
		return corpaction.NewCorpActionHandler(do.MustInvoke[usecase.CorpActionUsecase](i), do.MustInvoke[validator.Validator](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*keystats.KeystatsHandler, error) {
		return keystats.NewKeystatsHandler(do.MustInvoke[usecase.KeystatsUsecase](i), do.MustInvoke[validator.Validator](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*priceperformance.PricePerformanceHandler, error) {
		return priceperformance.NewPricePerformanceHandler(do.MustInvoke[usecase.PricePerformanceUsecase](i), do.MustInvoke[validator.Validator](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*chart.ChartbitHandler, error) {
		return chart.NewChartbitHandler(do.MustInvoke[usecase.ChartbitUsecase](i), do.MustInvoke[validator.Validator](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*fundachart.FundaChartHandler, error) {
		return fundachart.NewFundaChartHandler(do.MustInvoke[usecase.FundaChartUsecase](i), do.MustInvoke[validator.Validator](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*fundachart.FundaChartMetricsHandler, error) {
		return fundachart.NewFundaChartMetricsHandler(do.MustInvoke[usecase.FundaChartMetricsUsecase](i), do.MustInvoke[validator.Validator](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*findata.FindataFinancialHandler, error) {
		return findata.NewFindataFinancialHandler(do.MustInvoke[usecase.FindataFinancialUsecase](i), do.MustInvoke[validator.Validator](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*indexsummary.IndexSummaryHandler, error) {
		return indexsummary.NewIndexSummaryHandler(do.MustInvoke[usecase.IndexSummaryUsecase](i), do.MustInvoke[validator.Validator](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*runningtrade.RunningTradeHandler, error) {
		return runningtrade.NewRunningTradeHandler(do.MustInvoke[usecase.RunningTradeUsecase](i), do.MustInvoke[validator.Validator](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*orderbook.OrderBookHandler, error) {
		return orderbook.NewOrderBookHandler(do.MustInvoke[usecase.OrderBookUsecase](i), do.MustInvoke[validator.Validator](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*foreigndomestic.ForeignDomesticHandler, error) {
		return foreigndomestic.NewForeignDomesticHandler(do.MustInvoke[usecase.ForeignDomesticUsecase](i), do.MustInvoke[validator.Validator](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*historicalsummary.HistoricalSummaryHandler, error) {
		return historicalsummary.NewHistoricalSummaryHandler(do.MustInvoke[usecase.HistoricalSummaryUsecase](i), do.MustInvoke[validator.Validator](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*activity.ActivityHandler, error) {
		return activity.NewActivityHandler(do.MustInvoke[usecase.ActivityUsecase](i), do.MustInvoke[validator.Validator](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*brokertop.BrokerTopHandler, error) {
		return brokertop.NewBrokerTopHandler(do.MustInvoke[usecase.BrokerTopUsecase](i), do.MustInvoke[validator.Validator](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*orderqueue.OrderQueueHandler, error) {
		return orderqueue.NewOrderQueueHandler(do.MustInvoke[usecase.OrderQueueUsecase](i), do.MustInvoke[validator.Validator](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*stream.StreamHandler, error) {
		return stream.NewStreamHandler(do.MustInvoke[usecase.StreamUsecase](i), do.MustInvoke[validator.Validator](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*search.SearchHandler, error) {
		return search.NewSearchHandler(do.MustInvoke[usecase.SearchUsecase](i), do.MustInvoke[validator.Validator](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*deliveryhttp.Server, error) {
		cfg := do.MustInvoke[*config.Config](i)
		logger := do.MustInvoke[log.Logger](i)

		router := deliveryhttp.NewRouter(deliveryhttp.Handlers{
			Health:            do.MustInvoke[*health.HealthHandler](i),
			Trending:          do.MustInvoke[*trending.TrendingHandler](i),
			Notification:      do.MustInvoke[*notification.NotificationHandler](i),
			MarketMover:       do.MustInvoke[*mover.MarketMoverHandler](i),
			MarketSession:     do.MustInvoke[*session.MarketSessionHandler](i),
			Index:             do.MustInvoke[*index.IndexHandler](i),
			Sectors:           do.MustInvoke[*sectors.SectorsHandler](i),
			Stocks:            do.MustInvoke[*stocks.StocksHandler](i),
			CompanyProfile:    do.MustInvoke[*companyprofile.CompanyProfileHandler](i),
			Subsidiary:        do.MustInvoke[*subsidiary.SubsidiaryHandler](i),
			Shareholding:      do.MustInvoke[*shareholding.ShareholdingHandler](i),
			Network:           do.MustInvoke[*network.ShareholdingNetworkHandler](i),
			MajorHolder:       do.MustInvoke[*majorholder.MajorHolderHandler](i),
			MarketDetector:    do.MustInvoke[*marketdetector.MarketDetectorHandler](i),
			TopStock:          do.MustInvoke[*topstock.TopStockHandler](i),
			CorpAction:        do.MustInvoke[*corpaction.CorpActionHandler](i),
			Keystats:          do.MustInvoke[*keystats.KeystatsHandler](i),
			PricePerformance:  do.MustInvoke[*priceperformance.PricePerformanceHandler](i),
			Chart:             do.MustInvoke[*chart.ChartbitHandler](i),
			FundaChart:        do.MustInvoke[*fundachart.FundaChartHandler](i),
			FundaChartMetrics: do.MustInvoke[*fundachart.FundaChartMetricsHandler](i),
			Financial:         do.MustInvoke[*findata.FindataFinancialHandler](i),
			IndexSummary:      do.MustInvoke[*indexsummary.IndexSummaryHandler](i),
			RunningTrade:      do.MustInvoke[*runningtrade.RunningTradeHandler](i),
			OrderBook:         do.MustInvoke[*orderbook.OrderBookHandler](i),
			ForeignDomestic:   do.MustInvoke[*foreigndomestic.ForeignDomesticHandler](i),
			HistoricalSummary: do.MustInvoke[*historicalsummary.HistoricalSummaryHandler](i),
			Activity:          do.MustInvoke[*activity.ActivityHandler](i),
			BrokerTop:         do.MustInvoke[*brokertop.BrokerTopHandler](i),
			Stream:            do.MustInvoke[*stream.StreamHandler](i),
			Search:            do.MustInvoke[*search.SearchHandler](i),
			OrderQueue:        do.MustInvoke[*orderqueue.OrderQueueHandler](i),
		}, logger,
			deliveryhttp.WithRateLimit(cfg.RateLimit.Rate, cfg.RateLimit.Burst),
		)
		return deliveryhttp.NewServer(router,
			deliveryhttp.WithAddr(cfg.Port),
			deliveryhttp.WithReadTimeout(cfg.HTTP.ReadTimeout),
			deliveryhttp.WithWriteTimeout(cfg.HTTP.WriteTimeout),
			deliveryhttp.WithIdleTimeout(cfg.HTTP.IdleTimeout),
		), nil
	})
}

type pinger interface {
	Ping(context.Context) error
}

func pingService[T pinger](ctx context.Context, injector *do.RootScope, name string) error {
	svc, err := do.Invoke[T](injector)
	if err != nil {
		return fmt.Errorf("container: resolve %s: %w", name, err)
	}
	if err := svc.Ping(ctx); err != nil {
		return fmt.Errorf("container: %s unreachable: %w", name, err)
	}
	return nil
}

// pingInfra verifies database and Redis connectivity before the server starts.
func pingInfra(injector *do.RootScope) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return pingService[*database.Postgres](ctx, injector, "postgres")
	})
	g.Go(func() error {
		return pingService[*cache.Redis](ctx, injector, "redis")
	})

	return g.Wait()
}

func newLogger(cfg *config.Config) (log.Logger, error) {
	level, err := log.ParseLevel(cfg.Log.Level)
	if err != nil {
		return nil, err
	}
	format, err := log.ParseFormat(cfg.Log.Format)
	if err != nil {
		return nil, err
	}

	logger := log.New(
		log.WithLevel(level),
		log.WithFormat(format),
		log.WithAddSource(cfg.Log.AddSource),
	)
	log.SetDefault(logger)

	return logger, nil
}

func awaitShutdown(server *deliveryhttp.Server, injector *do.RootScope, logger log.Logger) error {
	errChan := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- fmt.Errorf("http server failed: %w", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, os.Interrupt)
	defer signal.Stop(sigChan)

	select {
	case err := <-errChan:
		logger.Error("server startup failed", "error", err)
		if report := injector.Shutdown(); !report.Succeed {
			logger.Error("container shutdown failed", "error", report)
		}
		return err
	case sig := <-sigChan:
		logger.Info("received shutdown signal", "signal", sig.String())
		if report := injector.Shutdown(); !report.Succeed {
			logger.Error("container shutdown failed", "error", report)
			return report
		}
	}

	logger.Info("server stopped")
	return nil
}

func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger, err := newLogger(cfg)
	if err != nil {
		return err
	}

	injector := New(cfg, logger)

	if err := pingInfra(injector); err != nil {
		return err
	}

	refresher, err := do.Invoke[*stockbit.Refresher](injector)
	if err != nil {
		return fmt.Errorf("container: construct stockbit refresher: %w", err)
	}
	refresher.Start()

	server, err := do.Invoke[*deliveryhttp.Server](injector)
	if err != nil {
		return fmt.Errorf("container: construct server: %w", err)
	}

	logger.Info("server started",
		"app", cfg.App.Name,
		"version", cfg.App.Version,
		"addr", cfg.Port,
	)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		statuses := injector.HealthCheckWithContext(ctx)
		failed := 0
		for name, err := range statuses {
			if err != nil {
				failed++
				logger.Warn("service health check failed", "service", name, "error", err)
			}
		}
		logger.Info("health check complete", "services", len(statuses), "failed", failed)
	}()

	return awaitShutdown(server, injector, logger)
}
