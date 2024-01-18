package gyroeclp

import (
	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/gyroscope/math"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var (
	ErrPoolPaused         = errors.New("pool is paused")
	ErrTokenInIsNotToken0 = errors.New("TOKEN_IN_IS_NOT_TOKEN_0")
	ErrInvalidReserve     = errors.New("invalid reserve")
	ErrInvalidAmountIn    = errors.New("invalid amount in")
)

type PoolSimulator struct {
	poolpkg.Pool

	paused bool

	_paramsAlpha  *int256.Int
	_paramsBeta   *int256.Int
	_paramsC      *int256.Int
	_paramsS      *int256.Int
	_paramsLambda *int256.Int
	_tauAlphaX    *int256.Int
	_tauAlphaY    *int256.Int
	_tauBetaX     *int256.Int
	_tauBetaY     *int256.Int
	_u            *int256.Int
	_v            *int256.Int
	_w            *int256.Int
	_z            *int256.Int
	_dSq          *int256.Int

	swapFeePercentage *uint256.Int
	scalingFactors    []*uint256.Int

	vault  string
	poolID string
}

func (s *PoolSimulator) CalcAmountOut(params poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	if s.paused {
		return nil, ErrPoolPaused
	}

	var tokenInIsToken0 bool
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenOut)
	if indexIn == 0 && indexOut == 1 {
		tokenInIsToken0 = true
	} else if indexIn == 1 && indexOut == 0 {
		tokenInIsToken0 = false
	} else {
		return nil, ErrTokenInIsNotToken0
	}

	scalingFactorTokenIn := s._scalingFactor(tokenInIsToken0)
	scalingFactorTokenOut := s._scalingFactor(!tokenInIsToken0)

	balanceTokenIn, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexIn])
	if overflow {
		return nil, ErrInvalidReserve
	}
	balanceTokenOut, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexOut])
	if overflow {
		return nil, ErrInvalidReserve
	}

	balanceTokenIn, err := _upscale(balanceTokenIn, scalingFactorTokenIn)
	if err != nil {
		return nil, ErrInvalidReserve
	}
	balanceTokenOut, err = _upscale(balanceTokenOut, scalingFactorTokenOut)
	if err != nil {
		return nil, ErrInvalidReserve
	}

	balances := s._balancesFromTokenInOut(balanceTokenIn, balanceTokenOut, tokenInIsToken0)

	eclpParams, derivedECLPParams := s.reconstructECLPParams()

	invariant := &vector2{}
	{
		currentInvariant, invErr, err := GyroECLPMath.calculateInvariantWithError(
			balances, eclpParams, derivedECLPParams,
		)
		if err != nil {
			return nil, err
		}

		invariant.X = new(int256.Int).Add(
			currentInvariant,
			new(int256.Int).Mul(GyroECLPMath._number_2, invErr),
		)

		invariant.Y = currentInvariant
	}

	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow {
		return nil, ErrInvalidAmountIn
	}
	feeAmount, err := math.GyroFixedPoint.MulUp(amountIn, s.swapFeePercentage)
	if err != nil {
		return nil, err
	}
	amountInAfterFee, err := math.GyroFixedPoint.Sub(amountIn, feeAmount)
	if err != nil {
		return nil, err
	}
	amountInAfterFee, err = _upscale(amountInAfterFee, scalingFactorTokenIn)
	if err != nil {
		return nil, err
	}

	amountOut, err := GyroECLPMath.calcOutGivenIn(
		balances, amountInAfterFee, tokenInIsToken0, eclpParams, derivedECLPParams, invariant,
	)
	if err != nil {
		return nil, err
	}

	amountOut, err = _downscaleDown(amountOut, scalingFactorTokenOut)
	if err != nil {
		return nil, err
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: params.TokenOut, Amount: amountOut.ToBig()},
		Fee:            &poolpkg.TokenAmount{Token: params.TokenAmountIn.Token, Amount: feeAmount.ToBig()},
		Gas:            defaultGas.Swap,
	}, nil
}

func (s *PoolSimulator) reconstructECLPParams() (*params, *derivedParams) {
	p := &params{
		Alpha:  s._paramsAlpha,
		Beta:   s._paramsBeta,
		C:      s._paramsC,
		S:      s._paramsS,
		Lambda: s._paramsLambda,
	}

	dp := &derivedParams{
		TauAlpha: &vector2{
			X: s._tauAlphaX,
			Y: s._tauAlphaY,
		},
		TauBeta: &vector2{
			X: s._tauBetaX,
			Y: s._tauBetaY,
		},
		U:   s._u,
		V:   s._v,
		W:   s._w,
		Z:   s._z,
		DSq: s._dSq,
	}

	return p, dp

}

func (s *PoolSimulator) _balancesFromTokenInOut(
	balanceTokenIn *uint256.Int,
	balanceTokenOut *uint256.Int,
	tokenInIsToken0 bool,
) []*uint256.Int {
	balances := make([]*uint256.Int, 2)
	if tokenInIsToken0 {
		balances[0] = balanceTokenIn
		balances[1] = balanceTokenOut
	} else {
		balances[0] = balanceTokenOut
		balances[1] = balanceTokenIn
	}

	return balances
}

func (s *PoolSimulator) _scalingFactor(token0 bool) *uint256.Int {
	if token0 {
		return s.scalingFactors[0]
	}
	return s.scalingFactors[1]
}

func _upscale(amount, scalingFactor *uint256.Int) (*uint256.Int, error) {
	return math.GyroFixedPoint.MulDown(amount, scalingFactor)
}

func _downscaleDown(amount, scalingFactor *uint256.Int) (*uint256.Int, error) {
	return math.GyroFixedPoint.DivDown(amount, scalingFactor)
}
