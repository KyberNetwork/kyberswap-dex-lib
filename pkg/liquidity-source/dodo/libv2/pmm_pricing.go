package libv2

import (
	"github.com/holiman/uint256"
)

type RState int

const (
	RStateOne      RState = 0
	RStateAboveOne RState = 1
	RStateBelowOne RState = 2
)

type PMMState struct {
	I         *uint256.Int
	K         *uint256.Int
	B         *uint256.Int
	Q         *uint256.Int
	B0        *uint256.Int
	Q0        *uint256.Int
	R         RState
	MtFeeRate *uint256.Int
	LpFeeRate *uint256.Int
}

// SellBaseToken https://github.com/DODOEX/contractV2/blob/c58c067c4038437610a9cc8aef8f8025e2af4f63/contracts/lib/PMMPricing.sol#L39
func SellBaseToken(state PMMState, payBaseAmount *uint256.Int) (receiveQuoteAmount *uint256.Int, newR RState) {
	if state.R == RStateOne {
		// case 1: R=1
		// R falls below one
		receiveQuoteAmount = _ROneSellBaseToken(state, payBaseAmount)
		newR = RStateBelowOne
	} else if state.R == RStateAboveOne {
		backToOnePayBase := SafeSub(state.B0, state.B)
		backToOneReceiveQuote := SafeSub(state.Q, state.Q0)
		// case 2: R>1
		// complex case, R status depends on trading amount
		if payBaseAmount.Cmp(backToOnePayBase) < 0 {
			// case 2.1: R status do not change
			receiveQuoteAmount = _RAboveSellBaseToken(state, payBaseAmount)
			newR = RStateAboveOne
			if receiveQuoteAmount.Cmp(backToOneReceiveQuote) > 0 {
				// [Important corner case!] may enter this branch when some precision problem happens. And consequently contribute to negative spare quote amount
				// to make sure spare quote>=0, mannually set receiveQuote=backToOneReceiveQuote
				receiveQuoteAmount = backToOneReceiveQuote
			}
		} else if payBaseAmount.Cmp(backToOnePayBase) == 0 {
			// case 2.2: R status changes to ONE
			receiveQuoteAmount = backToOneReceiveQuote
			newR = RStateOne
		} else {
			// case 2.3: R status changes to BELOW_ONE
			receiveQuoteAmount = SafeAdd(
				backToOneReceiveQuote, _ROneSellBaseToken(state, SafeSub(payBaseAmount, backToOnePayBase)),
			)
			newR = RStateBelowOne
		}
	} else {
		// state.R == RState.BELOW_ONE
		// case 3: R<1
		receiveQuoteAmount = _RBelowSellBaseToken(state, payBaseAmount)
		newR = RStateBelowOne
	}

	return
}

// SellQuoteToken https://github.com/DODOEX/contractV2/blob/c58c067c4038437610a9cc8aef8f8025e2af4f63/contracts/lib/PMMPricing.sol#L82
func SellQuoteToken(state PMMState, payQuoteAmount *uint256.Int) (receiveBaseAmount *uint256.Int, newR RState) {
	if state.R == RStateOne {
		receiveBaseAmount = _ROneSellQuoteToken(state, payQuoteAmount)
		newR = RStateAboveOne
	} else if state.R == RStateAboveOne {
		receiveBaseAmount = _RAboveSellQuoteToken(state, payQuoteAmount)
		newR = RStateAboveOne
	} else {
		backToOnePayQuote := SafeSub(state.Q0, state.Q)
		backToOneReceiveBase := SafeSub(state.B, state.B0)
		if payQuoteAmount.Cmp(backToOnePayQuote) < 0 {
			receiveBaseAmount = _RBelowSellQuoteToken(state, payQuoteAmount)
			newR = RStateBelowOne
			if receiveBaseAmount.Cmp(backToOneReceiveBase) > 0 {
				receiveBaseAmount = backToOneReceiveBase
			}
		} else if payQuoteAmount.Cmp(backToOnePayQuote) == 0 {
			receiveBaseAmount = backToOneReceiveBase
			newR = RStateOne
		} else {
			receiveBaseAmount = SafeAdd(
				backToOneReceiveBase, _ROneSellQuoteToken(state, SafeSub(payQuoteAmount, backToOnePayQuote)),
			)
			newR = RStateAboveOne
		}
	}

	return
}

