package ponsv2

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// referencePoolJSON mirrors the real, verified on-chain state of curve
// 0x4bd421eb79aca48d13793c7140582c8ef312d124 (pairToken TSLA, launched token
// "Milo") sampled at Robinhood-chain block 25970537, as captured in
// context/pons-v2/output/explorer.md. Tokens[0]=TSLA (quote), Tokens[1]=Milo
// (launched token).
const referencePoolJSON = `{
	"address": "0x4bd421eb79aca48d13793c7140582c8ef312d124",
	"exchange": "pons-v2",
	"type": "pons-v2",
	"timestamp": 0,
	"reserves": ["10402378355729293768", "999771364235373746376058260"],
	"tokens": [
		{"address": "0x322f0929c4625ed5bad873c95208d54e1c003b2d", "symbol": "TSLA", "decimals": 18, "swappable": true},
		{"address": "0x0b6ebf6e9d2d9a7e1f4e9d1453138781fd81ba95", "symbol": "Milo", "decimals": 18, "swappable": true}
	],
	"extra": "{\"quoteReserve\":\"10402378355729293768\",\"tokenReserve\":\"999771364235373746376058260\",\"graduated\":false}",
	"staticExtra": "{\"feeBps\":100,\"creatorTaxBps\":1000,\"reservedTokens\":\"285714285714285714285714285\",\"isNativeQuote\":false}"
}`

const quoteToken = "0x322f0929c4625ed5bad873c95208d54e1c003b2d"
const launchedToken = "0x0b6ebf6e9d2d9a7e1f4e9d1453138781fd81ba95"

func mustNewSimulator(t *testing.T, poolJSON string) *PoolSimulator {
	t.Helper()
	var ep entity.Pool
	require.NoError(t, json.Unmarshal([]byte(poolJSON), &ep))
	sim, err := NewPoolSimulator(ep)
	require.NoError(t, err)
	return sim
}

// TestCalcAmountOut_Buy_NoClamp exercises a small buy, entirely below the
// sellableTokens() ceiling, cross-checked against a from-scratch port of
// PonsV2BondingCurve.buy() run in Python during discovery (see
// context/pons-v2/output/explorer.md evidence).
func TestCalcAmountOut_Buy_NoClamp(t *testing.T) {
	sim := mustNewSimulator(t, referencePoolJSON)

	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: quoteToken, Amount: bigFromString("1000000000000000000")},
		TokenOut:      launchedToken,
	})
	require.NoError(t, err)
	require.Equal(t, "78796200954251240723387962", result.TokenAmountOut.Amount.String())
	require.Nil(t, result.RemainingTokenAmountIn)
	// feeBps(100) + creatorTaxBps(1000) = 11% of the 1e18 quote input, always
	// reported on the quote leg (Tokens[0]) regardless of swap direction.
	require.Equal(t, quoteToken, result.Fee.Token)
	require.Equal(t, "110000000000000000", result.Fee.Amount.String())

	swapInfo, ok := result.SwapInfo.(*SwapInfo)
	require.True(t, ok)
	require.True(t, swapInfo.IsBuy)
	require.Equal(t, "11292378355729293768", swapInfo.NewQuoteReserve.String())
	require.Equal(t, "920975163281122505652670298", swapInfo.NewTokenReserve.String())

	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: swapInfo})
	require.Equal(t, "11292378355729293768", sim.quoteReserve.String())
	require.Equal(t, "920975163281122505652670298", sim.tokenReserve.String())
}

// TestCalcAmountOut_Sell exercises a small sell (token -> quote), full input
// always consumed, no partial fill.
func TestCalcAmountOut_Sell(t *testing.T) {
	sim := mustNewSimulator(t, referencePoolJSON)

	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: launchedToken, Amount: bigFromString("1000000000000000000000")},
		TokenOut:      quoteToken,
	})
	require.NoError(t, err)
	require.Equal(t, "9260224694928", result.TokenAmountOut.Amount.String())
	require.Nil(t, result.RemainingTokenAmountIn)
	// Fee/tax on a sell are deducted from the quote OUTPUT (grossQuoteOut),
	// but still reported on the quote leg (Tokens[0]), same as a buy.
	require.Equal(t, quoteToken, result.Fee.Token)
	require.Equal(t, "1144522153305", result.Fee.Amount.String())

	swapInfo, ok := result.SwapInfo.(*SwapInfo)
	require.True(t, ok)
	require.False(t, swapInfo.IsBuy)
	require.Equal(t, "10402367950982445535", swapInfo.NewQuoteReserve.String())
	require.Equal(t, "999772364235373746376058260", swapInfo.NewTokenReserve.String())
}

