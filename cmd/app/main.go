package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/KyberNetwork/kyberswap-error/pkg/transformers"
	"github.com/KyberNetwork/reload"
	gincache "github.com/chenyahui/gin-cache"
	"github.com/chenyahui/gin-cache/persist"
	"github.com/getsentry/sentry-go"
	"github.com/gin-contrib/timeout"
	"github.com/gin-gonic/gin"
	redisv8 "github.com/go-redis/redis/v8"
	cmap "github.com/orcaman/concurrent-map"
	"github.com/patrickmn/go-cache"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/KyberNetwork/router-service/internal/pkg/api"
	"github.com/KyberNetwork/router-service/internal/pkg/config"
	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	errorsPkg "github.com/KyberNetwork/router-service/internal/pkg/errors"
	"github.com/KyberNetwork/router-service/internal/pkg/job"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/reloadconfig"
	"github.com/KyberNetwork/router-service/internal/pkg/repository"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/setting"
	"github.com/KyberNetwork/router-service/internal/pkg/server"
	httppkg "github.com/KyberNetwork/router-service/internal/pkg/server/http"
	"github.com/KyberNetwork/router-service/internal/pkg/service"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/clientdata"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/factory"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/getroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/l2feecalculator"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/validateroute"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/envvar"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/eth"
	timeutil "github.com/KyberNetwork/router-service/internal/pkg/utils/time"
	"github.com/KyberNetwork/router-service/internal/pkg/validator"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	cryptopkg "github.com/KyberNetwork/router-service/pkg/crypto"
	"github.com/KyberNetwork/router-service/pkg/crypto/keystorage"
	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/KyberNetwork/router-service/pkg/redis"
	"github.com/KyberNetwork/router-service/pkg/util/env"
)

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

	// wDb to write to secondary Redis
	wDb, err := redis.New(&cfg.Redis)
	if err != nil {
		return err
	}

	// rDb to read-only secondary Redis read-replica.
	rDb, err := redis.New(&cfg.ReadOnlyRedis)
	if err != nil {
		return err
	}

	// k8sDb to read/write to primary Redis (sentinel) in our K8S cluster
	k8sDb, err := redis.NewSentinel(&cfg.RedisSentinel)
	if err != nil {
		// it's ok to not be able to connect primary Redis
		logger.Warnf("Can not connect to primary redis (sentinel) in k8s: %v", err)
		k8sDb = nil
	}

	_, err = metrics.InitClient(newMetricsConfig(cfg))

	// init repositories
	tokenDataStoreRepo := repository.NewTokenDataStoreRedisRepository(rDb)
	tokenCacheRepo := repository.NewTokenCacheRepository(
		tokenDataStoreRepo,
		cache.New(cache.NoExpiration, cache.NoExpiration),
	)

	poolDataStoreRepo := repository.NewPoolDataStoreRedisRepository(rDb)
	priceDataStoreRepo := repository.NewPriceDataStoreRedisRepository(rDb)

	var (
		rDbClient   *redisv8.Client
		wDbClient   *redisv8.Client
		k8sDbClient *redisv8.ClusterClient
	)

	if rDb != nil {
		rDbClient = rDb.Client
	}

	if wDb != nil {
		wDbClient = wDb.Client
	}

	if k8sDb != nil {
		k8sDbClient = k8sDb.Client
	}

	routeCacheRepo := repository.NewRouteCacheRedisRepository(
		rDbClient,
		wDbClient,
		k8sDbClient,
	)
	routeRepo := repository.NewRouteRedisRepository(rDb)
	scanStateRepo := repository.NewScannerStateRedisRepository(rDb)

	// sealer

	// init validators
	getPoolsParamsValidator := validator.NewGetPoolsParamsValidator()
	getTokensParamsValidator := validator.NewGetTokensParamsValidator()
	getRoutesParamsValidator := validator.NewGetRouteParamsValidator()
	buildRouteParamsValidator := validator.NewBuildRouteParamsValidator(timeutil.NowFunc)

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
	poolFactoryConfig := factory.PoolFactoryConfig{ChainID: cfg.Common.ChainID}
	poolFactory := factory.NewPoolFactory(poolFactoryConfig)

	getPoolsUseCase := usecase.NewGetPoolsUseCase(poolDataStoreRepo)
	getTokensUseCase := usecase.NewGetTokens(tokenCacheRepo, poolDataStoreRepo, priceDataStoreRepo)

	cacheRouteConfig := newCacheRouteConfig(cfg)
	cacheRouteUseCase := usecase.NewCacheRouteUseCase(cacheRouteConfig, routeCacheRepo)

	getRoutesUseCase := getroute.NewGetRoutesUseCase(
		cacheRouteUseCase,
		validateRouteUseCase,
		poolFactory,
		poolDataStoreRepo,
		tokenCacheRepo,
		priceDataStoreRepo,
		routeRepo,
		scanStateRepo,
		cfg.UseCase.GetRoutes,
	)

	buildRouteUseCase := usecase.NewBuildRouteUseCase(
		tokenCacheRepo,
		priceDataStoreRepo,
		clientDataEncoder,
		encoder,
		timeutil.NowFunc,
		usecase.BuildRouteConfig{ChainID: valueobject.ChainID(cfg.Common.ChainID)},
	)

	l2FeeCalculator := l2feecalculator.NewL2FeeCalculator(valueobject.ChainID(cfg.Common.ChainID))
	l2FeeCalculatorUseCase := usecase.NewL2FeeCalculatorUseCase(l2FeeCalculator, scanStateRepo)

	// init services
	zapLogger, err := logger.GetDesugaredZapLoggerDelegate(lg)
	if err != nil {
		return err
	}
	ginServer, router, _ := httppkg.GinServer(cfg.Http, zapLogger)

	routeSvc := service.NewRoute(
		configLoader,
		router,
		wDb,
		rDb,
		k8sDb,
		cfg.Gas,
		cfg.Common,
		poolDataStoreRepo,
		tokenDataStoreRepo,
		priceDataStoreRepo,
		cfg.EnableDexes,
		cfg.Epsilon,
		cfg.CachePoints,
		cfg.CacheRanges,
		poolFactory,
		validateRouteUseCase,
		l2FeeCalculatorUseCase,
		clientDataEncoder,
		encoder,
		cfg.BlacklistedPools,
		cfg.FeatureFlags,
	)

	service.SetupStatsRoute(rDb, router)

	apiHandlersWithConfig := apiHandlersFactory(cfg.API.DefaultTTL)

	getPoolsHandlers := apiHandlersWithConfig(api.GetPools(getPoolsParamsValidator, getPoolsUseCase), cfg.API.GetPools)
	getTokensHandlers := apiHandlersWithConfig(api.GetTokens(getTokensParamsValidator, getTokensUseCase), cfg.API.GetTokens)
	getRoutesHandlers := apiHandlersWithConfig(api.GetRoutes(getRoutesParamsValidator, getRoutesUseCase), cfg.API.GetRoutes)
	buildRouteHandlers := apiHandlersWithConfig(api.BuildRoute(buildRouteParamsValidator, buildRouteUseCase, timeutil.NowFunc), cfg.API.BuildRoute)
	getPublicKeyHandlers := apiHandlersWithConfig(api.GetPublicKey(keyPairUseCase), cfg.API.GetPublicKey)

	v1 := router.Group("/api/v1")

	v1Health := v1.Group("/health")
	v1Health.GET("/live", func(c *gin.Context) { c.AbortWithStatusJSON(http.StatusOK, "OK") })
	v1Health.GET("/ready", func(c *gin.Context) { c.AbortWithStatusJSON(http.StatusOK, "OK") })

	v1.GET("/pools", getPoolsHandlers...)
	v1.GET("/tokens", getTokensHandlers...)
	v1.GET("/routes", getRoutesHandlers...)
	v1.POST("/route/build", buildRouteHandlers...)
	v1.GET("/keys/publics/:keyId", getPublicKeyHandlers...)

	transformer := transformers.RestTransformerInstance()
	transformer.RegisterTransformFunc(constant.DomainErrCodeTokensAreIdentical, errorsPkg.NewRestAPIErrTokensAreIdentical)

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
		return applyLatestConfigForAPI(ctx, getRoutesUseCase, routeSvc, configLoader)
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

	// init eth client
	ethClient, err := eth.NewClient(cfg.Common.RPCs)
	if err != nil {
		logger.Errorf("error when initing RPC client cause by %v", err)
		return err
	}

	// init redis client
	rds, err := redis.New(&cfg.Redis)
	if err != nil {
		return err
	}

	// init repository
	poolDatastoreRepo := repository.NewPoolDataStoreRedisRepository(rds)
	poolCacheRepo := repository.NewPoolCacheCMapRepository(cmap.New(), cmap.New())
	poolRepo := repository.NewPoolRepository(poolDatastoreRepo, poolCacheRepo)
	statsRepo := repository.NewStatsRedisRepository(rds)
	routeRepo := repository.NewRouteRedisRepository(rds)
	rpcRepo := repository.NewRPCRepository(ethClient, repository.RPCRepositoryConfig{
		RPCs:             cfg.Common.RPCs,
		MulticallAddress: cfg.Common.Address.Multicall,
	})

	// init service
	rpcService := service.NewRPC(cfg.Common)
	commonService := service.NewCommon(rds, cfg.Common, rpcService)
	scanSvc := service.NewScanService(rpcRepo)

	jobs := []service.IService{
		rpcService,
		commonService,
		service.NewStats(scanSvc, poolRepo, statsRepo),
	}

	// init use case
	getAllPoolAddressesUseCase := usecase.NewGetAllPoolAddressesUseCase(poolRepo)
	indexPoolsUseCase := usecase.NewIndexPoolsUseCase(
		poolRepo,
		routeRepo,
		usecase.IndexPoolsConfig{
			WhitelistedTokensByAddress: cfg.WhitelistedTokensByAddress(),
			ChunkSize:                  cfg.IndexPoolsChunkSize,
		},
	)
	indexPoolsJob := job.NewIndexPoolsJob(
		getAllPoolAddressesUseCase,
		indexPoolsUseCase,
		job.IndexPoolsJobConfig{
			IndexPoolsJobIntervalSec: cfg.IndexPoolsJobIntervalSec,
		},
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
		logger.Infoln("Starting scanner")

		var wg sync.WaitGroup
		for _, s := range jobs {
			wg.Add(1)
			go func(s service.IService) {
				defer wg.Done()
				s.UpdateData(ctx)
			}(s)
		}

		wg.Wait()

		return nil
	})

	g.Go(func() error {
		logger.Infof("Starting indexer")
		indexPoolsJob.Run(ctx)
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

func newCacheRouteConfig(cfg *config.Config) usecase.CacheRouteConfig {
	cachePoints := make([]usecase.CachePointConfig, 0, len(cfg.CachePoints))
	for _, cachePoint := range cfg.CachePoints {
		cachePoints = append(cachePoints, usecase.CachePointConfig{
			Amount: int64(cachePoint.Amount),
			TTL:    time.Duration(cachePoint.TTL) * time.Second,
		})
	}

	cacheRanges := make([]usecase.CacheRangeConfig, 0, len(cfg.CacheRanges))
	for _, cacheRange := range cfg.CacheRanges {
		cacheRanges = append(cacheRanges, usecase.CacheRangeConfig{
			FromUSD: cacheRange.FromUSD,
			ToUSD:   cacheRange.ToUSD,
			TTL:     time.Duration(cacheRange.TTL) * time.Second,
		})
	}

	return usecase.CacheRouteConfig{
		CachePoints:     cachePoints,
		CacheRanges:     cacheRanges,
		KeyPrefix:       cfg.Redis.Prefix,
		DefaultCacheTTL: 10 * time.Second,
	}
}

func apiHandlersFactory(defaultTTL time.Duration) func(mainHandler gin.HandlerFunc, config api.ItemConfig) []gin.HandlerFunc {
	cacheStore := persist.NewMemoryStore(defaultTTL)

	return func(mainHandler gin.HandlerFunc, config api.ItemConfig) []gin.HandlerFunc {
		var handlers []gin.HandlerFunc

		if config.IsCacheEnabled {
			handlers = append(handlers, gincache.CacheByRequestURI(cacheStore, config.TTL))
		}

		if config.IsTimeoutEnabled {
			handlers = append(handlers,
				timeout.New(
					timeout.WithTimeout(config.Timeout),
					timeout.WithHandler(mainHandler),
					timeout.WithResponse(api.TimeoutHandler),
				),
			)
		} else {
			handlers = append(handlers, mainHandler)
		}

		return handlers
	}
}

func applyLatestConfigForAPI(
	ctx context.Context,
	getRoutesUseCase *getroute.GetRoutesUseCase,
	routeSvc *service.RouteService,
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

	logger.Infoln("Applying new config to GetRoutesUseCase")
	if err = getRoutesUseCase.ApplyConfig(cfg.EnableDexes, cfg.BlacklistedPools, cfg.FeatureFlags, cfg.WhitelistedTokens); err != nil {
		logger.Warnf("reload GetRoutesUseCase's config error cause by <%v>", err)
	}

	logger.Infoln("Applying new config to RouteService")
	if err = routeSvc.ApplyConfig(ctx); err != nil {
		logger.Warnf("reload RouteService's config error cause by <%v>", err)
	}

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
	indexPoolsJob.ApplyConfig(cfg.IndexPoolsJobIntervalSec)

	logger.Infoln("Applying new config to IndexPoolsUseCase")
	indexPoolsUseCase.ApplyConfig(cfg.WhitelistedTokensByAddress(), cfg.IndexPoolsChunkSize)

	return nil
}
