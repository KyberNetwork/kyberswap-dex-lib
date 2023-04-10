package getroute

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strings"
	"sync"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/core"
	"github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase"
	usecasecore "github.com/KyberNetwork/router-service/internal/pkg/usecase/core"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/factory"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/spfav2"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/validateroute"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/eth"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/requestid"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type GetRoutesUseCase struct {
	cacheRoute    *usecase.CacheRouteUseCase
	validateRoute *validateroute.ValidateRouteUseCase
	poolFactory   *factory.PoolFactory

	poolRepository         usecase.IPoolRepository
	tokenRepository        usecase.ITokenRepository
	priceRepository        usecase.IPriceRepository
	routeRepository        usecase.IRouteRepository
	scannerStateRepository usecase.IScannerStateRepository

	config usecase.GetRoutesConfig
	mu     sync.RWMutex
}

type findRouteParams struct {
	query dto.GetRoutesQuery

	tokenIn  entity.Token
	tokenOut entity.Token
	gasToken entity.Token

	tokenInPrice  entity.Price
	tokenOutPrice entity.Price
	gasTokenPrice entity.Price

	gasPrice *big.Float

	sources        []string
	pools          []entity.Pool
	tokenByAddress map[string]entity.Token
	priceByAddress map[string]entity.Price
}

func NewGetRoutesUseCase(
	cacheRoute *usecase.CacheRouteUseCase,
	validateRoute *validateroute.ValidateRouteUseCase,
	poolFactory *factory.PoolFactory,
	poolRepository usecase.IPoolRepository,
	tokenRepository usecase.ITokenRepository,
	priceRepository usecase.IPriceRepository,
	routeRepository usecase.IRouteRepository,
	scannerStateRepository usecase.IScannerStateRepository,
	config usecase.GetRoutesConfig,
) *GetRoutesUseCase {
	return &GetRoutesUseCase{
		cacheRoute:             cacheRoute,
		validateRoute:          validateRoute,
		poolFactory:            poolFactory,
		poolRepository:         poolRepository,
		tokenRepository:        tokenRepository,
		priceRepository:        priceRepository,
		routeRepository:        routeRepository,
		scannerStateRepository: scannerStateRepository,
		config:                 config,
	}
}

// Handle returns the best route with signature
func (uc *GetRoutesUseCase) Handle(ctx context.Context, query dto.GetRoutesQuery) (*dto.GetRoutesResult, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "GetRoutesUseCase.Handle")
	defer span.Finish()

	// Step 1: Prepare data for finding route
	// - convert Ether to WETH (tokenIn and tokenOut)
	// - pool set
	// - token set (poolTokens + gasToken + tokenIn + tokenOut)
	// - price set
	// - gas price

	tokenInAddress, err := eth.ConvertEtherToWETH(query.TokenIn, uc.config.ChainID)
	if err != nil {
		return nil, err
	}

	tokenOutAddress, err := eth.ConvertEtherToWETH(query.TokenOut, uc.config.ChainID)
	if err != nil {
		return nil, err
	}

	gasTokenAddress := strings.ToLower(uc.config.GasTokenAddress)

	sources := uc.getSources(query.IncludedSources, query.ExcludedSources)

	pools, err := uc.listPools(ctx, tokenInAddress, tokenOutAddress, sources)
	if err != nil {
		return nil, err
	}

	if len(pools) == 0 {
		return nil, errors.Wrap(usecase.ErrRouteNotFound, "poolSet empty")
	}

	tokenAddresses := getTokenAddresses(pools, gasTokenAddress, tokenInAddress, tokenOutAddress)

	tokenByAddress, err := uc.getTokenByAddress(ctx, tokenAddresses)
	if err != nil {
		return nil, err
	}

	if _, ok := tokenByAddress[tokenInAddress]; !ok {
		return nil, errors.Wrapf(usecase.ErrRouteNotFound, "tokenIn not found >> tokenInAddress: [%s]", tokenInAddress)
	}

	if _, ok := tokenByAddress[tokenOutAddress]; !ok {
		return nil, errors.Wrapf(usecase.ErrRouteNotFound, "tokenOut not found >> tokenOutAddress: [%s]", tokenOutAddress)
	}

	priceByAddress, err := uc.getPriceByAddress(ctx, tokenAddresses)
	if err != nil {
		return nil, err
	}

	gasPrice, err := uc.getGasPrice(ctx, query.GasPrice)
	if err != nil {
		return nil, err
	}

	// Step 2: Find route
	// - Look up from cache, returns cached route if there is a valid one
	// - Perform SPFA to find route
	// - Set found route to cache

	params := &findRouteParams{
		query: query,

		tokenIn:  tokenByAddress[tokenInAddress],
		tokenOut: tokenByAddress[tokenOutAddress],
		gasToken: tokenByAddress[gasTokenAddress],

		tokenInPrice:  priceByAddress[tokenInAddress],
		tokenOutPrice: priceByAddress[tokenOutAddress],
		gasTokenPrice: priceByAddress[gasTokenAddress],

		gasPrice: gasPrice,

		sources:        sources,
		pools:          pools,
		tokenByAddress: tokenByAddress,
		priceByAddress: priceByAddress,
	}

	routeSummary, err := uc.findRouteWithCache(ctx, params)
	if err != nil {
		return nil, err
	}

	return &dto.GetRoutesResult{
		RouteSummary:  routeSummary,
		RouterAddress: uc.config.RouterAddress,
	}, nil
}