// TestCalcAmountOut_Buy_ClampedAndRefunded is the high-risk path flagged in
// explorer.md: a buy sized past sellableTokens() must fill only up to the
// sellable ceiling, re-derive spent via getAmountIn grossed back up through
// the fee rate (ceiling rounding), and refund the remainder rather than
// reverting.
func TestCalcAmountOut_Buy_ClampedAndRefunded(t *testing.T) {
	sim := mustNewSimulator(t, referencePoolJSON)

	sellable := sim.sellableTokens()
	require.Equal(t, "714057078521088032090343975", sellable.String())

	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: quoteToken, Amount: bigFromString("30210810836259220523")},
		TokenOut:      launchedToken,
	})
	require.NoError(t, err)

	// Clamped exactly to sellableTokens().
	require.Equal(t, sellable.String(), result.TokenAmountOut.Amount.String())
	require.NotNil(t, result.RemainingTokenAmountIn)
	require.Equal(t, "999999999999999998", result.RemainingTokenAmountIn.Amount.String())
	// Fee is computed on the re-derived, grossed-up `spent` (not the
	// originally offered `received`), since only `spent` is actually pulled
	// by the curve once the buy clamps to sellableTokens().
	require.Equal(t, quoteToken, result.Fee.Token)
	require.Equal(t, "3213189191988514257", result.Fee.Amount.String())

	swapInfo, ok := result.SwapInfo.(*SwapInfo)
	require.True(t, ok)
	require.Equal(t, "36400000000000000036", swapInfo.NewQuoteReserve.String())
	// tokenReserve is drawn down to exactly ReservedTokens once fully clamped.
	require.Equal(t, "285714285714285714285714285", swapInfo.NewTokenReserve.String())

	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: swapInfo})
	require.True(t, sim.sellableTokens().IsZero())
}

// TestCalcAmountOut_Graduated verifies both buy and sell are rejected once
// graduated() is true.
func TestCalcAmountOut_Graduated(t *testing.T) {
	graduatedPoolJSON := `{
		"address": "0x5e40dd2c72b76b2e51dbb33b1526b3b2e99ff016",
		"exchange": "pons-v2",
		"type": "pons-v2",
		"timestamp": 0,
		"reserves": ["9680000000000000000", "0"],
		"tokens": [
			{"address": "0x322f0929c4625ed5bad873c95208d54e1c003b2d", "symbol": "TSLA", "decimals": 18, "swappable": true},
			{"address": "0x91c4b7a5c655c2fcd7bc173c1e8e2469352695c2", "symbol": "TOK", "decimals": 18, "swappable": true}
		],
		"extra": "{\"quoteReserve\":\"9680000000000000000\",\"tokenReserve\":\"0\",\"graduated\":true}",
		"staticExtra": "{\"feeBps\":100,\"creatorTaxBps\":1000,\"reservedTokens\":\"0\",\"isNativeQuote\":false}"
	}`
	sim := mustNewSimulator(t, graduatedPoolJSON)

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: quoteToken, Amount: bigFromString("1000000000000000000")},
		TokenOut:      "0x91c4b7a5c655c2fcd7bc173c1e8e2469352695c2",
	})
	require.ErrorIs(t, err, ErrPoolGraduated)

	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0x91c4b7a5c655c2fcd7bc173c1e8e2469352695c2", Amount: bigFromString("1000000000000000000")},
		TokenOut:      quoteToken,
	})
	require.ErrorIs(t, err, ErrPoolGraduated)
}

