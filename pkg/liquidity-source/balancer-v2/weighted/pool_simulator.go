package weighted

import (
	"errors"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/math"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var (
	ErrTokenNotRegistered = errors.New("TOKEN_NOT_REGISTERED")
	ErrInvalidReserve     = errors.New("invalid reserve")
	ErrInvalidAmountIn    = errors.New("invalid amount in")
)

var (
	defaultGas = Gas{Swap: 10}
)

type (
	PoolSimulator struct {
		poolpkg.Pool

		// poolID       string
		// vaultAddress string

		swapFeePercentage *uint256.Int
		scalingFactors    []*uint256.Int
		normalizedWeights []*uint256.Int
	}
	Gas struct {
		Swap int64
	}
)

func (s *PoolSimulator) CalcAmountOut(
	tokenAmountIn poolpkg.TokenAmount,
	tokenOut string,
) (*poolpkg.CalcAmountOutResult, error) {
	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)

	if indexIn == -1 || indexOut == -1 {
		return nil, ErrTokenNotRegistered
	}

	reserveIn, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexIn])
	if overflow {
		return nil, ErrInvalidReserve
	}

	reserveOut, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexOut])
	if overflow {
		return nil, ErrInvalidReserve
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrInvalidAmountIn
	}

	scalingFactorTokenIn := s.scalingFactors[indexIn]
	scalingFactorTokenOut := s.scalingFactors[indexOut]
	normalizedWeightIn := s.normalizedWeights[indexIn]
	normalizedWeightOut := s.normalizedWeights[indexOut]

	balanceTokenIn, err := _upscale(reserveIn, scalingFactorTokenIn)
	if err != nil {
		return nil, err
	}
	balanceTokenOut, err := _upscale(reserveOut, scalingFactorTokenOut)
	if err != nil {
		return nil, err
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

	upScaledAmountOut, err := math.WeightedMath.CalcOutGivenIn(
		balanceTokenIn,
		normalizedWeightIn,
		balanceTokenOut,
		normalizedWeightOut,
		upScaledAmountIn,
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

func _upscale(amount *uint256.Int, scalingFactor *uint256.Int) (*uint256.Int, error) {
	return math.FixedPoint.MulDown(amount, scalingFactor)
}

func _downscaleDown(amount *uint256.Int, scalingFactor *uint256.Int) (*uint256.Int, error) {
	return math.FixedPoint.DivDown(amount, scalingFactor)
}
