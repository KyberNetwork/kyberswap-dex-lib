package tickerdualpool

import (
	"errors"

	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/dualpool"
)

var ErrInvalidQuoteState = errors.New("invalid DualPool quote state")

// temporaryLiquidity reuses the same allocation while reverse pricing probes inputs.
type temporaryLiquidity struct {
	state         dualpool.Extra
	positions     []position
	tickSpacing   int
	zeroForOne    bool
	fee, protocol uint64
}

func prepareTemporaryLiquidity(state dualpool.Extra, tickSpacing int, zeroForOne bool) (*temporaryLiquidity, error) {
	if state.SqrtPriceX96 == nil || state.Balance0 == nil || state.Balance1 == nil {
		return nil, dualpool.ErrStateNotSet
	}
	if !state.Live {
		return nil, dualpool.ErrPoolNotLive
	}
	if tickSpacing <= 0 || tickSpacing > 32767 {
		return nil, ErrInvalidQuoteState
	}
	if len(state.Buckets) == 0 {
		return nil, dualpool.ErrNoDistribution
	}
	if len(state.Buckets) > 8 {
		return nil, ErrInvalidQuoteState
	}
	if state.ProtocolFee&0xfff > 1000 || state.ProtocolFee>>12 > 1000 {
		return nil, ErrInvalidQuoteState
	}
	var weight uint64
	for _, b := range state.Buckets {
		if b.WeightBps == 0 || b.TickLower < -887272 || b.TickUpper > 887272 || b.TickLower >= b.TickUpper || int(b.TickLower)%tickSpacing != 0 || int(b.TickUpper)%tickSpacing != 0 {
			return nil, ErrInvalidQuoteState
		}
		weight += uint64(b.WeightBps)
		// Solidity computes balance * weight before division using checked arithmetic.
		var product uint256.Int
		w := uint256.NewInt(uint64(b.WeightBps))
		if _, overflow := product.MulOverflow(state.Balance0, w); overflow {
			return nil, ErrInvalidQuoteState
		}
		if _, overflow := product.MulOverflow(state.Balance1, w); overflow {
			return nil, ErrInvalidQuoteState
		}
	}
	if weight != totalWeightBps {
		return nil, ErrInvalidQuoteState
	}
	positions, err := dualpool.AllocatePositions(state.Buckets, state.SqrtPriceX96, state.Balance0, state.Balance1)
	if err != nil {
		return nil, err
	}
	// PoolManager limits gross liquidity at each initialized tick. Overlapping
	// buckets share that limit even when their positions use different salts.
	minCompressed := -887272 / tickSpacing
	if -887272%tickSpacing != 0 {
		minCompressed--
	}
	numTicks := 887272/tickSpacing - minCompressed + 1
	maxPerTick := new(uint256.Int).Div(uMaxUint128, uint256.NewInt(uint64(numTicks)))
	grossAtTick := make(map[int]*uint256.Int, len(positions)*2)
	for _, position := range positions {
		for _, tick := range []int{position.TickLower, position.TickUpper} {
			gross := grossAtTick[tick]
			if gross == nil {
				gross = new(uint256.Int)
				grossAtTick[tick] = gross
			}
			if _, overflow := gross.AddOverflow(gross, position.Liquidity); overflow || gross.Cmp(maxPerTick) > 0 {
				return nil, ErrInvalidQuoteState
			}
		}
	}
	fee := dualpool.EffectiveSwapFee(state.LpFee, state.ProtocolFee, zeroForOne)
	if fee >= maxSwapFee {
		return nil, ErrInvalidQuoteState
	}
	protocol := uint64(state.ProtocolFee & 0xfff)
	if !zeroForOne {
		protocol = uint64(state.ProtocolFee >> 12)
	}
	return &temporaryLiquidity{state: state, positions: positions, tickSpacing: tickSpacing, zeroForOne: zeroForOne, fee: fee, protocol: protocol}, nil
}

func (p *temporaryLiquidity) swap(amountIn *uint256.Int) (*swapResult, error) {
	if amountIn == nil || amountIn.IsZero() || amountIn.BitLen() > 127 {
		return nil, ErrInvalidQuoteState
	}
	result, err := swapExactInWithSpacing(p.positions, p.state.SqrtPriceX96, int(p.state.Tick), p.zeroForOne, amountIn, p.fee, p.tickSpacing, p.protocol)
	if err != nil {
		return nil, err
	}
	if result.amountOut.BitLen() > 127 {
		return nil, ErrInvalidQuoteState
	}
	return result, nil
}

