package getcustomroute

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/business"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/getroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type aggregator struct {
	poolFactory            IPoolFactory
	tokenRepository        ITokenRepository
	priceRepository        IPriceRepository
	onchainpriceRepository IOnchainPriceRepository
	poolRepository         IPoolRepository

	routeFinder findroute.IFinder
}

func NewCustomAggregator(
	poolFactory IPoolFactory,
	tokenRepository ITokenRepository,
	priceRepository IPriceRepository,
	onchainpriceRepository IOnchainPriceRepository,
	poolRepository IPoolRepository,
	routeFinder findroute.IFinder,
) *aggregator {
	return &aggregator{
		poolFactory:            poolFactory,
		tokenRepository:        tokenRepository,
		priceRepository:        priceRepository,
		onchainpriceRepository: onchainpriceRepository,
		poolRepository:         poolRepository,
		routeFinder:            routeFinder,
	}
}

func (a *aggregator) Aggregate(ctx context.Context, params *types.AggregateParams, poolIds []string) (*valueobject.RouteSummary, error) {
	// Step 1: get pool set
	poolEntities, err := a.poolRepository.FindByAddresses(ctx, poolIds)
	if err != nil {
		return nil, err
	}

	poolByAddress := make(map[string]poolpkg.IPoolSimulator, len(poolIds))
	poolInterfaces := a.poolFactory.NewPools(ctx, poolEntities, common.Hash{}) // Not use AEVM in custom route
	for i := range poolInterfaces {
		poolByAddress[poolInterfaces[i].GetAddress()] = poolInterfaces[i]
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

	var priceUSDByAddress map[string]float64

	// only get price from onchain-price-service if enabled
	var priceByAddress map[string]*routerEntity.OnchainPrice
	if a.onchainpriceRepository != nil {
		priceByAddress, err = a.onchainpriceRepository.FindByAddresses(ctx, tokenAddresses)
		if err != nil {
			return nil, err
		}

		// if we're using native price from onchain-price-service, then only need USD price for the native token
		// TODO: get this from onchain-price-service once API available
		priceUSDByAddress = map[string]float64{params.GasToken.Address: params.GasTokenPriceUSD}
	} else {
		priceUSDByAddress, err = a.getPriceUSDByAddress(ctx, tokenAddresses)
		if err != nil {
			return nil, err
		}
	}

	var limits = make(map[string]map[string]*big.Int)
	limits[pooltypes.PoolTypes.KyberPMM] = make(map[string]*big.Int)
	limits[pooltypes.PoolTypes.Synthetix] = make(map[string]*big.Int)
	limits[pooltypes.PoolTypes.NativeV1] = make(map[string]*big.Int)
	for _, pool := range poolInterfaces {
		dexLimit, avail := limits[pool.GetType()]
		if !avail {
			continue
		}
		limitMap := pool.CalculateLimit()
		for k, v := range limitMap {
			dexLimit[k] = v
		}
	}

	// Step 3: finds best route
	return a.findBestRoute(ctx, params, tokenByAddress, priceUSDByAddress, priceByAddress, &types.FindRouteState{
		Pools:     poolByAddress,
		SwapLimit: a.poolFactory.NewSwapLimit(limits),
	})
}

func (a *aggregator) ApplyConfig(config getroute.Config) {}

// findBestRoute find the best route and summarize it
func (a *aggregator) findBestRoute(
	ctx context.Context,
	params *types.AggregateParams,
	tokenByAddress map[string]*entity.Token,
	priceUSDByAddress map[string]float64,
	priceByAddress map[string]*routerEntity.OnchainPrice,
	state *types.FindRouteState,
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

	data := findroute.NewFinderData(ctx, tokenByAddress, priceUSDByAddress, priceByAddress, state)
	defer data.ReleaseResources()
	routes, err := a.routeFinder.Find(ctx, input, data)
	if err != nil {
		return nil, errors.WithMessagef(getroute.ErrRouteNotFound, "find route failed: [%v]", err)
	}

	bestRoute := extractBestRoute(routes)

	if bestRoute == nil || len(bestRoute.Paths) == 0 {
		return nil, getroute.ErrRouteNotFound
	}

	data.Refresh()
	return a.summarizeRoute(ctx, bestRoute, params, state.Pools, data.SwapLimits)
}

func (a *aggregator) summarizeRoute(
	_ context.Context,
	route *valueobject.Route,
	params *types.AggregateParams,
	poolByAddress map[string]poolpkg.IPoolSimulator,
	swapLimits map[string]poolpkg.SwapLimit,
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
		tokenAmountIn := *path.Input.ToDexLibAmount()

		for swapIdx, swapPoolAddress := range path.PoolAddresses {
			// Step 2.1.1: take the pool with fresh data
			pool, ok := poolBucket.GetPool(swapPoolAddress)
			if !ok {
				return nil, errors.WithMessagef(
					getroute.ErrInvalidSwap,
					"aggregator.summarizeRoute > pool not found [%s]",
					swapPoolAddress,
				)
			}
			swapLimit := swapLimits[pool.GetType()]
			// Step 2.1.2: simulate c swap through the pool
			result, err := poolpkg.CalcAmountOut(pool, tokenAmountIn, path.Tokens[swapIdx+1].Address, swapLimit)

			if err != nil {
				return nil, errors.WithMessagef(
					getroute.ErrInvalidSwap,
					"aggregator.summarizeRoute > swap failed > pool: [%s] > error : [%v]",
					swapPoolAddress,
					err,
				)
			}

			// Step 2.1.3: check if result is valid
			if !result.IsValid() {
				return nil, errors.WithMessagef(
					getroute.ErrInvalidSwap,
					"aggregator.summarizeRoute > invalid swap > pool : [%s]",
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
				SwapLimit:      swapLimit,
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
		Extra:        route.Extra,
	}, nil
}

// getTokenByAddress receives a list of address and returns a map of address to entity.Token
func (a *aggregator) getTokenByAddress(ctx context.Context, tokenAddresses []string) (map[string]*entity.Token, error) {
	tokens, err := a.tokenRepository.FindByAddresses(ctx, tokenAddresses)
	if err != nil {
		return nil, err
	}

	tokenByAddress := make(map[string]*entity.Token, len(tokens))
	for _, token := range tokens {
		tokenByAddress[token.Address] = token
	}

	return tokenByAddress, nil
}

// getPriceUSDByAddress receives a list of address and returns a map of address to its preferred price in USD
func (a *aggregator) getPriceUSDByAddress(ctx context.Context, tokenAddresses []string) (map[string]float64, error) {
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
