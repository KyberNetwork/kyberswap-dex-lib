package classical

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dodo/libv1"
)

// ============ R = 1 cases ============

// https://github.com/DODOEX/dodo-smart-contract/blob/d983485948a55d0ee846951e02cf911633b08d96/contracts/impl/Pricing.sol#L29
func (p *PoolSimulator) _ROneSellBaseToken(amount, targetQuoteTokenAmount *uint256.Int) *uint256.Int {
	i := p.getOraclePrice()
	Q2 := libv1.SolveQuadraticFunctionForTrade(
		targetQuoteTokenAmount,
		targetQuoteTokenAmount,
		libv1.DecimalMathMul(i, amount),
		false,
		p._K_(),
	)

	return libv1.SafeSub(targetQuoteTokenAmount, Q2)
}

// https://github.com/DODOEX/dodo-smart-contract/blob/d983485948a55d0ee846951e02cf911633b08d96/contracts/impl/Pricing.sol#L47
func (p *PoolSimulator) _ROneBuyBaseToken(amount, targetBaseTokenAmount *uint256.Int) *uint256.Int {
	if amount.Cmp(targetBaseTokenAmount) >= 0 {
		panic(ErrBaseBalanceNotEnough)
	}
	B2 := libv1.SafeSub(targetBaseTokenAmount, amount)
	return p._RAboveIntegrate(targetBaseTokenAmount, targetBaseTokenAmount, B2)
}

// ============ R < 1 cases ============

// https://github.com/DODOEX/dodo-smart-contract/blob/d983485948a55d0ee846951e02cf911633b08d96/contracts/impl/Pricing.sol#L60
func (p *PoolSimulator) _RBelowSellBaseToken(amount, quoteBalance, targetQuoteAmount *uint256.Int) *uint256.Int {
	i := p.getOraclePrice()
	Q2 := libv1.SolveQuadraticFunctionForTrade(
		targetQuoteAmount,
		quoteBalance,
		libv1.DecimalMathMul(i, amount),
		false,
		p._K_(),
	)

	return libv1.SafeSub(quoteBalance, Q2)
}

// https://github.com/DODOEX/dodo-smart-contract/blob/d983485948a55d0ee846951e02cf911633b08d96/contracts/impl/Pricing.sol#L76
func (p *PoolSimulator) _RBelowBuyBaseToken(amount, quoteBalance, targetQuoteAmount *uint256.Int) *uint256.Int {
	i := p.getOraclePrice()
	Q2 := libv1.SolveQuadraticFunctionForTrade(
		targetQuoteAmount,
		quoteBalance,
		libv1.DecimalMathMul(i, amount),
		true,
		p._K_(),
	)

	return libv1.SafeSub(Q2, quoteBalance)
}

// https://github.com/DODOEX/dodo-smart-contract/blob/d983485948a55d0ee846951e02cf911633b08d96/contracts/impl/Pricing.sol#L95
func (p *PoolSimulator) _RBelowBackToOne() *uint256.Int {
	spareBase := libv1.SafeSub(p.B, p.B0)
	price := p.getOraclePrice()
	fairAmount := libv1.DecimalMathMul(spareBase, price)
	newTargetQuote := libv1.SolveQuadraticFunctionForTarget(
		p.Q,
		p._K_(),
		fairAmount,
	)

	return libv1.SafeSub(newTargetQuote, p.Q)
}

// ============ R > 1 cases ============

// https://github.com/DODOEX/dodo-smart-contract/blob/d983485948a55d0ee846951e02cf911633b08d96/contracts/impl/Pricing.sol#L110
func (p *PoolSimulator) _RAboveBuyBaseToken(amount, baseBalance, targetBaseAmount *uint256.Int) *uint256.Int {
	if amount.Cmp(baseBalance) >= 0 {
		panic(ErrBaseBalanceNotEnough)
	}
	B2 := libv1.SafeSub(baseBalance, amount)
	return p._RAboveIntegrate(targetBaseAmount, baseBalance, B2)
}

// https://github.com/DODOEX/dodo-smart-contract/blob/d983485948a55d0ee846951e02cf911633b08d96/contracts/impl/Pricing.sol#L120
func (p *PoolSimulator) _RAboveSellBaseToken(amount, baseBalance, targetBaseAmount *uint256.Int) *uint256.Int {
	B1 := libv1.SafeAdd(baseBalance, amount)
	return p._RAboveIntegrate(targetBaseAmount, B1, baseBalance)
}

// https://github.com/DODOEX/dodo-smart-contract/blob/d983485948a55d0ee846951e02cf911633b08d96/contracts/impl/Pricing.sol#L132
func (p *PoolSimulator) _RAboveBackToOne() *uint256.Int {
	spareQuote := libv1.SafeSub(p.Q, p.Q0)
	price := p.getOraclePrice()
	fairAmount := libv1.DecimalMathDivFloor(spareQuote, price)
	newTargetBase := libv1.SolveQuadraticFunctionForTarget(
		p.B,
		p._K_(),
		fairAmount,
	)

	return libv1.SafeSub(newTargetBase, p.B)
}

// ============ Helper functions ============

// https://github.com/DODOEX/dodo-smart-contract/blob/d983485948a55d0ee846951e02cf911633b08d96/contracts/impl/Pricing.sol#L147
func (p *PoolSimulator) getExpectedTarget() (baseTarget, quoteTarget *uint256.Int) {
	Q := p.Q
	B := p.B
	if p.RStatus == rStatusOne {
		return p.B0, p.Q0
	} else if p.RStatus == rStatusBelowOne {
		payQuoteToken := p._RBelowBackToOne()
		return p.B0, libv1.SafeAdd(Q, payQuoteToken)
	} else if p.RStatus == rStatusAboveOne {
		payBaseToken := p._RAboveBackToOne()
		return libv1.SafeAdd(B, payBaseToken), p.Q0
	}

	panic(ErrInvalidRStatus)
}

// https://github.com/DODOEX/dodo-smart-contract/blob/d983485948a55d0ee846951e02cf911633b08d96/contracts/impl/Pricing.sol#L180
func (p *PoolSimulator) _RAboveIntegrate(B0, B1, B2 *uint256.Int) *uint256.Int {
	i := p.getOraclePrice()
	return libv1.GeneralIntegrate(B0, B1, B2, i, p._K_())
}
