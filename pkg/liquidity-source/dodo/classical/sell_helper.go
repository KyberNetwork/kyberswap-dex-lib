package classical

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dodo/libv1"
)

type DODOState struct {
	OraclePrice *uint256.Int
	K           *uint256.Int
	B           *uint256.Int
	Q           *uint256.Int
	BaseTarget  *uint256.Int
	QuoteTarget *uint256.Int
	RStatus     int
}

// https://github.com/DODOEX/contractV2/blob/c58c067c4038437610a9cc8aef8f8025e2af4f63/contracts/SmartRoute/helper/DODOSellHelper.sol#L149
func (p *PoolSimulator) querySellQuoteToken(amount *uint256.Int) (
	boughtAmount *uint256.Int,
	err error,
) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	var state DODOState
	state.BaseTarget, state.QuoteTarget = p.getExpectedTarget()
	state.RStatus = p.RStatus
	state.OraclePrice = p.OraclePrice
	state.Q = p.Q
	state.B = p.B
	state.K = p.K

	if p.RStatus == rStatusOne {
		boughtAmount = p._ROneSellQuoteToken(amount, &state)
	} else if p.RStatus == rStatusAboveOne {
		boughtAmount = p._RAboveSellQuoteToken(amount, &state)
	} else {
		backOneBase := libv1.SafeSub(state.B, state.BaseTarget)
		backOneQuote := libv1.SafeSub(state.QuoteTarget, state.Q)
		if amount.Cmp(backOneQuote) <= 0 {
			boughtAmount = p._RBelowSellQuoteToken(amount, &state)
		} else {
			boughtAmount = libv1.SafeAdd(
				backOneBase,
				p._ROneSellQuoteToken(libv1.SafeSub(amount, backOneQuote), &state),
			)
		}
	}

	boughtAmount = libv1.DecimalMathDivFloor(
		boughtAmount,
		libv1.SafeAdd(
			libv1.DecimalMathOne,
			libv1.SafeAdd(p.MtFeeRate, p.LpFeeRate),
		),
	)

	return
}

// https://github.com/DODOEX/contractV2/blob/c58c067c4038437610a9cc8aef8f8025e2af4f63/contracts/SmartRoute/helper/DODOSellHelper.sol#L186
func (p *PoolSimulator) _ROneSellQuoteToken(amount *uint256.Int, state *DODOState) *uint256.Int {
	i := libv1.DecimalMathDivFloor(libv1.DecimalMathOne, state.OraclePrice)
	B2 := libv1.SolveQuadraticFunctionForTrade(
		state.BaseTarget,
		state.BaseTarget,
		libv1.DecimalMathMul(i, amount),
		false,
		state.K,
	)

	return libv1.SafeSub(state.BaseTarget, B2)
}

// https://github.com/DODOEX/contractV2/blob/c58c067c4038437610a9cc8aef8f8025e2af4f63/contracts/SmartRoute/helper/DODOSellHelper.sol#L202
func (p *PoolSimulator) _RAboveSellQuoteToken(amount *uint256.Int, state *DODOState) *uint256.Int {
	i := libv1.DecimalMathDivFloor(libv1.DecimalMathOne, state.OraclePrice)
	B2 := libv1.SolveQuadraticFunctionForTrade(
		state.BaseTarget,
		state.B,
		libv1.DecimalMathMul(i, amount),
		false,
		state.K,
	)

	return libv1.SafeSub(state.B, B2)
}

// https://github.com/DODOEX/contractV2/blob/c58c067c4038437610a9cc8aef8f8025e2af4f63/contracts/SmartRoute/helper/DODOSellHelper.sol#L218
func (p *PoolSimulator) _RBelowSellQuoteToken(amount *uint256.Int, state *DODOState) *uint256.Int {
	Q1 := libv1.SafeAdd(state.Q, amount)
	i := libv1.DecimalMathDivFloor(libv1.DecimalMathOne, state.OraclePrice)
	return libv1.GeneralIntegrate(state.QuoteTarget, Q1, state.Q, i, state.K)
}
