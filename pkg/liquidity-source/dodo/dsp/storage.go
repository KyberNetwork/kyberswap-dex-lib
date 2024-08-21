package dsp

import (
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dodo/libv2"
)

// https://github.com/DODOEX/contractV2/blob/c58c067c4038437610a9cc8aef8f8025e2af4f63/contracts/DODOStablePool/impl/DSPStorage.sol#L70
func (p *PoolSimulator) getPMMState() libv2.PMMState {
	// This function is a bit different compare to the Solidity code
	// We don't run adjustedTarget here to avoid issue when cloning pool https://team-kyber.slack.com/archives/C061UNZDUVC/p1724213576872309
	// The adjustedTarget will be called in the NewPoolSimulator function when initializing the pool or in the UpdateState function when updating the pool

	return p.PMMState
}

func (p *PoolSimulator) UpdateStateSellBase(amountIn *uint256.Int, amountOut *uint256.Int) {
	// state.B = state.B + amountInF
	// state.Q = state.Q - outputAmountF
	p.B = number.Add(p.B, amountIn)
	p.Q = number.Sub(p.Q, amountOut)

	if p.R == libv2.RStateOne {
		p.R = libv2.RStateBelowOne
	} else if p.R == libv2.RStateAboveOne {
		backToOnePayBase := number.Sub(p.B0, p.B)

		if amountIn.Cmp(backToOnePayBase) < 0 {
			p.R = libv2.RStateAboveOne
		} else if amountIn.Cmp(backToOnePayBase) == 0 {
			p.R = libv2.RStateOne
		} else {
			p.R = libv2.RStateBelowOne
		}
	} else {
		p.R = libv2.RStateBelowOne
	}
}

func (p *PoolSimulator) UpdateStateSellQuote(amountIn *uint256.Int, amountOut *uint256.Int) {
	// state.B = state.B - amountOut
	// state.Q = state.Q + amountIn
	p.B = number.Sub(p.B, amountOut)
	p.Q = number.Add(p.Q, amountIn)

	if p.R == libv2.RStateOne {
		p.R = libv2.RStateAboveOne
	} else if p.R == libv2.RStateAboveOne {
		p.R = libv2.RStateAboveOne
	} else {
		backToOnePayQuote := number.Sub(p.Q0, p.Q)

		if amountIn.Cmp(backToOnePayQuote) < 0 {
			p.R = libv2.RStateBelowOne
		} else if amountIn.Cmp(backToOnePayQuote) == 0 {
			p.R = libv2.RStateOne
		} else {
			p.R = libv2.RStateAboveOne
		}
	}
}
