package uniswap

import (
	"context"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/KyberNetwork/router-service/internal/pkg/core"
	poolPkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/common"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
)

const (
	defaultUniswapMaxHops             uint32  = 3
	defaultUniswapMaxPaths            uint32  = 5
	defaultUniswapMaxPools            uint32  = 15
	defaultUniswapMaxRoutes           uint32  = 5
	defaultUniswapMaxPathsToGenerate  uint32  = 5
	defaultUniswapDistributionPercent uint32  = 5
	defaultUniswapMinPartUSD          float64 = 500
)

// uniswapFinder finds route using Uniswap auto router algorithm
type uniswapFinder struct {
	// maxHops max number of pools in a path
	maxHops uint32

	// max number of paths in a route
	maxPaths uint32

	// maxSwaps max total number of pools that a route consists
	maxPools uint32

	// maxRoutes max number of routes to return
	maxRoutes uint32

	// maxPathsToGenerate number of paths to generate for each path length
	// This is the K params in docs and algo
	// The total number of paths generated is <= maxPathsToGenerate * maxHop
	maxPathsToGenerate uint32

	// distributionPercent the portion of amountIn to split
	distributionPercent uint32

	// minPartUSD minimum amount in USD of each part
	minPartUSD float64
}

func NewUniswapFinder(maxHops, maxPaths, maxPools, maxRoutes, maxPathsToGenerate, distributionPercent uint32, minPartUSD float64) *uniswapFinder {
	return &uniswapFinder{maxHops, maxPaths, maxPools, maxRoutes, maxPathsToGenerate, distributionPercent, minPartUSD}
}

func NewDefaultUniswapFinder() *uniswapFinder {
	return NewUniswapFinder(
		defaultUniswapMaxHops,
		defaultUniswapMaxPaths,
		defaultUniswapMaxPools,
		defaultUniswapMaxRoutes,
		defaultUniswapMaxPathsToGenerate,
		defaultUniswapDistributionPercent,
		defaultUniswapMinPartUSD,
	)
}

func (f *uniswapFinder) Find(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
) ([]*core.Route, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[uniswap] Find")
	defer span.Finish()

	// Must be able to get info about tokenIn
	if _, ok := data.TokenByAddress[input.TokenInAddress]; !ok {
		return nil, findroute.ErrNoInfoTokenIn
	}
	// Must be able to get info about tokenOut
	if _, ok := data.TokenByAddress[input.TokenOutAddress]; !ok {
		return nil, findroute.ErrNoInfoTokenOut
	}

	// Step 1: Optimize graph traversal by using adjacent list
	tokenToPoolAddress := make(map[string][]string)
	for poolAddress, pool := range data.PoolByAddress {
		for _, fromToken := range pool.GetTokens() {
			tokenToPoolAddress[fromToken] = append(tokenToPoolAddress[fromToken], poolAddress)
		}
	}

	// Step 2: Find min number of hop from tokenA -> tokenOut for all tokenA
	hopsToTokenOut, err := common.MinHopsToTokenOut(data.PoolByAddress, data.TokenByAddress, tokenToPoolAddress, input.TokenOutAddress)
	if err != nil {
		return nil, err
	}

	if minHopFromTokenIn, ok := hopsToTokenOut[input.TokenInAddress]; !ok || minHopFromTokenIn > f.maxHops {
		return nil, nil
	}

	// Step 3: Find multiple best paths from tokenIn to tokenOut
	// it is fine if prices[token] is not set because it would default to zero
	tokenAmountIn := poolPkg.TokenAmount{
		Token:     input.TokenInAddress,
		Amount:    input.AmountIn,
		AmountUsd: utils.CalcTokenAmountUsd(input.AmountIn, data.TokenByAddress[input.TokenInAddress].Decimals, data.PriceUSDByAddress[input.TokenInAddress]),
	}

	paths, err := common.GenKthBestPaths(ctx, input, data, tokenAmountIn, tokenToPoolAddress, hopsToTokenOut, f.maxHops, f.maxPathsToGenerate)
	if err != nil {
		return nil, err
	}

	if input.SaveGas {
		return f.genSinglePathRoutes(ctx, input, data, tokenAmountIn, paths), nil
	}

	// Step 4: Pick several paths to form a route
	return f.genBestRoutes(ctx, input, data, tokenAmountIn, paths)
}