func (uc *GetRoutesUseCase) ApplyConfig(
	enabledDexes []string,
	blacklistedPools []string,
	featureFlags valueobject.FeatureFlags,
	whitelistedToken []valueobject.WhitelistedToken,
) error {

	uc.mu.Lock()
	uc.config.EnabledDexes = enabledDexes
	uc.config.BlacklistedPools = blacklistedPools
	uc.config.FeatureFlags = featureFlags
	uc.config.WhitelistedTokens = whitelistedToken
	uc.mu.Unlock()

	return nil
}

func (uc *GetRoutesUseCase) getSources(includedSources []string, excludedSources []string) []string {
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
func (uc *GetRoutesUseCase) listPools(
	ctx context.Context,
	tokenInAddress string,
	tokenOutAddress string,
	sources []string,
) ([]entity.Pool, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "GetRoutesUseCase.listPools")
	defer span.Finish()

	directPairKey := usecasecore.GenDirectPairKey(tokenInAddress, tokenOutAddress)

	whitelistI := uc.isWhiteListedToken(tokenInAddress)
	whitelistJ := uc.isWhiteListedToken(tokenOutAddress)
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
func (uc *GetRoutesUseCase) listCurveMetaBasePools(
	ctx context.Context,
	pools []entity.Pool,
) ([]entity.Pool, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "GetRoutesUseCase.listCurveMetaBasePools")
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
			logger.WithFields(logger.Fields{
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
func (uc *GetRoutesUseCase) getTokenByAddress(
	ctx context.Context,
	tokenAddresses []string,
) (map[string]entity.Token, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "GetRoutesUseCase.getTokenByAddress")
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
func (uc *GetRoutesUseCase) getPriceByAddress(
	ctx context.Context,
	tokenAddresses []string,
) (map[string]entity.Price, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "GetRoutesUseCase.getPriceByAddress")
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
func (uc *GetRoutesUseCase) getGasPrice(
	ctx context.Context,
	queryGasPrice *big.Float,
) (*big.Float, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "GetRoutesUseCase.getGasPrice")
	defer span.Finish()

	if queryGasPrice != nil {
		return queryGasPrice, nil
	}

	return uc.scannerStateRepository.GetGasPrice(ctx)
}

// findRouteWithCache gets route from cache, if cached route is valid, it returns the cached route
// otherwise, it performs SPFA to find the best route .
// The best route will be cached before returned
func (uc *GetRoutesUseCase) findRouteWithCache(
	ctx context.Context,
	params *findRouteParams,
) (*valueobject.RouteSummary, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "GetRoutesUseCase.findRouteWithCache")
	defer span.Finish()

	// amountIn is actual amount of token to be swapped
	// in case charge fee by currencyIn: amountIn = amountIn - extraFeeAmount
	// in case charge fee by currencyOut: amountOut = amountOut - extraFeeAmount will be included in summarizeRoute
	amountIn := usecasecore.CalcAmountInAfterFee(params.query.AmountIn, params.query.ExtraFee)

	tokenInPrice, _ := params.tokenInPrice.GetPreferredPrice()

	amountInUSD := utils.CalcTokenAmountUsd(amountIn, params.tokenIn.Decimals, tokenInPrice)

	if amountInUSD > constant.MaxAmountInUSD {
		return nil, usecase.ErrAmountInIsGreaterThanMaxAllowed
	}

	amountInUSDInt := int(math.Round(amountInUSD))

	routeCacheKey := uc.cacheRoute.GenKey(
		params.tokenIn.Address,
		params.tokenOut.Address,
		amountIn,
		params.tokenIn.Decimals,
		amountInUSDInt,
		params.query.SaveGas,
		params.sources,
		params.query.GasInclude,
	)
	summarizedRoute, err := uc.getRouteFromCache(ctx, routeCacheKey, params, amountIn)
	if err == nil {
		logger.WithFields(logger.Fields{
			"key": routeCacheKey.String(""),
		}).Info("cache hit")
		metrics.IncrFindRouteCacheCount(true, nil)

		return summarizedRoute, nil
	}

	logger.WithFields(logger.Fields{
		"key":   routeCacheKey.String(""),
		"error": err,
	}).Info("cache missed")

	route, err := uc.findRouteWithSPFA(ctx, params, amountIn)
	if err != nil {
		return nil, errors.Wrapf(usecase.ErrRouteNotFound, "find route with SPFA failed: [%s]", err.Error())
	}

	if route == nil || len(route.Paths) <= 0 {
		return nil, errors.Wrap(usecase.ErrRouteNotFound, "route is nil or contains no path")
	}

	if err = uc.validateRoute.ValidateRouteResult(*route); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Warnf("validate route failed")
	}

	routeSummary, err := uc.summarizeRoute(ctx, route, params)
	if err != nil {
		return nil, errors.Wrapf(usecase.ErrRouteNotFound, "summarize route failed: [%s]", err.Error())
	}

	if err = uc.setRouteToCache(ctx, routeCacheKey, route, amountIn, params.tokenIn.Decimals, amountInUSDInt); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Warnf("setRouteToCache failed")
	}

	return routeSummary, nil
}

