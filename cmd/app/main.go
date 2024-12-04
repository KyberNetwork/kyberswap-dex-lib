package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	aevmclient "github.com/KyberNetwork/aevm/client"
	"github.com/KyberNetwork/aggregator-encoding/pkg/decode"
	"github.com/KyberNetwork/aggregator-encoding/pkg/encode"
	"github.com/KyberNetwork/aggregator-encoding/pkg/encode/clientdata"
	"github.com/KyberNetwork/aggregator-encoding/pkg/encode/l1encode"
	"github.com/KyberNetwork/aggregator-encoding/pkg/encode/l2encode"
	"github.com/KyberNetwork/ethrpc"
	_ "github.com/KyberNetwork/kyber-trace-go/tools"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/pathfinder-lib/pkg/finderengine"
	"github.com/KyberNetwork/reload"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/getsentry/sentry-go"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/grafana/pyroscope-go"
	"github.com/urfave/cli/v2"
	_ "go.uber.org/automaxprocs"
	"golang.org/x/sync/errgroup"

	"github.com/KyberNetwork/router-service/internal/pkg/api"
	"github.com/KyberNetwork/router-service/internal/pkg/bootstrap"
	"github.com/KyberNetwork/router-service/internal/pkg/config"
	"github.com/KyberNetwork/router-service/internal/pkg/consumer"
	"github.com/KyberNetwork/router-service/internal/pkg/job"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/reloadconfig"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/blackjack"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/erc20balanceslot"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/executorbalance"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/gas"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/l2fee"
	onchainprice "github.com/KyberNetwork/router-service/internal/pkg/repository/onchain-price"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/pool"
	poolservice "github.com/KyberNetwork/router-service/internal/pkg/repository/pool-service"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/poolrank"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/route"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/setting"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/token"
	"github.com/KyberNetwork/router-service/internal/pkg/server"
	httppkg "github.com/KyberNetwork/router-service/internal/pkg/server/http"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase"
	aevmclientuc "github.com/KyberNetwork/router-service/internal/pkg/usecase/aevmclient"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/buildroute"
	erc20balanceslotuc "github.com/KyberNetwork/router-service/internal/pkg/usecase/erc20balanceslot"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/getcustomroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/getpools"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/getroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/indexpools"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/poolfactory"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/poolmanager"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/poolpublisher"
	trackexecutor "github.com/KyberNetwork/router-service/internal/pkg/usecase/trackexecutorbalance"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/validateroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/validateroute/synthetix"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/envvar"
	timeutil "github.com/KyberNetwork/router-service/internal/pkg/utils/time"
	"github.com/KyberNetwork/router-service/internal/pkg/validator"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	cryptopkg "github.com/KyberNetwork/router-service/pkg/crypto"
	"github.com/KyberNetwork/router-service/pkg/crypto/keystorage"
	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/KyberNetwork/router-service/pkg/redis"
	"github.com/KyberNetwork/router-service/pkg/util/env"
)

type IGetRouteUseCase interface {
	ApplyConfig(config getroute.Config)
}

type IPoolFactory interface {
	ApplyConfig(config poolfactory.Config)
}

type IPoolManager interface {
	ApplyConfig(config poolmanager.Config)
}

type IBuildRouteUseCase interface {
	ApplyConfig(config buildroute.Config)
}

