package pools

import (
	"fmt"
	"math"

	"github.com/holiman/uint256"

	ekubomath "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting"
)

type MevResistPoolSwapState = BasePoolSwapState
type MevResistPoolState = BasePoolState

type MevResistPool struct {
	*BasePool
	lastTick         int32
	swappedThisBlock bool
}

func NewMevResistPool(key *PoolKey, state *BasePoolState) *MevResistPool {
	return &MevResistPool{
		BasePool:         NewBasePool(key, state),
		lastTick:         state.ActiveTick,
		swappedThisBlock: false,
	}
}

func (p *MevResistPool) CloneState() any {
	cloned := *p
	cloned.BasePool = p.BasePool.CloneState().(*BasePool)
	return &cloned
}

func (p *MevResistPool) SetSwapState(state quoting.SwapState) {
	p.BasePoolSwapState = state.(*MevResistPoolSwapState)
	p.swappedThisBlock = true
}

func (p *MevResistPool) Quote(amount *uint256.Int, isToken1 bool) (*quoting.Quote, error) {
	quote, err := p.BasePool.Quote(amount, isToken1)
	if err != nil {
		return nil, err
	}
	quote.SwapInfo.Forward = &p.key.Config.Extension

	tickAfterSwap := ekubomath.ApproximateSqrtRatioToTick(quote.SwapInfo.SwapStateAfter.(*BasePoolSwapState).SqrtRatio)

	poolConfig := &p.key.Config
	approximateFeeMultiplier := math.Abs(float64(tickAfterSwap-p.lastTick)) / float64(poolConfig.TickSpacing)

	fixedPointAdditionalFee := uint64(min(math.Round(approximateFeeMultiplier*float64(poolConfig.Fee)), math.MaxUint64))

	if !p.swappedThisBlock {
		quote.Gas += quoting.GasAccumulatingMevResistFees
	}

	quote.Gas += quoting.ExtraBaseGasMevResistSwap

	if fixedPointAdditionalFee == 0 {
		return quote, nil
	}

	calculatedAmount := quote.CalculatedAmount

	if amount.Sign() >= 0 {
		// exact input, remove the additional fee from the output
		calculatedAmount.Sub(calculatedAmount, ekubomath.ComputeFee(calculatedAmount, fixedPointAdditionalFee))
	} else {
		// exact output, add the additional fee to the output
		inputAmountFee := ekubomath.ComputeFee(calculatedAmount, poolConfig.Fee)
		inputAmount := inputAmountFee.Sub(calculatedAmount, inputAmountFee)

		bf, err := ekubomath.AmountBeforeFee(inputAmount, fixedPointAdditionalFee)
		if err != nil {
			return nil, fmt.Errorf("amount before fee calculation: %w", err)
		}

		calculatedAmount.Add(calculatedAmount, bf.Sub(bf, inputAmount))
	}

	return quote, nil
}
