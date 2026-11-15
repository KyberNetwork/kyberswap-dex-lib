package btrprop

import (
	"math/big"
	"slices"
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

// TestInSwapGateABI pins the swap gate probe (selector 0x0d0e0323) and its bool decode.
func TestInSwapGateABI(t *testing.T) {
	call, err := poolABI.Pack("inSwapGate", common.HexToAddress(usdc))
	require.NoError(t, err)
	require.Equal(t, []byte{0x0d, 0x0e, 0x03, 0x23}, call[:4])
	out, err := poolABI.Unpack("inSwapGate", common.LeftPadBytes([]byte{1}, 32))
	require.NoError(t, err)
	require.Equal(t, true, out[0])
}

// getAsset's tuple must decode into rpcAssetResult.Asset field for field (v2 layout: covBits,
// kneeBits at the tail), or every refresh loses the cap clamp.
func TestAssetUnpack(t *testing.T) {
	in := rpcAsset{
		Reserves: big.NewInt(7), Liabilities: big.NewInt(9), MinLiquidity: big.NewInt(1),
		LiquidityIndexWad: big.NewInt(1), Flags: SwapEnabledBit | KneeBit, MaxLiabWeightBps: 10000,
		CovBits: 3049886662, KneeBits: 2585,
	}
	raw, err := poolABI.Methods["getAsset"].Outputs.Pack(in)
	require.NoError(t, err)
	var out rpcAssetResult
	require.NoError(t, poolABI.UnpackIntoInterface(&out, "getAsset", raw))
	require.Zero(t, out.Asset.Reserves.Cmp(big.NewInt(7)))
	require.Zero(t, out.Asset.Liabilities.Cmp(big.NewInt(9)))
	require.Equal(t, SwapEnabledBit|KneeBit, out.Asset.Flags)
	require.Equal(t, uint32(3049886662), out.Asset.CovBits)
	require.Equal(t, uint16(2585), out.Asset.KneeBits)
}

// Vectors are CovLib.cap read from the linked library on Monad (0x4c85...C567, `cast call`,
// 2026-11-13); bits are live v2 covBits of USDC, WMON, cbBTC and USDe.
func TestCoverageRoomMatchesCovLib(t *testing.T) {
	for _, c := range []struct {
		bits uint32
		knee uint16
		neg  bool
		kNeg uint64
		r, l string
		want string
	}{
		{3049886662, 0, true, 5000, "1000000000000", "1000000000000", "367601636094"},
		{3049886662, 0, false, 0, "1000000000000", "1000000000000", "451545702540"},
		{1771521287, 0, false, 0, "5000000000000000000000", "3000000000000000000000", "27650620716364722999"},
		{1771521287, 0, true, 1000, "5000000000000000000000", "3000000000000000000000", "3793055390652036944000"},
		{1765229831, 0, false, 0, "100000000", "90000000", "50829518"},
		{2228357001, 0, true, 800, "123456789012", "100000000000", "82730967509"},
		{2228357001, 0, false, 0, "123456789012", "100000000000", "0"},
		{2228357001, 0, false, 0, "0", "100000000000", "119236490208"},
		{2228357001, 0, true, 800, "0", "0", "0"},
		{2228357001, 0, true, 800, "5000", "0", "5000"},
		{2228357001, 2585, true, 800, "123456789012", "100000000000", "81015727161"},
		{2228357001, 2585, true, 3000, "123456789012", "100000000000", "65344686483"},
		{1771521287, 2585, true, 800, "5000000000000000000000", "3000000000000000000000", "3798915879378588290000"},
	} {
		r, l := uint256.MustFromDecimal(c.r), uint256.MustFromDecimal(c.l)
		got := coverageRoom(c.bits, c.knee, c.neg, c.kNeg, r, l)
		require.Equal(t, c.want, got.Dec(), "bits=%d knee=%d neg=%v r=%s l=%s", c.bits, c.knee, c.neg, c.r, c.l)
	}
}

// The grid is sized inside both caps: input room of tokenIn, output room of tokenOut at the rate.
func TestClampToCaps(t *testing.T) {
	book := func(r, l string, bits uint32) *rpcAssetResult {
		rr, _ := new(big.Int).SetString(r, 10)
		ll, _ := new(big.Int).SetString(l, 10)
		return &rpcAssetResult{Asset: rpcAsset{Reserves: rr, Liabilities: ll, CovBits: bits}}
	}
	pp := &pairProbe{}
	maxIn, bal := *uint256.NewInt(1_000_000_000_000), uint256.NewInt(1_000_000_000_000)
	// Input room 451545702540 (USDC-like book) binds first.
	got := pp.clampToCaps(maxIn, bal, book("1000000000000", "1000000000000", 3049886662), nil)
	require.Equal(t, "451545702540", got.Dec())
	// A tight output room scales maxIn down by room/balance.
	got = pp.clampToCaps(maxIn, bal, nil, book("1000000000000", "1000000000000", 3049886662))
	require.Less(t, got.Uint64(), maxIn.Uint64())
	require.Greater(t, got.Uint64(), uint64(0))
	// Unreadable books leave the grid alone.
	got = pp.clampToCaps(maxIn, bal, nil, nil)
	require.Equal(t, maxIn.Dec(), got.Dec())
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
	g := gridAmounts(*uint256.NewInt(1_000_000), nil)
	require.Len(t, g, len(gridPermille))
	require.Equal(t, uint64(1_000_000), g[len(g)-1].Uint64())
	// A tiny maxIn collapses neighbouring permilles: zeros and repeats drop out.
	g = gridAmounts(*uint256.NewInt(15), nil)
	for k := range g {
		require.False(t, g[k].IsZero())
		if k > 0 {
			require.True(t, g[k].Gt(&g[k-1]))
		}
	}
}

// capCurve is a concave body whose output stops growing at 0.98*maxIn (1e6), like
// the live pairs where the core's ~92.7% output cap bites at 0.98-0.995 of maxIn.
func capCurve(x uint64) uint64 {
	const onset = 980_000
	x = min(x, onset)
	return x - x*x/(4_000_000)
}

// sampleLadder mimics a refresh: grid plus knee samples (prev nil = cold) probed
// on capCurve, then cut at the first flat output.
func sampleLadder(prev []Point, withKnee bool) []Point {
	maxIn := *uint256.NewInt(1_000_000)
	var knee []uint256.Int
	if withKnee {
		knee = kneeAmounts(prev, &maxIn)
	}
	amounts := gridAmounts(maxIn, knee)
	ins := make([]uint64, len(amounts))
	outs := make([]int64, len(amounts))
	for k := range amounts {
		ins[k] = amounts[k].Uint64()
		outs[k] = int64(capCurve(ins[k]))
	}
	return probe(ins, outs).ladder()
}

// worstKneeBps is the most negative error in bps (negative = underquote) of
// interpolate against capCurve over the last 11% below the onset.
func worstKneeBps(t *testing.T, pts []Point) int64 {
	worst := int64(0)
	var res uint256.Int
	for x := uint64(880_000); x <= 980_000; x += 500 {
		require.NoError(t, interpolate(pts, uint256.NewInt(x), &res))
		exact := int64(capCurve(x))
		worst = min(worst, (int64(res.Uint64())-exact)*10_000/exact)
	}
	return worst
}

// The cap onset falls inside the last grid segment, where one chord cut up to
// 136-140 bps off the live quotes. The knee samples must fix that and must keep
// doing so when each refresh feeds on its own output (no collapse of the spread).
func TestKneeSamplesBracketCapOnset(t *testing.T) {
	require.Less(t, worstKneeBps(t, sampleLadder(nil, false)), int64(-100), "without knee samples the last chord underquotes")

	var pts []Point
	for cycle := range 6 { // cycle 0 is the cold start
		pts = sampleLadder(pts, true)
		require.Greater(t, worstKneeBps(t, pts), int64(-30), "cycle %d", cycle)
	}
}

func TestMergeProbesKeepsInvariants(t *testing.T) {
	grid := pts(1000, 900, 2000, 1700, 3000, 2000)
	// in:   10 revert  20 zero  500  700 same out  1000 dup  1500 flat vs 2000  2500  3000 dup  4000 past
	pp := probe(
		[]uint64{10, 15, 20, 500, 700, 1000, 1500, 2500, 3000, 4000},
		[]int64{9, -1, 0, 450, 450, 900, 1700, 1900, 2000, 2000},
	)
	got := pp.mergeInto(slices.Clone(grid))
	var ins []uint64
	for _, p := range got {
		ins = append(ins, p.In.Uint64())
	}
	require.Equal(t, []uint64{10, 500, 1000, 2000, 2500, 3000}, ins)
	require.True(t, validPoints(got))
	require.Equal(t, grid[len(grid)-1], got[len(got)-1], "probes never move the cut")
}

// The ladder comes from append, so it has spare capacity and Insert shifts it in
// place. A probe near the cut must still merge after a smaller one was inserted.
func TestMergeProbesWithSpareCapacity(t *testing.T) {
	grid := append(make([]Point, 0, 8), pts(1000, 900, 2000, 1700, 3000, 2000)...)
	got := probe([]uint64{500, 2500}, []int64{450, 1900}).mergeInto(grid)
	var ins []uint64
	for _, p := range got {
		ins = append(ins, p.In.Uint64())
	}
	require.Equal(t, []uint64{500, 1000, 2000, 2500, 3000}, ins)
}

// A dust probe below the first grid sample replaces the long backward extension.
func TestMergeProbesAddsDustSample(t *testing.T) {
	got := probe([]uint64{10}, []int64{9}).mergeInto(pts(1000, 900, 2000, 1700))
	require.Len(t, got, 3)
	require.Equal(t, uint64(10), got[0].In.Uint64())
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
