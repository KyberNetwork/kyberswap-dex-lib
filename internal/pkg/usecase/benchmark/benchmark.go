package benchmark

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/KyberNetwork/router-service/internal/pkg/config"
	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/core"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/repository"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/gas"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase"
	usecasecore "github.com/KyberNetwork/router-service/internal/pkg/usecase/core"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/factory"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/KyberNetwork/router-service/pkg/redis"
)

type benchmarkUseCase struct {
	poolFactory     *factory.PoolFactory
	poolRepository  usecase.IPoolRepository
	tokenRepository usecase.ITokenRepository
	priceRepository usecase.IPriceRepository
	routeRepository usecase.IRouteRepository
	gasRepository   usecase.IGasRepository

	logger logger.Logger

	config usecase.BenchmarkConfig
}

func newMockBenchmarkUseCase(configFile string) (*benchmarkUseCase, error) {
	configLoader, err := config.NewConfigLoader(configFile)
	if err != nil {
		return nil, err
	}

	cfg, err := configLoader.Get()
	if err != nil {
		return nil, err
	}

	lg, err := logger.InitLogger(cfg.Log.Configuration, logger.LoggerBackendZap)
	if err != nil {
		return nil, err
	}

	// rDb to read-only secondary Redis read-replica.
	rDb, err := redis.New(&cfg.Redis)
	if err != nil {
		return nil, err
	}

	ethClient := ethrpc.New(cfg.Common.RPC)

	poolRepo := repository.NewPoolDataStoreRedisRepository(rDb)
	tokenDataStoreRepo := repository.NewTokenDataStoreRedisRepository(rDb)
	priceDataStoreRepo := repository.NewPriceDataStoreRedisRepository(rDb)
	routeRepo := repository.NewRouteRedisRepository(rDb)
	gasRepo := gas.NewRedisRepository(rDb.Client, ethClient, cfg.Repository.Gas.Redis)
	poolFactoryConfig := factory.PoolFactoryConfig{ChainID: cfg.Common.ChainID}
	poolFactory := factory.NewPoolFactory(poolFactoryConfig)

	c := usecase.BenchmarkConfig{
		GetRoutesConfig:            cfg.UseCase.GetRoutes,
		WhitelistedTokensByAddress: cfg.WhitelistedTokensByAddress(),
	}

	return &benchmarkUseCase{
		poolFactory:     poolFactory,
		poolRepository:  poolRepo,
		tokenRepository: tokenDataStoreRepo,
		priceRepository: priceDataStoreRepo,
		routeRepository: routeRepo,
		gasRepository:   gasRepo,
		logger:          lg,
		config:          c,
	}, nil
}

func (uc *benchmarkUseCase) getSources(includedSources []string, excludedSources []string) []string {
	sources := make([]string, 0, len(uc.config.EnabledDexes))
	includedSourcesLen := len(includedSources)
	excludedSourcesLen := len(excludedSources)

	for _, source := range uc.config.EnabledDexes {
		if excludedSourcesLen > 0 && utils.StringContains(excludedSources, source) {
			continue
		}

		if includedSourcesLen > 0 && !utils.StringContains(includedSources, source) {
			continue
		}

		sources = append(sources, source)
	}

	return sources
}

// listPools prepares pool set for finding route
func (uc *benchmarkUseCase) listPools(
	ctx context.Context,
	tokenInAddress string,
	tokenOutAddress string,
	sources []string,
) ([]entity.Pool, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "benchmarkUseCase.listPools")
	defer span.Finish()

	directPairKey := usecasecore.GenDirectPairKey(tokenInAddress, tokenOutAddress)

	whitelistI := uc.config.WhitelistedTokensByAddress[tokenInAddress]
	whitelistJ := uc.config.WhitelistedTokensByAddress[tokenOutAddress]
	bestPools, err := uc.routeRepository.GetBestPools(ctx, directPairKey, tokenInAddress, tokenOutAddress, uc.config.GetBestPoolsOptions, whitelistI, whitelistJ)
	if err != nil {
		return nil, err
	}

	poolIDs := uc.filterBlacklistedPools(bestPools.PoolIds)

	pools, err := uc.poolRepository.FindByAddresses(ctx, poolIDs)
	if err != nil {
		return nil, err
	}

	filteredPools := filterPools(
		pools,
		PoolFilterSources(sources),
		PoolFilterHasReserveOrAmplifiedTvl,
	)

	curveMetaBasePools, err := uc.listCurveMetaBasePools(ctx, filteredPools)
	if err != nil {
		return nil, err
	}

	return append(filteredPools, curveMetaBasePools...), nil
}