// getRouteFromCache gets route from cache then redistributes input amount and validates the route
func (uc *GetRoutesUseCase) getRouteFromCache(
	ctx context.Context,
	routeCacheKey valueobject.RouteCacheKey,
	params *findRouteParams,
	amountIn *big.Int,
) (*valueobject.RouteSummary, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "GetRoutesUseCase.getRouteFromCache")
	defer span.Finish()

	cachedRoute, err := uc.cacheRoute.Get(ctx, routeCacheKey)
	if err != nil {
		metrics.IncrFindRouteCacheCount(false, []string{"reason:getCachedRouteFailed"})
		return nil, err
	}

	tokenInPrice, _ := params.tokenInPrice.GetPreferredPrice()

	if err = cachedRoute.RedistributeInputAmount(amountIn, params.tokenIn.Decimals, tokenInPrice); err != nil {
		metrics.IncrFindRouteCacheCount(false, []string{"reason:redistributeInputAmountFailed"})
		return nil, err
	}

	route, err := cachedRoute.ToRoute(
		uc.poolFactory.NewPools(params.pools),
		uc.poolFactory.NewPools(params.pools),
	)
	if err != nil {
		metrics.IncrFindRouteCacheCount(false, []string{"reason:convertCachedRouteFailed"})
		return nil, err
	}

	routeSummary, err := uc.summarizeRoute(
		ctx,
		route,
		params,
	)
	if err != nil {
		metrics.IncrFindRouteCacheCount(false, []string{"reason:summarizeCachedRouteFailed"})
		return nil, err
	}

	if routeSummary.GetPriceImpact() >= uc.config.Epsilon {
		metrics.IncrFindRouteCacheCount(
			false,
			[]string{
				"reason:priceImpactIsGreaterThanEpsilon",
				fmt.Sprintf("priceImpact:%f", routeSummary.GetPriceImpact()),
			},
		)
		return nil, errors.Wrapf(
			usecase.ErrPriceImpactIsGreaterThanEpsilon,
			"priceImpact: [%f]",
			routeSummary.GetPriceImpact(),
		)
	}

	return routeSummary, nil
}

