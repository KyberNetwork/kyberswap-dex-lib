package common

import (
	"context"
	"fmt"
	"math/big"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

// BestPathAmongAddedPaths try to swap through a previously-found path by adding more amountIn to that path.
// Because we are reusing the path, we can disregard gas fee
func BestPathAmongAddedPaths(
	ctx context.Context,
	input findroute.Input,
	data findroute.FinderData,
	tokenAmountIn valueobject.TokenAmount,
	addedPaths []*valueobject.Path,
) (*valueobject.Path, error) {
	var (
		bestPath      *valueobject.Path
		bestAmountOut = valueobject.TokenAmount{
			Token:     input.TokenOutAddress,
			Amount:    constant.Zero,
			AmountUsd: 0,
		}

		err error

		intermediateResults sync.Map // map[int]pookpkg.TokenAmount
		wg                  errgroup.Group
	)

	for itr, path := range addedPaths {
		_itr, _path := itr, path
		wg.Go(func() error {
			amountOut, _, err := _path.CalcAmountOut(ctx, data.PoolBucket, tokenAmountIn, data.SwapLimits)
			if err != nil {
				return nil
			}
			intermediateResults.Store(_itr, amountOut)
			return nil
		})
	}

	wg.Wait()

	for itr, path := range addedPaths {
		_amountOut, ok := intermediateResults.Load(itr)
		if !ok {
			continue
		}
		amountOut := _amountOut.(valueobject.TokenAmount)
		// only compare token amount (not AmountUsd) as fee should be disregarded here
		if amountOut.Token == input.TokenOutAddress && amountOut.Amount.Cmp(bestAmountOut.Amount) > 0 {
			bestAmountOut = amountOut
			bestPath = path
		}
	}
	if bestPath == nil {
		return nil, fmt.Errorf("cannot find path among added paths")
	}

	// clone the best path and disregard gas fee as the path would be merged into an existing path anyway
	bestPath, err = valueobject.NewPath(
		ctx,
		data.PoolBucket,
		bestPath.PoolAddresses,
		bestPath.Tokens,
		tokenAmountIn,
		input.TokenOutAddress,
		data.PriceUSDByAddress[input.TokenOutAddress],
		data.TokenNativeBuyPrice(input.TokenOutAddress),
		data.TokenByAddress[input.TokenOutAddress].Decimals,
		valueobject.GasOption{
			GasFeeInclude: false,
			Price:         big.NewFloat(0),
			TokenPrice:    0,
		},
		data.SwapLimits,
	)
	if err != nil {
		return nil, fmt.Errorf("error initalizing new best path among added paths")
	}
	bestPath.TotalGas = 0
	return bestPath, nil
}
