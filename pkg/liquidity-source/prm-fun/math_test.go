package prmfun

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

// TestQuoteBuy_LiveVector_NormalPath ports a real on-chain cross-check: block 57379101,
// curve 0xaea1eaf948e97581fbe9d8dea2951c327669af57, router.quoteMemeExactIn(meme, buy=true,
// useEth=true, 0.001 ETH) returned output=747638974590231692804552,
// inputUsed=1000000000000000 - matching an independent hand-computation bit-for-bit.
func TestQuoteBuy_LiveVector_NormalPath(t *testing.T) {
	t.Parallel()
	result := QuoteBuy(
		uint256.MustFromDecimal("1000000000000000"),             // deskIn: 0.001 ETH
		uint256.MustFromDecimal("1062330437710576052235807612"), // virtualMeme
		uint256.MustFromDecimal("1405714531301211559"),          // virtualDesk
		uint256.MustFromDecimal("4336228956090614430859054"),    // memeSold
		uint256.MustFromDecimal("5714531301211559"),             // deskRaised
		uint256.MustFromDecimal("4200000000000000000"),          // graduationDesk (4.2 ETH)
		uSaleSupply,
	)
	assert.Equal(t, uint256.MustFromDecimal("747638974590231692804552"), result.MemeOut)
	assert.Equal(t, uint256.MustFromDecimal("1000000000000000"), result.DeskUsed)
	assert.Equal(t, uint256.MustFromDecimal("10000000000000"), result.Fee) // 1% of deskIn
}

// TestQuoteSell_LiveVector_NormalPath ports a second real on-chain cross-check: same
// curve/block, router.quoteMemeExactIn(meme, buy=false, useEth=true, 1000 MEME) returned
// output=1310003014677, inputUsed=1000e18 - reproduced bit-for-bit.
func TestQuoteSell_LiveVector_NormalPath(t *testing.T) {
	t.Parallel()
	gross, result := QuoteSell(
		uint256.MustFromDecimal("1000000000000000000000"), // memeIn: 1000 MEME
		uint256.MustFromDecimal("1062330437710576052235807612"),
		uint256.MustFromDecimal("1405714531301211559"),
	)
	assert.Equal(t, uint256.MustFromDecimal("1323235368361"), gross)
	assert.Equal(t, uint256.MustFromDecimal("1310003014677"), result.DeskOut)
	assert.Equal(t, uint256.MustFromDecimal("13232353684"), result.Fee)
}

// TestQuoteBuy_GraduationCap exercises MemeCurve.sol's sale-completing-buy branch: net
// (post-fee) exceeds the remaining room to graduationDesk, so the trade caps at exactly
// `room`, deskUsed is grossed back up from the capped net, and memeOut is the entire
// remaining supply regardless of curve shape.
func TestQuoteBuy_GraduationCap(t *testing.T) {
	t.Parallel()
	// room = 1000 - 900 = 100; a 200 deskIn crosses it even after the 1% fee.
	result := QuoteBuy(
		uint256.MustFromDecimal("200"),
		uint256.MustFromDecimal("1000000"), // virtualMeme (synthetic)
		uint256.MustFromDecimal("1000000"), // virtualDesk (synthetic)
		uint256.MustFromDecimal("500"),     // memeSold
		uint256.MustFromDecimal("900"),     // deskRaised
		uint256.MustFromDecimal("1000"),    // graduationDesk
		uint256.MustFromDecimal("1000"),    // saleSupply
	)
	assert.Equal(t, uint256.MustFromDecimal("102"), result.DeskUsed) // ceil(100*10000/9900)
	assert.Equal(t, uint256.MustFromDecimal("2"), result.Fee)        // deskUsed - room
	assert.Equal(t, uint256.MustFromDecimal("500"), result.MemeOut)  // all remaining supply
}

// TestQuoteBuy_RemainingSupplyClamp exercises the defensive memeOut>remaining clamp that
// MemeCurve.sol's quoteBuy also carries. Under an honestly-initialized curve this branch is
// unreachable (the graduation-cap branch triggers first), but QuoteBuy accepts raw reserves
// from the caller and must not assume that invariant - so the reserves here are deliberately
// out of proportion to drive an in-bounds buy whose curve-implied memeOut exceeds supply.
func TestQuoteBuy_RemainingSupplyClamp(t *testing.T) {
	t.Parallel()
	result := QuoteBuy(
		uint256.MustFromDecimal("10"),
		uint256.MustFromDecimal("1000"), // virtualMeme, disproportionate vs. remaining(=5)
		uint256.MustFromDecimal("10"),   // virtualDesk, disproportionately small
		uint256.MustFromDecimal("995"),  // memeSold -> remaining = 1000-995 = 5
		uint256.MustFromDecimal("0"),    // deskRaised -> room = 1_000_000, not crossed
		uint256.MustFromDecimal("1000000"),
		uint256.MustFromDecimal("1000"),
	)
	assert.Equal(t, uint256.MustFromDecimal("5"), result.MemeOut) // clamped to remaining
}