// findRouteWithSPFA performs SPFA to find the best route
// if saveGas, it finds and returns the best single path route
// otherwise, it finds the best single path route and the best multiple path route and returns the better one
func (uc *GetRoutesUseCase) findRouteWithSPFA(
	ctx context.Context,
	params *findRouteParams,
	amountIn *big.Int,
) (*core.Route, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "GetRoutesUseCase.findRouteWithSPFA")
	defer span.Finish()

	preferredPriceUSDByAddress := make(map[string]float64, len(params.priceByAddress))
	for address, price := range params.priceByAddress {
		preferredPrice, _ := price.GetPreferredPrice()
		preferredPriceUSDByAddress[address] = preferredPrice
	}

	gasTokenPriceUSD, _ := params.gasTokenPrice.GetPreferredPrice()

	input := findroute.Input{
		TokenInAddress:   params.tokenIn.Address,
		TokenOutAddress:  params.tokenOut.Address,
		AmountIn:         amountIn,
		GasPrice:         params.gasPrice,
		GasTokenPriceUSD: gasTokenPriceUSD,
		SaveGas:          params.query.SaveGas,
		GasInclude:       params.query.GasInclude,
	}

	var finder findroute.IFinder = spfav2.NewSPFAv2Finder(
		uc.config.SPFAFinderOptions.MaxHops,
		uc.config.SPFAFinderOptions.DistributionPercent,
		uc.config.SPFAFinderOptions.MaxPathsInRoute,
		uc.config.SPFAFinderOptions.MaxPathsToGenerate,
		uc.config.SPFAFinderOptions.MaxPathsToReturn,
		uc.config.SPFAFinderOptions.MinPartUSD,
		uc.config.SPFAFinderOptions.MinThresholdAmountInUSD,
		uc.config.SPFAFinderOptions.MaxThresholdAmountInUSD,
	)

	bestRoutes, err := finder.Find(
		ctx,
		input,
		findroute.FinderData{
			PoolByAddress:     uc.poolFactory.NewPoolByAddress(params.pools),
			TokenByAddress:    params.tokenByAddress,
			PriceUSDByAddress: preferredPriceUSDByAddress,
		})
	if err != nil {
		return nil, err
	}

	return extractBestRoute(bestRoutes), nil
}

// setRouteToCache caches best route
func (uc *GetRoutesUseCase) setRouteToCache(
	ctx context.Context,
	routeCacheKey valueobject.RouteCacheKey,
	route *core.Route,
	amountIn *big.Int,
	tokenInDecimals uint8,
	amountInUSDInt int,
) error {
	span, ctx := tracer.StartSpanFromContext(ctx, "GetRoutesUseCase.setRouteToCache")
	defer span.Finish()

	cachedRoute, err := route.ToCachedRoute()
	if err != nil {
		return err
	}

	return uc.cacheRoute.Set(ctx, routeCacheKey, cachedRoute, amountIn, tokenInDecimals, amountInUSDInt)
}