// ============ R = 1 cases ============

// https://github.com/DODOEX/contractV2/blob/c58c067c4038437610a9cc8aef8f8025e2af4f63/contracts/lib/PMMPricing.sol#L116
func _ROneSellBaseToken(state PMMState, payBaseAmount *uint256.Int) *uint256.Int {
	return SolveQuadraticFunctionForTrade(
		state.Q0,
		state.Q0,
		payBaseAmount,
		state.I,
		state.K,
	)
}

// https://github.com/DODOEX/contractV2/blob/c58c067c4038437610a9cc8aef8f8025e2af4f63/contracts/lib/PMMPricing.sol#L135
func _ROneSellQuoteToken(state PMMState, payQuoteAmount *uint256.Int) *uint256.Int {
	return SolveQuadraticFunctionForTrade(
		state.B0,
		state.B0,
		payQuoteAmount,
		DecimalMathReciprocalFloor(state.I),
		state.K,
	)
}

// ============ R < 1 cases ============

// https://github.com/DODOEX/contractV2/blob/c58c067c4038437610a9cc8aef8f8025e2af4f63/contracts/lib/PMMPricing.sol#L154
func _RBelowSellQuoteToken(state PMMState, payQuoteAmount *uint256.Int) *uint256.Int {
	return GeneralIntegrate(
		state.Q0,
		SafeAdd(state.Q, payQuoteAmount),
		state.Q,
		DecimalMathReciprocalFloor(state.I),
		state.K,
	)
}

// https://github.com/DODOEX/contractV2/blob/c58c067c4038437610a9cc8aef8f8025e2af4f63/contracts/lib/PMMPricing.sol#L171
func _RBelowSellBaseToken(state PMMState, payBaseAmount *uint256.Int) *uint256.Int {
	return SolveQuadraticFunctionForTrade(
		state.Q0,
		state.Q,
		payBaseAmount,
		state.I,
		state.K,
	)
}

// ============ R > 1 cases ============

// https://github.com/DODOEX/contractV2/blob/c58c067c4038437610a9cc8aef8f8025e2af4f63/contracts/lib/PMMPricing.sol#L190
func _RAboveSellBaseToken(state PMMState, payBaseAmount *uint256.Int) *uint256.Int {
	return GeneralIntegrate(
		state.B0,
		SafeAdd(state.B, payBaseAmount),
		state.B,
		state.I,
		state.K,
	)
}

// https://github.com/DODOEX/contractV2/blob/c58c067c4038437610a9cc8aef8f8025e2af4f63/contracts/lib/PMMPricing.sol#L207
func _RAboveSellQuoteToken(state PMMState, payQuoteAmount *uint256.Int) *uint256.Int {
	return SolveQuadraticFunctionForTrade(
		state.B0,
		state.B,
		payQuoteAmount,
		DecimalMathReciprocalFloor(state.I),
		state.K,
	)
}

// ============ Helper functions ============

// AdjustedTarget https://github.com/DODOEX/contractV2/blob/c58c067c4038437610a9cc8aef8f8025e2af4f63/contracts/lib/PMMPricing.sol#L226
func AdjustedTarget(state *PMMState) {
	if state.R == RStateBelowOne {
		state.Q0 = SolveQuadraticFunctionForTarget(
			state.Q,
			SafeSub(state.B, state.B0),
			state.I,
			state.K,
		)
	} else if state.R == RStateAboveOne {
		state.B0 = SolveQuadraticFunctionForTarget(
			state.B,
			SafeSub(state.Q, state.Q0),
			DecimalMathReciprocalFloor(state.I),
			state.K,
		)
	}
}
