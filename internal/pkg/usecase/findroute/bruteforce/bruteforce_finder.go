package bruteforce

import (
	"context"
	"fmt"
	"math/big"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/core"
	poolPkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/factory"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/common"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

const (
	defaultBruteforceMaxHops             uint32  = 3
	defaultBruteforceDistributionPercent uint32  = 5
	defaultBruteforceMinPartUSD          float64 = 500

	// number of paths to generate in BestPathExactIn, not meant to be configurable
	defaultBruteforceMaxPathsToGenerate uint32 = 1
)

// bruteforceFinder finds route using Shortest spfaPath Faster Algorithm (SPFA)
type bruteforceFinder struct {
	// maxHops maximum hops performed
	maxHops uint32

	// distributionPercent the portion of amountIn to split. It should be a divisor of 100.
	//  e.g distributionPercent = 5, we split amountIn into portions of 5%, 10%, 15%, ..., 100%
	distributionPercent uint32

	// minPartUSD minimum amount in USD of each part
	minPartUSD float64

	originalPools []entity.Pool

	poolFactory *factory.PoolFactory
}

func NewBruteforceFinder(maxHops, distributionPercent uint32, minPartUSD float64, pool []entity.Pool, poolFactory *factory.PoolFactory) *bruteforceFinder {
	return &bruteforceFinder{maxHops, distributionPercent, minPartUSD, pool, poolFactory}
}

func NewDefaultBruteforceFinder(pool []entity.Pool, uc *factory.PoolFactory) *bruteforceFinder {
	return NewBruteforceFinder(defaultBruteforceMaxHops, defaultBruteforceDistributionPercent, defaultBruteforceMinPartUSD, pool, uc)
}

func (f *bruteforceFinder) Find(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
) ([]*core.Route, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[spfa] Find")
	defer span.Finish()

	bestRoute, err := f.bestRouteExactIn(ctx, input, data)
	if err != nil {
		return nil, err
	}
	if bestRoute == nil {
		return nil, nil
	}

	return []*core.Route{bestRoute}, nil
}

func (f *bruteforceFinder) bestBruteforceRoute(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
	tokenAmountIn poolPkg.TokenAmount,
	tokenToPoolAddress map[string][]string,
	hopsToTokenOut map[string]uint32,
) (*core.Route, error) {
	var (
		bestBruteforceRoute = core.NewEmptyRouteFromPoolData(poolPkg.TokenAmount{
			Token:     input.TokenInAddress,
			Amount:    input.AmountIn,
			AmountUsd: 0,
		}, input.TokenOutAddress, f.poolFactory.NewPoolByAddress(f.originalPools))
	)
	splits, err := f.generateSplits(input, data, tokenAmountIn)
	//fmt.Println(len(splits), "???", input.AmountIn)
	if err != nil {
		return nil, err
	}

	for _, split := range splits {
		data.PoolByAddress = f.poolFactory.NewPoolByAddress(f.originalPools)
		currentBestRoute := core.NewEmptyRouteFromPoolData(poolPkg.TokenAmount{
			Token:     input.TokenInAddress,
			Amount:    input.AmountIn,
			AmountUsd: 0,
		}, input.TokenOutAddress, data.PoolByAddress)
		for _, amountInPerSplit := range split {
			bestPath, err := f.bestPathExactIn(ctx, input, data, amountInPerSplit, tokenToPoolAddress, hopsToTokenOut)
			if err != nil {
				logger.WithFields(logger.Fields{"error": err}).
					Debugf("failed to find best path. tokenIn %v tokenOut %v amountIn %v amountInUsd %v",
						input.TokenInAddress, input.TokenOutAddress, amountInPerSplit.Amount, amountInPerSplit.AmountUsd)
			}
			bestAddedPath, err := common.BestPathAmongAddedPaths(input, data, amountInPerSplit, currentBestRoute.Paths)
			if err == nil && bestAddedPath.CompareTo(bestPath, input.GasInclude) < 0 {
				bestPath = bestAddedPath
			}
			if bestPath == nil {
				return nil, nil
			}
			if ok := currentBestRoute.AddPath(bestPath); !ok {
				return nil, fmt.Errorf("cannot add path to bestMultiPathRoute")
			}
		}

		if currentBestRoute.CompareTo(bestBruteforceRoute, input.GasInclude) > 0 {
			bestBruteforceRoute = currentBestRoute
		}
	}
	return bestBruteforceRoute, nil
}

func (f *bruteforceFinder) bestRouteExactIn(ctx context.Context, input findroute.Input, data findroute.FinderData) (*core.Route, error) {
	span, _ := tracer.StartSpanFromContext(ctx, "[bf] bestRouteExactIn")
	defer span.Finish()

	// Must be able to get info about tokenIn
	if _, ok := data.TokenByAddress[input.TokenInAddress]; !ok {
		return nil, findroute.ErrNoInfoTokenIn
	}
	// Must be able to get info about tokenOut
	if _, ok := data.TokenByAddress[input.TokenOutAddress]; !ok {
		return nil, findroute.ErrNoInfoTokenOut
	}

	// Optimize graph traversal by using adjacent list
	tokenToPoolAddress := make(map[string][]string)
	for poolAddress, pool := range data.PoolByAddress {
		for _, fromToken := range pool.GetTokens() {
			tokenToPoolAddress[fromToken] = append(tokenToPoolAddress[fromToken], poolAddress)
		}
	}

	// it is fine if prices[token] is not set because it would default to zero
	tokenAmountIn := poolPkg.TokenAmount{
		Token:     input.TokenInAddress,
		Amount:    input.AmountIn,
		AmountUsd: utils.CalcTokenAmountUsd(input.AmountIn, data.TokenByAddress[input.TokenInAddress].Decimals, data.PriceUSDByAddress[input.TokenInAddress]),
	}
	hopsToTokenOut, err := common.MinHopsToTokenOut(data.PoolByAddress, data.TokenByAddress, tokenToPoolAddress, input.TokenOutAddress)
	if err != nil {
		return nil, err
	}

	if minHopFromTokenIn, ok := hopsToTokenOut[input.TokenInAddress]; !ok || minHopFromTokenIn > f.maxHops {
		return nil, nil
	}

	bestSinglePathRoute, errFindSinglePathRoute := f.bestSinglePathRoute(ctx, input, data, tokenAmountIn, tokenToPoolAddress, hopsToTokenOut)

	// if SaveGas option enabled, consider only single-path route
	if input.SaveGas && errFindSinglePathRoute != nil {
		return nil, errFindSinglePathRoute
	}
	if input.SaveGas && errFindSinglePathRoute == nil {
		return bestSinglePathRoute, nil
	}

	var (
		bestMultiPathRoute    *core.Route
		errFindMultiPathRoute error
	)

	bestMultiPathRoute, errFindMultiPathRoute = f.bestBruteforceRoute(ctx, input, data, tokenAmountIn, tokenToPoolAddress, hopsToTokenOut)

	//fmt.Println(bestMultiPathRoute.Output.AmountUsd, f.bruteforce)
	// cannot find any route
	if errFindSinglePathRoute != nil && errFindMultiPathRoute != nil {
		return nil, nil
	}

	// return the better route between bestSinglePathRoute and bestMultiPathRoute
	if errFindSinglePathRoute != nil || bestSinglePathRoute == nil || len(bestSinglePathRoute.Paths) == 0 {
		return bestMultiPathRoute, nil
	}

	if errFindMultiPathRoute != nil || bestMultiPathRoute == nil || len(bestMultiPathRoute.Paths) == 0 {
		return bestSinglePathRoute, nil
	}

	if bestSinglePathRoute.CompareTo(bestMultiPathRoute, input.GasInclude) > 0 {
		return bestSinglePathRoute, nil
	}

	return bestMultiPathRoute, nil
}

func (f *bruteforceFinder) bestSinglePathRoute(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
	tokenAmountIn poolPkg.TokenAmount,
	tokenToPoolAddress map[string][]string,
	hopsToTokenOut map[string]uint32,
) (*core.Route, error) {
	bestPath, err := f.bestPathExactIn(ctx, input, data, tokenAmountIn, tokenToPoolAddress, hopsToTokenOut)
	if err != nil {
		return nil, err
	}
	bestSinglePathRoute := core.NewRouteFromPaths(poolPkg.TokenAmount{
		Token:     input.TokenInAddress,
		Amount:    input.AmountIn,
		AmountUsd: 0,
	}, input.TokenOutAddress, data.PoolByAddress, []*core.Path{bestPath})
	return bestSinglePathRoute, nil
}

// bestPathExactIn Find the best path to token out
// we represent graph node as pair (token, hops) because we want to handle negative cycles
// edges are now from (X, hop) to (Y, hop + 1) => make the graph a DAG => no cycle
// Perform SPFA from (tokenIn,0) to find the best path to token out
// Because we are performing SPFA and that only edges between (X, hop) -> (Y, hop+1) exist
// => The order of traversal looks like: (, 0) ... (, 0) (, 1) ... (, 1) ... (, hop-1), ... (,hop-1), (,hop)... (, hop)
func (f *bruteforceFinder) bestPathExactIn(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
	tokenAmountIn poolPkg.TokenAmount,
	tokenToPoolAddress map[string][]string,
	hopsToTokenOut map[string]uint32,
) (*core.Path, error) {
	span, _ := tracer.StartSpanFromContext(ctx, "[bf] bestPathExactIn")
	defer span.Finish()

	// Must be able to get info about tokenIn
	if _, ok := data.TokenByAddress[input.TokenInAddress]; !ok {
		return nil, findroute.ErrNoInfoTokenIn
	}
	// Must be able to get info about tokenOut
	if _, ok := data.TokenByAddress[input.TokenOutAddress]; !ok {
		return nil, findroute.ErrNoInfoTokenOut
	}

	// only pick one best path, so set maxPathsToGenerate = 1.
	paths, err := common.GenKthBestPaths(ctx, input, data, tokenAmountIn, tokenToPoolAddress, hopsToTokenOut, f.maxHops, defaultBruteforceMaxPathsToGenerate)
	if err != nil {
		return nil, err
	}
	var bestPath *core.Path
	for _, path := range paths {
		if path != nil && path.CompareTo(bestPath, input.GasInclude) < 0 {
			bestPath = path
		}
	}
	return bestPath, nil
}

// generateSplits spawn all possible splits
func (f *bruteforceFinder) generateSplits(input findroute.Input, data findroute.FinderData, totalAmountIn poolPkg.TokenAmount) ([][]poolPkg.TokenAmount, error) {
	tokenInPrice := data.PriceUSDByAddress[input.TokenInAddress]
	tokenInDecimal := data.TokenByAddress[input.TokenInAddress].Decimals

	if f.distributionPercent == constant.OneHundredPercent || tokenInPrice == 0 || totalAmountIn.AmountUsd <= f.minPartUSD {
		return [][]poolPkg.TokenAmount{{totalAmountIn}}, nil
	}

	var (
		// f.distributionPercent should be a divisor of 100
		// maxNumSplits is the max number of splits with each split contains a portion of f.distributionPercent% of amountIn
		// But we need to account for the f.MinPartUsd requirement by merging these splits
		//maxNumSplits        = int64(constant.OneHundredPercent / f.distributionPercent)
		n                   = 100 / f.distributionPercent
		splits              = generateArraySumN(int(n), int(DefaultMaxNumSplitBruteForce))
		result              [][]poolPkg.TokenAmount
		cumulativeSumAmount *big.Int
		scaledSplit         []poolPkg.TokenAmount
	)
	for _, split := range splits {
		// validAmounts will be set to false if exist any amount < min part usd
		validAmounts := true
		cumulativeSumAmount = big.NewInt(0)
		scaledSplit = []poolPkg.TokenAmount{}

		for index, num := range split {
			percentOfAmountIn := big.NewInt(int64(uint32(num) * f.distributionPercent))
			amount := new(big.Int).Div(
				new(big.Int).Mul(percentOfAmountIn, totalAmountIn.Amount),
				big.NewInt(100),
			)
			amountUsd := utils.CalcTokenAmountUsd(amount, tokenInDecimal, tokenInPrice)
			// if this split amount by usd lower than MinPartUsd, then break and ignore that split
			// edge case: if MinPartUsd > AmountIn, only accept no split (split = 1).
			if amountUsd < f.minPartUSD && len(split) > 1 {
				validAmounts = false
				break
			}
			// Because we only take the integer part of the division of amount calculation (totalAmountIn * percentAmountIn / 100)
			//  multiplications here can result in a loss of precision in the amounts (e.g. taking 50% of 101)
			// This should be reconciled by adding the remainder to the last portion
			if index == len(split)-1 {
				amount = new(big.Int).Sub(totalAmountIn.Amount, cumulativeSumAmount)
				amountUsd = utils.CalcTokenAmountUsd(amount, tokenInDecimal, tokenInPrice)
			}
			cumulativeSumAmount = new(big.Int).Add(cumulativeSumAmount, amount)

			scaledSplit = append(scaledSplit, poolPkg.TokenAmount{
				Token:     totalAmountIn.Token,
				Amount:    amount,
				AmountUsd: amountUsd,
			})
		}

		// Only append to result if we generated valid amounts
		if validAmounts {
			result = append(result, scaledSplit)
			//fmt.Println(scaledSplit)
		}

	}
	return result, nil
}
