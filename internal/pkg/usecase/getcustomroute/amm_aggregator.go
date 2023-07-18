package getcustomroute

import (
	"context"
	"math/big"

	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	poolpkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/business"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/getroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

// ammAggregator finds best route within amm liquidity sources
type ammAggregator struct {
	tokenRepository ITokenRepository
	priceRepository IPriceRepository
	poolManager     IPoolManager

	routeFinder findroute.IFinder
}

func NewCustomAMMAggregator(
	tokenRepository ITokenRepository,
	priceRepository IPriceRepository,
	poolManager IPoolManager,
	routeFinder findroute.IFinder,
) *ammAggregator {
	return &ammAggregator{
		tokenRepository: tokenRepository,
		priceRepository: priceRepository,
		poolManager:     poolManager,
		routeFinder:     routeFinder,
	}
}

func (a *ammAggregator) Aggregate(ctx context.Context, params *types.AggregateParams, poolIds []string) (*valueobject.RouteSummary, error) {
	// Step 1: get pool set
	poolByAddress, err := a.getPoolByAddress(ctx, params, poolIds)
	if err != nil {
		return nil, err
	}

	if len(poolByAddress) == 0 {
		return nil, getroute.ErrPoolSetEmpty
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

func (a *ammAggregator) ApplyConfig(config getroute.Config) {}

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
		PoolBucket:        valueobject.NewPoolBucket(poolByAddress),
		TokenByAddress:    tokenByAddress,
		PriceUSDByAddress: priceUSDByAddress,
	}

	routes, err := a.routeFinder.Find(ctx, input, data)
	if err != nil {
		return nil, errors.Wrapf(getroute.ErrRouteNotFound, "find route failed: [%v]", err)
	}

	bestRoute := extractBestRoute(routes)

	if bestRoute == nil || len(bestRoute.Paths) == 0 {
		return nil, getroute.ErrRouteNotFound
	}

	return a.summarizeRoute(ctx, bestRoute, params, poolByAddress)
}

func (a *ammAggregator) summarizeRoute(
	_ context.Context,
	route *valueobject.Route,
	params *types.AggregateParams,
	poolByAddress map[string]poolpkg.IPool,
) (*valueobject.RouteSummary, error) {
	// Step 1: prepare pool data
	poolBucket := valueobject.NewPoolBucket(poolByAddress)

	var (
		amountOut = new(big.Int).Set(constant.Zero)
		gas       = business.BaseGas
	)

	// Step 2: summarize route
	summarizedRoute := make([][]valueobject.Swap, 0, len(route.Paths))
	for _, path := range route.Paths {

		// Step 2.1: summarize path
		summarizedPath := make([]valueobject.Swap, 0, len(path.PoolAddresses))

		// Step 2.1.0: prepare input of the first swap
		tokenAmountIn := path.Input

		for swapIdx, swapPoolAddress := range path.PoolAddresses {
			// Step 2.1.1: take the pool with fresh data
			pool, ok := poolBucket.GetPool(swapPoolAddress)
			if !ok {
				return nil, errors.Wrapf(
					getroute.ErrInvalidSwap,
					"ammAggregator.summarizeRoute > pool not found [%s]",
					swapPoolAddress,
				)
			}

			// Step 2.1.2: simulate c swap through the pool
			result, err := poolpkg.CalcAmountOut(pool, tokenAmountIn, path.Tokens[swapIdx+1].Address)
			if err != nil {
				return nil, errors.Wrapf(
					getroute.ErrInvalidSwap,
					"ammAggregator.summarizeRoute > swap failed > pool: [%s] > error : [%v]",
					swapPoolAddress,
					err,
				)
			}

			// Step 2.1.3: check if result is valid
			if !result.IsValid() {
				return nil, errors.Wrapf(
					getroute.ErrInvalidSwap,
					"ammAggregator.summarizeRoute > invalid swap > pool : [%s]",
					swapPoolAddress,
				)
			}

			//Step 2.1.4: clone the pool before updating it (do not modify IPool returned by `poolManager`)
			pool = poolBucket.ClonePool(swapPoolAddress)

			// Step 2.1.5: update balance of the pool
			updateBalanceParams := poolpkg.UpdateBalanceParams{
				TokenAmountIn:  tokenAmountIn,
				TokenAmountOut: *result.TokenAmountOut,
				Fee:            *result.Fee,
				SwapInfo:       result.SwapInfo,
			}
			pool.UpdateBalance(updateBalanceParams)

			// Step 2.1.6: summarize the swap
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

			// Step 2.1.7: add up gas fee
			gas += result.Gas

			// Step 2.1.8: update input of the next swap is output of current swap
			tokenAmountIn = *result.TokenAmountOut
		}

		// Step 2.2: add up amountOut
		amountOut.Add(amountOut, tokenAmountIn.Amount)
		summarizedRoute = append(summarizedRoute, summarizedPath)
	}

	// amountOut is actual amount of token to be received
	// in case charge fee by currencyIn: amountIn = amountIn - extraFeeAmount
	// in case charge fee by currencyOut: amountOut = amountOut - extraFeeAmount will be included in summarizeRoute
	amountOut, err := calcAmountOutAfterFee(amountOut, params.ExtraFee)
	if err != nil {
		return nil, err
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
	poolIds []string,
) (map[string]poolpkg.IPool, error) {
	ammSources := a.filterAMMSources(params.Sources)

	return a.poolManager.GetPoolByAddress(
		ctx,
		poolIds,
		ammSources,
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

func (a *ammAggregator) filterAMMSources(sources []string) []string {
	ammSources := make([]string, 0, len(sources))
	for _, source := range sources {
		if valueobject.IsAMMSource(valueobject.Exchange(source)) {
			ammSources = append(ammSources, source)
		}
	}

	return ammSources
}

func calcAmountOutAfterFee(amountOut *big.Int, extraFee valueobject.ExtraFee) (*big.Int, error) {
	if extraFee.ChargeFeeBy != valueobject.ChargeFeeByCurrencyOut {
		return amountOut, nil
	}

	actualFeeAmount := extraFee.CalcActualFeeAmount(amountOut)

	if actualFeeAmount.Cmp(constant.Zero) > 0 && actualFeeAmount.Cmp(amountOut) > 0 {
		return nil, getroute.ErrFeeAmountIsGreaterThanAmountOut
	}

	return new(big.Int).Sub(amountOut, actualFeeAmount), nil
}
