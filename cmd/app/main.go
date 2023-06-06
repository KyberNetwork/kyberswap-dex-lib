package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/reload"
	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/KyberNetwork/router-service/internal/pkg/api"
	"github.com/KyberNetwork/router-service/internal/pkg/config"
	"github.com/KyberNetwork/router-service/internal/pkg/job"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/reloadconfig"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/gas"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/poolrank"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/price"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/route"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/setting"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/token"
	"github.com/KyberNetwork/router-service/internal/pkg/server"
	httppkg "github.com/KyberNetwork/router-service/internal/pkg/server/http"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/clientdata"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/spfav2"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/getcustomroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/getroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/poolfactory"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/poolmanager"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/validateroute"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/envvar"
	timeutil "github.com/KyberNetwork/router-service/internal/pkg/utils/time"
	"github.com/KyberNetwork/router-service/internal/pkg/validator"
	cryptopkg "github.com/KyberNetwork/router-service/pkg/crypto"
	"github.com/KyberNetwork/router-service/pkg/crypto/keystorage"
	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/KyberNetwork/router-service/pkg/redis"
	"github.com/KyberNetwork/router-service/pkg/util/env"
)

type IGetRouteUseCase interface {
	ApplyConfig(config getroute.Config)
}

type IPoolManager interface {
	ApplyConfig(config poolmanager.Config)
}