// TODO: refactor main file -> separate to many folders with per folder is application. The main file should contains call root action per application.
func main() {

	if env.StringFromEnv(envvar.PYROSCOPEEnabled, "") != "" {
		pyroscopeServer := env.StringFromEnv(envvar.PYROSCOPEHost, "")
		if pyroscopeServer == "" {
			log.Fatal("pyroscope server is not set")
			return
		}
		pyroscope.Start(pyroscope.Config{
			ApplicationName: env.StringFromEnv(envvar.OTELService, "router-service-unknown"),

			// replace this with the address of pyroscope server
			ServerAddress: pyroscopeServer,

			// you can disable logging by setting this to nil
			Logger: pyroscope.StandardLogger,

			// you can provide static tags via a map:
			Tags: map[string]string{
				"hostname": env.StringFromEnv(envvar.OTELService, "router-service-unknown"),
				"env":      env.StringFromEnv(envvar.OTELEnv, ""),
				"version":  env.StringFromEnv(envvar.OTELServiceVersion, ""),
			},

			ProfileTypes: []pyroscope.ProfileType{
				// these profile types are enabled by default:
				pyroscope.ProfileCPU,
				pyroscope.ProfileAllocObjects,
				pyroscope.ProfileAllocSpace,
				pyroscope.ProfileInuseObjects,
				pyroscope.ProfileInuseSpace,

				// these profile types are optional:
				// pyroscope.ProfileGoroutines,
				// pyroscope.ProfileMutexCount,
				// pyroscope.ProfileMutexDuration,
				// pyroscope.ProfileBlockCount,
				// pyroscope.ProfileBlockDuration,
			},
		})
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
			{
				Name:   "executortracker",
				Usage:  "Track executor's tokens & pool approval, to support optimize building route",
				Action: executorTrackerAction,
			},
			{
				Name:    "liquidityScoresIndexer",
				Aliases: []string{"liqIndexer"},
				Usage:   "Index pools by liquidity scores",
				Action:  liquidityScoreIndexerAction,
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

	tokenGroupConfigPath := env.StringFromEnv(envvar.TokenGroupConfigPath, "")
	correlatedPairsConfigPath := env.StringFromEnv(envvar.CorrelatedPairsConfigPath, "")

	configLoader, err := config.NewConfigLoader(configFile, []string{tokenGroupConfigPath, correlatedPairsConfigPath})
	if err != nil {
		return err
	}

	cfg, err := configLoader.Get()
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
		logger.Warnf(ctx, "[apiAction] Config could not be reloaded: %s", err)
	} else {
		logger.Info(ctx, "Config reloaded")
	}

	if err := cfg.Validate(); err != nil {
		logger.Errorf(ctx, "failed to validate config, err: %v", err)
		panic(err)
	}

	if err := sentry.Init(sentry.ClientOptions{
		Dsn: cfg.Log.SentryDSN,
	}); err != nil {
		logger.Errorf(ctx, "sentry.Init error cause by %v", err)
		return err
	}

	// Flush buffered events before the program terminates.
	defer sentry.Flush(2 * time.Second)
	defer metrics.Flush()

	routerRedisClient, err := redis.New(&cfg.Redis)
	if err != nil {
		return err
	}

	poolRedisClient, err := redis.New(&cfg.PoolRedis)
	if err != nil {
		logger.Errorf(ctx, "fail to init redis client to pool service")
		return err
	}

	ethClient := ethrpc.New(cfg.Common.RPC)

	// init repositories
	poolRankRepository := poolrank.NewRedisRepository(routerRedisClient.Client, cfg.Repository.PoolRank)
	routeRedisRepository := route.NewRedisRepository(routerRedisClient.Client, cfg.Repository.Route.Redis)
	routeRepository, err := route.NewRistrettoRepository(routeRedisRepository, cfg.Repository.Route.RistrettoConfig)
	gasRepository, err := gas.NewRistrettoRepository(
		gas.NewRedisRepository(routerRedisClient.Client, ethClient, cfg.Repository.Gas.Redis),
		cfg.Repository.Gas.Ristretto)

	l1FeeParamsRepository := l2fee.NewRedisRepository(routerRedisClient.Client,
		l2fee.RedisL1FeeRepositoryConfig{Prefix: cfg.Redis.Prefix})
	l1FeeCalculator := usecase.NewL1FeeCalculator(l1FeeParamsRepository, common.HexToAddress(cfg.Encoder.RouterAddress))

	tokenRepository := token.NewGoCacheRepository(
		token.NewRedisRepository(poolRedisClient.Client, cfg.Repository.Token.Redis),
		cfg.Repository.Token.GoCache,
	)

	var onchainpriceRepository getroute.IOnchainPriceRepository
	grpcRepository, err := onchainprice.NewGRPCRepository(
		cfg.Repository.OnchainPrice.Grpc,
		cfg.Common.ChainID,
		tokenRepository,
		cfg.Common.GasTokenAddress)
	if err != nil {
		return err
	}

	onchainpriceRepository, err = onchainprice.NewRistrettoRepository(grpcRepository,
		cfg.Repository.OnchainPrice.Ristretto)
	if err != nil {
		return err
	}

	go onchainpriceRepository.RefreshCacheNativePriceInUSD(ctx)

	poolServiceClient, err := poolservice.NewGRPCClient(cfg.Repository.PoolService)
	poolRepository, err := pool.NewRedisRepository(poolRedisClient.Client, poolServiceClient, cfg.Repository.Pool)

	blackjackRepo, err := blackjack.NewGRPCClient(cfg.Repository.Blackjack.GRPCClient)
	if err != nil {
		return err
	}

	executorBalanceRepository := executorbalance.NewRedisRepository(
		routerRedisClient.Client,
		executorbalance.Config{
			Prefix: cfg.Redis.Prefix,
		},
	)
	// sealer

	// init validators
	slippageValidator := validator.NewSlippageValidator(cfg.Validator.SlippageValidatorConfig)
	getPoolsParamsValidator := validator.NewGetPoolsParamsValidator()
	getTokensParamsValidator := validator.NewGetTokensParamsValidator()
	getRoutesParamsValidator := validator.NewGetRouteParamsValidator()
	getRouteEncodeParamsValidator := validator.NewGetRouteEncodeParamsValidator(timeutil.NowFunc,
		cfg.Validator.GetRouteEncodeParams, blackjackRepo, slippageValidator)
	buildRouteParamsValidator := validator.NewBuildRouteParamsValidator(timeutil.NowFunc,
		cfg.Validator.BuildRouteParams, blackjackRepo, slippageValidator)

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
	l1Encoder := l1encode.NewEncoder(l1encode.Config{
		RouterAddress:               cfg.Encoder.RouterAddress,
		ExecutorAddress:             cfg.Encoder.ExecutorAddress,
		ChainID:                     cfg.Encoder.ChainID,
		IsPositiveSlippageEnabled:   cfg.Encoder.IsPositiveSlippageEnabled,
		MinimumPSThreshold:          cfg.Encoder.MinimumPSThreshold,
		ExecutorAddressByClientID:   cfg.Encoder.ExecutorAddressByClientID,
		PartnerPositiveSlippageInfo: cfg.Encoder.PartnerPositiveSlippageInfo,
	})
	l2Encoder := l2encode.NewEncoder(l2encode.Config{
		RouterAddress:               cfg.Encoder.RouterAddress,
		ExecutorAddress:             cfg.Encoder.L2ExecutorAddress,
		ChainID:                     cfg.Encoder.ChainID,
		IsPositiveSlippageEnabled:   cfg.Encoder.IsPositiveSlippageEnabled,
		MinimumPSThreshold:          cfg.Encoder.MinimumPSThreshold,
		FunctionSelectorMappingID:   cfg.Encoder.FunctionSelectorMappingID,
		ExecutorAddressByClientID:   cfg.Encoder.ExecutorAddressByClientID,
		PartnerPositiveSlippageInfo: cfg.Encoder.PartnerPositiveSlippageInfo,
	})
	encodeBuilder := encode.NewEncodeBuilder(l1Encoder, l2Encoder)

	validateRouteUseCase := validateroute.NewValidateRouteUseCase()
	validateRouteUseCase.RegisterValidator(synthetix.NewSynthetixValidator())

	getPoolsUseCase := getpools.NewGetPoolsUseCase(poolRepository)
	GetPoolsIncludingBasePools := getpools.NewGetPoolsIncludingBasePools(poolRepository)
	getTokensUseCase := usecase.NewGetTokens(tokenRepository, onchainpriceRepository)

	var (
		balanceSlotsUseCase erc20balanceslotuc.ICache
		aevmClient          aevmclientuc.IAEVMClientUseCase
		poolsPublisher      poolmanager.IPoolsPublisher
	)
	if cfg.AEVMEnabled {
		balanceSlotsUseCase, aevmClient, poolsPublisher, err = initializeAEVMComponents(ctx, cfg, routerRedisClient)
		if err != nil {
			return fmt.Errorf("could not initilize AEVM components, perhaps AEVM is not confitured: %w", err)
		}
		defer aevmClient.Close()
	}

	poolFactory := poolfactory.NewPoolFactory(cfg.UseCase.PoolFactory, aevmClient, balanceSlotsUseCase)
	poolManager, err := poolmanager.NewPointerSwapPoolManager(ctx, poolRepository, poolFactory, poolRankRepository,
		GetPoolsIncludingBasePools, cfg.UseCase.PoolManager, aevmClient, poolsPublisher, balanceSlotsUseCase)
	if err != nil {
		return err
	}

	pathFinder, routeFinalizer, err := getroute.InitializeFinderEngine(cfg.UseCase.GetRoute, aevmClient)
	if err != nil {
		return err
	}

	finderEngine := finderengine.NewPathFinderEngine(pathFinder, routeFinalizer)

	customRouteConfig := getcustomroute.ReplaceAggregatorConfig(cfg.UseCase.GetRoute, cfg.UseCase.GetCustomRoute)
	customRoutePathFinder, customRouteRouteFinalizer, err := getroute.InitializeFinderEngine(customRouteConfig,
		aevmClient)
	if err != nil {
		return err
	}

	customRouteFinderEngine := finderengine.NewPathFinderEngine(customRoutePathFinder, customRouteRouteFinalizer)

	getRouteUseCase := getroute.NewUseCase(
		poolRankRepository,
		tokenRepository,
		onchainpriceRepository,
		routeRepository,
		gasRepository,
		poolManager,
		finderEngine,
		cfg.UseCase.GetRoute,
	)
	getBundledRouteUseCase := getroute.NewBundledUseCase(
		poolRankRepository,
		tokenRepository,
		onchainpriceRepository,
		routeRepository,
		gasRepository,
		poolManager,
		poolFactory,
		finderEngine,
		cfg.UseCase.GetRoute,
	)

	rfqHandlerByPoolType := make(map[string]poolpkg.IPoolRFQ)
	for _, s := range cfg.UseCase.BuildRoute.RFQ {
		rfqHandler, err := bootstrap.NewRFQHandler(s, cfg.Common)
		if err != nil {
			return fmt.Errorf("can not create RFQ handler: %v, err: %v", s.Handler, err)
		}

		rfqHandlerByPoolType[s.Handler] = rfqHandler
	}

	gasEstimator := buildroute.NewGasEstimator(ethClient, gasRepository, onchainpriceRepository,
		cfg.Common.GasTokenAddress,
		cfg.Common.RouterAddress)
	buildRouteUseCase := buildroute.NewBuildRouteUseCase(
		tokenRepository,
		poolRepository,
		executorBalanceRepository,
		onchainpriceRepository,
		gasEstimator,
		l1FeeCalculator,
		rfqHandlerByPoolType,
		clientDataEncoder,
		encodeBuilder,
		cfg.UseCase.BuildRoute,
	)

	getCustomRoutesUseCase := getcustomroute.NewCustomRoutesUseCase(
		poolFactory,
		tokenRepository,
		onchainpriceRepository,
		gasRepository,
		poolRepository,
		customRouteFinderEngine,
		getcustomroute.Config{
			ChainID:           customRouteConfig.ChainID,
			RouterAddress:     customRouteConfig.RouterAddress,
			GasTokenAddress:   customRouteConfig.GasTokenAddress,
			AvailableSources:  customRouteConfig.AvailableSources,
			UnscalableSources: customRouteConfig.UnscalableSources,
			Aggregator:        customRouteConfig.Aggregator,
		},
	)
	l1Decoder := &decode.Decoder{}
	l2Decoder := decode.NewL2Decoder(decode.L2DecoderConfig{
		FunctionSelectorMappingID: cfg.Encoder.FunctionSelectorMappingID,
	})

	removePoolIndex := usecase.NewRemovePoolIndexUseCase(poolRankRepository)

	// init services
	ginServer, router, _ := httppkg.GinServer(cfg.Http, cfg.Log.Configuration, logger.LoggerBackendZap)

	// Only profiling in dev
	if cfg.Pprof {
		pprof.Register(ginServer)
	}

	router.GET(
		"/route/encode",
		api.GetRouteEncode(
			getRouteEncodeParamsValidator,
			getRouteUseCase,
			buildRouteUseCase,
			getTokensUseCase,
			timeutil.NowFunc,
		),
	)

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
	v1Debug.POST("/decode", api.DecodeSwapData(l1Decoder, l2Decoder))
	v1Debug.DELETE("/pool-index", api.RemovePoolsFromIndex(removePoolIndex))

	v1.GET("/pools", api.GetPools(getPoolsParamsValidator, getPoolsUseCase))
	v1.GET("/tokens", api.GetTokens(getTokensParamsValidator, getTokensUseCase))
	v1.GET("/routes", api.GetRoutes(getRoutesParamsValidator, getRouteUseCase))
	v1.POST("/route/build", api.BuildRoute(buildRouteParamsValidator, buildRouteUseCase, timeutil.NowFunc))

	if cfg.BundledRouteEnabled {
		getBundledRoutesHandler := api.GetBundledRoutes(getRoutesParamsValidator, getBundledRouteUseCase)
		v1.GET("/bundled-routes", getBundledRoutesHandler)
		v1.POST("/bundled-routes", getBundledRoutesHandler)
	}

	v1.GET("/keys/publics/:keyId", api.GetPublicKey(keyPairUseCase))

	reloadManager := reload.NewManager()

	// Run hot-reload manager.
	// Add all app reloaders in order.
	reloadManager.RegisterReloader(0, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		// If configuration fails ignore reload with a warning.
		err = configLoader.Reload(ctx)
		if err != nil {
			logger.Warnf(ctx, "[apiAction] Config could not be reloaded: %s", err)
			return nil
		}

		return nil
	}))

	reloadManager.RegisterReloader(100, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		return applyLatestConfigForAPI(
			ctx,
			configLoader,
			getRouteUseCase,
			getBundledRouteUseCase,
			buildRouteUseCase,
			poolFactory,
			poolManager,
			buildRouteParamsValidator,
			getRouteEncodeParamsValidator,
			aevmClient,
			finderEngine,
		)
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
	tokenGroupConfigPath := env.StringFromEnv(envvar.TokenGroupConfigPath, "")
	correlatedPairsConfigPath := env.StringFromEnv(envvar.CorrelatedPairsConfigPath, "")

	configLoader, err := config.NewConfigLoader(configFile, []string{tokenGroupConfigPath, correlatedPairsConfigPath})
	if err != nil {
		return err
	}
	cfg, err := configLoader.Get()
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
		logger.Warnf(ctx, "[indexerAction] Config could not be reloaded: %s", err)
	} else {
		logger.Infoln(ctx, "[indexerAction] Config reloaded")
	}

	ethClient := ethrpc.New(cfg.Common.RPC)
	ethClient.SetMulticallContract(common.HexToAddress(cfg.Common.MulticallAddress))

	// init redis client
	routerRedisClient, err := redis.New(&cfg.Redis)
	if err != nil {
		logger.Errorf(ctx, "fail to init redis client for indexer")
		return err
	}

	poolRedisClient, err := redis.New(&cfg.PoolRedis)
	if err != nil {
		logger.Errorf(ctx, "[indexerAction] fail to init redis client to pool service")
		return err
	}

	poolEventRedisClient, err := redis.New(&cfg.PoolEventRedis)
	if err != nil {
		logger.Errorf(ctx, "[indexerAction] fail to init redis client to pool service")
		return err
	}

	poolServiceClient, err := poolservice.NewGRPCClient(cfg.Repository.PoolService)
	// init repository
	poolRepo, _ := pool.NewRedisRepository(poolRedisClient.Client, poolServiceClient, cfg.Repository.Pool)
	poolRankRepository := poolrank.NewRedisRepository(routerRedisClient.Client, cfg.Repository.PoolRank)
	gasRepository := gas.NewRedisRepository(routerRedisClient.Client, ethClient,
		gas.RedisRepositoryConfig{Prefix: cfg.Redis.Prefix})

	tokenRepository := token.NewGoCacheRepository(
		token.NewRedisRepository(poolRedisClient.Client, cfg.Repository.Token.Redis),
		cfg.Repository.Token.GoCache,
	)

	var onchainpriceRepository getroute.IOnchainPriceRepository
	grpcRepository, err := onchainprice.NewGRPCRepository(
		cfg.Repository.OnchainPrice.Grpc,
		cfg.Common.ChainID,
		tokenRepository,
		cfg.Common.GasTokenAddress)
	if err != nil {
		return err
	}

	onchainpriceRepository, err = onchainprice.NewRistrettoRepository(grpcRepository,
		cfg.Repository.OnchainPrice.Ristretto)
	if err != nil {
		return err
	}

	go onchainpriceRepository.RefreshCacheNativePriceInUSD(ctx)

	// init use case
	getAllPoolAddressesUseCase := usecase.NewGetAllPoolAddressesUseCase(poolRepo)
	indexPoolsUseCase := indexpools.NewIndexPoolsUseCase(
		poolRepo,
		poolRankRepository,
		onchainpriceRepository,
		cfg.UseCase.IndexPools,
	)
	poolEventStreamConsumer := consumer.NewPoolEventsStreamConsumer(poolEventRedisClient.Client,
		&cfg.Job.IndexPools.PoolEvent.ConsumerConfig)

	indexPoolsJob := job.NewIndexPoolsJob(
		getAllPoolAddressesUseCase,
		indexPoolsUseCase,
		poolEventStreamConsumer,
		cfg.Job.IndexPools,
	)

	updateSuggestedGasPriceUseCase := usecase.NewUpdateSuggestedGasPrice(gasRepository)
	updateSuggestedGasPriceJob := job.NewUpdateSuggestedGasPriceJob(
		updateSuggestedGasPriceUseCase,
		cfg.Job.UpdateSuggestedGasPrice,
	)

	var updateL1FeeJob *job.UpdateL1FeeJob
	if cfg.Job.UpdateL1Fee.Interval > 0 {
		l1FeeParamsRepository := l2fee.NewRedisRepository(routerRedisClient.Client,
			l2fee.RedisL1FeeRepositoryConfig{Prefix: cfg.Redis.Prefix})

		updateL1FeeUseCase := usecase.NewUpdateL1FeeParams(
			cfg.Common.ChainID,
			ethClient,
			cfg.Job.UpdateL1Fee.OracleAddress,
			l1FeeParamsRepository,
		)
		updateL1FeeJob = job.NewUpdateL1FeeJob(
			updateL1FeeUseCase,
			cfg.Job.UpdateL1Fee.Interval,
		)
	}

	reloadManager := reload.NewManager()

	// Run hot-reload manager.
	// Add all app reloaders in order.
	reloadManager.RegisterReloader(0, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		// If configuration fails ignore reload with a warning.
		err = configLoader.Reload(ctx)
		if err != nil {
			logger.Warnf(ctx, "[indexerAction] Config could not be reloaded: %s", err)
			return nil
		}
		return nil
	}))

	reloadManager.RegisterReloader(100, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		return applyLatestConfigForIndexer(ctx, indexPoolsUseCase, indexPoolsJob, configLoader)
	}))

	g, ctx := errgroup.WithContext(ctx)

	// run jobs
	g.Go(func() error {
		return reloadManager.Run(ctx)
	})

	g.Go(func() error {
		indexPoolsJob.Run(ctx)

		return nil
	})

	g.Go(func() error {
		updateSuggestedGasPriceJob.Run(ctx)

		return nil
	})

	if updateL1FeeJob != nil {
		g.Go(func() error {
			updateL1FeeJob.Run(ctx)

			return nil
		})
	}

	// Register notifier
	reloadChan := make(chan string)
	reloadManager.RegisterNotifier(reload.NotifierChan(reloadChan))

	g.Go(func() error {
		reloadConfigReporter.Report(ctx, reloadChan)
		return nil
	})

	return g.Wait()
}

