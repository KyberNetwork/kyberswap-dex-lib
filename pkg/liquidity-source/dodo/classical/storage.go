package classical

import (
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
)

// https://github.com/DODOEX/dodo-smart-contract/blob/d983485948a55d0ee846951e02cf911633b08d96/contracts/impl/Storage.sol#L94
func (p *PoolSimulator) getOraclePrice() *uint256.Int {
	return p.OraclePrice
}

func (p *PoolSimulator) _K_() *uint256.Int {
	return p.K
}

func (p *PoolSimulator) UpdateStateSellBase(amountIn *uint256.Int, amountOut *uint256.Int) {
	// state.B = state.B + amountInF
	// state.Q = state.Q - outputAmountF
	p.B = number.Add(p.B, amountIn)
	p.Q = number.Sub(p.Q, amountOut)

	if p.RStatus == rStatusOne {
		p.RStatus = rStatusBelowOne
	} else if p.RStatus == rStatusAboveOne {
		backToOnePayBase := number.Sub(p.B0, p.B)

		if amountIn.Cmp(backToOnePayBase) < 0 {
			p.RStatus = rStatusAboveOne
		} else if amountIn.Cmp(backToOnePayBase) == 0 {
			p.RStatus = rStatusOne
		} else {
			p.RStatus = rStatusBelowOne
		}
	} else {
		p.RStatus = rStatusBelowOne
	}
}

func (p *PoolSimulator) UpdateStateBuyBase(amountIn *uint256.Int, amountOut *uint256.Int) {
	// state.B = state.B - amountOut
	// state.Q = state.Q + amountIn
	p.B = number.Sub(p.B, amountOut)
	p.Q = number.Add(p.Q, amountIn)

	if p.RStatus == rStatusOne {
		p.RStatus = rStatusAboveOne
	} else if p.RStatus == rStatusAboveOne {
		p.RStatus = rStatusAboveOne
	} else {
		backToOnePayQuote := number.Sub(p.Q0, p.Q)

		if amountIn.Cmp(backToOnePayQuote) < 0 {
			p.RStatus = rStatusBelowOne
		} else if amountIn.Cmp(backToOnePayQuote) == 0 {
			p.RStatus = rStatusOne
		} else {
			p.RStatus = rStatusAboveOne
		}
	}
}