// TODO: refactor main file -> separate to many folders with per folder is application. The main file should contains call root action per application.
func main() {

	if env.StringFromEnv(envvar.DDEnabled, "") != "" {
		addr := net.JoinHostPort(
			env.StringFromEnv(envvar.DDAgentHost, ""),
			"8126",
		)

		samplerRate := env.ParseFloatFromEnv(envvar.DDSamplerRate, 0.2, 0, 1)
		tracer.Start(
			tracer.WithEnv(env.StringFromEnv(envvar.DDEnv, "")),
			tracer.WithService(env.StringFromEnv(envvar.DDService, "")),
			tracer.WithServiceVersion(env.StringFromEnv(envvar.DDVersion, "")),
			tracer.WithAgentAddr(addr),
			tracer.WithSampler(tracer.NewRateSampler(samplerRate)),
		)
		defer tracer.Stop()
	}

	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   "internal/pkg/config/default.yaml",
				Usage:   "Configuration file",
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "api",
				Aliases: []string{},
				Usage:   "Run api server",
				Action:  apiAction,
			},
			{
				Name:    "indexer",
				Aliases: []string{},
				Usage:   "Index pools",
				Action:  indexerAction,
			},
		}}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func apiAction(c *cli.Context) (err error) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	configFile := c.String("config")

	configLoader, err := config.NewConfigLoader(configFile)
	if err != nil {
		return err
	}

	cfg, err := configLoader.Get()
	if err != nil {
		return err
	}

	lg, err := logger.InitLogger(cfg.Log.Configuration, logger.LoggerBackendZap)
	if err != nil {
		return err
	}

	// Initialize config reloader
	restSettingRepo := setting.NewRestRepository(cfg.ReloadConfig.HttpUrl)
	reloadConfigUseCase := usecase.NewReloadConfigUseCase(restSettingRepo)
	reloadConfigFetcher := reloadconfig.NewReloadConfigFetcher(cfg.ReloadConfig, reloadConfigUseCase)
	reloadConfigReporter := reloadconfig.NewReloadConfigReporter(cfg.ReloadConfig, reloadConfigUseCase)

	configLoader.SetRemoteConfigFetcher(reloadConfigFetcher)

	// reload config with remote config. Ignore error with a warning
	err = configLoader.Reload(ctx)
	if err != nil {
		logger.Warnf("Config could not be reloaded: %s", err)
	} else {
		logger.Info("Config reloaded")
	}

	if err := cfg.Validate(); err != nil {
		logger.Errorf("failed to validate config, err: %v", err)
		panic(err)
	}

	if err := sentry.Init(sentry.ClientOptions{
		Dsn: cfg.Log.SentryDSN,
	}); err != nil {
		logger.Errorf("sentry.Init error cause by %v", err)
		return err
	}

	// Flush buffered events before the program terminates.
	defer sentry.Flush(2 * time.Second)

	routerRedisClient, err := redis.New(&cfg.Redis)
	if err != nil {
		return err
	}

	poolRedisClient, err := redis.New(&cfg.PoolRedis)
	if err != nil {
		logger.Errorf("fail to init redis client to pool service")
		return err
	}

	_, err = metrics.InitClient(newMetricsConfig(cfg))

	ethClient := ethrpc.New(cfg.Common.RPC)

	// init repositories
	tokenDataStoreRepo := token.NewRedisRepository(poolRedisClient.Client, cfg.Repository.Token.Redis)
	tokenCacheRepo := token.NewGoCacheRepository(
		tokenDataStoreRepo,
		cfg.Repository.Token.GoCache,
	)

	poolDataStoreRepo := pool.NewRedisRepository(poolRedisClient.Client, cfg.Repository.Pool.Redis)
	priceDataStoreRepo := price.NewRedisRepository(poolRedisClient.Client, cfg.Repository.Price.Redis)

	gasRepository := gas.NewRedisRepository(routerRedisClient.Client, ethClient, cfg.Repository.Gas.Redis)
	poolRankRepository := poolrank.NewRedisRepository(routerRedisClient.Client, cfg.Repository.PoolRank.Redis)
	routeRepository := route.NewRedisCacheRepository(routerRedisClient.Client, cfg.Repository.Route.RedisCache)

	tokenRepository := token.NewGoCacheRepository(
		token.NewRedisRepository(poolRedisClient.Client, cfg.Repository.Token.Redis),
		cfg.Repository.Token.GoCache,
	)
	poolRepository := pool.NewRedisRepository(poolRedisClient.Client, cfg.Repository.Pool.Redis)
	priceRepository := price.NewRedisRepository(poolRedisClient.Client, cfg.Repository.Price.Redis)
	// sealer

	// init validators
	getPoolsParamsValidator := validator.NewGetPoolsParamsValidator()
	getTokensParamsValidator := validator.NewGetTokensParamsValidator()
	getRoutesParamsValidator := validator.NewGetRouteParamsValidator()
	buildRouteParamsValidator := validator.NewBuildRouteParamsValidator(timeutil.NowFunc, cfg.Validator.BuildRouteParams)

	// init use cases
	keyStorage, err := getKeyStorage(cfg.KeyPair.StorageFilePath)
	if err != nil {
		return err
	}

	signer := cryptopkg.NewLocalSigner(keyStorage)
	keyPairUseCase := usecase.NewGetPublicKeyUseCase(keyStorage)

	clientDataEncoder := clientdata.NewEncoder(
		signer,
		cfg.KeyPair.KeyIDForSealingData.ClientData,
	)
	encoder := encode.NewEncoder(cfg.Encoder)

	validateRouteUseCase := validateroute.NewValidateRouteUseCase()
	validateRouteUseCase.RegisterValidator(validateroute.NewSynthetixValidator())

	getPoolsUseCase := usecase.NewGetPoolsUseCase(poolDataStoreRepo)
	getTokensUseCase := usecase.NewGetTokens(tokenCacheRepo, poolDataStoreRepo, priceDataStoreRepo)

	poolFactory := poolfactory.NewPoolFactory(cfg.UseCase.PoolFactory)
	poolManager, err := poolmanager.NewPointerSwapPoolManager(poolRepository, poolFactory, poolRankRepository, cfg.UseCase.PoolManager)
	if err != nil {
		return err
	}

	getRouteUseCase := getroute.NewUseCase(
		poolRankRepository,
		tokenRepository,
		priceRepository,
		routeRepository,
		gasRepository,
		poolManager,
		cfg.UseCase.GetRoute,
	)

	buildRouteUseCase := usecase.NewBuildRouteUseCase(
		tokenCacheRepo,
		priceDataStoreRepo,
		clientDataEncoder,
		encoder,
		timeutil.NowFunc,
		cfg.UseCase.BuildRoute,
	)

	routeFinder := spfav2.NewSPFAv2Finder(
		cfg.UseCase.GetRoute.AmmAggregator.FinderOptions.MaxHops,
		cfg.UseCase.GetRoute.AmmAggregator.FinderOptions.DistributionPercent,
		cfg.UseCase.GetRoute.AmmAggregator.FinderOptions.MaxPathsInRoute,
		cfg.UseCase.GetRoute.AmmAggregator.FinderOptions.MaxPathsToGenerate,
		cfg.UseCase.GetRoute.AmmAggregator.FinderOptions.MaxPathsToReturn,
		cfg.UseCase.GetRoute.AmmAggregator.FinderOptions.MinPartUSD,
		cfg.UseCase.GetRoute.AmmAggregator.FinderOptions.MinThresholdAmountInUSD,
		cfg.UseCase.GetRoute.AmmAggregator.FinderOptions.MaxThresholdAmountInUSD,
	)
	getCustomRoutesUseCase := getcustomroute.NewCustomRoutesUseCase(
		tokenRepository,
		priceRepository,
		gasRepository,
		poolManager,
		routeFinder,
		getcustomroute.Config{
			ChainID:          cfg.UseCase.GetRoute.ChainID,
			RouterAddress:    cfg.UseCase.GetRoute.RouterAddress,
			GasTokenAddress:  cfg.UseCase.GetRoute.GasTokenAddress,
			AvailableSources: cfg.UseCase.GetRoute.AvailableSources,
		},
	)

	// init services
	zapLogger, err := logger.GetDesugaredZapLoggerDelegate(lg)
	if err != nil {
		return err
	}
	ginServer, router, _ := httppkg.GinServer(cfg.Http, zapLogger)

	v1 := router.Group("/api/v1")

	v1Health := v1.Group("/health")
	v1Health.GET("/live", func(c *gin.Context) { c.AbortWithStatusJSON(http.StatusOK, "OK") })
	v1Health.GET("/ready", func(c *gin.Context) { c.AbortWithStatusJSON(http.StatusOK, "OK") })

	v1Debug := v1.Group("/debug")
	v1Debug.GET("/config", func(c *gin.Context) {
		currentConfig, _ := configLoader.Get()
		c.JSON(http.StatusOK, currentConfig)
	})
	v1Debug.GET("/custom-routes", api.GetCustomRoutes(getRoutesParamsValidator, getCustomRoutesUseCase))

	v1.GET("/pools", api.GetPools(getPoolsParamsValidator, getPoolsUseCase))
	v1.GET("/tokens", api.GetTokens(getTokensParamsValidator, getTokensUseCase))
	v1.GET("/routes", api.GetRoutes(getRoutesParamsValidator, getRouteUseCase))
	v1.POST("/route/build", api.BuildRoute(buildRouteParamsValidator, buildRouteUseCase, timeutil.NowFunc))
	v1.GET("/keys/publics/:keyId", api.GetPublicKey(keyPairUseCase))

	reloadManager := reload.NewManager()

	// Run hot-reload manager.
	// Add all app reloaders in order.
	reloadManager.RegisterReloader(0, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		logger.Infof("Received reloading signal: <%s>", id)

		// If configuration fails ignore reload with a warning.
		err = configLoader.Reload(ctx)
		if err != nil {
			logger.Warnf("Config could not be reloaded: %s", err)
			return nil
		}

		logger.Infoln("Config reloaded")
		return nil
	}))

	reloadManager.RegisterReloader(100, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		logger.Infof("Received reloading signal: <%s>", id)
		return applyLatestConfigForAPI(ctx, configLoader, getRouteUseCase, poolManager)
	}))

	httpServer := &http.Server{Handler: ginServer, Addr: cfg.Http.BindAddress}

	// use pointer for reloadManager to avoid "Function returns lock by value": https://stackoverflow.com/a/69281722/2667212
	apiServer := server.NewServer(httpServer, cfg, reloadConfigReporter, &reloadManager)
	return apiServer.Run(ctx)
}

