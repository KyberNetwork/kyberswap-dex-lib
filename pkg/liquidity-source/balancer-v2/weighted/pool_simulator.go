//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple PoolSimulator
//msgp:ignore Gas
//msgp:shim *uint256.Int as:[]byte using:msgpencode.EncodeUint256/msgpencode.DecodeUint256

package weighted

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/math"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrTokenNotRegistered       = errors.New("TOKEN_NOT_REGISTERED")
	ErrInvalidReserve           = errors.New("invalid reserve")
	ErrInvalidAmountIn          = errors.New("invalid amount in")
	ErrInvalidSwapFeePercentage = errors.New("invalid swap fee percentage")
	ErrPoolPaused               = errors.New("pool is paused")
	ErrMaxTotalInRatio          = errors.New("MAX_TOTAL_IN_RATIO")
	ErrMaxTotalOutRatio         = errors.New("MAX_TOTAL_OUT_RATIO")
	ErrOverflow                 = errors.New("OVERFLOW")
)

var (
	defaultGas = Gas{Swap: 80000}

	_MAX_IN_RATIO  = uint256.NewInt(0.3e18)
	_MAX_OUT_RATIO = uint256.NewInt(0.3e18)
)

type (
	PoolSimulator struct {
		poolpkg.Pool

		paused bool

		swapFeePercentage *uint256.Int
		scalingFactors    []*uint256.Int
		normalizedWeights []*uint256.Int

		vault       string
		poolID      string
		poolTypeVer int

		totalAmountsIn          []*uint256.Int
		scaledMaxTotalAmountsIn []*uint256.Int

		totalAmountsOut          []*uint256.Int
		scaledMaxTotalAmountsOut []*uint256.Int
	}

	Gas struct {
		Swap int64
	}
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var (
		extra       Extra
		staticExtra StaticExtra

		tokens   = make([]string, len(entityPool.Tokens))
		reserves = make([]*big.Int, len(entityPool.Tokens))

		totalAmountsIn          = make([]*uint256.Int, len(entityPool.Tokens))
		scaledMaxTotalAmountsIn = make([]*uint256.Int, len(entityPool.Tokens))

		totalAmountsOut          = make([]*uint256.Int, len(entityPool.Tokens))
		scaledMaxTotalAmountsOut = make([]*uint256.Int, len(entityPool.Tokens))
	)

	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	for idx := 0; idx < len(entityPool.Tokens); idx++ {
		tokens[idx] = entityPool.Tokens[idx].Address
		reserves[idx] = bignumber.NewBig10(entityPool.Reserves[idx])
	}

	scaledInitialBalances, err := _upscaleArray(staticExtra.PoolTypeVer, reserves, staticExtra.ScalingFactors)
	if err != nil {
		return nil, err
	}
	for idx := 0; idx < len(entityPool.Tokens); idx++ {
		totalAmountsIn[idx] = number.Zero
		totalAmountsOut[idx] = number.Zero

		maxIn, err := math.FixedPoint.MulDown(scaledInitialBalances[idx], _MAX_IN_RATIO)
		if err != nil {
			return nil, err
		}
		scaledMaxTotalAmountsIn[idx] = maxIn

		maxOut, err := math.FixedPoint.MulDown(scaledInitialBalances[idx], _MAX_OUT_RATIO)
		if err != nil {
			return nil, err
		}
		scaledMaxTotalAmountsOut[idx] = maxOut
	}

	poolInfo := poolpkg.PoolInfo{
		Address:     entityPool.Address,
		Exchange:    entityPool.Exchange,
		Type:        entityPool.Type,
		Tokens:      tokens,
		Reserves:    reserves,
		Checked:     true,
		BlockNumber: uint64(entityPool.BlockNumber),
	}

	return &PoolSimulator{
		Pool:                     poolpkg.Pool{Info: poolInfo},
		paused:                   extra.Paused,
		swapFeePercentage:        extra.SwapFeePercentage,
		scalingFactors:           staticExtra.ScalingFactors,
		normalizedWeights:        staticExtra.NormalizedWeights,
		vault:                    staticExtra.Vault,
		poolID:                   staticExtra.PoolID,
		poolTypeVer:              staticExtra.PoolTypeVer,
		totalAmountsIn:           totalAmountsIn,
		scaledMaxTotalAmountsIn:  scaledMaxTotalAmountsIn,
		totalAmountsOut:          totalAmountsOut,
		scaledMaxTotalAmountsOut: scaledMaxTotalAmountsOut,
	}, nil
}