// listCurveMetaBasePools collects base pools of curveMeta pools
// - collects already fetched curveBase and curvePainOracle pools
// - for each curveMeta pool
//   - decode its staticExtra to get its basePool address
//   - if it hasn't been fetched, fetch the pool data
func (uc *benchmarkUseCase) listCurveMetaBasePools(
	ctx context.Context,
	pools []entity.Pool,
) ([]entity.Pool, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "benchmarkUseCase.listCurveMetaBasePools")
	defer span.Finish()

	var (
		// alreadyFetchedSet contains fetched pool ids
		alreadyFetchedSet = map[string]bool{}

		// poolAddresses contains pool addresses to fetch
		poolAddresses []string
	)

	for _, pool := range pools {
		if pool.Type == constant.PoolTypes.CurveBase {
			alreadyFetchedSet[pool.Address] = true
		}

		if pool.Type == constant.PoolTypes.CurvePlainOracle {
			alreadyFetchedSet[pool.Address] = true
		}
	}

	for _, pool := range pools {
		if pool.Type != constant.PoolTypes.CurveMeta {
			continue
		}

		var staticExtra struct {
			BasePool string `json:"basePool"`
		}

		if err := json.Unmarshal([]byte(pool.StaticExtra), &staticExtra); err != nil {
			uc.logger.WithFields(logger.Fields{
				"pool.Address": pool.Address,
				"pool.Type":    pool.Type,
				"error":        err,
			}).Warn("unable to unmarshal staticExtra")

			continue
		}

		if _, ok := alreadyFetchedSet[staticExtra.BasePool]; ok {
			continue
		}

		poolAddresses = append(poolAddresses, staticExtra.BasePool)
	}

	return uc.poolRepository.FindByAddresses(ctx, poolAddresses)
}

// getTokenByAddress fetches token data and returns a map from token address to entity.Token
func (uc *benchmarkUseCase) getTokenByAddress(
	ctx context.Context,
	tokenAddresses []string,
) (map[string]entity.Token, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "benchmarkUseCase.getTokenByAddress")
	defer span.Finish()

	tokens, err := uc.tokenRepository.FindByAddresses(ctx, tokenAddresses)
	if err != nil {
		return nil, err
	}

	tokenByAddress := make(map[string]entity.Token, len(tokens))
	for _, token := range tokens {
		tokenByAddress[token.Address] = token
	}

	return tokenByAddress, nil
}

// getPriceByAddress fetch price data and return a map from token address to price in USD of the token
func (uc *benchmarkUseCase) getPriceByAddress(
	ctx context.Context,
	tokenAddresses []string,
) (map[string]entity.Price, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "benchmarkUseCase.getPriceByAddress")
	defer span.Finish()

	prices, err := uc.priceRepository.FindByAddresses(ctx, tokenAddresses)
	if err != nil {
		return nil, err
	}

	priceByAddress := make(map[string]entity.Price, len(prices))
	for _, price := range prices {
		priceByAddress[price.Address] = price
	}

	return priceByAddress, nil
}

// getGasPrice returns gas price, preferred custom gasPrice
func (uc *benchmarkUseCase) getGasPrice(
	ctx context.Context,
	queryGasPrice *big.Float,
) (*big.Float, error) {
	if queryGasPrice != nil {
		return queryGasPrice, nil
	}

	suggestedGasPrice, err := uc.gasRepository.GetSuggestedGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	return new(big.Float).SetInt(suggestedGasPrice), nil
}

func (uc *benchmarkUseCase) filterBlacklistedPools(poolIDs []string) []string {
	blacklistedPoolSet := make(map[string]bool, len(uc.config.BlacklistedPools))
	for _, blacklistedPool := range uc.config.BlacklistedPools {
		blacklistedPoolSet[blacklistedPool] = true
	}

	filtered := make([]string, 0, len(poolIDs))

	for _, poolID := range poolIDs {
		if blacklistedPoolSet[poolID] {
			continue
		}

		filtered = append(filtered, poolID)
	}

	return filtered
}

// getTokenAddresses extracts addresses of pool tokens, combines with addresses and returns
func getTokenAddresses(pools []entity.Pool, addresses ...string) []string {
	tokenAddressSet := make(map[string]bool, len(pools)+len(addresses))

	for _, pool := range pools {
		for _, token := range pool.Tokens {
			tokenAddressSet[token.Address] = true
		}
	}

	for _, address := range addresses {
		tokenAddressSet[address] = true
	}

	tokenAddresses := make([]string, 0, len(tokenAddressSet))
	for tokenAddress := range tokenAddressSet {
		tokenAddresses = append(tokenAddresses, tokenAddress)
	}

	return tokenAddresses
}

func extractBestRoute(routes []*core.Route) *core.Route {
	if len(routes) == 0 {
		return nil
	}

	return routes[0]
}

func filterPools(pools []entity.Pool, filters ...PoolFilter) []entity.Pool {
	filteredPools := make([]entity.Pool, 0, len(pools))

	for _, pool := range pools {
		valid := true

		for _, filter := range filters {
			if !filter(pool) {
				valid = false
				break
			}
		}

		if !valid {
			continue
		}

		filteredPools = append(filteredPools, pool)
	}

	return filteredPools
}

type PoolFilter func(pool entity.Pool) bool

func PoolFilterSources(sources []string) PoolFilter {
	sourceSet := make(map[string]bool, len(sources))

	for _, source := range sources {
		sourceSet[source] = true
	}

	return func(pool entity.Pool) bool {
		return sourceSet[pool.Exchange]
	}
}

func PoolFilterHasReserveOrAmplifiedTvl(pool entity.Pool) bool {
	return pool.HasReserves() || pool.HasAmplifiedTvl()
}