func indexerAction(c *cli.Context) (err error) {
	ctx := context.Background()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// load config
	configFile := c.String("config")
	configLoader, err := config.NewConfigLoader(configFile)
	if err != nil {
		return err
	}
	cfg, err := configLoader.Get()
	if err != nil {
		return err
	}

	// init logger
	_, err = logger.InitLogger(cfg.Log.Configuration, logger.LoggerBackendZap)
	if err != nil {
		return err
	}

	// init metrics client
	_, err = metrics.InitClient(newMetricsConfig(cfg))

	// Initialize config reloader
	restSettingRepo := setting.NewRestRepository(cfg.ReloadConfig.HttpUrl)
	reloadConfigUseCase := usecase.NewReloadConfigUseCase(restSettingRepo)
	reloadConfigFetcher := reloadconfig.NewReloadConfigFetcher(cfg.ReloadConfig, reloadConfigUseCase)
	reloadConfigReporter := reloadconfig.NewReloadConfigReporter(cfg.ReloadConfig, reloadConfigUseCase)

	configLoader.SetRemoteConfigFetcher(reloadConfigFetcher)

	// reload config with remote config. Ignore error with a warning
	err = configLoader.Reload(ctx)
	if err != nil {
		logger.Warnf("Config could not be reloaded: %s", err)
	} else {
		logger.Infoln("Config reloaded")
	}

	ethClient := ethrpc.New(cfg.Common.RPC)

	// init redis client
	routerRedisClient, err := redis.New(&cfg.Redis)
	if err != nil {
		logger.Errorf("fail to init redis client for indexer")
		return err
	}

	poolRedisClient, err := redis.New(&cfg.PoolRedis)
	if err != nil {
		logger.Errorf("fail to init redis client to pool service")
		return err
	}

	// init repository
	poolRepo := pool.NewRedisRepository(poolRedisClient.Client, cfg.Repository.Pool.Redis)
	poolRankRepository := poolrank.NewRedisRepository(routerRedisClient.Client, cfg.Repository.PoolRank.Redis)
	gasRepository := gas.NewRedisRepository(routerRedisClient.Client, ethClient, gas.RedisRepositoryConfig{Prefix: cfg.Redis.Prefix})

	// init use case
	getAllPoolAddressesUseCase := usecase.NewGetAllPoolAddressesUseCase(poolRepo)
	indexPoolsUseCase := usecase.NewIndexPoolsUseCase(
		poolRepo,
		poolRankRepository,
		cfg.UseCase.IndexPools,
	)
	updateSuggestedGasPriceUseCase := usecase.NewUpdateSuggestedGasPrice(gasRepository)

	indexPoolsJob := job.NewIndexPoolsJob(
		getAllPoolAddressesUseCase,
		indexPoolsUseCase,
		cfg.Job.IndexPools,
	)
	updateSuggestedGasPriceJob := job.NewUpdateSuggestedGasPriceJob(
		updateSuggestedGasPriceUseCase,
		cfg.Job.UpdateSuggestedGasPrice,
	)

	reloadManager := reload.NewManager()

	// Run hot-reload manager.
	// Add all app reloaders in order.
	reloadManager.RegisterReloader(0, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		logger.Infof("Received reloading signal: <%s>", id)

		// If configuration fails ignore reload with a warning.
		err = configLoader.Reload(ctx)
		if err != nil {
			logger.Warnf("Config could not be reloaded: %s", err)
			return nil
		}

		logger.Infoln("Config reloaded")
		return nil
	}))

	reloadManager.RegisterReloader(100, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		logger.Infof("Received reloading signal: <%s>", id)
		logger.Infoln("ScanConfigService reloaded")

		return applyLatestConfigForIndexer(ctx, indexPoolsUseCase, indexPoolsJob, configLoader)
	}))

	g, ctx := errgroup.WithContext(ctx)

	// run jobs
	g.Go(func() error {
		logger.Infof("Starting reload manager")
		return reloadManager.Run(ctx)
	})

	g.Go(func() error {
		logger.Info("Starting indexPoolsJobs")

		indexPoolsJob.Run(ctx)

		return nil
	})

	g.Go(func() error {
		logger.Info("Starting updateSuggestedGasPriceJob")

		updateSuggestedGasPriceJob.Run(ctx)

		return nil
	})

	// Register notifier
	reloadChan := make(chan string)
	reloadManager.RegisterNotifier(reload.NotifierChan(reloadChan))

	g.Go(func() error {
		logger.Infoln("Starting reload config reporter")
		reloadConfigReporter.Report(ctx, reloadChan)
		return nil
	})

	return g.Wait()
}

