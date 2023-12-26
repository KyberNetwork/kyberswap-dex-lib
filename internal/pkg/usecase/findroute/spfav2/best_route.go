package spfav2

import (
	"context"
	"math"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/common"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func (f *spfav2Finder) bestRouteExactIn(ctx context.Context, input findroute.Input, data findroute.FinderData) (*valueobject.Route, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "spfav2Finder.bestRouteExactIn")
	defer span.End()

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
	for poolAddress := range data.PoolBucket.PerRequestPoolsByAddress {
		for _, fromToken := range data.PoolBucket.PerRequestPoolsByAddress[poolAddress].GetTokens() {
			tokenToPoolAddress[fromToken] = append(tokenToPoolAddress[fromToken], poolAddress)
		}
	}

	hopsToTokenOut, err := f.findMinHopsToTokenOut(
		data.PoolBucket.PerRequestPoolsByAddress,
		data.TokenByAddress,
		tokenToPoolAddress,
		input.TokenInAddress,
		input.TokenOutAddress,
	)
	if err != nil {
		return nil, err
	}

	if minHopFromTokenIn, ok := hopsToTokenOut[input.TokenInAddress]; !ok || minHopFromTokenIn > f.maxHops {
		return nil, nil
	}

	// it is fine if prices[token] is not set because it would default to zero
	tokenAmountIn := poolpkg.TokenAmount{
		Token:     input.TokenInAddress,
		Amount:    input.AmountIn,
		AmountUsd: utils.CalcTokenAmountUsd(input.AmountIn, data.TokenByAddress[input.TokenInAddress].Decimals, data.PriceUSDByAddress[input.TokenInAddress]),
	}

	if f.minThresholdAmountInUSD <= tokenAmountIn.AmountUsd && tokenAmountIn.AmountUsd <= f.maxThresholdAmountInUSD {
		return f.findrouteV2(ctx, input, data, tokenAmountIn, tokenToPoolAddress, hopsToTokenOut)
	} else {
		return f.findrouteV1(ctx, input, data, tokenAmountIn, tokenToPoolAddress, hopsToTokenOut)
	}
}

func (f *spfav2Finder) findMinHopsToTokenOut(
	poolByAddress map[string]poolpkg.IPoolSimulator,
	tokenByAddress map[string]entity.Token,
	tokenToPoolAddresses map[string][]string,
	tokenIn string,
	tokenOut string,
) (map[string]uint32, error) {
	if len(f.whitelistedTokenSet) == 0 {
		return common.MinHopsToTokenOut(
			poolByAddress,
			tokenByAddress,
			tokenToPoolAddresses,
			tokenOut,
		)
	}

	return common.MinHopsToTokenOutWithWhitelist(
		poolByAddress,
		tokenByAddress,
		tokenToPoolAddresses,
		f.whitelistedTokenSet,
		tokenIn,
		tokenOut,
	)
}

// split amount in into portions of f.distributionPercent% such that each split has value >= minUsdPerSplit
// if there are remaining amount after splitting, we add to the first split (because it is always the best possible path)
// e.g. distributionPercent = 10, but we need 30% amountIn to be > minUsdPerSplit -> split 40, 30, 30
func (f *spfav2Finder) splitAmountIn(input findroute.Input, data findroute.FinderData, totalAmountIn poolpkg.TokenAmount) []poolpkg.TokenAmount {
	tokenInPrice := data.PriceUSDByAddress[input.TokenInAddress]
	tokenInDecimal := data.TokenByAddress[input.TokenInAddress].Decimals

	if f.distributionPercent == constant.OneHundredPercent || tokenInPrice == 0 || totalAmountIn.AmountUsd <= f.minPartUSD {
		return []poolpkg.TokenAmount{totalAmountIn}
	}
	var (
		amountInBigInt = totalAmountIn.Amount
		amountInUsd    = totalAmountIn.AmountUsd

		// f.distributionPercent should be a divisor of 100
		// maxNumSplits is the max number of splits with each split contains a portion of f.distributionPercent% of amountIn
		// But we need to account for the f.MinPartUsd requirement by merging these splits
		maxNumSplits = int64(constant.OneHundredPercent / f.distributionPercent)

		amountInPerSplit = new(big.Int).Div(amountInBigInt, big.NewInt(maxNumSplits))
	)
	if amountInPerSplit.Cmp(constant.Zero) == 0 {
		amountInPerSplit = constant.One
	}

	var amountInPerSplitUsd = utils.CalcTokenAmountUsd(amountInPerSplit, tokenInDecimal, tokenInPrice)

	var minSplitsToMeetMinUsdRequirement int64
	if utils.Float64AlmostEqual(amountInPerSplitUsd, 0) {
		minSplitsToMeetMinUsdRequirement = maxNumSplits
	} else {
		minSplitsToMeetMinUsdRequirement = int64(math.Max(math.Ceil(f.minPartUSD/amountInPerSplitUsd), 1))
	}

	var (
		// the actual number of splits that we would make, considering the f.MinPartUSD requirement
		trueNumSplits           = maxNumSplits / minSplitsToMeetMinUsdRequirement
		trueAmountInPerSplit    = new(big.Int).Mul(amountInPerSplit, big.NewInt(minSplitsToMeetMinUsdRequirement))
		trueAmountInPerSplitUsd = amountInPerSplitUsd * float64(minSplitsToMeetMinUsdRequirement)

		// remaining amount after split, will be added to the first split
		remainingAmountIn    = new(big.Int).Sub(amountInBigInt, new(big.Int).Mul(trueAmountInPerSplit, big.NewInt(trueNumSplits)))
		remainingAmountInUsd = amountInUsd - trueAmountInPerSplitUsd*float64(trueNumSplits)

		splits = make([]poolpkg.TokenAmount, trueNumSplits)
	)

	splits[0] = poolpkg.TokenAmount{
		Token:     totalAmountIn.Token,
		Amount:    new(big.Int).Add(trueAmountInPerSplit, remainingAmountIn),
		AmountUsd: trueAmountInPerSplitUsd + remainingAmountInUsd,
	}
	for i := 1; i < int(trueNumSplits); i++ {
		splits[i] = poolpkg.TokenAmount{
			Token:     totalAmountIn.Token,
			Amount:    new(big.Int).Set(trueAmountInPerSplit),
			AmountUsd: trueAmountInPerSplitUsd,
		}
	}
	return splits
}
