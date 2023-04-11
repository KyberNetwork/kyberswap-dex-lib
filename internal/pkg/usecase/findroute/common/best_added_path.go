package common

import (
	"fmt"
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	poolPkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

// BestPathAmongAddedPaths try to swap through a previously-found path by adding more amountIn to that path.
// Because we are reusing the path, we can disregard gas fee
func BestPathAmongAddedPaths(
	input findroute.Input,
	data findroute.FinderData,
	tokenAmountIn poolPkg.TokenAmount,
	addedPaths []*valueobject.Path,
) (*valueobject.Path, error) {
	var (
		bestPath      *valueobject.Path = nil
		bestAmountOut                   = poolPkg.TokenAmount{
			Token:     input.TokenOutAddress,
			Amount:    constant.Zero,
			AmountUsd: 0,
		}

		amountOut poolPkg.TokenAmount
		err       error
	)

	for _, path := range addedPaths {
		amountOut, _, err = path.CalcAmountOut(data.PoolBucket, tokenAmountIn)
		if err != nil {
			continue
		}
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
		data.PoolBucket,
		bestPath.PoolAddresses,
		bestPath.Tokens,
		tokenAmountIn,
		input.TokenOutAddress,
		data.PriceUSDByAddress[input.TokenOutAddress],
		data.TokenByAddress[input.TokenOutAddress].Decimals,
		valueobject.GasOption{
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
