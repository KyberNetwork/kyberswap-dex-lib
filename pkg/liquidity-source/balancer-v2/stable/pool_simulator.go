package stable

import (
	"math/big"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/math"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolSimulator struct {
	poolpkg.Pool

	// poolID       string
	// vaultAddress string

	swapFeePercentage *uint256.Int
	amp               *uint256.Int
	scalingFactors    []*uint256.Int
}

func (s *PoolSimulator) CalcAmountOut(
	tokenAmountIn poolpkg.TokenAmount,
	tokenOut string,
) (*poolpkg.CalcAmountOutResult, error) {
	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)
	if indexIn == -1 || indexOut == -1 {
		return nil, ErrTokenNotRegistered
	}

	scalingFactorTokenIn := s.scalingFactors[indexIn]
	scalingFactorTokenOut := s.scalingFactors[indexOut]

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrInvalidAmountIn
	}
	feeAmount, err := math.FixedPoint.MulUp(amountIn, s.swapFeePercentage)
	if err != nil {
		return nil, err
	}
	amountInAfterFee, err := math.FixedPoint.Sub(amountIn, feeAmount)
	if err != nil {
		return nil, err
	}
	upScaledAmountIn, err := _upscale(amountInAfterFee, scalingFactorTokenIn)
	if err != nil {
		return nil, err
	}

	balances, err := _upscaleArray(s.Info.Reserves, s.scalingFactors)
	if err != nil {
		return nil, err
	}

	upScaledAmountOut, err := StableMath._calcOutGivenIn(
		s.amp,
		upScaledAmountIn,
		balances,
		indexIn,
		indexOut,
	)
	if err != nil {
		return nil, err
	}

	amountOut, err := _downscaleDown(upScaledAmountOut, scalingFactorTokenOut)
	if err != nil {
		return nil, err
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
		Gas:            defaultGas.Swap,
	}, nil
}

func _upscaleArray(reserves []*big.Int, scalingFactors []*uint256.Int) ([]*uint256.Int, error) {
	upscaled := make([]*uint256.Int, len(reserves))
	for i, reserve := range reserves {
		r, overflow := uint256.FromBig(reserve)
		if overflow {
			return nil, ErrInvalidReserve
		}

		upscaledI, err := _upscale(r, scalingFactors[i])
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

func _downscaleDown(amount *uint256.Int, scalingFactor *uint256.Int) (*uint256.Int, error) {
	return math.FixedPoint.DivDown(amount, scalingFactor)
}
