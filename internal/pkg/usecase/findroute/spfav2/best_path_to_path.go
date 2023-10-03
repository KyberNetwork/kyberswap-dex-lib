package spfav2

import (
	"context"
	"fmt"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/pkg/errors"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

func bestPathToPath(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
	tokenAmountIn poolpkg.TokenAmount,
	bestPaths []*entity.MinimalPath,
) []*valueobject.Path {
	var paths []*valueobject.Path
	span, _ := tracer.StartSpanFromContext(ctx, "spfav2Finder.bestPathToPath")
	defer span.Finish()
	calcAmountOutResultBySwapKey := make(map[string]*poolpkg.CalcAmountOutResult)

	for _, bestPath := range bestPaths {
		// construct path from bestPath and perform some sanity check regarding pool length and token addresses
		path, err := newPathFromBestPathWithoutInputOutput(data.TokenByAddress, input.TokenInAddress, input.TokenOutAddress, bestPath)
		if err != nil {
			logger.WithFields(logger.Fields{"error": err}).Debug("[spfav2Finder.bestPathToPath] does not pass sanity check")
			continue
		}

		path.Input = tokenAmountIn

		// calcAmountOut of the path, cache results of `pool.calcAmountOut` to save computation
		// if the path requires calculating amount out for same (tokenIn, tokenOut, pool, amount) (share the same pools with previously calculated paths),
		// we will read previous calcAmountOutResult from calcAmountOutResultBySwapKey
		// otherwise, compute as usual and save the result to calcAmountOutResultBySwapKey
		tokenAmountOut, totalGas, err := calcAmountOutAndUpdateCache(data.PoolBucket, tokenAmountIn, path, calcAmountOutResultBySwapKey)
		if err != nil {
			logger.WithFields(logger.Fields{"error": err}).Debug("[spfav2Finder.bestPathToPath] failed to calcAmountOut")
			continue
		}

		amountUSD := utils.CalcTokenAmountUsd(
			tokenAmountOut.Amount,
			data.TokenByAddress[input.TokenOutAddress].Decimals,
			data.PriceUSDByAddress[input.TokenOutAddress],
		)
		totalGasUSD := utils.CalcGasUsd(input.GasPrice, totalGas, input.GasTokenPriceUSD)
		tokenAmountOut.AmountUsd = amountUSD - totalGasUSD

		path.Output = tokenAmountOut
		path.TotalGas = totalGas

		paths = append(paths, path)
	}
	// fmt.Println("Number of calcAmountOut", len(calcAmountOutResultBySwapKey))
	return paths
}

func getSwapKey(tokenInAddress, tokenOutAddress, poolAddress, amountInStr string) string {
	return fmt.Sprintf("%s_%s_%s_%s", tokenInAddress, tokenOutAddress, poolAddress, amountInStr)
}

func newPathFromBestPathWithoutInputOutput(tokenByAddress map[string]entity.Token, tokenInAddress, tokenOutAddress string, bestPath *entity.MinimalPath) (*valueobject.Path, error) {
	var (
		tokenLen = len(bestPath.Tokens)
		poolLen  = len(bestPath.Pools)
	)

	if tokenLen < 2 {
		return nil, valueobject.ErrInvalidTokenLength
	}

	if poolLen+1 != tokenLen {
		return nil, valueobject.ErrInvalidPoolLength
	}

	if bestPath.Tokens[0] != tokenInAddress {
		return nil, valueobject.ErrInvalidTokenIn
	}

	if bestPath.Tokens[tokenLen-1] != tokenOutAddress {
		return nil, valueobject.ErrInvalidTokenOut
	}

	var tokens []entity.Token
	for _, tokenAddress := range bestPath.Tokens {
		token, ok := tokenByAddress[tokenAddress]
		if !ok {
			return nil, fmt.Errorf("cannot get info for token %v", tokenAddress)
		}
		tokens = append(tokens, token)
	}
	return &valueobject.Path{
		PoolAddresses: bestPath.Pools,
		Tokens:        tokens,
	}, nil
}

func calcAmountOutAndUpdateCache(
	poolBucket *valueobject.PoolBucket,
	tokenAmountIn poolpkg.TokenAmount,
	p *valueobject.Path,
	calcAmountOutResultBySwapKey map[string]*poolpkg.CalcAmountOutResult,
) (poolpkg.TokenAmount, int64, error) {
	var (
		currentAmount       = tokenAmountIn
		calcAmountOutResult *poolpkg.CalcAmountOutResult
		err                 error
		totalGas            int64
	)

	for i, poolAddr := range p.PoolAddresses {
		pool, ok := poolBucket.PerRequestPoolsByAddress[poolAddr]
		if !ok {
			return poolpkg.TokenAmount{}, 0, errors.Wrapf(
				findroute.ErrNoIPool,
				"[spfav2Finder.calcAmountOutAndUpdateCache] poolAddress: [%s]",
				poolAddr,
			)
		}
		// first, check if we performed the same calculation before
		// if yes, use the previous result
		swapKey := getSwapKey(p.Tokens[i].Address, p.Tokens[i+1].Address, poolAddr, currentAmount.Amount.String())
		calcAmountOutResult, ok = calcAmountOutResultBySwapKey[swapKey]
		// if no, then calculate amountOut as usual and save the result to `calcAmountOutResultBySwapKey`
		if !ok {
			calcAmountOutResult, err = pool.CalcAmountOut(currentAmount, p.Tokens[i+1].Address)
			if err != nil {
				return poolpkg.TokenAmount{}, 0, errors.Wrapf(
					valueobject.ErrInvalidSwap,
					"[spfav2Finder.calcAmountOutAndUpdateCache] CalcAmountOut returns error | poolAddress: [%s], exchange: [%s], tokenIn: [%s], amountIn: [%s], tokenOut: [%s], err: [%v]",
					pool.GetAddress(),
					pool.GetExchange(),
					currentAmount.Token,
					currentAmount.Amount,
					p.Tokens[i+1].Address,
					err,
				)
			}

			// save the result
			calcAmountOutResultBySwapKey[swapKey] = calcAmountOutResult
		}

		swapTokenAmountOut, gas := calcAmountOutResult.TokenAmountOut, calcAmountOutResult.Gas
		if swapTokenAmountOut == nil {
			return poolpkg.TokenAmount{}, 0, errors.Wrapf(
				valueobject.ErrInvalidSwap,
				"[spfav2Finder.calcAmountOutAndUpdateCache] returns nil | poolAddress: [%s], exchange: [%s], tokenIn: [%s], amountIn: [%s], tokenOut: [%s]",
				pool.GetAddress(),
				pool.GetExchange(),
				currentAmount.Token,
				currentAmount.Amount,
				p.Tokens[i+1].Address,
			)
		}

		currentAmount = *swapTokenAmountOut
		totalGas += gas
	}

	return currentAmount, totalGas, nil
}
