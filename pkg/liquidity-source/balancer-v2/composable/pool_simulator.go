package composable

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/math"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolSimulator struct {
	poolpkg.Pool

	regularSimulator *regularSimulator
	bptSimulator     *bptSimulator
}

func (s *PoolSimulator) CalcAmountOut(
	tokenAmountIn poolpkg.TokenAmount,
	tokenOut string,
) (*poolpkg.CalcAmountOutResult, error) {
	indexIn := s.GetTokenIndex(tokenAmountIn.Token)
	indexOut := s.GetTokenIndex(tokenOut)
	if indexIn == unknownInt || indexOut == unknownInt {
		return nil, ErrUnknownToken
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrOverflow
	}

	balances := make([]*uint256.Int, len(s.Info.Reserves))
	for i, reserve := range s.Info.Reserves {
		r, overflow := uint256.FromBig(reserve)
		if overflow {
			return nil, ErrOverflow
		}
		balances[i] = r
	}

	var (
		amountOut *uint256.Int
		fee       *poolpkg.TokenAmount
		err       error
	)
	if tokenAmountIn.Token == s.Info.Address || tokenOut == s.Info.Address {
		amountOut, fee, err = s.bptSimulator.swap(amountIn, balances, indexIn, indexOut)
	} else {
		amountOut, fee, err = s.regularSimulator.swap(amountIn, balances, indexIn, indexOut)
	}
	if err != nil {
		return nil, err
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: fee,
		Gas: DefaultGas.Swap,
	}, nil
}

func _downscaleDown(amount *uint256.Int, scalingFactor *uint256.Int) (*uint256.Int, error) {
	return math.FixedPoint.DivDown(amount, scalingFactor)
}

func _upscaleArray(balances []*uint256.Int, scalingFactors []*uint256.Int) ([]*uint256.Int, error) {
	upscaled := make([]*uint256.Int, len(balances))
	for i, balance := range balances {
		upscaledI, err := _upscale(balance, scalingFactors[i])
		if err != nil {
			return nil, err
		}
		upscaled[i] = upscaledI
	}
	return upscaled, nil
}

func _upscale(amount *uint256.Int, scalingFactor *uint256.Int) (*uint256.Int, error) {
	return math.FixedPoint.MulDown(amount, scalingFactor)
}

func _dropBptItem(amounts []*uint256.Int, bptIndex int) []*uint256.Int {
	amountsWithoutBpt := make([]*uint256.Int, len(amounts)-1)

	for i := 0; i < len(amountsWithoutBpt); i++ {
		if i < bptIndex {
			amountsWithoutBpt[i] = amounts[i]
			continue
		}
		amountsWithoutBpt[i] = amounts[i+1]
	}

	return amountsWithoutBpt
}
