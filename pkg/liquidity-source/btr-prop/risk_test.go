package btrprop

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// word packs a MarkWordLib leg word. mark is in 1e18 WAD; its lane float keeps 25 bits.
func word(obs, sigma, ttl uint64, mark uint64, halted, internal bool, depegCode uint64) *big.Int {
	m := new(big.Int).Lsh(new(big.Int).SetUint64(mark), 16)
	e := uint(0)
	if bl := m.BitLen(); bl > 25 {
		e = uint(bl - 25)
	}
	f := new(big.Int).Rsh(m, e)
	f.Or(f, new(big.Int).Lsh(big.NewInt(int64(e)), 25))
	w := new(big.Int).Set(f)
	put := func(v uint64, shift uint) { w.Or(w, new(big.Int).Lsh(new(big.Int).SetUint64(v), shift)) }
	put(obs, 32)
	put(sigma, 64)
	put(ttl, 107)
	if halted {
		put(1, 134)
	}
	if internal {
		put(1, 232)
	}
	put(depegCode, 234)
	return w
}

const wad = 1_000_000_000_000_000_000

// The live WMON leg word of 2026-11-15 (getLegMark on Monad) decodes to its feed fields.
func TestDecodeMarkLiveWord(t *testing.T) {
	w, _ := new(big.Int).SetString("627424820714972079307831745652209381336", 10)
	l, ok := decodeMark(w)
	require.True(t, ok)
	require.Equal(t, uint64(1794759928), l.obs)
	require.Equal(t, uint64(11264), l.sigma)
	require.Equal(t, uint64(180), l.ttl)
	require.False(t, l.halted || l.internal)
	require.Equal(t, "15426126662860800", l.mark.Dec())
	_, ok = decodeMark(new(big.Int))
	require.False(t, ok, "an empty word is an unreadable leg")
}

func TestTradableAt(t *testing.T) {
	const obs, ttl = 1000, 60
	for _, c := range []struct {
		name     string
		word     *big.Int
		base     bool
		at       uint64
		tradable bool
	}{
		{"fresh", word(obs, 100, ttl, wad, false, false, 0), false, obs + 5, true},
		{"ttl edge", word(obs, 100, ttl, wad, false, false, 0), false, obs + ttl, true},
		{"expired inside the window", word(obs, 100, ttl, wad, false, false, 0), false, obs + ttl + 1, false},
		{"halted feed", word(obs, 100, ttl, wad, true, false, 0), false, obs, false},
		{"dead mark", word(obs, 100, ttl, 0, false, false, 0), false, obs, false},
		{"no depeg code, any mark", word(obs, 100, ttl, wad/2, false, false, 0), false, obs, true},
		{"inside band (code 1 = 2.5%)", word(obs, 100, ttl, wad*98/100, false, false, 1), false, obs, true},
		{"above band", word(obs, 100, ttl, wad*103/100, false, false, 1), false, obs, false},
		{"below band", word(obs, 100, ttl, wad*97/100, false, false, 1), false, obs, false},
		{"wider band holds it", word(obs, 100, ttl, wad*103/100, false, false, 2), false, obs, true},
		{"INTERNAL without code reads code 1", word(obs, 100, ttl, wad*97/100, false, true, 0), false, obs, false},
		{"base is exempt from the breaker", word(obs, 100, ttl, wad*90/100, false, false, 1), true, obs, true},
		{"base still gated on TTL", word(obs, 100, ttl, wad, false, false, 0), true, obs + ttl + 1, false},
	} {
		l, ok := decodeMark(c.word)
		require.True(t, ok, c.name)
		l.base = c.base
		require.Equal(t, c.tradable, l.tradableAt(c.at), c.name)
	}
}

func TestLiquidRoom(t *testing.T) {
	a := func(minLiq int64) *rpcAsset { return &rpcAsset{MinLiquidity: big.NewInt(minLiq)} }
	require.Equal(t, "90", liquidRoom(big.NewInt(100), a(10)).Dec())
	require.True(t, liquidRoom(big.NewInt(10), a(10)).IsZero(), "liquid at the floor leaves no room")
	require.True(t, liquidRoom(big.NewInt(5), a(10)).IsZero())
	require.Nil(t, liquidRoom(nil, a(10)), "an unreadable liquid read is unknown, not zero")
	require.Nil(t, liquidRoom(big.NewInt(5), nil))
}

func TestCutAbove(t *testing.T) {
	p := pts(10, 9, 20, 18, 30, 27)
	require.Len(t, cutAbove(p, uint256.NewInt(18)), 2)
	require.Empty(t, cutAbove(p, new(uint256.Int)))
	require.Len(t, cutAbove(p, nil), 3)
}

