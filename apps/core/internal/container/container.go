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

	deliveryhttp "github.com/nofendian17/sbterm/apps/core/internal/delivery/http"
	"github.com/nofendian17/sbterm/apps/core/internal/delivery/http/admin"
	authhandler "github.com/nofendian17/sbterm/apps/core/internal/delivery/http/auth"
	"github.com/nofendian17/sbterm/apps/core/internal/delivery/http/companyprofile"
	"github.com/nofendian17/sbterm/apps/core/internal/delivery/http/health"
	appmw "github.com/nofendian17/sbterm/apps/core/internal/delivery/http/middleware"
	"github.com/nofendian17/sbterm/apps/core/internal/delivery/http/sector"
	"github.com/nofendian17/sbterm/apps/core/internal/delivery/http/stock"
	"github.com/nofendian17/sbterm/apps/core/internal/delivery/http/user"
	"github.com/nofendian17/sbterm/apps/core/internal/delivery/http/watchlist"
	"github.com/nofendian17/sbterm/apps/core/internal/infrastructure/cache"
	"github.com/nofendian17/sbterm/apps/core/internal/infrastructure/config"
	"github.com/nofendian17/sbterm/apps/core/internal/infrastructure/database"
	infraRepo "github.com/nofendian17/sbterm/apps/core/internal/infrastructure/repository"
	"github.com/nofendian17/sbterm/apps/core/internal/infrastructure/stockapi"
	"github.com/nofendian17/sbterm/apps/core/internal/infrastructure/token"
	"github.com/nofendian17/sbterm/apps/core/internal/repository"
	"github.com/nofendian17/sbterm/apps/core/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/log"
	appvalidator "github.com/nofendian17/sbterm/libs/pkg/validator"
)

func New(cfg *config.Config, logger log.Logger) *do.RootScope {
	injector := do.New()

	do.ProvideValue(injector, cfg)
	do.ProvideValue(injector, logger)

	provideCommon(injector)
	provideInfrastructure(injector)
	provideRepositories(injector)
	provideUsecases(injector)
	provideHandlers(injector)

	return injector
}

func provideCommon(injector *do.RootScope) {
	do.Provide(injector, func(i do.Injector) (appvalidator.Validator, error) {
		return appvalidator.New(), nil
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

	// StockSyncClient: HTTP adapter that calls apps/api (docs/api.md). The
	// same stockapi.Client also implements CompanyProfileSyncClient, so it
	// is provided once and aliased to both ports.
	do.Provide(injector, func(i do.Injector) (*stockapi.Client, error) {
		cfg := do.MustInvoke[*config.Config](i)
		return stockapi.NewClient(cfg.StockbitAPI.BaseURL, cfg.StockbitAPI.Timeout), nil
	})
	do.MustAs[*stockapi.Client, repository.StockSyncClient](injector)
	do.MustAs[*stockapi.Client, repository.CompanyProfileSyncClient](injector)
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

	do.Provide(injector, func(i do.Injector) (*infraRepo.UserRepository, error) {
		querier, err := do.MustInvoke[*database.Postgres](i).Querier()
		if err != nil {
			return nil, fmt.Errorf("container: get querier for user repo: %w", err)
		}
		return infraRepo.NewUserRepository(querier), nil
	})
	do.MustAs[*infraRepo.UserRepository, repository.UserRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.RBACRepository, error) {
		querier, err := do.MustInvoke[*database.Postgres](i).Querier()
		if err != nil {
			return nil, fmt.Errorf("container: get querier for rbac repo: %w", err)
		}
		return infraRepo.NewRBACRepository(querier), nil
	})
	do.MustAs[*infraRepo.RBACRepository, repository.RBACRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.WatchlistRepository, error) {
		querier, err := do.MustInvoke[*database.Postgres](i).Querier()
		if err != nil {
			return nil, fmt.Errorf("container: get querier for watchlist repo: %w", err)
		}
		return infraRepo.NewWatchlistRepository(querier), nil
	})
	do.MustAs[*infraRepo.WatchlistRepository, repository.WatchlistRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.SectorRepository, error) {
		querier, err := do.MustInvoke[*database.Postgres](i).Querier()
		if err != nil {
			return nil, fmt.Errorf("container: get querier for sector repo: %w", err)
		}
		return infraRepo.NewSectorRepository(querier), nil
	})
	do.MustAs[*infraRepo.SectorRepository, repository.SectorRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.StockRepository, error) {
		querier, err := do.MustInvoke[*database.Postgres](i).Querier()
		if err != nil {
			return nil, fmt.Errorf("container: get querier for stock repo: %w", err)
		}
		return infraRepo.NewStockRepository(querier), nil
	})
	do.MustAs[*infraRepo.StockRepository, repository.StockRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.CompanyProfileRepository, error) {
		querier, err := do.MustInvoke[*database.Postgres](i).Querier()
		if err != nil {
			return nil, fmt.Errorf("container: get querier for company profile repo: %w", err)
		}
		return infraRepo.NewCompanyProfileRepository(querier, do.MustInvoke[repository.TxManager](i)), nil
	})
	do.MustAs[*infraRepo.CompanyProfileRepository, repository.CompanyProfileRepository](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.RedisRefreshStore, error) {
		client := do.MustInvoke[*cache.Redis](i).Client()
		if client == nil {
			return nil, errors.New("container: redis client unavailable for refresh store")
		}
		return infraRepo.NewRedisRefreshStore(client), nil
	})
	do.MustAs[*infraRepo.RedisRefreshStore, repository.RefreshStore](injector)

	do.Provide(injector, func(i do.Injector) (*infraRepo.RedisPermissionCache, error) {
		client := do.MustInvoke[*cache.Redis](i).Client()
		if client == nil {
			return nil, errors.New("container: redis client unavailable for permission cache")
		}
		return infraRepo.NewRedisPermissionCache(client), nil
	})
	do.MustAs[*infraRepo.RedisPermissionCache, repository.PermissionCache](injector)
}

