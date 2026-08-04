package metronomeswap

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

// Fixture: real on-chain swap, tx 0x587e98585911758665038dc4b8417546c04706c9b261f9852ffa113aeeb5bc9a,
// block 25675390, msETH -> msUSD. Verified byte-exact via a Pool.quoteSwapOut replay at the
// same block (see context/metronome/docs/simulations/smoke-0x3364f53c...46.md).
// This is the WHY: these aren't synthetic numbers, they're what actually happened on-chain,
// so a change to the rounding/operation order that still "looks reasonable" will still fail here.
var (
	fixtureAmountIn = uint256.MustFromDecimal("589108703221595501")
	fixturePriceETH = uint256.MustFromDecimal("1861780000000000000000")
	fixturePriceUSD = uint256.MustFromDecimal("1000000000000000000")
	fixtureUsdValue = uint256.MustFromDecimal("1096790801483902071851")
	fixtureGross    = uint256.MustFromDecimal("1096790801483902071851") // priceUSD == 1e18 exactly, so usdValue == gross here
	fixtureFeeBps   = uint256.MustFromDecimal("4500000000000000")       // 45bps, WAD-scaled
	fixtureFee      = uint256.MustFromDecimal("4935558606677559323")
	fixtureNetOut   = uint256.MustFromDecimal("1091855242877224512528")
)

func TestQuoteTokenToUsd(t *testing.T) {
	got := quoteTokenToUsd(fixtureAmountIn, fixturePriceETH, 18)
	assert.Equal(t, fixtureUsdValue.String(), got.String())
}

func TestQuoteUsdToToken(t *testing.T) {
	got := quoteUsdToToken(fixtureUsdValue, fixturePriceUSD, 18)
	assert.Equal(t, fixtureGross.String(), got.String())
}

func TestQuoteUsdToToken_RoundsDown(t *testing.T) {
	// 7 usd-units(1e18) -> token with price 3 (1e18) and 18 decimals: floor(7e18*1e18/3e18) = floor(7e18/3) = 2333333333333333333 (not 2333333333333333334)
	amountInUsd := uint256.MustFromDecimal("7000000000000000000")
	price := uint256.NewInt(3000000000000000000)
	got := quoteUsdToToken(amountInUsd, price, 18)
	assert.Equal(t, "2333333333333333333", got.String())
}

func TestQuote_EndToEnd(t *testing.T) {
	got := quote(fixtureAmountIn, 18, fixturePriceETH, 18, fixturePriceUSD)
	assert.Equal(t, fixtureGross.String(), got.String())
}

func TestQuote_DifferentDecimals(t *testing.T) {
	// 1 token (6 decimals) priced at $2 -> token priced at $1 with 18 decimals: expect 2e18.
	amountIn := uint256.NewInt(1_000000) // 1.0 at 6 decimals
	priceIn := uint256.MustFromDecimal("2000000000000000000")
	priceOut := uint256.MustFromDecimal("1000000000000000000")
	got := quote(amountIn, 6, priceIn, 18, priceOut)
	assert.Equal(t, "2000000000000000000", got.String())
}

func TestSwapFee(t *testing.T) {
	got := swapFee(fixtureGross, fixtureFeeBps)
	assert.Equal(t, fixtureFee.String(), got.String())
}

func TestSwapFee_Zero(t *testing.T) {
	got := swapFee(fixtureGross, uint256.NewInt(0))
	assert.True(t, got.IsZero())
}

// TestSwapFee_RoundsHalfUp_NotFloor is a second real on-chain fixture (Pool.quoteSwapOut on a
// Tenderly vnet forked at chain head, msUSD->msETH, 300bps fee) that a floor-division
// implementation gets wrong by 1 wei. Metronome's `wadMul` (Aave-style WadRayMath) is
// round-half-up: (a*b + WAD/2) / WAD. The first fixture above (45bps) doesn't catch this bug
// because its remainder happens to fall under 0.5 — floor and round-half-up agree there by
// coincidence. This fixture's remainder is >= 0.5, so only the correct rounding passes both.
func TestSwapFee_RoundsHalfUp_NotFloor(t *testing.T) {
	gross := uint256.MustFromDecimal("540664366325628")
	feeBps := uint256.MustFromDecimal("30000000000000000") // 300bps

	got := swapFee(gross, feeBps)

	assert.Equal(t, "16219930989769", got.String(), "on-chain quoteSwapOut gave fee=16219930989769; floor division gives 16219930989768 (wrong)")
}

func TestFixture_NetAmountOutMatchesRealSwap(t *testing.T) {
	gross := quote(fixtureAmountIn, 18, fixturePriceETH, 18, fixturePriceUSD)
	fee := swapFee(gross, fixtureFeeBps)
	net := new(uint256.Int).Sub(gross, fee)

	assert.Equal(t, fixtureFee.String(), fee.String())
	assert.Equal(t, fixtureNetOut.String(), net.String())
}