func executorTrackerAction(c *cli.Context) (err error) {
	ctx := context.Background()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// load config
	configFile := c.String("config")
	tokenGroupConfigPath := env.StringFromEnv(envvar.TokenGroupConfigPath, "")
	correlatedPairsConfigPath := env.StringFromEnv(envvar.CorrelatedPairsConfigPath, "")

	configLoader, err := config.NewConfigLoader(configFile, []string{tokenGroupConfigPath, correlatedPairsConfigPath})
	if err != nil {
		return err
	}
	cfg, err := configLoader.Get()
	if err != nil {
		return err
	}

	ethClient := ethrpc.New(cfg.Common.RPC)
	ethClient.SetMulticallContract(common.HexToAddress(cfg.Common.MulticallAddress))

	// init redis client
	routerRedisClient, err := redis.New(&cfg.Redis)
	if err != nil {
		logger.Errorf(ctx, "[executorTrackerAction] fail to init redis client for track executor balance")
		return err
	}
	poolRedisClient, err := redis.New(&cfg.PoolRedis)
	if err != nil {
		logger.Errorf(ctx, "[executorTrackerAction] fail to init redis client to pool service")
		return err
	}

	// init repository
	poolFactory := poolfactory.NewPoolFactory(cfg.UseCase.PoolFactory, nil, nil)
	executorBalanceRepository := executorbalance.NewRedisRepository(
		routerRedisClient.Client,
		executorbalance.Config{
			Prefix: cfg.Redis.Prefix,
		},
	)
	poolServiceClient, err := poolservice.NewGRPCClient(cfg.Repository.PoolService)
	poolRepository, err := pool.NewRedisRepository(poolRedisClient.Client, poolServiceClient, cfg.Repository.Pool)

	// init usecase
	var trackExecutorAddresses []string

	// Only track either L1 or L2 address
	if valueobject.IsL2EncoderSupportedChains(cfg.Common.ChainID) {
		trackExecutorAddresses = []string{cfg.Encoder.L2ExecutorAddress}
	} else {
		trackExecutorAddresses = []string{cfg.Encoder.ExecutorAddress}
	}
	trackExecutorBalanceUseCase := trackexecutor.NewUseCase(
		ethClient,
		poolFactory,
		poolRepository,
		executorBalanceRepository,
		trackexecutor.Config{
			SubgraphURL:       cfg.UseCase.TrackExecutor.SubgraphURL,
			GasTokenAddress:   cfg.Common.GasTokenAddress,
			ExecutorAddresses: trackExecutorAddresses,
		},
	)

	// init job
	trackExecutorBalanceJob := job.NewExecutorBalanceFetcherJob(
		trackExecutorBalanceUseCase,
		job.TrackExecutorBalanceConfig{
			Interval: cfg.Job.TrackExecutorBalance.Interval,
		},
	)

	return trackExecutorBalanceJob.Run(ctx)
}

