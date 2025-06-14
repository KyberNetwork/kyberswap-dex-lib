package pools

import (
	"fmt"
	"math"
	"math/big"

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

func NewMevResistPool(key *PoolKey, state *BasePoolState, lastTick int32) *MevResistPool {
	return &MevResistPool{
		BasePool:         NewBasePool(key, state),
		lastTick:         lastTick,
		swappedThisBlock: false,
	}
}

func (p *MevResistPool) CloneState() any {
	cloned := *p
	clonedSwapState := *p.BasePoolSwapState
	cloned.BasePoolSwapState = &clonedSwapState
	return &cloned
}

func (p *MevResistPool) SetSwapState(state quoting.SwapState) {
	swapState := state.(*MevResistPoolSwapState)

	p.BasePoolSwapState = swapState
	p.swappedThisBlock = true
}

func (p *MevResistPool) Quote(amount *big.Int, isToken1 bool) (*quoting.Quote, error) {
	quote, err := p.BasePool.Quote(amount, isToken1)
	if err != nil {
		return nil, err
	}

	tickAfterSwap := ekubomath.ApproximateSqrtRatioToTick(quote.SwapInfo.SwapStateAfter.(*BasePoolSwapState).SqrtRatio)

	poolConfig := &p.key.Config
	approximateFeeMultiplier := math.Abs(float64(tickAfterSwap-p.lastTick)) / float64(poolConfig.TickSpacing)

	fixedPointAdditionalFee := uint64(math.Min(math.Round(approximateFeeMultiplier*float64(poolConfig.Fee)), float64(math.MaxUint64)))

	if !p.swappedThisBlock {
		quote.Gas += quoting.GasCostOfAccumulatingMevResistFees
	}

	quote.Gas += quoting.ExtraBaseGasCostOfMevResistSwap

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
		inputAmount := new(big.Int).Sub(calculatedAmount, inputAmountFee)

		bf, err := ekubomath.AmountBeforeFee(inputAmount, fixedPointAdditionalFee)
		if err != nil {
			return nil, fmt.Errorf("amount before fee calculation: %w", err)
		}

		calculatedAmount.Add(calculatedAmount, bf.Sub(bf, inputAmount))
	}

	return quote, nil
}