func getKeyStorage(storageFilePath string) (cryptopkg.KeyPairStorage, error) {
	keyStorage, err := keystorage.NewInMemoryStorageFromFile(storageFilePath)
	if err != nil {
		// handle fallback when we have not setup file key pair store
		return keystorage.NewInMemoryStorage(nil), nil
	}
	return keyStorage, err
}

func newMetricsConfig(cfg *config.Config) metrics.Config {
	host := cfg.DogstatsdHost

	if len(cfg.Metrics.Host) > 0 {
		host = cfg.Metrics.Host
	}

	return metrics.Config{
		Host:      host,
		Port:      cfg.Metrics.Port,
		Namespace: cfg.Metrics.Namespace,
	}
}

func applyLatestConfigForAPI(
	_ context.Context,
	configLoader *config.ConfigLoader,
	getRouteUseCase IGetRouteUseCase,
	poolManager IPoolManager,
) error {
	cfg, err := configLoader.Get()
	if err != nil {
		return err
	}

	logger.Infoln("Applying new log level")
	if err = logger.SetLogLevel(cfg.Log.ConsoleLevel); err != nil {
		logger.Warnf("reload Log level error cause by <%v>", err)
	}

	getRouteUseCase.ApplyConfig(cfg.UseCase.GetRoute)
	poolManager.ApplyConfig(cfg.UseCase.PoolManager)

	return nil
}

func applyLatestConfigForIndexer(
	_ context.Context,
	indexPoolsUseCase *usecase.IndexPoolsUseCase,
	indexPoolsJob *job.IndexPoolsJob,
	configLoader *config.ConfigLoader,
) error {
	cfg, err := configLoader.Get()
	if err != nil {
		return err
	}

	logger.Infoln("Applying new log level")
	if err = logger.SetLogLevel(cfg.Log.ConsoleLevel); err != nil {
		logger.Warnf("reload Log level error cause by <%v>", err)
	}

	logger.Infoln("Applying new config to IndexPoolsJob")
	indexPoolsJob.ApplyConfig(cfg.Job.IndexPools)

	logger.Infoln("Applying new config to IndexPoolsUseCase")
	indexPoolsUseCase.ApplyConfig(cfg.UseCase.IndexPools)

	return nil
}
