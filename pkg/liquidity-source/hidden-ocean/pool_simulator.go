package hiddenocean

import (
	"math"
	"math/big"
	"time"

	"github.com/KyberNetwork/int256"
	"github.com/KyberNetwork/uniswapv3-sdk-uint256/constants"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var _ = pool.RegisterFactory(DexType, NewPoolSimulator)

var (
	minSqrtRatioU256 = uint256.MustFromDecimal(minSqrtRatio)
	maxSqrtRatioU256 = uint256.MustFromDecimal(maxSqrtRatio)
	q96              = constants.Q96U256
)

type PoolSimulator struct {
	pool.Pool

	sqrtPriceX96 *uint256.Int
	liquidity    *uint256.Int
	fee          uint32
	sqrtPaX96    *uint256.Int
	sqrtPbX96    *uint256.Int
	gas          int64
}

func NewPoolSimulator(params pool.FactoryParams) (*PoolSimulator, error) {
	maxAge := lo.Ternary(params.Opts.StaleCheck, MaxAge, time.Duration(math.MaxInt64))
	if time.Since(time.Unix(params.EntityPool.Timestamp, 0)) > maxAge {
		return nil, ErrPoolStateStale
	}

	entityPool := params.EntityPool

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	tokens := make([]string, len(entityPool.Tokens))
	reserves := make([]*big.Int, len(entityPool.Reserves))
	for i, token := range entityPool.Tokens {
		tokens[i] = token.Address
	}
	for i, reserve := range entityPool.Reserves {
		reserves[i] = bignumber.NewBig(reserve)
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:     entityPool.Address,
				Exchange:    entityPool.Exchange,
				Type:        entityPool.Type,
				Tokens:      tokens,
				Reserves:    reserves,
				BlockNumber: entityPool.BlockNumber,
			},
		},
		sqrtPriceX96: extra.SqrtPriceX96,
		liquidity:    extra.Liquidity,
		fee:          extra.Fee,
		sqrtPaX96:    extra.SqrtPaX96,
		sqrtPbX96:    extra.SqrtPbX96,
		gas:          defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn := params.TokenAmountIn.Token
	tokenOut := params.TokenOut
	amountIn := params.TokenAmountIn.Amount

	if amountIn == nil || amountIn.Sign() <= 0 {
		return nil, ErrZeroAmountIn
	}
	if s.liquidity == nil || s.liquidity.IsZero() {
		return nil, ErrZeroLiquidity
	}

	// Determine swap direction
	tokenInIndex := s.GetTokenIndex(tokenIn)
	tokenOutIndex := s.GetTokenIndex(tokenOut)
	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return nil, ErrInvalidToken
	}
	zeroForOne := tokenInIndex == 0

	// Determine price limit based on direction
	var sqrtPriceLimitX96 uint256.Int
	if zeroForOne {
		sqrtPriceLimitX96.Set(s.sqrtPaX96)
	} else {
		sqrtPriceLimitX96.Set(s.sqrtPbX96)
	}

	// Clamp starting price into [sqrtPaX96, sqrtPbX96]
	var sqrtPriceCurrentX96 uint256.Int
	sqrtPriceCurrentX96.Set(s.sqrtPriceX96)
	if sqrtPriceCurrentX96.Cmp(s.sqrtPaX96) < 0 {
		sqrtPriceCurrentX96.Set(s.sqrtPaX96)
	}
	if sqrtPriceCurrentX96.Cmp(s.sqrtPbX96) > 0 {
		sqrtPriceCurrentX96.Set(s.sqrtPbX96)
	}

	// Check that swap can make progress
	if zeroForOne {
		if sqrtPriceLimitX96.Cmp(&sqrtPriceCurrentX96) >= 0 {
			return nil, ErrNoSwapLimit
		}
	} else {
		if sqrtPriceLimitX96.Cmp(&sqrtPriceCurrentX96) <= 0 {
			return nil, ErrNoSwapLimit
		}
	}

	// Convert amountIn to Int256 (positive = exact input)
	amountInU256, _ := uint256.FromBig(amountIn)
	var amountRemainingI256 int256.Int
	amountRemainingI256.SetFromBig(amountIn)

	// Call ComputeSwapStep
	var (
		sqrtRatioNextX96 v3Utils.Uint160
		stepAmountIn     uint256.Int
		stepAmountOut    uint256.Int
		stepFeeAmount    uint256.Int
	)

	err := v3Utils.ComputeSwapStep(
		&sqrtPriceCurrentX96,
		&sqrtPriceLimitX96,
		s.liquidity,
		&amountRemainingI256,
		constants.FeeAmount(s.fee),
		&sqrtRatioNextX96,
		&stepAmountIn,
		&stepAmountOut,
		&stepFeeAmount,
	)
	if err != nil {
		return nil, err
	}

	// Calculate total consumed input (amountIn + fee)
	var totalInput uint256.Int
	totalInput.Add(&stepAmountIn, &stepFeeAmount)

	// Calculate remaining (unconsumed) input
	var remainingAmount *big.Int
	if amountInU256.Cmp(&totalInput) > 0 {
		var rem uint256.Int
		rem.Sub(amountInU256, &totalInput)
		remainingAmount = rem.ToBig()
	}

	amountOutBig := stepAmountOut.ToBig()
	feeAmountBig := stepFeeAmount.ToBig()

	result := &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOutBig,
		},
		Fee: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: feeAmountBig,
		},
		Gas:      s.gas,
		SwapInfo: SwapInfo{},
	}

	if remainingAmount != nil && remainingAmount.Sign() > 0 {
		result.RemainingTokenAmountIn = &pool.TokenAmount{
			Token:  tokenIn,
			Amount: remainingAmount,
		}
	}

	return result, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	tokenIn := params.TokenAmountIn.Token
	amountIn := params.TokenAmountIn.Amount
	amountOut := params.TokenAmountOut.Amount

	tokenInIndex := s.GetTokenIndex(tokenIn)
	zeroForOne := tokenInIndex == 0

	// Update reserves: fees are transferred to feeReceiver (not kept in pool),
	// so the pool balance only increases by (amountIn - fee).
	feeAmount := params.Fee.Amount
	amountInLessFee := new(big.Int).Sub(amountIn, feeAmount)
	s.Info.Reserves[tokenInIndex] = new(big.Int).Add(s.Info.Reserves[tokenInIndex], amountInLessFee)
	tokenOutIndex := 1 - tokenInIndex
	s.Info.Reserves[tokenOutIndex] = new(big.Int).Sub(s.Info.Reserves[tokenOutIndex], amountOut)

	// Recompute sqrtPriceX96 using the consumed input (excluding fee)
	amountInLessFeeU256, _ := uint256.FromBig(amountInLessFee)

	// Recompute sqrtPriceX96: use GetNextSqrtPriceFromInput
	var newSqrtPrice uint256.Int
	// Start from clamped current price
	var sqrtPriceCurrentX96 uint256.Int
	sqrtPriceCurrentX96.Set(s.sqrtPriceX96)
	if sqrtPriceCurrentX96.Cmp(s.sqrtPaX96) < 0 {
		sqrtPriceCurrentX96.Set(s.sqrtPaX96)
	}
	if sqrtPriceCurrentX96.Cmp(s.sqrtPbX96) > 0 {
		sqrtPriceCurrentX96.Set(s.sqrtPbX96)
	}

	err := v3Utils.GetNextSqrtPriceFromInput(&sqrtPriceCurrentX96, s.liquidity, amountInLessFeeU256, zeroForOne, &newSqrtPrice)
	if err != nil {
		// Fallback: keep price at limit
		if zeroForOne {
			newSqrtPrice.Set(s.sqrtPaX96)
		} else {
			newSqrtPrice.Set(s.sqrtPbX96)
		}
	}

	s.sqrtPriceX96 = new(uint256.Int).Set(&newSqrtPrice)

	// Recompute range bounds based on new reserves and oracle price (approximated by current sqrtPriceX96)
	s.recomputeRange()
}

