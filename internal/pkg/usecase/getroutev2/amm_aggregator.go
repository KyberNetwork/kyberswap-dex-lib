package getroutev2

import (
	"context"
	"math/big"

	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/core"
	poolpkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/business"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/spfav2"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

// ammAggregator finds best route within amm liquidity sources
type ammAggregator struct {
	poolRankRepository IPoolRankRepository
	tokenRepository    ITokenRepository
	priceRepository    IPriceRepository
	poolManager        IPoolManager

	routeFinder findroute.IFinder

	config AmmAggregatorConfig
}

func NewAMMAggregator(
	poolRankRepository IPoolRankRepository,
	tokenRepository ITokenRepository,
	priceRepository IPriceRepository,
	poolManager IPoolManager,
	config AmmAggregatorConfig,
) *ammAggregator {
	routeFinder := spfav2.NewSPFAv2Finder(
		config.FinderOptions.MaxHops,
		config.FinderOptions.DistributionPercent,
		config.FinderOptions.MaxPathsInRoute,
		config.FinderOptions.MaxPathsToGenerate,
		config.FinderOptions.MaxPathsToReturn,
		config.FinderOptions.MinPartUSD,
		config.FinderOptions.MinThresholdAmountInUSD,
		config.FinderOptions.MaxThresholdAmountInUSD,
	)

	return &ammAggregator{
		poolRankRepository: poolRankRepository,
		tokenRepository:    tokenRepository,
		priceRepository:    priceRepository,
		poolManager:        poolManager,
		routeFinder:        routeFinder,
		config:             config,
	}
}

func (a *ammAggregator) Aggregate(ctx context.Context, params *types.AggregateParams) (*valueobject.RouteSummary, error) {
	// Step 1: get pool set
	poolByAddress, err := a.getPoolByAddress(ctx, params)
	if err != nil {
		return nil, err
	}

	if len(poolByAddress) == 0 {
		return nil, ErrPoolSetEmpty
	}

	// Step 2: collect tokens and price data
	tokenAddresses := collectTokenAddresses(
		poolByAddress,
		params.TokenIn.Address,
		params.TokenOut.Address,
		params.GasToken.Address,
	)

	tokenByAddress, err := a.getTokenByAddress(ctx, tokenAddresses)
	if err != nil {
		return nil, err
	}

	priceByAddress, err := a.getPriceUSDByAddress(ctx, tokenAddresses)
	if err != nil {
		return nil, err
	}

	// Step 3: finds best route
	return a.findBestRoute(ctx, params, poolByAddress, tokenByAddress, priceByAddress)
}

// findBestRoute find the best route and summarize it
func (a *ammAggregator) findBestRoute(
	ctx context.Context,
	params *types.AggregateParams,
	poolByAddress map[string]poolpkg.IPool,
	tokenByAddress map[string]entity.Token,
	priceUSDByAddress map[string]float64,
) (*valueobject.RouteSummary, error) {
	input := findroute.Input{
		TokenInAddress:   params.TokenIn.Address,
		TokenOutAddress:  params.TokenOut.Address,
		AmountIn:         params.AmountIn,
		GasPrice:         params.GasPrice,
		GasTokenPriceUSD: params.GasTokenPriceUSD,
		SaveGas:          params.SaveGas,
		GasInclude:       params.GasInclude,
	}

	data := findroute.FinderData{
		PoolByAddress:     poolByAddress,
		TokenByAddress:    tokenByAddress,
		PriceUSDByAddress: priceUSDByAddress,
	}

	routes, err := a.routeFinder.Find(ctx, input, data)
	if err != nil {
		return nil, err
	}

	bestRoute := extractBestRoute(routes)

	if bestRoute == nil || len(bestRoute.Paths) == 0 {
		return nil, ErrRouteNotFound
	}

	return a.summarizeRoute(ctx, bestRoute, params)
}

func (a *ammAggregator) summarizeRoute(
	ctx context.Context,
	route *core.Route,
	params *types.AggregateParams,
) (*valueobject.RouteSummary, error) {
	// Step 1: prepare pool data
	poolByAddress, err := a.poolManager.GetPoolByAddress(
		ctx,
		route.ExtractPoolAddresses(),
		PoolFilterSources(params.Sources),
		PoolFilterHasReserveOrAmplifiedTvl,
	)
	if err != nil {
		return nil, err
	}

	var (
		amountOut = new(big.Int).Set(constant.Zero)
		gas       = business.BaseGas
	)

	// Step 3: summarize route
	summarizedRoute := make([][]valueobject.Swap, 0, len(route.Paths))
	for _, path := range route.Paths {

		// Step 3.1: summarize path
		summarizedPath := make([]valueobject.Swap, 0, len(path.Pools))

		// Step 3.1.0: prepare input of the first swap
		tokenAmountIn := path.Input

		for swapIdx, swapPool := range path.Pools {
			// Step 3.1.1: take the pool with fresh data
			pool, ok := poolByAddress[swapPool.GetAddress()]
			if !ok {
				return nil, errors.Wrapf(
					ErrInvalidSwap,
					"ammAggregator.summarizeRoute > pool not found [%s]",
					swapPool.GetAddress(),
				)
			}

			// Step 3.1.2: simulate c swap through the pool
			result, err := pool.CalcAmountOut(tokenAmountIn, path.Tokens[swapIdx+1].Address)
			if err != nil {
				return nil, errors.Wrapf(
					ErrInvalidSwap,
					"ammAggregator.summarizeRoute > swap failed > pool: [%s] > error : [%v]",
					pool.GetAddress(),
					err,
				)
			}

			// Step 3.1.3: check if result is valid
			if !result.IsValid() {
				return nil, errors.Wrapf(
					ErrInvalidSwap,
					"ammAggregator.summarizeRoute > invalid swap > pool : [%s]",
					pool.GetAddress(),
				)
			}

			// Step 3.1.4: update balance of the pool
			updateBalanceParams := poolpkg.UpdateBalanceParams{
				TokenAmountIn:  tokenAmountIn,
				TokenAmountOut: *result.TokenAmountOut,
				Fee:            *result.Fee,
				SwapInfo:       result.SwapInfo,
			}
			pool.UpdateBalance(updateBalanceParams)

			// Step 3.1.5: summarize the swap
			swap := valueobject.Swap{
				Pool:              pool.GetAddress(),
				TokenIn:           tokenAmountIn.Token,
				TokenOut:          result.TokenAmountOut.Token,
				SwapAmount:        tokenAmountIn.Amount,
				AmountOut:         result.TokenAmountOut.Amount,
				LimitReturnAmount: constant.Zero,
				Exchange:          valueobject.Exchange(pool.GetExchange()),
				PoolLength:        len(pool.GetTokens()),
				PoolType:          pool.GetType(),
				PoolExtra:         pool.GetMetaInfo(tokenAmountIn.Token, result.TokenAmountOut.Token),
				Extra:             result.SwapInfo,
			}

			summarizedPath = append(summarizedPath, swap)

			// Step 3.1.6: add up gas fee
			gas += result.Gas

			// Step 3.1.7: update input of the next swap is output of current swap
			tokenAmountIn = *result.TokenAmountOut
		}

		// Step 3.2: add up amountOut
		amountOut.Add(amountOut, tokenAmountIn.Amount)
		summarizedRoute = append(summarizedRoute, summarizedPath)
	}

	return &valueobject.RouteSummary{
		TokenIn:      params.TokenIn.Address,
		AmountIn:     params.AmountIn,
		AmountInUSD:  utils.CalcTokenAmountUsd(params.AmountIn, params.TokenIn.Decimals, params.TokenInPriceUSD),
		TokenOut:     params.TokenOut.Address,
		AmountOut:    amountOut,
		AmountOutUSD: utils.CalcTokenAmountUsd(amountOut, params.TokenOut.Decimals, params.TokenOutPriceUSD),
		Gas:          gas,
		GasPrice:     params.GasPrice,
		GasUSD:       utils.CalcGasUsd(params.GasPrice, gas, params.GasTokenPriceUSD),
		ExtraFee:     params.ExtraFee,
		Route:        summarizedRoute,
	}, nil
}

func (a *ammAggregator) getPoolByAddress(
	ctx context.Context,
	params *types.AggregateParams,
) (map[string]poolpkg.IPool, error) {
	bestPoolIDs, err := a.poolRankRepository.FindBestPoolIDs(
		ctx,
		params.TokenIn.Address,
		params.TokenOut.Address,
		a.isWhitelistedToken(params.TokenIn.Address),
		a.isWhitelistedToken(params.TokenOut.Address),
		a.config.GetBestPoolsOptions,
	)
	if err != nil {
		return nil, err
	}

	ammSources := a.filterAMMSources(params.Sources)

	return a.poolManager.GetPoolByAddress(
		ctx,
		bestPoolIDs,
		PoolFilterSources(ammSources),
		PoolFilterHasReserveOrAmplifiedTvl,
	)
}

// getTokenByAddress receives a list of address and returns a map of address to entity.Token
func (a *ammAggregator) getTokenByAddress(ctx context.Context, tokenAddresses []string) (map[string]entity.Token, error) {
	tokens, err := a.tokenRepository.FindByAddresses(ctx, tokenAddresses)
	if err != nil {
		return nil, err
	}

	tokenByAddress := make(map[string]entity.Token, len(tokens))
	for _, token := range tokens {
		tokenByAddress[token.Address] = *token
	}

	return tokenByAddress, nil
}

// getPriceUSDByAddress receives a list of address and returns a map of address to its preferred price in USD
func (a *ammAggregator) getPriceUSDByAddress(ctx context.Context, tokenAddresses []string) (map[string]float64, error) {
	prices, err := a.priceRepository.FindByAddresses(ctx, tokenAddresses)
	if err != nil {
		return nil, err
	}

	priceUSDByAddress := make(map[string]float64, len(prices))
	for _, price := range prices {
		priceUSD, _ := price.GetPreferredPrice()

		priceUSDByAddress[price.Address] = priceUSD
	}

	return priceUSDByAddress, nil
}

func (a *ammAggregator) isWhitelistedToken(tokenAddress string) bool {
	_, contained := a.config.WhitelistedTokenSet[tokenAddress]

	return contained
}

func (a *ammAggregator) filterAMMSources(sources []string) []string {
	ammSources := make([]string, 0, len(sources))
	for _, source := range sources {
		if valueobject.IsAMMSource(valueobject.Exchange(source)) {
			ammSources = append(ammSources, source)
		}
	}

	return ammSources
}
