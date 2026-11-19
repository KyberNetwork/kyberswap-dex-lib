package aegisprop

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

// testHookV1 packs a ladder of one bid and one ask lane: price codes 0x1000|... give a delta of
// anchor*F/2^27; amplitude code m<<ampShift raw token0. Anchor 2000e6 raw (2000 USDC/ETH).
func testHookV1(inv0, inv1 uint64) *HookV1 {
	var hd, lv uint256.Int
	anchor := uint256.NewInt(2000e6)
	hd.Set(anchor)
	hd.Or(&hd, new(uint256.Int).Lsh(uint256.NewInt(7), 128))     // sequence 7
	hd.Or(&hd, new(uint256.Int).Lsh(uint256.NewInt(1<<31), 176)) // expiration far away
	hd.Or(&hd, new(uint256.Int).Lsh(uint256.NewInt(1), 242))     // bidLen 1
	hd.Or(&hd, new(uint256.Int).Lsh(uint256.NewInt(1), 245))     // askLen 1
	hd.Or(&hd, new(uint256.Int).Lsh(uint256.NewInt(40), 248))    // ampShift 40
	// price code: e=0, m=0x100 -> F=256 -> delta = ceil(anchor*256/2^27). amp code m=1000.
	lane := uint64(0x0100)<<16 | 1000
	lv.SetUint64(lane)
	lv.Or(&lv, new(uint256.Int).Lsh(uint256.NewInt(lane), 128))
	h := &HookV1{Header: hd, Levels: lv, MmId: 1, TakerFeePips: 1000, Token0Decimals: 18, Token1Decimals: 6,
		Inv0: *uint256.NewInt(inv0), Inv1: *uint256.NewInt(inv1)}
	h.Inv0.Mul(&h.Inv0, uint256.NewInt(1e12))
	h.decode()
	return h
}

func TestLaneDecode(t *testing.T) {
	h := testHookV1(1e6, 1e12)
	// delta = ceil(2000e6*256/2^27)
	delta := (uint64(2000e6)*256 + 1<<27 - 1) >> 27
	assert.Equal(t, 2000e6-delta, h.bids[0].price.Uint64(), "bids round away from the anchor")
	assert.Equal(t, 2000e6+delta, h.asks[0].price.Uint64())
	assert.Equal(t, uint64(1000)<<40, h.bids[0].amp.Uint64())
}

func TestBeforeSwapV1(t *testing.T) {
	h := testHookV1(1e6, 1e12)
	bs := func(h *HookV1, calcOut, zfo bool, amt uint64) (*uniswapv4.BeforeSwapResult, error) {
		return h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: calcOut, ZeroForOne: zfo, AmountSpecified: new(big.Int).SetUint64(amt)})
	}
	// Bid depth is 1000<<40 raw token0 (~1.1e15 wei).
	res, err := bs(h, true, true, 4e14)
	require.NoError(t, err)
	out := new(big.Int).Neg(res.DeltaUnspecified)
	assert.Positive(t, out.Sign())

	_, err = bs(h, true, true, 2e15)
	assert.ErrorIs(t, err, ErrPartialFill, "input past the book is not fully taken, core pool is empty")

	// UpdateBalance advances the cursor: two 4e14 swaps leave 2.99e14 of depth, so a third is partial.
	next := h.CloneState().(*HookV1)
	next.UpdateBalance(res.SwapInfo)
	res2, err := bs(next, true, true, 4e14)
	require.NoError(t, err)
	next.UpdateBalance(res2.SwapInfo)
	_, err = bs(next, true, true, 4e14)
	assert.ErrorIs(t, err, ErrPartialFill)
	assert.Equal(t, uint64(8e14), next.BidConsumed.Uint64())
	assert.Equal(t, h.Inv0.Uint64()+8e14, next.Inv0.Uint64(), "maker gains the input token0")
	again, _ := bs(h, true, true, 4e14)
	assert.Equal(t, res.DeltaUnspecified, again.DeltaUnspecified, "CloneState left the source untouched")
	assert.Equal(t, uint64(0), h.BidConsumed.Uint64())

	// CalcIn returns an input whose exact-in output covers the request.
	want := uint64(1_000_000)
	in, err := bs(h, false, true, want)
	require.NoError(t, err)
	f, err := h.fill(true, true, uint256.MustFromBig(in.DeltaUnspecified))
	require.NoError(t, err)
	assert.GreaterOrEqual(t, f.takerOut.Uint64(), want)
}

func TestInventoryCapsExactIn(t *testing.T) {
	// Maker holds 1 USDC: selling 1 ETH must be cut to what 1 USDC covers, so the exact-in is partial.
	h := testHookV1(1e6, 1e6)
	_, err := h.fill(true, true, uint256.NewInt(1e18))
	require.NoError(t, err)
	f, _ := h.fill(true, true, uint256.NewInt(1e18))
	assert.LessOrEqual(t, f.t1.Uint64(), uint64(1e6))
	assert.True(t, f.takerIn.Lt(uint256.NewInt(1e18)))
	h.Inv1.Clear()
	_, err = h.fill(true, true, uint256.NewInt(1e15))
	assert.ErrorIs(t, err, ErrYield, "no inventory of the output currency")
}

func TestUnsupportedAndExpired(t *testing.T) {
	h := testHookV1(1e6, 1e12)
	h.Unsupported = true
	_, err := h.fill(true, true, uint256.NewInt(1e15))
	assert.ErrorIs(t, err, ErrUnsupported)
	h.Unsupported = false
	h.quoteAt = int64(h.expiration)
	_, err = h.fill(true, true, uint256.NewInt(1e15))
	assert.ErrorIs(t, err, ErrYield, "ladder fills only while now < expiration")
}