// https://etherscan.io/address/0x065f5b35d4077334379847fe26f58b1029e51161#code#F7#L32
func (s *PoolSimulator) CalcAmountOut(params poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	if s.paused {
		return nil, ErrPoolPaused
	}

	tokenAmountIn := params.TokenAmountIn
	tokenOut := params.TokenOut

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

	balanceTokenIn, err := _upscale(s.poolTypeVer, reserveIn, scalingFactorTokenIn)
	if err != nil {
		return nil, err
	}
	balanceTokenOut, err := _upscale(s.poolTypeVer, reserveOut, scalingFactorTokenOut)
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

	if err := s.validateMaxInRatio(indexIn, amountInAfterFee); err != nil {
		return nil, err
	}

	upScaledAmountIn, err := _upscale(s.poolTypeVer, amountInAfterFee, scalingFactorTokenIn)
	if err != nil {
		return nil, err
	}

	upScaledAmountOut, err := s._onSwapGivenIn(
		balanceTokenIn,
		normalizedWeightIn,
		balanceTokenOut,
		normalizedWeightOut,
		upScaledAmountIn,
	)
	if err != nil {
		return nil, err
	}

	amountOut, err := _downscaleDown(s.poolTypeVer, upScaledAmountOut, scalingFactorTokenOut)
	if err != nil {
		return nil, err
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &poolpkg.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: feeAmount.ToBig(),
		},
		Gas: defaultGas.Swap,
	}, nil
}

func (s *PoolSimulator) CalcAmountIn(params poolpkg.CalcAmountInParams) (*poolpkg.CalcAmountInResult, error) {
	if s.paused {
		return nil, ErrPoolPaused
	}

	tokenAmountOut := params.TokenAmountOut
	tokenIn := params.TokenIn

	indexIn, indexOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenAmountOut.Token)

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

	amountOut, overflow := uint256.FromBig(tokenAmountOut.Amount)
	if overflow {
		return nil, ErrInvalidAmountIn
	}

	if err := s.validateMaxOutRatio(indexOut, amountOut); err != nil {
		return nil, err
	}

	scalingFactorTokenIn := s.scalingFactors[indexIn]
	scalingFactorTokenOut := s.scalingFactors[indexOut]
	normalizedWeightIn := s.normalizedWeights[indexIn]
	normalizedWeightOut := s.normalizedWeights[indexOut]

	balanceTokenIn, err := _upscale(s.poolTypeVer, reserveIn, scalingFactorTokenIn)
	if err != nil {
		return nil, err
	}
	balanceTokenOut, err := _upscale(s.poolTypeVer, reserveOut, scalingFactorTokenOut)
	if err != nil {
		return nil, err
	}

	upScaledAmountOut, err := _upscale(s.poolTypeVer, amountOut, scalingFactorTokenOut)
	if err != nil {
		return nil, err
	}

	upScaledAmountIn, err := s._onSwapGivenOut(
		balanceTokenIn,
		normalizedWeightIn,
		balanceTokenOut,
		normalizedWeightOut,
		upScaledAmountOut,
	)
	if err != nil {
		return nil, err
	}

	amountIn, err := _downscaleUp(s.poolTypeVer, upScaledAmountIn, scalingFactorTokenIn)
	if err != nil {
		return nil, err
	}

	amountInAfterFee, err := s._addSwapFeeAmount(amountIn)
	if err != nil {
		return nil, err
	}

	feeAmount, err := math.FixedPoint.Sub(amountInAfterFee, amountIn)
	if err != nil {
		return nil, err
	}

	return &poolpkg.CalcAmountInResult{
		TokenAmountIn: &poolpkg.TokenAmount{
			Token:  tokenIn,
			Amount: amountIn.ToBig(),
		},
		Fee: &poolpkg.TokenAmount{
			Token:  tokenIn,
			Amount: feeAmount.ToBig(),
		},
		Gas: defaultGas.Swap,
	}, nil
}