func getKeyStorage(storageFilePath string) (cryptopkg.KeyPairStorage, error) {
	keyStorage, err := keystorage.NewInMemoryStorageFromFile(storageFilePath)
	if err != nil {
		// handle fallback when we have not setup file key pair store
		return keystorage.NewInMemoryStorage(nil), nil
	}
	return keyStorage, err
}

func applyLatestConfigForAPI(
	ctx context.Context,
	configLoader *config.ConfigLoader,
	getRouteUseCase IGetRouteUseCase,
	getBundledRouteUseCase IGetRouteUseCase,
	buildRouteUseCase IBuildRouteUseCase,
	poolFactory IPoolFactory,
	poolManager IPoolManager,
	buildRouteParamsValidator api.IBuildRouteParamsValidator,
	getRouteEncodeParamsValidator api.IGetRouteEncodeParamsValidator,
	aevmClientUC aevmclientuc.IAEVMClientUseCase,
	finderEngine finderengine.IPathFinderEngine,
) error {
	cfg, err := configLoader.Get()
	if err != nil {
		return err
	}

	getRouteUseCase.ApplyConfig(cfg.UseCase.GetRoute)
	getBundledRouteUseCase.ApplyConfig(cfg.UseCase.GetRoute)
	buildRouteUseCase.ApplyConfig(cfg.UseCase.BuildRoute)
	poolFactory.ApplyConfig(cfg.UseCase.PoolFactory)
	poolManager.ApplyConfig(cfg.UseCase.PoolManager)
	buildRouteParamsValidator.ApplyConfig(cfg.Validator.BuildRouteParams)
	getRouteEncodeParamsValidator.ApplyConfig(cfg.Validator.GetRouteEncodeParams)
	if aevmClientUC != nil {
		serverURLs := strings.Split(cfg.AEVM.AEVMServerURLs, ",")
		aevmClientUC.ApplyConfig(aevmclientuc.Config{ServerURLs: serverURLs})
	}

	// Reload FinderEngine with new config
	pathFinder, routeFinalizer, err := getroute.InitializeFinderEngine(cfg.UseCase.GetRoute, aevmClientUC)
	if err != nil {
		return err
	}

	finderEngine.SetFinder(pathFinder)
	finderEngine.SetFinalizer(routeFinalizer)

	return nil
}