// TestCalcAmountOut_ReadyToGraduate_NotYetFlagged verifies the window where
// sellableTokens()==0 but the on-chain graduated flag hasn't flipped yet
// (auto-graduation can fail/lag): both buy and sell must still be rejected,
// matching PonsV2BondingCurve.sell()'s `graduated || readyToGraduate()` gate.
func TestCalcAmountOut_ReadyToGraduate_NotYetFlagged(t *testing.T) {
	readyPoolJSON := `{
		"address": "0x4bd421eb79aca48d13793c7140582c8ef312d124",
		"exchange": "pons-v2",
		"type": "pons-v2",
		"timestamp": 0,
		"reserves": ["36400000000000000036", "285714285714285714285714285"],
		"tokens": [
			{"address": "0x322f0929c4625ed5bad873c95208d54e1c003b2d", "symbol": "TSLA", "decimals": 18, "swappable": true},
			{"address": "0x0b6ebf6e9d2d9a7e1f4e9d1453138781fd81ba95", "symbol": "Milo", "decimals": 18, "swappable": true}
		],
		"extra": "{\"quoteReserve\":\"36400000000000000036\",\"tokenReserve\":\"285714285714285714285714285\",\"graduated\":false}",
		"staticExtra": "{\"feeBps\":100,\"creatorTaxBps\":1000,\"reservedTokens\":\"285714285714285714285714285\",\"isNativeQuote\":false}"
	}`
	sim := mustNewSimulator(t, readyPoolJSON)
	require.False(t, sim.graduated)
	require.True(t, sim.sellableTokens().IsZero())

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: launchedToken, Amount: bigFromString("1000000000000000000")},
		TokenOut:      quoteToken,
	})
	require.ErrorIs(t, err, ErrPoolGraduated)

	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: quoteToken, Amount: bigFromString("1000000000000000000")},
		TokenOut:      launchedToken,
	})
	require.ErrorIs(t, err, ErrPoolGraduated)
}

// TestCalcAmountOut_NativeQuote_IsNativeQuoteFlag verifies SwapInfo.IsNativeQuote
// mirrors StaticExtra.IsNativeQuote on both buy and sell, since
// aggregator-encoding relies on this flag (not just the wrapped-native token
// address) to decide native msg.value/payout handling -- a curve whose
// pairToken happens to be the real WETH ERC-20 must NOT be treated as native.
func TestCalcAmountOut_NativeQuote_IsNativeQuoteFlag(t *testing.T) {
	nativeQuotePoolJSON := `{
		"address": "0x497063554d2be2f6e6a38a50a5d734ba29e425f0",
		"exchange": "pons-v2",
		"type": "pons-v2",
		"timestamp": 0,
		"reserves": ["10402378355729293768", "999771364235373746376058260"],
		"tokens": [
			{"address": "0x0bd7d308f8e1639fab988df18a8011f41eacad73", "symbol": "WROBIN", "decimals": 18, "swappable": true},
			{"address": "0x0b6ebf6e9d2d9a7e1f4e9d1453138781fd81ba95", "symbol": "Milo", "decimals": 18, "swappable": true}
		],
		"extra": "{\"quoteReserve\":\"10402378355729293768\",\"tokenReserve\":\"999771364235373746376058260\",\"graduated\":false}",
		"staticExtra": "{\"feeBps\":100,\"creatorTaxBps\":1000,\"reservedTokens\":\"285714285714285714285714285\",\"isNativeQuote\":true}"
	}`
	sim := mustNewSimulator(t, nativeQuotePoolJSON)
	wrappedNative := "0x0bd7d308f8e1639fab988df18a8011f41eacad73"

	buyResult, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: wrappedNative, Amount: bigFromString("1000000000000000000")},
		TokenOut:      launchedToken,
	})
	require.NoError(t, err)
	buySwapInfo, ok := buyResult.SwapInfo.(*SwapInfo)
	require.True(t, ok)
	require.True(t, buySwapInfo.IsNativeQuote)

	sellResult, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: launchedToken, Amount: bigFromString("1000000000000000000000")},
		TokenOut:      wrappedNative,
	})
	require.NoError(t, err)
	sellSwapInfo, ok := sellResult.SwapInfo.(*SwapInfo)
	require.True(t, ok)
	require.True(t, sellSwapInfo.IsNativeQuote)
}

func bigFromString(s string) *big.Int {
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic("invalid big int literal: " + s)
	}
	return v
}