// Version = 1: https://etherscan.io/address/0x6df50e37a6aefb9024a7284ef1c9e1e8e7c4f7b8#code#F1#L165
//
// Version > 1: https://etherscan.io/address/0x065f5b35d4077334379847fe26f58b1029e51161#code#F3#L117
func (s *PoolSimulator) _onSwapGivenIn(
	balanceTokenIn *uint256.Int,
	normalizedWeightIn *uint256.Int,
	balanceTokenOut *uint256.Int,
	normalizedWeightOut *uint256.Int,
	upScaledAmountIn *uint256.Int,
) (*uint256.Int, error) {
	if s.poolTypeVer == poolTypeVer1 {
		return math.WeightedMath.CalcOutGivenInV1(
			balanceTokenIn,
			normalizedWeightIn,
			balanceTokenOut,
			normalizedWeightOut,
			upScaledAmountIn,
		)
	}

	return math.WeightedMath.CalcOutGivenIn(
		balanceTokenIn,
		normalizedWeightIn,
		balanceTokenOut,
		normalizedWeightOut,
		upScaledAmountIn,
	)
}

// Version = 1: https://etherscan.io/address/0x6df50e37a6aefb9024a7284ef1c9e1e8e7c4f7b8#code#F1#L182
//
// Version > 1: https://etherscan.io/address/0x065f5b35d4077334379847fe26f58b1029e51161#code#F3#L132
func (s *PoolSimulator) _onSwapGivenOut(
	balanceTokenIn *uint256.Int,
	normalizedWeightIn *uint256.Int,
	balanceTokenOut *uint256.Int,
	normalizedWeightOut *uint256.Int,
	upScaledAmountOut *uint256.Int,
) (*uint256.Int, error) {
	if s.poolTypeVer == poolTypeVer1 {
		return math.WeightedMath.CalcInGivenOutV1(
			balanceTokenIn,
			normalizedWeightIn,
			balanceTokenOut,
			normalizedWeightOut,
			upScaledAmountOut,
		)
	}

	return math.WeightedMath.CalcInGivenOut(
		balanceTokenIn,
		normalizedWeightIn,
		balanceTokenOut,
		normalizedWeightOut,
		upScaledAmountOut,
	)
}

// Version = 1: https://etherscan.io/address/0x6df50e37a6aefb9024a7284ef1c9e1e8e7c4f7b8#code#F28#L454
//
// Version > 1: https://etherscan.io/address/0x065f5b35d4077334379847fe26f58b1029e51161#code#F14#L619
func (s *PoolSimulator) _addSwapFeeAmount(amount *uint256.Int) (*uint256.Int, error) {
	// This returns amount + fee amount, so we round up (favoring a higher fee amount).
	return math.FixedPoint.DivUp(amount, math.FixedPoint.Complement(s.swapFeePercentage))
}

func (s *PoolSimulator) validateMaxInRatio(tokenIndex int, amountIn *uint256.Int) error {
	sum := new(uint256.Int).Add(s.totalAmountsIn[tokenIndex], amountIn)
	upscaledSum, err := _upscale(s.poolTypeVer, sum, s.scalingFactors[tokenIndex])
	if err != nil {
		return err
	}

	if upscaledSum.Gt(s.scaledMaxTotalAmountsIn[tokenIndex]) {
		return ErrMaxTotalInRatio
	}

	return nil
}