// quoteTemporaryLiquidity prices and settles temporary positions without mutating state.
// Derived hooks must check their own withdrawal limits before calling it.
func quoteTemporaryLiquidity(state dualpool.Extra, tickSpacing int, zeroForOne bool, amountIn *uint256.Int) (SwapInfo, error) {
	if amountIn == nil {
		return SwapInfo{}, dualpool.ErrStateNotSet
	}
	prepared, err := prepareTemporaryLiquidity(state, tickSpacing, zeroForOne)
	if err != nil {
		return SwapInfo{}, err
	}
	result, err := prepared.swap(amountIn)
	if err != nil {
		return SwapInfo{}, err
	}
	return prepared.finish(result)
}

// finish reconstructs teardown balances only for the selected swap, not every search probe.
func (p *temporaryLiquidity) finish(result *swapResult) (SwapInfo, error) {
	state, zeroForOne := p.state, p.zeroForOne
	positions := p.positions
	minted := [2]*uint256.Int{new(uint256.Int), new(uint256.Int)}
	returned := [2]*uint256.Int{new(uint256.Int), new(uint256.Int)}
	balances := [2]*uint256.Int{state.Balance0.Clone(), state.Balance1.Clone()}
	for i, p := range positions {
		start, err := positionAmounts(p, state.SqrtPriceX96, true)
		if err != nil {
			return SwapInfo{}, err
		}
		end, err := positionAmounts(p, result.sqrtPriceX96, false)
		if err != nil {
			return SwapInfo{}, err
		}
		for side := range balances {
			if _, overflow := minted[side].AddOverflow(minted[side], start[side]); overflow {
				return SwapInfo{}, ErrInvalidQuoteState
			}
			if _, overflow := returned[side].AddOverflow(returned[side], end[side]); overflow {
				return SwapInfo{}, ErrInvalidQuoteState
			}
			if balances[side].Cmp(start[side]) < 0 {
				return SwapInfo{}, ErrInvalidQuoteState
			}
			balances[side].Sub(balances[side], start[side])
			if _, overflow := balances[side].AddOverflow(balances[side], end[side]); overflow {
				return SwapInfo{}, ErrInvalidQuoteState
			}
		}
		inputSide := 0
		if !zeroForOne {
			inputSide = 1
		}
		if _, overflow := returned[inputSide].AddOverflow(returned[inputSide], result.positionFees[i]); overflow {
			return SwapInfo{}, ErrInvalidQuoteState
		}
		if _, overflow := balances[inputSide].AddOverflow(balances[inputSide], result.positionFees[i]); overflow {
			return SwapInfo{}, ErrInvalidQuoteState
		}
	}
	return SwapInfo{Mint0: minted[0], Mint1: minted[1], Burn0: returned[0], Burn1: returned[1], Balance0: balances[0], Balance1: balances[1], ZeroForOne: zeroForOne, AmountIn: result.amountIn, AmountOut: result.amountOut,
		SqrtPriceX96: result.sqrtPriceX96, Tick: int32(result.tick)}, nil
}

// Minting a position rounds its token cost up; burning rounds returned tokens down.
func positionAmounts(p position, price *uint256.Int, roundUp bool) ([2]*uint256.Int, error) {
	result := [2]*uint256.Int{new(uint256.Int), new(uint256.Int)}
	if price.Cmp(p.SqrtUpper) < 0 {
		lower := price
		if price.Cmp(p.SqrtLower) < 0 {
			lower = p.SqrtLower
		}
		if err := v3Utils.GetAmount0DeltaV2(lower, p.SqrtUpper, p.Liquidity, roundUp, result[0]); err != nil {
			return result, err
		}
	}
	if price.Cmp(p.SqrtLower) > 0 {
		upper := price
		if price.Cmp(p.SqrtUpper) > 0 {
			upper = p.SqrtUpper
		}
		if err := v3Utils.GetAmount1DeltaV2(p.SqrtLower, upper, p.Liquidity, roundUp, result[1]); err != nil {
			return result, err
		}
	}
	return result, nil
}

// SwapInfo adds Ticker's temporary-liquidity movements to the swap result.
type SwapInfo struct {
	Balance0, Balance1                *uint256.Int
	Mint0, Mint1, Burn0, Burn1        *uint256.Int
	ZeroForOne                        bool
	AmountIn, AmountOut, SqrtPriceX96 *uint256.Int
	Tick                              int32
}
