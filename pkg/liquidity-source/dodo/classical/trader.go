package classical

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dodo/libv1"
)

// https://github.com/DODOEX/dodo-smart-contract/blob/d983485948a55d0ee846951e02cf911633b08d96/contracts/impl/Trader.sol#L161
func (p *PoolSimulator) _querySellBaseToken(amount *uint256.Int) (
	receiveQuote *uint256.Int,
	lpFeeQuote *uint256.Int,
	mtFeeQuote *uint256.Int,
	newRStatus int,
	newQuoteTarget *uint256.Int,
	newBaseTarget *uint256.Int,
	err error,
) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	newBaseTarget, newQuoteTarget = p.getExpectedTarget()

	sellBaseAmount := amount

	if p.RStatus == rStatusOne {
		receiveQuote = p._ROneSellBaseToken(sellBaseAmount, newQuoteTarget)
		newRStatus = rStatusBelowOne
	} else if p.RStatus == rStatusAboveOne {
		backToOnePayBase := libv1.SafeSub(newBaseTarget, p.B)
		backToOneReceiveQuote := libv1.SafeSub(p.Q, newQuoteTarget)
		if sellBaseAmount.Cmp(backToOnePayBase) < 0 {
			receiveQuote = p._RAboveSellBaseToken(sellBaseAmount, p.B, newBaseTarget)
			newRStatus = rStatusAboveOne
			if receiveQuote.Cmp(backToOneReceiveQuote) > 0 {
				receiveQuote = backToOneReceiveQuote
			}
		} else if sellBaseAmount.Cmp(backToOnePayBase) == 0 {
			receiveQuote = backToOneReceiveQuote
			newRStatus = rStatusOne
		} else {
			receiveQuote = libv1.SafeAdd(
				backToOneReceiveQuote,
				p._ROneSellBaseToken(libv1.SafeSub(sellBaseAmount, backToOnePayBase), newQuoteTarget),
			)
			newRStatus = rStatusBelowOne
		}
	} else {
		receiveQuote = p._RBelowSellBaseToken(sellBaseAmount, p.Q, newQuoteTarget)
		newRStatus = rStatusBelowOne
	}

	lpFeeQuote = libv1.DecimalMathMul(receiveQuote, p.LpFeeRate)
	mtFeeQuote = libv1.DecimalMathMul(receiveQuote, p.MtFeeRate)
	receiveQuote = libv1.SafeSub(
		libv1.SafeSub(receiveQuote, lpFeeQuote),
		mtFeeQuote,
	)

	return receiveQuote, lpFeeQuote, mtFeeQuote, newRStatus, newQuoteTarget, newBaseTarget, err
}

// https://github.com/DODOEX/dodo-smart-contract/blob/d983485948a55d0ee846951e02cf911633b08d96/contracts/impl/Trader.sol#L222
func (p *PoolSimulator) _queryBuyBaseToken(amount *uint256.Int) (
	payQuote *uint256.Int,
	lpFeeBase *uint256.Int,
	mtFeeBase *uint256.Int,
	newRStatus int,
	newQuoteTarget *uint256.Int,
	newBaseTarget *uint256.Int,
	err error,
) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	newBaseTarget, newQuoteTarget = p.getExpectedTarget()

	lpFeeBase = libv1.DecimalMathMul(amount, p.LpFeeRate)
	mtFeeBase = libv1.DecimalMathMul(amount, p.MtFeeRate)
	buyBaseAmount := libv1.SafeAdd(
		libv1.SafeAdd(amount, lpFeeBase),
		mtFeeBase,
	)

	if p.RStatus == rStatusOne {
		payQuote = p._ROneBuyBaseToken(buyBaseAmount, newBaseTarget)
		newRStatus = rStatusAboveOne
	} else if p.RStatus == rStatusAboveOne {
		payQuote = p._RAboveBuyBaseToken(buyBaseAmount, p.B, newBaseTarget)
		newRStatus = rStatusAboveOne
	} else if p.RStatus == rStatusBelowOne {
		backToOnePayQuote := libv1.SafeSub(newQuoteTarget, p.Q)
		backToOneReceiveBase := libv1.SafeSub(p.B, newBaseTarget)
		if buyBaseAmount.Cmp(backToOneReceiveBase) < 0 {
			payQuote = p._RBelowBuyBaseToken(buyBaseAmount, p.Q, newQuoteTarget)
			newRStatus = rStatusBelowOne
		} else if buyBaseAmount.Cmp(backToOneReceiveBase) == 0 {
			payQuote = backToOnePayQuote
			newRStatus = rStatusOne
		} else {
			payQuote = libv1.SafeAdd(
				backToOnePayQuote,
				p._ROneBuyBaseToken(libv1.SafeSub(buyBaseAmount, backToOneReceiveBase), newBaseTarget),
			)
			newRStatus = rStatusAboveOne
		}
	}

	return payQuote, lpFeeBase, mtFeeBase, newRStatus, newQuoteTarget, newBaseTarget, err
}