func applyLatestConfigForIndexer(
	ctx context.Context,
	indexPoolsUseCase *indexpools.IndexPoolsUseCase,
	indexPoolsJob *job.IndexPoolsJob,
	configLoader *config.ConfigLoader,
) error {
	cfg, err := configLoader.Get()
	if err != nil {
		return err
	}

	indexPoolsJob.ApplyConfig(cfg.Job.IndexPools)

	indexPoolsUseCase.ApplyConfig(cfg.UseCase.IndexPools)

	return nil
}

func applyLatestConfigForLiquidityScoreIndexer(
	ctx context.Context,
	tradesGenerator *indexpools.TradeDataGenerator,
	configLoader *config.ConfigLoader,
) error {
	cfg, err := configLoader.Get()
	if err != nil {
		return err
	}

	tradesGenerator.ApplyConfig(cfg.UseCase.TradeDataGenerator)

	return nil
}

func liquidityScoreIndexerAction(c *cli.Context) (err error) {
	ctx := context.Background()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// load config
	configFile := c.String("config")

	configLoader, err := config.NewConfigLoader(configFile, []string{})
	if err != nil {
		return err
	}
	cfg, err := configLoader.Get()
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
		logger.Warnf(ctx, "[apiAction] Config could not be reloaded: %s", err)
	} else {
		logger.Info(ctx, "Config reloaded")
	}

	if err := cfg.Validate(); err != nil {
		logger.Errorf(ctx, "failed to validate config, err: %v", err)
		panic(err)
	}

	// init redis client
	poolRedisClient, err := redis.New(&cfg.PoolRedis)
	if err != nil {
		logger.Errorf(ctx, "[indexerAction] fail to init redis client to pool service")
		return err
	}
	routerRedisClient, err := redis.New(&cfg.Redis)
	if err != nil {
		return err
	}

	poolServiceClient, err := poolservice.NewGRPCClient(cfg.Repository.PoolService)
	poolRepository, err := pool.NewRedisRepository(
		poolRedisClient.Client,
		poolServiceClient, cfg.Repository.Pool)
	poolRankRepo := poolrank.NewRedisRepository(poolRedisClient.Client, cfg.Repository.PoolRank)

	tokenRepository := token.NewGoCacheRepository(
		token.NewRedisRepository(poolRedisClient.Client, cfg.Repository.Token.Redis),
		cfg.Repository.Token.GoCache,
	)

	var (
		balanceSlotsUseCase erc20balanceslotuc.ICache
		aevmClient          aevmclientuc.IAEVMClientUseCase
	)
	if cfg.AEVMEnabled {
		balanceSlotsUseCase, aevmClient, _, err = initializeAEVMComponents(ctx, cfg, routerRedisClient)
		if err != nil {
			return fmt.Errorf("could not initilize AEVM components, perhaps AEVM is not confitured: %w", err)
		}
		defer aevmClient.Close()
	}

	var onchainpriceRepository indexpools.IOnchainPriceRepository
	grpcRepository, err := onchainprice.NewGRPCRepository(
		cfg.Repository.OnchainPrice.Grpc,
		cfg.Common.ChainID,
		tokenRepository,
		cfg.Common.GasTokenAddress)
	if err != nil {
		return err
	}

	onchainpriceRepository, err = onchainprice.NewRistrettoRepository(grpcRepository, cfg.Repository.OnchainPrice.Ristretto)
	if err != nil {
		return err
	}
	go onchainpriceRepository.RefreshCacheNativePriceInUSD(ctx)

	getPools := getpools.NewGetPoolsIncludingBasePools(poolRepository)
	poolFactory := poolfactory.NewPoolFactory(cfg.UseCase.PoolFactory, aevmClient, balanceSlotsUseCase)
	tradeGenerator := indexpools.NewTradeDataGenerator(poolRepository, onchainpriceRepository, tokenRepository, getPools, aevmClient, poolFactory, cfg.UseCase.TradeDataGenerator)
	updatePoolScores := indexpools.NewUpdatePoolsScore(poolRankRepo, cfg.UseCase.UpdateLiquidityScoreConfig)
	indexJob := job.NewLiquidityScoreIndexPoolsJob(tradeGenerator, updatePoolScores, cfg.Job.LiquidityScoreIndexPools)

	reloadManager := reload.NewManager()

	// Run hot-reload manager.
	// Add all app reloaders in order.
	reloadManager.RegisterReloader(0, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		// If configuration fails ignore reload with a warning.
		err = configLoader.Reload(ctx)
		if err != nil {
			logger.Warnf(ctx, "[indexerAction] Config could not be reloaded: %s", err)
			return nil
		}
		return nil
	}))

	reloadManager.RegisterReloader(100, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		return applyLatestConfigForLiquidityScoreIndexer(ctx, tradeGenerator, configLoader)
	}))

	g, ctx := errgroup.WithContext(ctx)

	// run jobs
	g.Go(func() error {
		return reloadManager.Run(ctx)
	})

	// run jobs
	g.Go(func() error {
		indexJob.Run(ctx)

		return nil
	})

	// Register notifier
	reloadChan := make(chan string)
	reloadManager.RegisterNotifier(reload.NotifierChan(reloadChan))

	g.Go(func() error {
		reloadConfigReporter.Report(ctx, reloadChan)
		return nil
	})

	return g.Wait()
}