// refine refuses a path through a depegged, expired or unreadable leg, including an interior
// hop, and cuts the ladder at the output leg's settle-time room.
func TestGuardRefine(t *testing.T) {
	a, b, hub := common.HexToAddress("0x01"), common.HexToAddress("0x02"), common.HexToAddress("0x03")
	leg := func(mark uint64, code uint64) *markLeg {
		l, _ := decodeMark(word(1000, 100, 60, mark, false, false, code))
		return l
	}
	ladder := pts(10, 9, 20, 18, 30, 27)
	mk := func(hubLeg *markLeg, room *uint256.Int) *guard {
		return &guard{
			tokens: []common.Address{a, b, hub},
			legs:   []*markLeg{leg(wad, 1), leg(wad, 1), hubLeg},
			rooms:  []*uint256.Int{nil, room, nil},
			now:    1000, window: 10,
		}
	}
	hops := []common.Address{a, hub, b}
	require.Len(t, mk(leg(wad, 1), nil).refine(0, 1, hops, ladder), 3)
	require.Nil(t, mk(leg(wad*90/100, 1), nil).refine(0, 1, hops, ladder), "depegged interior hop")
	require.Len(t, mk(leg(wad*90/100, 1), nil).refine(0, 1, nil, ladder), 3, "without a hop list only the endpoints are checked")
	require.Nil(t, mk(nil, nil).refine(0, 1, hops, ladder), "unreadable interior word")
	g := mk(leg(wad, 1), nil)
	g.window = 100
	require.Nil(t, g.refine(0, 1, hops, ladder), "TTL expires inside the window")
	require.Len(t, mk(leg(wad, 1), uint256.NewInt(20)).refine(0, 1, hops, ladder), 2, "liquid floor cuts the top")
}

// driftPool is a synthetic pool whose quote is the base curve less the path's staleness premium
// at block time t, the way the live quote drifts. Samples are taken at t=now.
func driftPool(g *guard, legs []int, x, t uint64) uint64 {
	var s uint64
	for _, i := range legs {
		s += g.legs[i].staleTermAt(t)
	}
	return x * 99 / 100 * (pbps - s) / pbps
}

// The ladder never promises more than the pool pays at any second of the window (no push in
// between), for volatile spokes, a stable-ish leg, a fresh mark and one already well past grace.
func TestRefineNeverOverquotesAcrossWindow(t *testing.T) {
	a, b, hub := common.HexToAddress("0x01"), common.HexToAddress("0x02"), common.HexToAddress("0x03")
	for _, c := range []struct {
		name         string
		obsA, obsB   uint64
		sigA, sigB   uint64
		ttl          uint64
		wantHaircut  bool
		maxHaircutBp uint64
	}{
		{"fresh volatile", 1000, 1000, 11264, 11264, 180, true, 25},
		{"aged volatile", 970, 960, 11264, 6000, 180, true, 40},
		{"stable-ish", 995, 995, 200, 150, 180, true, 5},
		{"short ttl", 1000, 1000, 11264, 11264, 8, true, 25},
	} {
		wa, _ := decodeMark(word(c.obsA, c.sigA, c.ttl, wad, false, false, 0))
		wb, _ := decodeMark(word(c.obsB, c.sigB, c.ttl, wad, false, false, 0))
		wh, _ := decodeMark(word(1000, 100, c.ttl, wad, false, false, 0))
		wh.base = true
		g := &guard{
			tokens: []common.Address{a, b, hub}, legs: []*markLeg{wa, wb, wh},
			rooms: make([]*uint256.Int, 3), now: 1000, window: uint64(MaxAge.Seconds()),
		}
		path := []int{0, 1}
		var raw []Point
		for _, x := range []uint64{1e6, 1e9, 1e12} {
			raw = append(raw, Point{In: *uint256.NewInt(x), Out: *uint256.NewInt(driftPool(g, path, x, g.now))})
		}
		got := g.refine(0, 1, []common.Address{a, hub, b}, raw)
		require.Len(t, got, len(raw), c.name)
		for k, p := range got {
			x := p.In.Uint64()
			for ts := g.now; ts <= g.now+g.window; ts++ {
				require.LessOrEqual(t, p.Out.Uint64(), driftPool(g, path, x, ts), "%s x=%d t=%d", c.name, x, ts)
			}
			// The cost is bounded: the haircut is at most maxHaircutBp of the sampled output.
			require.GreaterOrEqual(t, p.Out.Uint64()*10_000, raw[k].Out.Uint64()*(10_000-c.maxHaircutBp), c.name)
		}
	}
}

// A refreshed mark (age inside grace) has no premium to grow into the window beyond sqrt(excess)
// of the seconds past grace; a base-only path has none at all.
func TestRefineBaseOnlyPathUntouched(t *testing.T) {
	a, hub := common.HexToAddress("0x01"), common.HexToAddress("0x03")
	wa, _ := decodeMark(word(1000, 11264, 180, wad, false, false, 0))
	wh, _ := decodeMark(word(1000, 11264, 180, wad, false, false, 0))
	wa.base, wh.base = true, true
	g := &guard{tokens: []common.Address{a, hub}, legs: []*markLeg{wa, wh}, rooms: make([]*uint256.Int, 2), now: 1000, window: 6}
	in := pts(1000, 990, 2000, 1980)
	got := g.refine(0, 1, nil, in)
	require.Len(t, got, 2)
	// Only the 1 PBPS rounding allowance applies.
	require.Equal(t, uint64(990*(pbps-1)/pbps), got[0].Out.Uint64())
}

func TestStaleTermMatchesPricingLib(t *testing.T) {
	// STALE_Z*sigma*floor(sqrt(excess))/BPS: sigma 11264 (1.13%), 100 s past grace -> 53166.
	l, _ := decodeMark(word(900, 11264, 180, wad, false, false, 0))
	require.Equal(t, uint64(97), l.staleExcessAt(1000), "age 100 less the 3 s grace")
	require.Equal(t, uint64(472*11264*9/10_000), l.staleTermAt(1000))
	require.Zero(t, l.staleTermAt(903), "inside the grace")
	l.base = true
	require.Zero(t, l.staleTermAt(1000))
}
