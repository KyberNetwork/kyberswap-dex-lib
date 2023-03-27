package common

import (
	"fmt"
	"math/big"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/core"
	poolPkg "github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/core/pool"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/usecase/findroute"
)

// BestPathAmongAddedPaths try to swap through a previously-found path by adding more amountIn to that path.
// Because we are reusing the path, we can disregard gas fee
func BestPathAmongAddedPaths(
	input findroute.Input,
	data findroute.FinderData,
	tokenAmountIn poolPkg.TokenAmount,
	addedPaths []core.Path,
) (*core.Path, error) {
	var (
		bestPath      *core.Path = nil
		bestAmountOut            = poolPkg.TokenAmount{
			Token:     input.TokenOutAddress,
			Amount:    constant.Zero,
			AmountUsd: 0,
		}

		amountOut poolPkg.TokenAmount
		err       error
	)

	for _, path := range addedPaths {
		amountOut, err = path.TrySwap(tokenAmountIn)
		if err != nil {
			continue
		}
		// only compare token amount (not AmountUsd) as fee should be disregarded here
		if amountOut.Token == input.TokenOutAddress && amountOut.Amount.Cmp(bestAmountOut.Amount) > 0 {
			bestAmountOut = amountOut
			bestPath = &path
		}
	}
	if bestPath == nil {
		return nil, fmt.Errorf("cannot find path among added paths")
	}

	// clone the best path and disregard gas fee as the path would be merged into an existing path anyway
	bestPath, err = core.NewPath(
		bestPath.Pools,
		bestPath.Tokens,
		tokenAmountIn,
		input.TokenOutAddress,
		data.PriceUSDByAddress[input.TokenOutAddress],
		data.TokenByAddress[input.TokenOutAddress].Decimals,
		core.GasOption{
			GasFeeInclude: false,
			Price:         big.NewFloat(0),
			TokenPrice:    0,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error initalizing new best path among added paths")
	}
	bestPath.TotalGas = 0
	return bestPath, nil
}
