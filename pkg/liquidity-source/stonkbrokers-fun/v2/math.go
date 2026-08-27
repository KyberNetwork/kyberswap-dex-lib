package stonkbrokersfunv2

import (
	"github.com/holiman/uint256"

	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// maxPriceAgeSecs ports SafeLaunchTwapLib.MAX_PRICE_AGE (48 hours) -- the
// fail-closed staleness window shared by every oracle read on the buy path
// (direct feed AND the ETH/USD leg of TWAP mode).
const maxPriceAgeSecs = 48 * 60 * 60

var (
	tenE8     = uint256.NewInt(1e8)
	tenE18    = uint256.NewInt(1e18)
	oneLsh128 = new(uint256.Int).Lsh(uint256.NewInt(1), 128)
	// denomToken0Branch = (1 << 128) * 1e18, the fused denominator used by
	// SafeLaunchTwapLib.quoteUsd8's quoteIsToken0==true branch.
	denomToken0Branch = new(uint256.Int).Mul(new(uint256.Int).Lsh(uint256.NewInt(1), 128), tenE18)
)

// CurrentTaxBps ports StonkSafeLaunchpadV2.currentTaxBps exactly (see
// docs/source/stonksafelaunchpadv2-.../StonkSafeLaunchpadV2.sol lines
// ~1008-1021). Callers must already have gated on armed/!graduated (matches
// on-chain's early return of 0 in that case -- CalcAmountOut never reaches
// here for a non-tradeable pool since it returns a sentinel error first).
func CurrentTaxBps(now uint64, se StaticExtra) uint16 {
	if now <= se.StartTime {
		return se.BufferTaxBps
	}
	elapsed := now - se.StartTime
	if elapsed < uint64(se.BufferSecs) {
		return se.BufferTaxBps
	}
	minutesElapsed := (elapsed - uint64(se.BufferSecs)) / 60
	decayed := minutesElapsed * uint64(se.DecayPerMinuteBps)
	if decayed >= uint64(se.StartTaxBps) {
		if se.OpenEnded {
			return se.PostTaxBps
		}
		return 0
	}
	return se.StartTaxBps - uint16(decayed)
}

// CalcBuyAmountOut ports StonkSafeLaunchpadV2._buy's core constant-product
// math over virtual reserves (lines ~728-776): tax is taken on the input
// (fee-on-input), then the AMM swap runs on the net amount.
//
//	tax   = quoteIn * taxBps / BPS
//	net   = quoteIn - tax
//	tokensOut = net * vToken / (vQuote + net)     // Math.mulDiv, floor
//
// Returns the post-trade virtual reserves too (newVQuote, newVToken) so the
// caller can feed them into the graduation/mcap check without recomputing.
func CalcBuyAmountOut(quoteIn, vQuote, vToken *uint256.Int, taxBps uint16) (
	tokensOut, tax, newVQuote, newVToken *uint256.Int, err error,
) {
	if quoteIn == nil || quoteIn.IsZero() {
		return nil, nil, nil, nil, ErrZeroTrade
	}

	tax = big256.MulDivDown(new(uint256.Int), quoteIn, uint256.NewInt(uint64(taxBps)), bpsU256)
	net := new(uint256.Int).Sub(quoteIn, tax)
	if net.IsZero() {
		return nil, nil, nil, nil, ErrZeroTrade
	}

	denom := new(uint256.Int).Add(vQuote, net)
	tokensOut = big256.MulDivDown(new(uint256.Int), net, vToken, denom)
	if tokensOut.IsZero() {
		return nil, nil, nil, nil, ErrZeroTrade
	}
	if tokensOut.Cmp(vToken) >= 0 {
		// Unreachable for a well-formed constant-product curve (net/(vQuote+net) < 1
		// strictly), guarded defensively against corrupt/stale tracker state.
		return nil, nil, nil, nil, ErrSlippageExceeded
	}

	newVQuote = new(uint256.Int).Add(vQuote, net)
	newVToken = new(uint256.Int).Sub(vToken, tokensOut)
	return tokensOut, tax, newVQuote, newVToken, nil
}

var bpsU256 = uint256.NewInt(bps)

// ppmU256 is the 1e6 denominator LaunchModes.maxBuyPpm is expressed against
// (StonkSafeLaunchpadV2._buy divides loadedSupply*capPpm by 1_000_000).
var ppmU256 = uint256.NewInt(1_000_000)

// McapUsd8FromReserves ports StonkSafeLaunchpadV2.mcapUsd8 (lines
// ~1023-1028), evaluated against the given (post-trade) virtual reserves.
//
//	mcapQuoteWei = vQuote * loadedSupply / vToken   // Math.mulDiv, floor
//	usd8         = mcapQuoteWei * quoteUsd8 / 10**quoteDecimals
func McapUsd8FromReserves(vQuote, vToken, loadedSupply *uint256.Int, quoteUsd8 uint64, quoteDecimals uint8) *uint256.Int {
	if vToken == nil || vToken.IsZero() {
		return uint256.NewInt(0)
	}
	mcapQuoteWei := big256.MulDivDown(new(uint256.Int), vQuote, loadedSupply, vToken)
	return big256.MulDivDown(new(uint256.Int), mcapQuoteWei, uint256.NewInt(quoteUsd8), big256.TenPow(quoteDecimals))
}

// feedUsd8 ports SafeLaunchTwapLib._feedUsd8 exactly: fail-closed on a
// non-positive answer, on staleness (> 48h old), and on unscalable/zero/
// overflowing results after rescaling to 1e8.
func feedUsd8(r OracleReading, now uint64) (uint64, error) {
	if !r.Ok || r.Answer == nil || r.Answer.IsZero() {
		return 0, ErrBadOracleAnswer
	}
	updatedAt := r.UpdatedAt
	if now < updatedAt {
		updatedAt = now // clock-skew guard; never treat a future timestamp as making the reading stale
	}
	if now-updatedAt > maxPriceAgeSecs {
		return 0, ErrStalePrice
	}

	var usd8 *uint256.Int
	if r.Decimals == 8 {
		usd8 = r.Answer
	} else {
		usd8 = big256.MulDivDown(new(uint256.Int), r.Answer, tenE8, big256.TenPow(r.Decimals))
	}
	if usd8.IsZero() || !usd8.IsUint64() {
		return 0, ErrBadOracleAnswer
	}
	return usd8.Uint64(), nil
}

// DirectFeedUsd8 ports SafeLaunchTwapLib.directFeedUsd8 -- the oracle path
// used by feed-mode lanes (WETH and the tokenized-stock lanes GME/NVDA/
// AAPL/SPCX/USO; StaticExtra.QuoteUsdFeed != "").
func DirectFeedUsd8(r OracleReading, now uint64) (uint64, error) {
	return feedUsd8(r, now)
}

// TwapQuoteUsd8 ports SafeLaunchTwapLib.quoteUsd8 -- the oracle path used by
// TWAP-mode lanes (STONK, USDG; StaticExtra.TwapPool != ""). Live-verified
// against the USDG pad: reproduces the on-chain
// mcapUsd8() USD mark bit-for-bit from a raw observe()+latestRoundData()
// snapshot.
func TwapQuoteUsd8(t TwapReading, window uint32, quoteIsToken0 bool, quoteDecimals uint8, now uint64) (uint64, error) {
	if !t.Ok {
		return 0, ErrBadOracleAnswer
	}
	eth8, err := feedUsd8(t.EthUsd, now)
	if err != nil {
		return 0, err
	}

	delta := t.TickCumulativeNow - t.TickCumulativeOld
	w := int64(window)
	if w == 0 {
		return 0, ErrBadOracleAnswer
	}
	avgTick := delta / w // Go truncates toward zero, same as Solidity's int division
	if delta < 0 && delta%w != 0 {
		avgTick-- // floor correction, matching SafeLaunchTwapLib.quoteUsd8
	}
	if avgTick < -887272 || avgTick > 887272 {
		return 0, ErrBadOracleAnswer
	}

	var sqrtP uint256.Int
	if err := uniswapv3.GetSqrtRatioAtTick(int(avgTick), &sqrtP); err != nil {
		return 0, ErrBadOracleAnswer
	}

	// token1-per-token0 raw price, X128 fixed point: mulDiv(sqrtP, sqrtP, 1<<64).
	ratioX128 := big256.MulDivDown(new(uint256.Int), &sqrtP, &sqrtP, oneLsh64)

	quoteScaled := new(uint256.Int).Mul(uint256.NewInt(eth8), big256.TenPow(quoteDecimals))

	var usd8 *uint256.Int
	if quoteIsToken0 {
		usd8 = big256.MulDivDown(new(uint256.Int), ratioX128, quoteScaled, denomToken0Branch)
	} else {
		tmp := big256.MulDivDown(new(uint256.Int), quoteScaled, oneLsh128, ratioX128)
		usd8 = new(uint256.Int).Div(tmp, tenE18)
	}
	if usd8.IsZero() || !usd8.IsUint64() {
		return 0, ErrBadOracleAnswer
	}
	return usd8.Uint64(), nil
}

var oneLsh64 = new(uint256.Int).Lsh(uint256.NewInt(1), 64)
