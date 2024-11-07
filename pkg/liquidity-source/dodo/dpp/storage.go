package dpp

import (
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dodo/libv2"
)

// https://github.com/DODOEX/contractV2/blob/c58c067c4038437610a9cc8aef8f8025e2af4f63/contracts/DODOPrivatePool/impl/DPPStorage.sol#L51
func (p *PoolSimulator) getPMMState() libv2.PMMState {
	p.Lock()
	defer p.Unlock()

	libv2.AdjustedTarget(&p.PMMState)
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
