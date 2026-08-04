package metronomeswap

import (
	"github.com/holiman/uint256"

	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// quoteTokenToUsd mirrors Metronome's MasterOracle.quoteTokenToUsd:
//
//	(amountIn * priceInUsd) / 10**decimals
//
// priceInUsd is always 1e18-scaled regardless of the token's own decimals (confirmed
// against MasterOracle's verified source: getPriceInUsd returns a WAD-scaled USD price
// per whole token). This must stay a standalone floored division — MasterOracle.quote()
// chains this into quoteUsdToToken as two SEPARATE floor divisions, not one combined
// formula, and the two-step order must match exactly to reproduce on-chain rounding to
// the wei (see context/metronome/docs/simulations/smoke-*.md for the verified fixture).
func quoteTokenToUsd(amountIn, priceInUsd *uint256.Int, decimals uint8) *uint256.Int {
	return big256.MulDivDown(new(uint256.Int), amountIn, priceInUsd, big256.TenPow(decimals))
}

// quoteUsdToToken mirrors Metronome's MasterOracle.quoteUsdToToken:
//
//	(amountInUsd * 10**decimals) / priceInUsd
func quoteUsdToToken(amountInUsd, priceInUsd *uint256.Int, decimals uint8) *uint256.Int {
	return big256.MulDivDown(new(uint256.Int), amountInUsd, big256.TenPow(decimals), priceInUsd)
}

// quote mirrors Metronome's MasterOracle.quote: the gross (pre-fee) amountOut for a
// swap of amountIn of tokenIn into tokenOut, purely via oracle USD prices — there is no
// bonding curve and no pooled reserves in this protocol.
func quote(
	amountIn *uint256.Int, decimalsIn uint8, priceInUsd *uint256.Int,
	decimalsOut uint8, priceOutUsd *uint256.Int,
) *uint256.Int {
	usdValue := quoteTokenToUsd(amountIn, priceInUsd, decimalsIn)
	return quoteUsdToToken(usdValue, priceOutUsd, decimalsOut)
}

// swapFee mirrors Metronome's swap fee: `_amountOut.wadMul(_swapFee)` in Pool.quoteSwapOut,
// where wadMul comes from Metronome's WadRayMath library (Aave-style):
//
//	wadMul(a, b) = (a*b + WAD/2) / WAD   — ROUND-HALF-UP, not floor.
//
// feeBps1e18 is FeeProvider.swapFees(tokenIn, tokenOut) — a WAD-scaled fraction (e.g.
// 45bps == 4_500_000_000_000_000), NOT basis points out of 10_000. Fee is taken from the
// gross (pre-fee) output amount.
//
// This is NOT big256.MulWadDown (floor) — that was the first implementation here and it
// matched the byte-exact msETH->msUSD fixture from dex-explorer by coincidence (that
// fixture's fractional remainder happened to be < 0.5) but was proven wrong by dex-verify's
// step-4 Tenderly match on a msUSD->msETH row (300bps fee) where the real remainder is
// >= 0.5: on-chain gave fee=16219930989769, floor-division predicts 16219930989768 — off by
// exactly 1 wei. Round-half-up matches both directions exactly.
func swapFee(grossAmountOut, feeBps1e18 *uint256.Int) *uint256.Int {
	quotient := big256.MulDivDown(new(uint256.Int), grossAmountOut, feeBps1e18, big256.BONE)
	remainder := new(uint256.Int).MulMod(grossAmountOut, feeBps1e18, big256.BONE)
	remainder.Add(remainder, remainder) // 2*remainder, safe: remainder < BONE < 2^255

	if remainder.Cmp(big256.BONE) >= 0 {
		quotient.AddUint64(quotient, 1)
	}
	return quotient
}
