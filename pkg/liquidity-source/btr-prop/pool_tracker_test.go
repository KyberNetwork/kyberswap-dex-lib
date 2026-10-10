package btrprop

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// TestSwapQuoteUnpack pins the getSwapQuote decode path the tracker relies on:
// the single unnamed tuple output must land in rpcSwapQuoteResult.Quote, else
// every ladder probe fails to unpack and the pool never quotes.
func TestSwapQuoteUnpack(t *testing.T) {
	in := rpcSwapQuote{
		AmountOut: big.NewInt(24004), AmountIn: big.NewInt(1e18), SpreadPbps: 2666,
		ProtoFee: big.NewInt(3), LpFee: big.NewInt(30), MarkPrice: big.NewInt(1), MidPrice: big.NewInt(1),
		CovToll:    big.NewInt(16),
		RouteHops:  []common.Address{common.HexToAddress(wmon), common.HexToAddress(usdc)},
		HopAmounts: []*big.Int{big.NewInt(1e18), big.NewInt(24004)},
		HopPrices:  []*big.Int{big.NewInt(1)},
	}
	raw, err := poolABI.Methods["getSwapQuote"].Outputs.Pack(in)
	require.NoError(t, err)
	var out rpcSwapQuoteResult
	require.NoError(t, poolABI.UnpackIntoInterface(&out, "getSwapQuote", raw))
	require.Zero(t, out.Quote.AmountOut.Cmp(in.AmountOut))
	require.Equal(t, in.SpreadPbps, out.Quote.SpreadPbps)
	require.Len(t, out.Quote.RouteHops, 2)
}

func probe(ins []uint64, outs []int64) *pairProbe {
	pp := &pairProbe{}
	for k := range ins {
		pp.ins = append(pp.ins, *uint256.NewInt(ins[k]))
		pp.ok = append(pp.ok, outs[k] >= 0)
		q := new(rpcSwapQuoteResult)
		if outs[k] >= 0 {
			q.Quote.AmountOut = big.NewInt(outs[k])
		}
		pp.res = append(pp.res, q)
	}
	return pp
}

// swap_qe takes the whole input but pays at most the capped output, so the
// ladder must end before the output flattens (live USDC->WMON: 4000 USDC and
// 20000 USDC both return 143195 WMON).
func TestLadderStopsAtSaturation(t *testing.T) {
	got := probe([]uint64{1, 2, 3, 4, 5}, []int64{10, 20, 30, 30, 30}).ladder()
	require.Len(t, got, 3)
	require.Equal(t, uint64(30), got[2].Out.Uint64())
}

func TestLadderStopsAtRevertAndSkipsLeadingZero(t *testing.T) {
	got := probe([]uint64{1, 2, 3, 4}, []int64{0, 20, -1, 40}).ladder()
	require.Len(t, got, 1, "a revert ends the ladder; a leading zero fill is skipped")
	require.Equal(t, uint64(2), got[0].In.Uint64())
	require.Empty(t, probe([]uint64{1}, []int64{-1}).ladder())
}

// The grid is sized from the best (largest out/in) probe: a probe that hit the
// liquidity cap or rounded down has a worse rate and must not set the scale.
func TestMaxInUsesBestRate(t *testing.T) {
	pp := probe([]uint64{10, 1000, 100000}, []int64{99, 9900, 50000}) // rates 9.9, 9.9, 0.5
	bal := uint256.NewInt(1_000_000)
	require.Equal(t, uint64(1_000_000*1000/9900), maxInOf(pp, bal))
	require.True(t, maxInOf(probe([]uint64{10}, []int64{0}), bal) == 0)
}

func TestGridAmountsStrictlyIncreasing(t *testing.T) {
	g := gridAmounts(*uint256.NewInt(1_000_000))
	require.Len(t, g, len(gridPermille))
	require.Equal(t, uint64(1_000_000), g[len(g)-1].Uint64())
	// A tiny maxIn collapses neighbouring permilles: zeros and repeats drop out.
	g = gridAmounts(*uint256.NewInt(15))
	for k := range g {
		require.False(t, g[k].IsZero())
		if k > 0 {
			require.True(t, g[k].Gt(&g[k-1]))
		}
	}
}

func TestUnitAmount(t *testing.T) {
	require.Equal(t, uint64(1e4), u64(unitAmount(6, -2)))
	require.Equal(t, uint64(1), u64(unitAmount(1, -2)))
	require.Equal(t, uint64(1e8), u64(unitAmount(6, 2)))
}

func maxInOf(pp *pairProbe, bal *uint256.Int) uint64 {
	res := pp.maxIn(bal)
	return res.Uint64()
}

func u64(v uint256.Int) uint64 { return v.Uint64() }
