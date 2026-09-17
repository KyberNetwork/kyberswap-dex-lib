package deepstateob

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	orderbook "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/order-book"
)

// TestBuildLevels_PriceAndSizeDirection locks in the token0/token1 <->
// price/size direction derived in context/deepstate/output/tracker.md from
// DeepstateV1.sol's _executeFill NatSpec: quantity is always denominated in
// token0 (base); price is token1-per-token0 at a tick; LevelsFrom[0] (swap
// FROM token0) uses the raw price/size directly, LevelsFrom[1] (swap FROM
// token1) must invert price and convert size into token1 units.
//
// token0 = USDG (6 decimals), token1 = NVDA (18 decimals), matching the one
// live market on Robinhood Chain (see output/explorer.md).
func TestBuildLevels_PriceAndSizeDirection(t *testing.T) {
	const dec0, dec1 = 6, 18

	leaves := []decodedNode{
		{tick: 0, quantity: uint256.NewInt(1_000_000), nonce: 1}, // 1.0 token0 at tick 0
	}

	bidLevels := buildLevels(leaves, dec0, dec1, false)
	require := assert.New(t)
	require.Len(bidLevels, 2) // dummy sentinel + 1 real level
	require.Equal(orderbook.Level{0, 0}, bidLevels[0])
	require.InDelta(1.0, bidLevels[1].Size(), 1e-15)
	require.InDelta(1e-12, bidLevels[1].Price(), 1e-24) // token1-real per token0-real at tick 0, dec0-dec1=-12

	askLevels := buildLevels(leaves, dec0, dec1, true)
	require.Len(askLevels, 2)
	require.InDelta(1e-12, askLevels[1].Size(), 1e-24) // size converted to token1 units: 1.0 * 1e-12
	require.InDelta(1e12, askLevels[1].Price(), 1e-3)  // inverted price: token0-real per token1-real
}

func TestBuildLevels_AggregatesSameTick(t *testing.T) {
	leaves := []decodedNode{
		{tick: 10, quantity: uint256.NewInt(1_000_000), nonce: 2},
		{tick: 10, quantity: uint256.NewInt(2_000_000), nonce: 1}, // earlier nonce, same tick -- must sum, not overwrite
	}
	levels := buildLevels(leaves, 6, 6, false)
	assert.Len(t, levels, 2)
	assert.InDelta(t, 3.0, levels[1].Size(), 1e-9)
}

func TestBuildLevels_SortsBestPriceFirst(t *testing.T) {
	leaves := []decodedNode{
		{tick: 5, quantity: uint256.NewInt(1_000_000), nonce: 1},
		{tick: -5, quantity: uint256.NewInt(1_000_000), nonce: 2},
		{tick: 0, quantity: uint256.NewInt(1_000_000), nonce: 3},
	}

	bidLevels := buildLevels(leaves, 6, 6, false) // best bid = highest tick first
	assert.Greater(t, bidLevels[1].Price(), bidLevels[2].Price())
	assert.Greater(t, bidLevels[2].Price(), bidLevels[3].Price())

	// LevelsFrom[1].Price() is token0-per-token1 (inverted tick price), so the
	// lowest-tick leaf (best ask) yields the *highest* Level.Price() -- the
	// most token0 a taker gets per token1 in -- and must still come first.
	askLevels := buildLevels(leaves, 6, 6, true)
	assert.Greater(t, askLevels[1].Price(), askLevels[2].Price())
	assert.Greater(t, askLevels[2].Price(), askLevels[3].Price())
}