// trackedQuotePoolJSON mirrors the live state of curve
// 0xce942131d2ed8a1e6c73f41f5f29fba154041916 (robinhood chain 4663, block
// 118814263): getReserves() = (10657669321075322920, 999806095666329503613463375),
// trackedQuote() = 2066568268183609, feeBps 100, creatorTaxBps 100. Of the
// quote reserve only trackedQuote is real quote; the rest is phantomQuote.
// Every expected number in the tests below is the result of the same sell/buy
// sequence sent to the curve on a Tenderly fork at that block (eth_call
// bisection of the sell limit and state reads after each transaction).
const trackedQuotePoolJSON = `{
	"address": "0xce942131d2ed8a1e6c73f41f5f29fba154041916",
	"exchange": "pons-v2",
	"type": "pons-v2",
	"timestamp": 0,
	"reserves": ["10657669321075322920", "999806095666329503613463375"],
	"tokens": [
		{"address": "0x5e81213613b6b86eab4c6c50d718d34359459786", "symbol": "Q", "decimals": 18, "swappable": true},
		{"address": "0x1c1dd94a0888e5448e50367c246158be2f05375a", "symbol": "T", "decimals": 18, "swappable": true}
	],
	"extra": "{\"quoteReserve\":\"10657669321075322920\",\"tokenReserve\":\"999806095666329503613463375\",\"trackedQuote\":\"2066568268183609\",\"graduated\":false}",
	"staticExtra": "{\"feeBps\":100,\"creatorTaxBps\":100,\"reservedTokens\":\"285714285714285714289544789\",\"isNativeQuote\":false}"
}`

// trackedQuotePoolWithoutTrackedQuoteJSON is the same pool as tracked before
// Extra.TrackedQuote existed.
const trackedQuotePoolWithoutTrackedQuoteJSON = `{
	"address": "0xce942131d2ed8a1e6c73f41f5f29fba154041916",
	"exchange": "pons-v2",
	"type": "pons-v2",
	"timestamp": 0,
	"reserves": ["10657669321075322920", "999806095666329503613463375"],
	"tokens": [
		{"address": "0x5e81213613b6b86eab4c6c50d718d34359459786", "symbol": "Q", "decimals": 18, "swappable": true},
		{"address": "0x1c1dd94a0888e5448e50367c246158be2f05375a", "symbol": "T", "decimals": 18, "swappable": true}
	],
	"extra": "{\"quoteReserve\":\"10657669321075322920\",\"tokenReserve\":\"999806095666329503613463375\",\"graduated\":false}",
	"staticExtra": "{\"feeBps\":100,\"creatorTaxBps\":100,\"reservedTokens\":\"285714285714285714289544789\",\"isNativeQuote\":false}"
}`

const (
	trackedQuoteQuote = "0x5e81213613b6b86eab4c6c50d718d34359459786"
	trackedQuoteToken = "0x1c1dd94a0888e5448e50367c246158be2f05375a"
)

func calcSell(sim *PoolSimulator, amount string) (*pool.CalcAmountOutResult, error) {
	return sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: trackedQuoteToken, Amount: bigFromString(amount)},
		TokenOut:      trackedQuoteQuote,
	})
}

// TestCalcAmountOut_Sell_RejectedAboveTrackedQuote is the sell faulty-pool-v2
// blocked: priced on the full quote reserve it pays 24259866948353260, ~12x
// the 2066568268183609 of real quote in the curve, and the curve reverts with
// an arithmetic underflow.
func TestCalcAmountOut_Sell_RejectedAboveTrackedQuote(t *testing.T) {
	sim := mustNewSimulator(t, trackedQuotePoolJSON)

	_, err := calcSell(sim, "2327693551619090177812385")
	require.ErrorIs(t, err, ErrInsufficientLiquidity)
}

// TestCalcAmountOut_Sell_TrackedQuoteBoundary pins the exact on-chain limit
// found by bisection: quoteOut (grossQuoteOut - fee - tax) <= trackedQuote.
// grossQuoteOut is above trackedQuote at the last accepted amount, so the
// check is on quoteOut, not on grossQuoteOut.
func TestCalcAmountOut_Sell_TrackedQuoteBoundary(t *testing.T) {
	sim := mustNewSimulator(t, trackedQuotePoolJSON)

	// Last amount the curve accepts: it pays exactly trackedQuote.
	result, err := calcSell(sim, "197862348108825965639777")
	require.NoError(t, err)
	require.Equal(t, "2066568268183609", result.TokenAmountOut.Amount.String())
	swapInfo, ok := result.SwapInfo.(*SwapInfo)
	require.True(t, ok)
	require.True(t, swapInfo.NewTrackedQuote.IsZero())

	// One more wei of input pays 1 wei above trackedQuote: the curve reverts.
	_, err = calcSell(sim, "197862348108825965639778")
	require.ErrorIs(t, err, ErrInsufficientLiquidity)
}