// recomputeRange recalculates sqrtPaX96 and sqrtPbX96 from current state.
// This mirrors _computeRange from the Solidity contract.
func (s *PoolSimulator) recomputeRange() {
	if s.liquidity == nil || s.liquidity.IsZero() {
		return
	}

	sqrtP := s.sqrtPriceX96
	L := s.liquidity
	balance0 := s.Info.Reserves[0]
	balance1 := s.Info.Reserves[1]

	// Compute sqrtPa: sqrtPa = sqrtP - balance1 * Q96 / L
	bal1U256, _ := uint256.FromBig(balance1)
	var yOverL_Q96 uint256.Int
	// yOverL_Q96 = balance1 * Q96 / L
	var numerator uint256.Int
	numerator.Mul(bal1U256, q96)
	yOverL_Q96.Div(&numerator, L)

	var sqrtPaX96 uint256.Int
	if yOverL_Q96.Cmp(sqrtP) >= 0 {
		sqrtPaX96.Set(sqrtP)
	} else {
		sqrtPaX96.Sub(sqrtP, &yOverL_Q96)
		if sqrtPaX96.Cmp(minSqrtRatioU256) < 0 {
			sqrtPaX96.Set(minSqrtRatioU256)
		}
	}

	// Compute sqrtPb: sqrtPb = (L * sqrtP) / (L - balance0 * sqrtP / Q96)
	bal0U256, _ := uint256.FromBig(balance0)
	var xTimesSqrtP uint256.Int
	var tmp uint256.Int
	tmp.Mul(bal0U256, sqrtP)
	xTimesSqrtP.Div(&tmp, q96)

	var sqrtPbX96 uint256.Int
	if xTimesSqrtP.Cmp(L) >= 0 {
		sqrtPbX96.Set(sqrtP)
	} else {
		var num uint256.Int
		num.Mul(L, sqrtP)
		var denom uint256.Int
		denom.Sub(L, &xTimesSqrtP)
		sqrtPbX96.Div(&num, &denom)

		maxMinusOne := new(uint256.Int).Sub(maxSqrtRatioU256, uint256.NewInt(1))
		if sqrtPbX96.Cmp(maxMinusOne) > 0 {
			sqrtPbX96.Set(maxMinusOne)
		}
		sqrtPPlus1 := new(uint256.Int).Add(sqrtP, uint256.NewInt(1))
		if sqrtPbX96.Cmp(sqrtPPlus1) <= 0 {
			sqrtPbX96.Set(sqrtPPlus1)
		}
	}

	// Ensure pa < pb
	if sqrtPaX96.Cmp(&sqrtPbX96) >= 0 {
		sqrtPaX96.Set(sqrtP)
		sqrtPbX96.Set(sqrtP)
	}

	s.sqrtPaX96 = new(uint256.Int).Set(&sqrtPaX96)
	s.sqrtPbX96 = new(uint256.Int).Set(&sqrtPbX96)
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	return nil
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:     s.Info.Address,
				Exchange:    s.Info.Exchange,
				Type:        s.Info.Type,
				Tokens:      s.Info.Tokens, // string slice, shared is fine
				BlockNumber: s.Info.BlockNumber,
			},
		},
		sqrtPriceX96: new(uint256.Int).Set(s.sqrtPriceX96),
		liquidity:    new(uint256.Int).Set(s.liquidity),
		fee:          s.fee,
		sqrtPaX96:    new(uint256.Int).Set(s.sqrtPaX96),
		sqrtPbX96:    new(uint256.Int).Set(s.sqrtPbX96),
		gas:          s.gas,
	}

	cloned.Info.Reserves = make([]*big.Int, len(s.Info.Reserves))
	for i, r := range s.Info.Reserves {
		cloned.Info.Reserves[i] = new(big.Int).Set(r)
	}

	return cloned
}

// uint256FromBigInt converts *big.Int to *uint256.Int, handling nil.
func uint256FromBigInt(v *big.Int) *uint256.Int {
	if v == nil {
		return uint256.NewInt(0)
	}
	u, _ := uint256.FromBig(v)
	return u
}
