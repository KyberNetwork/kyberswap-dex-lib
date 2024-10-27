package stable

import (
	"errors"
	"math/big"

	"github.com/bytedance/sonic"
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
	ErrNotTwoTokens             = errors.New("not two tokens")
)

type PoolSimulator struct {
	poolpkg.Pool

	paused bool

	swapFeePercentage *uint256.Int
	amp               *uint256.Int

	scalingFactors []*uint256.Int

	vault    string
	poolID   string
	poolSpec uint8

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

	if err := sonic.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	if err := sonic.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
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
		BlockNumber: entityPool.BlockNumber,
	}

	return &PoolSimulator{
		Pool:              poolpkg.Pool{Info: poolInfo},
		paused:            extra.Paused,
		swapFeePercentage: extra.SwapFeePercentage,
		amp:               extra.Amp,
		scalingFactors:    extra.ScalingFactors,
		vault:             staticExtra.Vault,
		poolID:            staticExtra.PoolID,
		poolSpec:          staticExtra.PoolSpecialization,
		poolType:          staticExtra.PoolType,
		poolTypeVer:       staticExtra.PoolTypeVer,
	}, nil
}

// https://etherscan.io/address/0x06df3b2bbb68adc8b0e302443692037ed9f91b42#code#F6#L46
func (s *PoolSimulator) CalcAmountOut(params poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	if s.paused {
		return nil, ErrPoolPaused
	}

	// NOTE: if pool specialization is not "General", then the pool must have 2 tokens
	// https://etherscan.io/address/0x06df3b2bbb68adc8b0e302443692037ed9f91b42#code#F1#L130
	if s.poolSpec != poolSpecializationGeneral && len(s.Info.Tokens) != 2 {
		return nil, ErrNotTwoTokens
	}

	tokenAmountIn, tokenOut := params.TokenAmountIn, params.TokenOut

	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)
	if indexIn == -1 || indexOut == -1 {
		return nil, ErrTokenNotRegistered
	}

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
	amountIn, err = _upscale(amountInAfterFee, s.scalingFactors[indexIn])
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

	amountOut, err := math.StableMath.CalcOutGivenIn(
		invariant,
		s.amp,
		amountIn,
		balances,
		indexIn,
		indexOut,
	)
	if err != nil {
		return nil, err
	}

	// amountOut tokens are exiting the Pool, so we round down.
	amountOut, err = _downscaleDown(amountOut, s.scalingFactors[indexOut])
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

// https://etherscan.io/address/0x06df3b2bbb68adc8b0e302443692037ed9f91b42#code#F6#L65
func (s *PoolSimulator) CalcAmountIn(params poolpkg.CalcAmountInParams) (*poolpkg.CalcAmountInResult, error) {
	if s.paused {
		return nil, ErrPoolPaused
	}

	// NOTE: if pool specialization is not "General", then the pool must have 2 tokens
	// https://etherscan.io/address/0x06df3b2bbb68adc8b0e302443692037ed9f91b42#code#F1#L130
	if s.poolSpec != poolSpecializationGeneral && len(s.Info.Tokens) != 2 {
		return nil, ErrNotTwoTokens
	}

	tokenAmountOut, tokenIn := params.TokenAmountOut, params.TokenIn

	indexIn, indexOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenAmountOut.Token)
	if indexIn == -1 || indexOut == -1 {
		return nil, ErrTokenNotRegistered
	}

	amountOut, overflow := uint256.FromBig(tokenAmountOut.Amount)
	if overflow {
		return nil, ErrInvalidAmountOut
	}

	amountOut, err := _upscale(amountOut, s.scalingFactors[indexOut])
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

	amountIn, err := math.StableMath.CalcInGivenOut(
		invariant,
		s.amp,
		amountOut,
		balances,
		indexIn,
		indexOut,
	)
	if err != nil {
		return nil, err
	}

	// amountIn tokens are entering the Pool, so we round up.
	amountIn, err = _downscaleUp(amountIn, s.scalingFactors[indexIn])
	if err != nil {
		return nil, err
	}

	// Fees are added after scaling happens, to reduce the complexity of the rounding direction analysis.
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

func (s *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return PoolMetaInfo{
		Vault:         s.vault,
		PoolID:        s.poolID,
		TokenOutIndex: s.GetTokenIndex(tokenOut),
		BlockNumber:   s.Info.BlockNumber,
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

// https://etherscan.io/address/0x06df3b2bbb68adc8b0e302443692037ed9f91b42#code#F13#L490
// The exact implementation is just `return amount.divUp(FixedPoint.ONE.sub(_swapFeePercentage));`
// But we use Complement to avoid negative value.
func (s *PoolSimulator) _addSwapFeeAmount(amount *uint256.Int) (*uint256.Int, error) {
	// This returns amount + fee amount, so we round up (favoring a higher fee amount).
	return math.FixedPoint.DivUp(amount, math.FixedPoint.Complement(s.swapFeePercentage))
}

// MetaStable: https://etherscan.io/address/0x063c624672e390363b25f0c6c68ad9067c34595b#code#F30#L49
//
// Stable Version 1: https://etherscan.io/address/0x06df3b2bbb68adc8b0e302443692037ed9f91b42#code#F8#L49
//
// Stable Version 2: https://etherscan.io/address/0x13f2f70a951fb99d48ede6e25b0bdf06914db33f#code#F5#L57
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

// https://etherscan.io/address/0x063c624672e390363b25f0c6c68ad9067c34595b#code#F31#L530
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

// https://etherscan.io/address/0x063c624672e390363b25f0c6c68ad9067c34595b#code#F31#L518
func _upscale(amount *uint256.Int, scalingFactor *uint256.Int) (*uint256.Int, error) {
	return math.FixedPoint.MulDown(amount, scalingFactor)
}

// https://etherscan.io/address/0x063c624672e390363b25f0c6c68ad9067c34595b#code#F32#L540
func _downscaleDown(amount *uint256.Int, scalingFactor *uint256.Int) (*uint256.Int, error) {
	return math.FixedPoint.DivDown(amount, scalingFactor)
}

// https://etherscan.io/address/0x063c624672e390363b25f0c6c68ad9067c34595b#code#F32#L558
func _downscaleUp(amount *uint256.Int, scalingFactor *uint256.Int) (*uint256.Int, error) {
	return math.FixedPoint.DivUp(amount, scalingFactor)
}
