package gyroscope2clp

import (
	"errors"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/gyroscope/math"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var (
	ErrPoolPaused      = errors.New("pool is paused")
	ErrInvalidToken    = errors.New("invalid token")
	ErrInvalidReserve  = errors.New("invalid reserve")
	ErrInvalidAmountIn = errors.New("invalid amount in")
)

type PoolSimulator struct {
	poolpkg.Pool

	// paused: `getPausedState`
	paused bool

	// scalingFactors: 10^(18-decimals)
	scalingFactors []*uint256.Int

	// swapFeePercentage: `getMicsData`
	swapFeePercentage *uint256.Int

	// sqrtParameters: `getSqrtParameters`
	sqrtParameters []*uint256.Int
}

func (s *PoolSimulator) CalcAmountOut(
	params poolpkg.CalcAmountOutParams,
) (*poolpkg.CalcAmountOutResult, error) {
	if s.paused {
		return nil, ErrPoolPaused
	}

	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenOut)
	if indexIn == -1 || indexOut == -1 {
		return nil, ErrInvalidToken
	}

	scalingFactorTokenIn, scalingFactorTokenOut := s.scalingFactors[indexIn], s.scalingFactors[indexOut]

	balanceTokenIn, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexIn])
	if overflow {
		return nil, ErrInvalidReserve
	}

	balanceTokenOut, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexOut])
	if overflow {
		return nil, ErrInvalidReserve
	}

	balanceTokenIn, err := s._upscale(balanceTokenIn, scalingFactorTokenIn)
	if err != nil {
		return nil, ErrInvalidReserve
	}

	balanceTokenOut, err = s._upscale(balanceTokenOut, scalingFactorTokenOut)
	if err != nil {
		return nil, ErrInvalidReserve
	}

	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow {
		return nil, ErrInvalidAmountIn
	}

	_, virtualParamIn, virtualParamOut, err := s._calculateCurrentValues(balanceTokenIn, balanceTokenOut, indexIn == 0)
	if err != nil {
		return nil, err
	}

	feeAmount, err := math.GyroFixedPoint.MulUp(amountIn, s.swapFeePercentage)
	if err != nil {
		return nil, err
	}

	amountInAfterFee, err := math.GyroFixedPoint.Sub(amountIn, feeAmount)
	if err != nil {
		return nil, err
	}

	amountInAfterFee, err = s._upscale(amountInAfterFee, scalingFactorTokenIn)
	if err != nil {
		return nil, err
	}

	amountOut, err := Gyro2CLPMath._calcOutGivenIn(balanceTokenIn, balanceTokenOut, amountInAfterFee, virtualParamIn, virtualParamOut)
	if err != nil {
		return nil, err
	}

	amountOut, err = s._downscaleDown(amountOut, scalingFactorTokenOut)
	if err != nil {
		return nil, err
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: params.TokenOut, Amount: amountOut.ToBig()},
	}, nil
}

func (s *PoolSimulator) _upscale(amount, scalingFactor *uint256.Int) (*uint256.Int, error) {
	return math.GyroFixedPoint.MulDown(amount, scalingFactor)
}

func (s *PoolSimulator) _downscaleDown(amount, scalingFactor *uint256.Int) (*uint256.Int, error) {
	return math.GyroFixedPoint.DivDown(amount, scalingFactor)
}

func (s *PoolSimulator) _calculateCurrentValues(
	balanceTokenIn,
	balanceTokenOut *uint256.Int,
	tokenInIsToken0 bool,
) (
	*uint256.Int,
	*uint256.Int,
	*uint256.Int,
	error,
) {
	var balances []*uint256.Int
	if tokenInIsToken0 {
		balances = []*uint256.Int{balanceTokenIn, balanceTokenOut}
	} else {
		balances = []*uint256.Int{balanceTokenOut, balanceTokenIn}
	}

	currentInvariant, err := Gyro2CLPMath._calculateInvariant(balances, s.sqrtParameters[0], s.sqrtParameters[1])
	if err != nil {
		return nil, nil, nil, err
	}

	virtualParam, err := s._getVirtualParameters(s.sqrtParameters, currentInvariant)
	if err != nil {
		return nil, nil, nil, err
	}

	var virtualParamIn, virtualParamOut *uint256.Int
	if tokenInIsToken0 {
		virtualParamIn, virtualParamOut = virtualParam[0], virtualParam[1]
	} else {
		virtualParamIn, virtualParamOut = virtualParam[1], virtualParam[0]
	}

	return currentInvariant, virtualParamIn, virtualParamOut, nil
}

// _getVirtualParameters
// https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/contracts/2clp/Gyro2CLPPool.sol#L108C14-L108C35
func (s *PoolSimulator) _getVirtualParameters(sqrtParams []*uint256.Int, invariant *uint256.Int) ([]*uint256.Int, error) {
	virtualParameters0, err := s._virtualParameters(true, sqrtParams[1], invariant)
	if err != nil {
		return nil, err
	}

	virtualParameters1, err := s._virtualParameters(false, sqrtParams[0], invariant)
	if err != nil {
		return nil, err
	}

	return []*uint256.Int{virtualParameters0, virtualParameters1}, nil
}

// _virtualParameters
// https://github.com/gyrostable/concentrated-lps/blob/7e9bd3b20dd52663afca04ca743808b1d6a9521f/contracts/2clp/Gyro2CLPPool.sol#L119
func (s *PoolSimulator) _virtualParameters(parameter0 bool, sqrtParam, invariant *uint256.Int) (*uint256.Int, error) {
	if parameter0 {
		return Gyro2CLPMath._calculateVirtualParameter0(invariant, sqrtParam)
	}

	return Gyro2CLPMath._calculateVirtualParameter1(invariant, sqrtParam)
}