func (s *PoolSimulator) validateMaxOutRatio(tokenIndex int, amountOut *uint256.Int) error {
	sum := new(uint256.Int).Add(s.totalAmountsOut[tokenIndex], amountOut)
	upscaledSum, err := _upscale(s.poolTypeVer, sum, s.scalingFactors[tokenIndex])
	if err != nil {
		return err
	}

	if upscaledSum.Gt(s.scaledMaxTotalAmountsOut[tokenIndex]) {
		return ErrMaxTotalOutRatio
	}

	return nil
}

func (s *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {
	for idx, token := range s.Info.Tokens {
		if token == params.TokenAmountIn.Token {
			s.Info.Reserves[idx] = new(big.Int).Add(
				s.Info.Reserves[idx],
				params.TokenAmountIn.Amount,
			)

			s.totalAmountsIn[idx] = new(uint256.Int).Add(
				s.totalAmountsIn[idx],
				uint256.MustFromBig(params.TokenAmountIn.Amount),
			)
		}

		if token == params.TokenAmountOut.Token {
			s.Info.Reserves[idx] = new(big.Int).Sub(
				s.Info.Reserves[idx],
				params.TokenAmountOut.Amount,
			)
		}
	}
}

func (s *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return PoolMetaInfo{
		Vault:         s.vault,
		PoolID:        s.poolID,
		TokenOutIndex: s.GetTokenIndex(tokenOut),
		BlockNumber:   s.Info.BlockNumber,
	}
}

// Version = 1: https://etherscan.io/address/0x6df50e37a6aefb9024a7284ef1c9e1e8e7c4f7b8#code#F27#L529
//
// Version > 1: https://etherscan.io/address/0x065f5b35d4077334379847fe26f58b1029e51161#code#F13#L681
func _upscale(poolTypeVer int, amount *uint256.Int, scalingFactor *uint256.Int) (*uint256.Int, error) {
	if poolTypeVer == poolTypeVer1 {
		return math.Math.Mul(amount, scalingFactor)
	}

	return math.FixedPoint.MulDown(amount, scalingFactor)
}

// Version = 1: https://etherscan.io/address/0x6df50e37a6aefb9024a7284ef1c9e1e8e7c4f7b8#code#F28#L547
//
// Version > 1: https://etherscan.io/address/0x065f5b35d4077334379847fe26f58b1029e51161#code#F14#L706
func _downscaleDown(poolTypeVer int, amount *uint256.Int, scalingFactor *uint256.Int) (*uint256.Int, error) {
	if poolTypeVer == poolTypeVer1 {
		return math.Math.DivDown(amount, scalingFactor)
	}

	return math.FixedPoint.DivDown(amount, scalingFactor)
}

// Version = 1: https://etherscan.io/address/0x6df50e37a6aefb9024a7284ef1c9e1e8e7c4f7b8#code#F28#L565
//
// Version > 1: https://etherscan.io/address/0x065f5b35d4077334379847fe26f58b1029e51161#code#F14#L727
func _downscaleUp(poolTypeVer int, amount *uint256.Int, scalingFactor *uint256.Int) (*uint256.Int, error) {
	if poolTypeVer == poolTypeVer1 {
		return math.Math.DivUp(amount, scalingFactor)
	}

	return math.FixedPoint.DivUp(amount, scalingFactor)
}

func _upscaleArray(poolTypeVer int, balances []*big.Int, scalingFactors []*uint256.Int) ([]*uint256.Int, error) {
	upscaled := make([]*uint256.Int, len(balances))
	for i, balance := range balances {
		b, overflow := uint256.FromBig(balance)
		if overflow {
			return nil, ErrOverflow
		}

		upscaledI, err := _upscale(poolTypeVer, b, scalingFactors[i])
		if err != nil {
			return nil, err
		}
		upscaled[i] = upscaledI
	}
	return upscaled, nil
}