// TestCalcAmountOut_Sell_BelowTrackedQuote_SameQuote: a sell below the limit
// quotes the same amount with and without the new field.
func TestCalcAmountOut_Sell_BelowTrackedQuote_SameQuote(t *testing.T) {
	withField := mustNewSimulator(t, trackedQuotePoolJSON)
	withoutField := mustNewSimulator(t, trackedQuotePoolWithoutTrackedQuoteJSON)

	for _, amount := range []string{"100000000000000000000000", "1000000000000000000000"} {
		a, err := calcSell(withField, amount)
		require.NoError(t, err)
		b, err := calcSell(withoutField, amount)
		require.NoError(t, err)
		require.Equal(t, b.TokenAmountOut.Amount.String(), a.TokenAmountOut.Amount.String())
		require.Equal(t, b.Fee.Amount.String(), a.Fee.Amount.String())
	}

	result, err := calcSell(withField, "100000000000000000000000")
	require.NoError(t, err)
	require.Equal(t, "1044549681207182", result.TokenAmountOut.Amount.String())
}

// TestCalcAmountOut_Sell_MissingTrackedQuote_KeepsOldBehaviour: a pool tracked
// before the field existed quotes the oversized sell as before (no check, no
// tracked quote carried in SwapInfo).
func TestCalcAmountOut_Sell_MissingTrackedQuote_KeepsOldBehaviour(t *testing.T) {
	sim := mustNewSimulator(t, trackedQuotePoolWithoutTrackedQuoteJSON)
	require.Nil(t, sim.trackedQuote)

	result, err := calcSell(sim, "2327693551619090177812385")
	require.NoError(t, err)
	require.Equal(t, "24259866948353260", result.TokenAmountOut.Amount.String())
	swapInfo, ok := result.SwapInfo.(*SwapInfo)
	require.True(t, ok)
	require.Nil(t, swapInfo.NewTrackedQuote)

	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: swapInfo})
	require.Nil(t, sim.trackedQuote)

	buy, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: trackedQuoteQuote, Amount: bigFromString("500000000000000")},
		TokenOut:      trackedQuoteToken,
	})
	require.NoError(t, err)
	buySwapInfo, ok := buy.SwapInfo.(*SwapInfo)
	require.True(t, ok)
	require.Nil(t, buySwapInfo.NewTrackedQuote)
}

// TestUpdateBalance_SellThenBuy_KeepsTrackedQuote replays on the simulator the
// sequence sent to the curve on the fork: sell 1e23 tokens, then buy with
// 5e14 quote. A sell lowers trackedQuote by quoteOut only (fee and tax stay
// in it); a buy raises it by the whole spent quote. The limit of the next sell
// after the first sell was bisected on the fork too.
func TestUpdateBalance_SellThenBuy_KeepsTrackedQuote(t *testing.T) {
	sim := mustNewSimulator(t, trackedQuotePoolJSON)

	sell, err := calcSell(sim, "100000000000000000000000")
	require.NoError(t, err)
	require.Equal(t, "1044549681207182", sell.TokenAmountOut.Amount.String())
	sellInfo, ok := sell.SwapInfo.(*SwapInfo)
	require.True(t, ok)
	require.Equal(t, "1022018586976427", sellInfo.NewTrackedQuote.String())
	require.Equal(t, "10656603454053682940", sellInfo.NewQuoteReserve.String())

	// The next sell is checked against the lowered trackedQuote.
	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: sellInfo})
	require.Equal(t, "1022018586976427", sim.trackedQuote.String())

	result, err := calcSell(sim, "97862348108825848411468")
	require.NoError(t, err)
	require.Equal(t, "1022018586976427", result.TokenAmountOut.Amount.String())
	_, err = calcSell(sim, "97862348108825848411469")
	require.ErrorIs(t, err, ErrInsufficientLiquidity)

	buy, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: trackedQuoteQuote, Amount: bigFromString("500000000000000")},
		TokenOut:      trackedQuoteToken,
	})
	require.NoError(t, err)
	require.Equal(t, "45974447816269793710203", buy.TokenAmountOut.Amount.String())
	buyInfo, ok := buy.SwapInfo.(*SwapInfo)
	require.True(t, ok)
	require.Equal(t, "1522018586976427", buyInfo.NewTrackedQuote.String())
	// QuoteReserve after the buy as read from the fork (0x93e59a086302aefc).
	require.Equal(t, "10657093454053682940", buyInfo.NewQuoteReserve.String())

	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: buyInfo})
	require.Equal(t, "1522018586976427", sim.trackedQuote.String())

	require.Equal(t, "10657093454053682940", sim.quoteReserve.String())
}
