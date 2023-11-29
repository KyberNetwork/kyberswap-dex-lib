package stable

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/math"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrInvalidSwapFeePercentage = errors.New("invalid swap fee percentage")
	ErrPoolPaused               = errors.New("pool is paused")
	ErrInvalidAmp               = errors.New("invalid amp")
)

type PoolSimulator struct {
	poolpkg.Pool

	paused bool

	swapFeePercentage *uint256.Int
	amp               *uint256.Int
	scalingFactors    []*uint256.Int

	poolType    string
	poolTypeVer int
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var (
		extra       Extra
		staticExtra StaticExtra

		tokens   = make([]string, len(entityPool.Tokens))
		reserves = make([]*big.Int, len(entityPool.Tokens))
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
		Pool:              poolpkg.Pool{Info: poolInfo},
		paused:            extra.Paused,
		swapFeePercentage: extra.SwapFeePercentage,
		amp:               extra.Amp,
		scalingFactors:    staticExtra.ScalingFactors,
		poolType:          staticExtra.PoolType,
		poolTypeVer:       staticExtra.PoolTypeVer,
	}, nil
}

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

	invariant, err := calculateInvariant(s.poolType, s.poolTypeVer, s.amp, balances)
	if err != nil {
		return nil, err
	}

	upScaledAmountOut, err := math.StableMath.CalcOutGivenIn(
		invariant,
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

func (s *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return PoolMetaInfo{
		T: s.poolType,
		V: s.poolTypeVer,
	}
}

func (s *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {
	for idx, token := range s.Info.Tokens {
		if token == params.TokenAmountIn.Token {
			s.Info.Reserves[idx] = new(big.Int).Add(
				s.Info.Reserves[idx],
				params.TokenAmountIn.Amount,
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

func calculateInvariant(
	poolType string,
	poolTypeVer int,
	amp *uint256.Int,
	balances []*uint256.Int,
) (*uint256.Int, error) {
	if poolType == poolTypeMetaStable {
		return math.StableMath.CalculateInvariantV1(amp, balances, true)
	}

	if poolTypeVer == poolTypeVer1 {
		return math.StableMath.CalculateInvariantV1(amp, balances, true)
	}

	return math.StableMath.CalculateInvariantV2(amp, balances)
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