func provideUsecases(injector *do.RootScope) {
	do.Provide(injector, func(i do.Injector) (*token.JWTTokenService, error) {
		cfg := do.MustInvoke[*config.Config](i)
		store := do.MustInvoke[repository.RefreshStore](i)
		return token.NewJWTTokenService(
			cfg.Auth.JWTSecret,
			cfg.Auth.AccessTokenTTL,
			cfg.Auth.RefreshTokenTTL,
			store,
		), nil
	})
	do.MustAs[*token.JWTTokenService, repository.TokenIssuer](injector)

	do.Provide(injector, func(i do.Injector) (usecase.HealthUsecase, error) {
		return usecase.NewHealthUsecase(do.MustInvoke[repository.HealthRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.AuthUsecase, error) {
		cfg := do.MustInvoke[*config.Config](i)
		return usecase.NewAuthUsecase(
			do.MustInvoke[repository.UserRepository](i),
			do.MustInvoke[repository.TokenIssuer](i),
			do.MustInvoke[repository.TxManager](i),
			usecase.AuthConfig{
				DefaultUserTTL: cfg.Auth.DefaultUserTTL,
				BcryptCost:     cfg.Auth.BcryptCost,
			},
		), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.UserUsecase, error) {
		return usecase.NewUserUsecase(do.MustInvoke[repository.UserRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.WatchlistUsecase, error) {
		return usecase.NewWatchlistUsecase(do.MustInvoke[repository.WatchlistRepository](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.StockUsecase, error) {
		return usecase.NewStockUsecase(
			do.MustInvoke[repository.StockRepository](i),
			do.MustInvoke[repository.SectorRepository](i),
			do.MustInvoke[repository.StockSyncClient](i),
			do.MustInvoke[log.Logger](i),
		), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.SectorUsecase, error) {
		return usecase.NewSectorUsecase(
			do.MustInvoke[repository.SectorRepository](i),
		), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.CompanyProfileUsecase, error) {
		return usecase.NewCompanyProfileUsecase(
			do.MustInvoke[repository.CompanyProfileRepository](i),
			do.MustInvoke[repository.CompanyProfileSyncClient](i),
		), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.RBACUsecase, error) {
		return usecase.NewRBACUsecase(
			do.MustInvoke[repository.RBACRepository](i),
			do.MustInvoke[repository.PermissionCache](i),
			do.MustInvoke[log.Logger](i),
		), nil
	})

	do.Provide(injector, func(i do.Injector) (usecase.AdminUsecase, error) {
		return usecase.NewAdminUsecase(
			do.MustInvoke[repository.UserRepository](i),
			do.MustInvoke[usecase.RBACUsecase](i),
			do.MustInvoke[repository.PermissionCache](i),
			do.MustInvoke[log.Logger](i),
		), nil
	})
}

func provideHandlers(injector *do.RootScope) {
	do.Provide(injector, func(i do.Injector) (*health.HealthHandler, error) {
		return health.NewHealthHandler(do.MustInvoke[usecase.HealthUsecase](i)), nil
	})

	do.Provide(injector, func(i do.Injector) (*authhandler.AuthHandler, error) {
		return authhandler.NewAuthHandler(
			do.MustInvoke[usecase.AuthUsecase](i),
			do.MustInvoke[appvalidator.Validator](i),
		), nil
	})

	do.Provide(injector, func(i do.Injector) (*user.UserHandler, error) {
		return user.NewUserHandler(
			do.MustInvoke[usecase.UserUsecase](i),
			do.MustInvoke[appvalidator.Validator](i),
		), nil
	})

	do.Provide(injector, func(i do.Injector) (*watchlist.WatchlistHandler, error) {
		return watchlist.NewWatchlistHandler(
			do.MustInvoke[usecase.WatchlistUsecase](i),
			do.MustInvoke[appvalidator.Validator](i),
		), nil
	})

	do.Provide(injector, func(i do.Injector) (*admin.AdminHandler, error) {
		return admin.NewAdminHandler(
			do.MustInvoke[usecase.AdminUsecase](i),
			do.MustInvoke[appvalidator.Validator](i),
		), nil
	})

	do.Provide(injector, func(i do.Injector) (*stock.StockHandler, error) {
		return stock.NewStockHandler(
			do.MustInvoke[usecase.StockUsecase](i),
			do.MustInvoke[appvalidator.Validator](i),
		), nil
	})

	do.Provide(injector, func(i do.Injector) (*companyprofile.CompanyProfileHandler, error) {
		return companyprofile.NewCompanyProfileHandler(
			do.MustInvoke[usecase.CompanyProfileUsecase](i),
			do.MustInvoke[appvalidator.Validator](i),
		), nil
	})

	do.Provide(injector, func(i do.Injector) (*sector.SectorHandler, error) {
		return sector.NewSectorHandler(
			do.MustInvoke[usecase.SectorUsecase](i),
			do.MustInvoke[appvalidator.Validator](i),
		), nil
	})

	// Server
	do.Provide(injector, func(i do.Injector) (*deliveryhttp.Server, error) {
		cfg := do.MustInvoke[*config.Config](i)
		logger := do.MustInvoke[log.Logger](i)

		router := deliveryhttp.NewRouter(deliveryhttp.Handlers{
			Health:         do.MustInvoke[*health.HealthHandler](i),
			Auth:           do.MustInvoke[*authhandler.AuthHandler](i),
			User:           do.MustInvoke[*user.UserHandler](i),
			Watchlist:      do.MustInvoke[*watchlist.WatchlistHandler](i),
			Admin:          do.MustInvoke[*admin.AdminHandler](i),
			Stock:          do.MustInvoke[*stock.StockHandler](i),
			CompanyProfile: do.MustInvoke[*companyprofile.CompanyProfileHandler](i),
			Sector:         do.MustInvoke[*sector.SectorHandler](i),
		}, appmw.AuthDeps{
			Verifier: do.MustInvoke[*token.JWTTokenService](i),
			Loader:   do.MustInvoke[repository.UserRepository](i),
			Checker:  do.MustInvoke[usecase.RBACUsecase](i),
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
		close(errChan)
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, os.Interrupt)
	defer signal.Stop(sigChan)

	select {
	case err, ok := <-errChan:
		if !ok {
			// Server exited cleanly (shouldn't happen without signal, but handle gracefully)
			return nil
		}
		logger.Error("server startup failed", "error", err)
		if report := injector.Shutdown(); !report.Succeed {
			logger.Error("container shutdown failed", "error", report)
		}
		return err
	case sig := <-sigChan:
		logger.Info("received shutdown signal", "signal", sig.String())
		// Gracefully shut down the HTTP server first — this unblocks the
		// goroutine waiting on ListenAndServe so errChan gets closed.
		if err := server.Shutdown(); err != nil {
			logger.Error("http server shutdown failed", "error", err)
		}
		// Drain the error channel (server returns ErrServerClosed which we ignore).
		<-errChan

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

	// Run migrations
	if err := database.RunMigrations(context.Background(), cfg.Database.URL); err != nil {
		logger.Error("migrations failed", "error", err)
		return err
	}
	logger.Info("migrations applied")

	injector := New(cfg, logger)

	if err := pingInfra(injector); err != nil {
		return err
	}

	server, err := do.Invoke[*deliveryhttp.Server](injector)
	if err != nil {
		return fmt.Errorf("container: construct server: %w", err)
	}

	logger.Info("server started",
		"app", cfg.App.Name,
		"version", cfg.App.Version,
		"addr", cfg.Port,
	)

	return awaitShutdown(server, injector, logger)
}
