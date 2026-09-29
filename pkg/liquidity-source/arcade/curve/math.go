package curve

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// BuyResult mirrors ArcadeV4Curve.BuyResult.
type BuyResult struct {
	TokensOut   *uint256.Int
	ActualGross *uint256.Int
	Fee         *uint256.Int
	Refund      *uint256.Int
}

// SellResult mirrors ArcadeV4Curve.SellResult.
type SellResult struct {
	GrossOut *uint256.Int
	UsdcOut  *uint256.Int
	Fee      *uint256.Int
}

func zero() *uint256.Int { return new(uint256.Int) }

// SimulateBuy ports ArcadeV4Curve.simulateBuy exactly, including the cap path that
// clips a buy crossing CURVE_SUPPLY to the remaining tokens and refunds the rest.
// A zero result (tokensOut == 0) makes the hook revert.
func SimulateBuy(tokensSold, realUsdcReserve, grossUsdcIn *uint256.Int) BuyResult {
	r := BuyResult{TokensOut: zero(), ActualGross: zero(), Fee: zero(), Refund: zero()}
	if grossUsdcIn.IsZero() || tokensSold.Cmp(curveSupply) >= 0 {
		return r
	}

	fee := big256.MulDivDown(new(uint256.Int), grossUsdcIn, uTradeFeeBps, uFeeDenominator)
	netIn := new(uint256.Int).Sub(grossUsdcIn, fee)

	currentUsdc := new(uint256.Int).Add(virtualUsdcReserve, realUsdcReserve)
	currentTokens := new(uint256.Int).Sub(virtualTokenReserve, tokensSold)

	newUsdcReserve := new(uint256.Int).Add(currentUsdc, netIn)
	newTokenReserve := new(uint256.Int).Div(kConstant, newUsdcReserve)
	if newTokenReserve.Cmp(currentTokens) > 0 { // cannot happen with netIn >= 0; defensive
		return r
	}
	desiredOut := new(uint256.Int).Sub(currentTokens, newTokenReserve)

	maxOut := new(uint256.Int).Sub(curveSupply, tokensSold)

	if desiredOut.Cmp(maxOut) <= 0 {
		r.TokensOut = desiredOut
		r.ActualGross = new(uint256.Int).Set(grossUsdcIn)
		r.Fee = fee
		return r
	}

	capTokenReserve := new(uint256.Int).Sub(currentTokens, maxOut)
	capUsdcReserve := big256.DivUp(kConstant, capTokenReserve)
	actualNet := new(uint256.Int).Sub(capUsdcReserve, currentUsdc)
	denominator := uint256.NewInt(feeDenominator - tradeFeeBps)
	actualGross := big256.MulDivUp(new(uint256.Int), actualNet, uFeeDenominator, denominator)
	if actualGross.Cmp(grossUsdcIn) > 0 {
		actualGross = new(uint256.Int).Set(grossUsdcIn)
	}
	actualFee := big256.MulDivDown(new(uint256.Int), actualGross, uTradeFeeBps, uFeeDenominator)

	r.TokensOut = maxOut
	r.ActualGross = actualGross
	r.Fee = actualFee
	r.Refund = new(uint256.Int).Sub(grossUsdcIn, actualGross)
	return r
}

// SimulateSell ports ArcadeV4Curve.simulateSell exactly (including its dust clip to
// the real reserve and the no-op on degenerate floor rounding).
func SimulateSell(tokensSold, realUsdcReserve, tokensIn *uint256.Int) SellResult {
	r := SellResult{GrossOut: zero(), UsdcOut: zero(), Fee: zero()}
	if tokensIn.IsZero() {
		return r
	}
	in := tokensIn
	if in.Cmp(tokensSold) > 0 {
		in = tokensSold
	}

	currentUsdc := new(uint256.Int).Add(virtualUsdcReserve, realUsdcReserve)
	currentTokens := new(uint256.Int).Sub(virtualTokenReserve, tokensSold)

	newTokenReserve := new(uint256.Int).Add(currentTokens, in)
	newUsdcReserve := new(uint256.Int).Div(kConstant, newTokenReserve)
	if newUsdcReserve.Cmp(currentUsdc) >= 0 {
		return r
	}
	grossOut := new(uint256.Int).Sub(currentUsdc, newUsdcReserve)
	if grossOut.Cmp(realUsdcReserve) > 0 {
		grossOut = new(uint256.Int).Set(realUsdcReserve)
	}

	fee := big256.MulDivDown(new(uint256.Int), grossOut, uTradeFeeBps, uFeeDenominator)
	r.GrossOut = grossOut
	r.UsdcOut = new(uint256.Int).Sub(grossOut, fee)
	r.Fee = fee
	return r
}

// CurrentSnipeBps ports ArcadeHook._currentSnipeBps for a launch still on its curve.
func (e *Extra) CurrentSnipeBps(now int64) uint64 {
	if e.SnipeStartBps == 0 || e.SnipeDecaySeconds == 0 || e.SnipeLaunchedAt == 0 {
		return 0
	}
	if e.Status == statusGraduated {
		return 0
	}
	elapsed := uint64(0)
	if uint64(now) > e.SnipeLaunchedAt {
		elapsed = uint64(now) - e.SnipeLaunchedAt
	}
	if elapsed >= uint64(e.SnipeDecaySeconds) {
		return 0
	}
	return uint64(e.SnipeStartBps) * (uint64(e.SnipeDecaySeconds) - elapsed) / uint64(e.SnipeDecaySeconds)
}