// summarizeRoute converts *core.Route into valueobject.RouteSummary
func (uc *GetRoutesUseCase) summarizeRoute(
	ctx context.Context,
	route *core.Route,
	params *findRouteParams,
) (*valueobject.RouteSummary, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "GetRoutesUseCase.summarizeRoute")
	defer span.Finish()

	pools := uc.poolFactory.NewPools(params.pools)

	poolByAddress := make(map[string]pool.IPool, len(pools))
	for _, pool := range pools {
		poolByAddress[pool.GetAddress()] = pool
	}

	var (
		amountOut = constant.Zero
		gas       = uc.config.BaseGas
	)

	summarizedRoute := make([][]valueobject.Swap, 0, len(route.Paths))
	for _, path := range route.Paths {
		gas += path.TotalGas

		summarizedPath := make([]valueobject.Swap, 0, len(path.Pools))
		swapIn := path.Input
		for swapIdx, swapPool := range path.Pools {
			freshPool, ok := poolByAddress[swapPool.GetAddress()]
			if !ok {
				logger.WithFields(logger.Fields{
					"pool.Address": swapPool.GetAddress(),
					"request.id":   requestid.RequestIDFromCtx(ctx),
				}).Error("[GetRoutesUseCase.summarizeRoute] pool not found")

				return nil, usecase.ErrPoolNotFound
			}

			calcAmountOutResult, err := freshPool.CalcAmountOut(swapIn, path.Tokens[swapIdx+1].Address)
			if err != nil {
				logger.WithFields(logger.Fields{
					"tokenIn":      swapIn.Token,
					"amountIn":     swapIn.Amount.String(),
					"pool.Address": freshPool.GetAddress(),
					"request.id":   requestid.RequestIDFromCtx(ctx),
				}).Error("[GetRoutesUseCase.summarizeRoute] invalid swap")

				return nil, usecase.ErrInvalidSwap
			}
			swapOut, swapFee := calcAmountOutResult.TokenAmountOut, calcAmountOutResult.Fee
			if swapOut == nil || swapOut.Amount == nil || swapOut.Amount.Cmp(constant.Zero) <= 0 {
				logger.WithFields(logger.Fields{
					"tokenIn":      swapIn.Token,
					"amountIn":     swapIn.Amount.String(),
					"pool.Address": freshPool.GetAddress(),
					"request.id":   requestid.RequestIDFromCtx(ctx),
				}).Error("[GetRoutesUseCase.summarizeRoute] invalid swap")

				return nil, usecase.ErrInvalidSwap
			}

			swap := valueobject.Swap{
				Pool:              freshPool.GetAddress(),
				TokenIn:           swapIn.Token,
				TokenOut:          swapOut.Token,
				SwapAmount:        swapIn.Amount,
				AmountOut:         swapOut.Amount,
				LimitReturnAmount: constant.Zero,
				Exchange:          valueobject.Exchange(swapPool.GetExchange()),
				PoolLength:        len(swapPool.GetTokens()),
				PoolType:          swapPool.GetType(),
				PoolExtra:         swapPool.GetMetaInfo(swapIn.Token, swapOut.Token),
				Extra:             calcAmountOutResult.SwapInfo,
			}

			summarizedPath = append(summarizedPath, swap)
			updateBalanceParams := pool.UpdateBalanceParams{
				TokenAmountIn:  swapIn,
				TokenAmountOut: *swapOut,
				Fee:            *swapFee,
				SwapInfo:       calcAmountOutResult.SwapInfo,
			}
			swapPool.UpdateBalance(updateBalanceParams)
			swapIn = *swapOut

			metrics.IncrDexHitRate(string(swap.Exchange))
			metrics.IncrPoolTypeHitRate(swap.PoolType)
		}

		amountOut = new(big.Int).Add(amountOut, swapIn.Amount)
		summarizedRoute = append(summarizedRoute, summarizedPath)
	}

	// amountOut is actual amount of token to be received
	// in case charge fee by currencyIn: amountIn = amountIn - extraFeeAmount
	// in case charge fee by currencyOut: amountOut = amountOut - extraFeeAmount will be included in summarizeRoute
	amountOut, err := calcAmountOutAfterFee(amountOut, params.query.ExtraFee)
	if err != nil {
		return nil, err
	}

	tokenInPrice, tokenInMarketPriceAvailable := params.tokenInPrice.GetPreferredPrice()
	tokenOutPrice, tokenOutMarketPriceAvailable := params.tokenOutPrice.GetPreferredPrice()
	gasTokenPrice, _ := params.gasTokenPrice.GetPreferredPrice()

	metrics.IncrRequestPairCount(params.tokenIn.Address, params.tokenOut.Address, params.query.AmountIn.String())

	return &valueobject.RouteSummary{
		TokenIn:                      params.query.TokenIn,
		AmountIn:                     params.query.AmountIn,
		AmountInUSD:                  utils.CalcTokenAmountUsd(params.query.AmountIn, params.tokenIn.Decimals, tokenInPrice),
		TokenInMarketPriceAvailable:  tokenInMarketPriceAvailable,
		TokenOut:                     params.query.TokenOut,
		AmountOut:                    amountOut,
		AmountOutUSD:                 utils.CalcTokenAmountUsd(amountOut, params.tokenOut.Decimals, tokenOutPrice),
		TokenOutMarketPriceAvailable: tokenOutMarketPriceAvailable,
		Gas:                          gas,
		GasPrice:                     params.gasPrice,
		GasUSD:                       utils.CalcGasUsd(params.gasPrice, gas, gasTokenPrice),
		ExtraFee:                     params.query.ExtraFee,
		Route:                        summarizedRoute,
	}, nil
}

func (uc *GetRoutesUseCase) filterBlacklistedPools(poolIDs []string) []string {
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

func (uc *GetRoutesUseCase) isWhiteListedToken(token string) bool {
	for _, whitelistedToken := range uc.config.WhitelistedTokens {
		if strings.EqualFold(whitelistedToken.Address, token) {
			return true
		}
	}

	return false
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

func calcAmountOutAfterFee(amountOut *big.Int, extraFee valueobject.ExtraFee) (*big.Int, error) {
	if extraFee.ChargeFeeBy != valueobject.ChargeFeeByCurrencyOut {
		return amountOut, nil
	}

	actualFeeAmount := extraFee.CalcActualFeeAmount(amountOut)

	if actualFeeAmount.Cmp(constant.Zero) > 0 && actualFeeAmount.Cmp(amountOut) > 0 {
		return nil, usecase.ErrFeeAmountIsGreaterThanAmountOut
	}

	return new(big.Int).Sub(amountOut, actualFeeAmount), nil
}