func initializeAEVMComponents(ctx context.Context, cfg *config.Config, routerRedisClient *redis.Redis) (erc20balanceslotuc.ICache, aevmclientuc.IAEVMClientUseCase, poolmanager.IPoolsPublisher, error) {
	balanceSlotsRepo := erc20balanceslot.NewRedisRepository(routerRedisClient.Client,
		cfg.Repository.ERC20BalanceSlot.Redis)
	rpcClient, err := rpc.Dial(cfg.AEVM.RPC)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("could not dial JSON-RPC node %w", err)
	}
	var balanceSlotsProbe *erc20balanceslotuc.MultipleStrategy
	if cfg.AEVM.UseHoldersListAsFallback {
		tokenHoldersRedis, err := redis.New(&cfg.AEVM.TokenHoldersRedis)
		if err != nil {
			return nil, nil, nil, err
		}
		holdersListRepo := erc20balanceslot.NewHoldersListRedisRepositoryWithCache(tokenHoldersRedis,
			cfg.AEVM.CachedHoldersListTTLSec)
		watchlistRepo := erc20balanceslot.NewWatchlistRedisRepository(tokenHoldersRedis)
		balanceSlotsProbe = erc20balanceslotuc.NewMultipleStrategyWithHoldersListAsFallback(rpcClient,
			common.HexToAddress(cfg.AEVM.SimulationWallet), holdersListRepo, watchlistRepo)
	} else {
		balanceSlotsProbe = erc20balanceslotuc.NewMultipleStrategy(rpcClient,
			common.HexToAddress(cfg.AEVM.SimulationWallet))
	}
	balanceSlotsUseCase := erc20balanceslotuc.NewCache(balanceSlotsRepo, balanceSlotsProbe,
		cfg.AEVM.PredefinedBalanceSlots, cfg.Common.ChainID)
	if err := balanceSlotsUseCase.PreloadFromEmbedded(context.Background()); err != nil {
		logger.Errorf(ctx, "could not preload balance slots %s", err)
		return nil, nil, nil, err
	}

	serverURLs := strings.Split(cfg.AEVM.AEVMServerURLs, ",")
	publishingPoolsURLs := strings.Split(cfg.AEVM.AEVMPublishingPoolsURLs, ",")
	logger.Infof(ctx, "AEVMServerURLs = %+v AEVMPublishingPoolsURLs = %+v", serverURLs, publishingPoolsURLs)
	aevmClient, err := aevmclientuc.NewClient(
		aevmclientuc.Config{
			ServerURLs:          serverURLs,
			PublishingPoolsURLs: publishingPoolsURLs,

			RetryOnTimeoutMs:          cfg.AEVM.RetryOnTimeoutMs,
			FindrouteRetryOnTimeoutMs: cfg.AEVM.FindrouteRetryOnTimeoutMs,
			MaxRetry:                  cfg.AEVM.MaxRetry,
		},
		func(url string) (aevmclient.Client, error) { return aevmclient.NewGRPCClient(url) },
	)
	if err != nil {
		return nil, nil, nil, err
	}

	poolsPublisher, err := poolpublisher.NewPoolPublisher(aevmClient, poolmanager.NState)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("could not NewPoolPublisher: %w", err)
	}

	return balanceSlotsUseCase, aevmClient, poolsPublisher, nil
}
