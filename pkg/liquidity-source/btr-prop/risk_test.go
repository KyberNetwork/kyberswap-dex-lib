package btrprop

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// word packs a leg word. mark is in 1e18 WAD; its lane float keeps 25 bits.
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

// refinePts is refine without the haircut inputs.
func (g *guard) refinePts(from, to int, hops []common.Address, spread uint64, pts []Point) []Point {
	res, _ := g.refine(from, to, hops, spread, pts)
	return res
}

// bookOf is a getAsset result with the fields the route rule reads.
func bookOf(anchor common.Address, lane uint8) *rpcAssetResult {
	return &rpcAssetResult{Asset: rpcAsset{Anchor: anchor, OracleBits: lane}}
}

// refine refuses a path through a depegged, expired, uncertain or unreadable leg, including an
// interior hop, and cuts the ladder at the output leg's settle-time room.
func TestGuardRefine(t *testing.T) {
	a, b, hub := common.HexToAddress("0x01"), common.HexToAddress("0x02"), common.HexToAddress("0x03")
	leg := func(mark uint64, code uint64) *markLeg {
		l, _ := decodeMark(word(1000, 100, 60, mark, false, false, code))
		return l
	}
	ladder := pts(1_000_000, 900_000, 2_000_000, 1_800_000, 3_000_000, 2_700_000)
	mk := func(hubLeg *markLeg, room *uint256.Int) *guard {
		return &guard{
			tokens: []common.Address{a, b, hub},
			legs:   []*markLeg{leg(wad, 1), leg(wad, 1), hubLeg},
			assets: []*rpcAssetResult{bookOf(hub, 1), bookOf(hub, 2), bookOf(common.Address{}, 0)},
			base:   hub,
			rooms:  []*uint256.Int{nil, room, nil},
			now:    1000, window: 10,
		}
	}
	hops := []common.Address{a, hub, b}
	require.Len(t, mk(leg(wad, 1), nil).refinePts(0, 1, hops, 3000, ladder), 3)
	require.Nil(t, mk(leg(wad*90/100, 1), nil).refinePts(0, 1, hops, 3000, ladder), "depegged interior hop")
	require.Len(t, mk(leg(wad*90/100, 1), nil).refinePts(0, 1, nil, 3000, ladder), 3, "without a hop list only the endpoints are checked")
	require.Nil(t, mk(nil, nil).refinePts(0, 1, hops, 3000, ladder), "unreadable interior word")
	g := mk(leg(wad, 1), nil)
	g.window = 100
	require.Nil(t, g.refinePts(0, 1, hops, 3000, ladder), "TTL expires inside the window")
	require.Len(t, mk(leg(wad, 1), uint256.NewInt(2_000_000)).refinePts(0, 1, hops, 3000, ladder), 2, "liquid floor cuts the top")
	g = mk(leg(wad, 1), nil)
	g.assets[0] = nil
	require.Nil(t, g.refinePts(0, 1, hops, 3000, ladder), "unreadable book cannot clear the route rule")
	require.Nil(t, mk(leg(wad, 1), nil).refinePts(0, 1, []common.Address{a, common.HexToAddress("0x99"), b}, 3000, ladder), "route through an unknown node")
}

// An INTERNAL (or any) leg whose mark confidence interval exceeds the limit is refused.
func TestTradableAtConfidence(t *testing.T) {
	w := word(1000, 100, 60, wad, false, true, 0)
	w.Or(w, new(big.Int).Lsh(big.NewInt(1000), 91))
	l, _ := decodeMark(w)
	require.True(t, l.tradableAt(1000))
	w.Or(w, new(big.Int).Lsh(big.NewInt(1), 91)) // conf 1001
	l, _ = decodeMark(w)
	require.Equal(t, uint64(1001), l.conf)
	require.False(t, l.tradableAt(1000))
}

// The route rule: no ascending and no descending chain may price two legs off one oracle lane.
func TestDistinctFeeds(t *testing.T) {
	a, c, hub, b := common.HexToAddress("0x01"), common.HexToAddress("0x02"), common.HexToAddress("0x03"), common.HexToAddress("0x04")
	mk := func(laneA, laneC, laneB uint8) *guard {
		return &guard{
			tokens: []common.Address{a, c, hub, b}, base: hub,
			assets: []*rpcAssetResult{bookOf(c, laneA), bookOf(hub, laneC), bookOf(common.Address{}, 0), bookOf(hub, laneB)},
		}
	}
	route := []int{0, 1, 2, 3} // a -> c -> hub -> b
	require.True(t, mk(1, 2, 3).distinctFeeds(route))
	require.False(t, mk(5, 5, 3).distinctFeeds(route), "ascent chain repeats a lane")
	require.True(t, mk(1, 2, 1).distinctFeeds(route), "a lane shared across the turn is fine")
	require.True(t, mk(7, 7, 7).distinctFeeds([]int{0, 2}), "single-child chains have nothing to collide with")
	g := mk(1, 2, 3)
	g.assets[3] = nil
	require.False(t, g.distinctFeeds(route))
	// Descent chain: hub -> c -> a read downward.
	d := &guard{
		tokens: []common.Address{a, c, hub}, base: hub,
		assets: []*rpcAssetResult{bookOf(c, 4), bookOf(hub, 4), bookOf(common.Address{}, 0)},
	}
	require.False(t, d.distinctFeeds([]int{2, 1, 0}), "descent chain repeats a lane")
}

// driftPool is a synthetic pool: the pool pays half the quoted spread on the output,
// out = gross - ceil(gross*spread/2e6), and the spread is a fixed part plus the path's
// staleness premium at block time t. Samples are taken at t=now.
const driftBase = 3000

func driftSpread(g *guard, legs []int, t uint64) uint64 {
	s := uint64(driftBase)
	for _, i := range legs {
		s += g.legs[i].staleTermAt(t)
	}
	return s
}

func driftPool(g *guard, legs []int, x, t uint64) uint64 {
	gross := x * 99 / 100
	return gross - (gross*driftSpread(g, legs, t)+2*pbps-1)/(2*pbps)
}

// window is the haircut window of a quote age seconds after the sample.
func window(age uint64) uint64 { return age + uint64(InclusionAllowance/time.Second) }

// scaled is out*num/den rounded down, as the simulator pays it.
func scaled(out, num, den uint64) uint64 { return out * num / den }

// stales replays every leg of g.
func stales(g *guard) []StaleLeg {
	res := make([]StaleLeg, len(g.legs))
	for i, l := range g.legs {
		if l != nil {
			res[i] = l.stale()
		}
	}
	return res
}

// A quote at any age up to MaxAge never promises more than the pool pays at any second until
// it lands (no push in between), for volatile spokes, a stable-ish leg, a fresh mark and one
// already well past the keeper's allowance; and the haircut is sized by the quote's own age.
func TestHaircutNeverOverquotesUntilLanding(t *testing.T) {
	a, b, hub := common.HexToAddress("0x01"), common.HexToAddress("0x02"), common.HexToAddress("0x03")
	maxAge := uint64(MaxAge / time.Second)
	for _, c := range []struct {
		name         string
		obsA, obsB   uint64
		sigA, sigB   uint64
		ttl          uint64
		maxHaircutBp uint64
	}{
		{"fresh volatile", 1000, 1000, 11264, 11264, 180, 22},
		{"aged volatile", 970, 960, 11264, 6000, 180, 25},
		{"stable-ish", 995, 995, 200, 150, 180, 5},
		{"short ttl", 1000, 1000, 11264, 11264, 20, 22},
	} {
		wa, _ := decodeMark(word(c.obsA, c.sigA, c.ttl, wad, false, false, 0))
		wb, _ := decodeMark(word(c.obsB, c.sigB, c.ttl, wad, false, false, 0))
		wh, _ := decodeMark(word(1000, 100, c.ttl, wad, false, false, 0))
		wh.base = true
		g := &guard{
			tokens: []common.Address{a, b, hub}, legs: []*markLeg{wa, wb, wh},
			assets: []*rpcAssetResult{bookOf(hub, 1), bookOf(hub, 2), bookOf(common.Address{}, 0)}, base: hub,
			rooms: make([]*uint256.Int, 3), now: 1000, window: window(maxAge),
		}
		path := []int{0, 1}
		var raw []Point
		for _, x := range []uint64{1e6, 1e9, 1e12} {
			raw = append(raw, Point{In: *uint256.NewInt(x), Out: *uint256.NewInt(driftPool(g, path, x, g.now))})
		}
		got, cut := g.refine(0, 1, []common.Address{a, hub, b}, driftSpread(g, path, g.now), raw)
		require.Equal(t, raw, got, c.name)
		require.Equal(t, []uint16{0, 1, 2}, cut.Legs, c.name)
		prev := uint64(0)
		for age := uint64(0); age <= maxAge; age++ {
			num, den, ok := cut.factor(stales(g), g.now, window(age))
			require.True(t, ok, c.name)
			for _, p := range got {
				x, out := p.In.Uint64(), scaled(p.Out.Uint64(), num, den)
				for ts := g.now; ts <= g.now+window(age); ts++ {
					require.LessOrEqual(t, out, driftPool(g, path, x, ts), "%s x=%d age=%d t=%d", c.name, x, age, ts)
				}
				// The cost is bounded: the haircut is at most maxHaircutBp of the sampled output.
				require.GreaterOrEqual(t, out*10_000, p.Out.Uint64()*(10_000-c.maxHaircutBp), c.name)
			}
			require.GreaterOrEqual(t, den-num, prev, "%s: an older quote never pays a smaller haircut", c.name)
			prev = den - num
		}
	}
}

// The haircut follows the quote's age (WETH-like leg, sigma 4608, base hub). A mark already some
// seconds old at the sample pays less than a fresh one for the same window: the premium grows
// with the square root of the mark's age.
func TestHaircutByAge(t *testing.T) {
	a, hub := common.HexToAddress("0x01"), common.HexToAddress("0x03")
	cutAt := func(obs, w uint64) uint64 {
		wa, _ := decodeMark(word(obs, 4608, 345, wad, false, false, 0))
		wh, _ := decodeMark(word(1000, 208, 1020, wad, false, false, 0))
		wh.base = true
		g := &guard{tokens: []common.Address{a, hub}, legs: []*markLeg{wa, wh}, rooms: make([]*uint256.Int, 2), now: 1000, window: window(10)}
		_, cut := g.refine(0, 1, nil, 3000, pts(1_000_000, 990_000))
		num, den, ok := cut.factor(stales(g), g.now, w)
		require.True(t, ok)
		return den - num
	}
	term := func(isqrt uint64) uint64 { return 472*4608*isqrt/10_000 + 1 }
	require.Equal(t, term(2), cutAt(1000, window(0)), "fresh mark, quote at the sample: isqrt(9-3)")
	require.Equal(t, term(3), cutAt(1000, window(3)), "isqrt(12-3)")
	require.Equal(t, term(4), cutAt(1000, window(10)), "at MaxAge: isqrt(19-3)")
	require.Equal(t, term(3), cutAt(1000, 17), "the fixed 17 s window this replaces: isqrt(14)")
	require.Equal(t, term(3)-term(1)+1, cutAt(995, window(3)), "mark 5 s old at the sample: isqrt(14)-isqrt(2)")
}

// A path of base legs only has no premium to grow; just the rounding allowance applies.
func TestRefineBaseOnlyPathUntouched(t *testing.T) {
	a, hub := common.HexToAddress("0x01"), common.HexToAddress("0x03")
	wa, _ := decodeMark(word(1000, 11264, 180, wad, false, false, 0))
	wh, _ := decodeMark(word(1000, 11264, 180, wad, false, false, 0))
	wa.base, wh.base = true, true
	g := &guard{tokens: []common.Address{a, hub}, legs: []*markLeg{wa, wh}, rooms: make([]*uint256.Int, 2), now: 1000, window: 6}
	in := pts(1_000_000, 990_000, 2_000_000, 1_980_000)
	got, cut := g.refine(0, 1, nil, 3000, in)
	require.Equal(t, in, got)
	den := uint64(2*pbps - 3000)
	for _, w := range []uint64{0, 9, 19, 1000} {
		num, d, ok := cut.factor(stales(g), g.now, w)
		require.True(t, ok)
		require.Equal(t, [2]uint64{den - 1, den}, [2]uint64{num, d})
	}
}

// A larger sampled spread gives a larger haircut for the same drift (the ratio falls as the
// spread grows), so the largest spread over a pair's samples is the safe one to use.
func TestRefineSpreadMonotone(t *testing.T) {
	a, hub := common.HexToAddress("0x01"), common.HexToAddress("0x03")
	wa, _ := decodeMark(word(1000, 11264, 180, wad, false, false, 0))
	wh, _ := decodeMark(word(1000, 100, 180, wad, false, false, 0))
	wh.base = true
	g := &guard{tokens: []common.Address{a, hub}, legs: []*markLeg{wa, wh}, rooms: make([]*uint256.Int, 2), now: 1000, window: 8}
	share := func(spread uint64) float64 {
		_, cut := g.refine(0, 1, nil, spread, pts(1_000_000, 990_000))
		num, den, ok := cut.factor(stales(g), g.now, 8)
		require.True(t, ok)
		return float64(num) / float64(den)
	}
	require.Less(t, share(30000), share(1000))
}

func TestStaleTermFormula(t *testing.T) {
	// premium = 472*sigma*floor(sqrt(excess))/10000 PBPS.
	l, _ := decodeMark(word(900, 11264, 180, wad, false, false, 0))
	require.Equal(t, uint64(97), l.stale().excessAt(1000), "age 100 less the 3 s allowance")
	require.Equal(t, uint64(472*11264*9/10_000), l.staleTermAt(1000))
	require.Zero(t, l.staleTermAt(903), "inside the allowance")
	l.base = true
	require.Zero(t, l.staleTermAt(1000))
}

// getLegMark's three outputs, baseToken and getLiquidReserves must decode into the tracker's
// targets, or every leg reads as unreadable and every pair is refused.
func TestLegReadsUnpack(t *testing.T) {
	w, _ := new(big.Int).SetString("627424820714972079307831745652209381336", 10)
	raw, err := poolABI.Methods["getLegMark"].Outputs.Pack(w, true, big.NewInt(15426126662860800))
	require.NoError(t, err)
	var lm rpcLegMark
	require.NoError(t, poolABI.UnpackIntoInterface(&lm, "getLegMark", raw))
	require.Zero(t, lm.Word.Cmp(w))
	require.True(t, lm.Priced)

	raw, err = poolABI.Methods["baseToken"].Outputs.Pack(common.HexToAddress(usdc))
	require.NoError(t, err)
	var base common.Address
	require.NoError(t, poolABI.UnpackIntoInterface(&base, "baseToken", raw))
	require.Equal(t, common.HexToAddress(usdc), base)

	raw, err = poolABI.Methods["getLiquidReserves"].Outputs.Pack(big.NewInt(7))
	require.NoError(t, err)
	var liq *big.Int
	require.NoError(t, poolABI.UnpackIntoInterface(&liq, "getLiquidReserves", raw))
	require.Equal(t, int64(7), liq.Int64())
}

// The haircut can never raise a quote: over extreme spreads, sigmas, ages and TTLs every refined
// output is at most its sampled output, every factor is a share in (0, 1), and a haircut that
// would reach the whole spread refuses the ladder instead of wrapping.
func TestRefineNeverRaisesOutput(t *testing.T) {
	a, hub := common.HexToAddress("0x01"), common.HexToAddress("0x03")
	raw := pts(1_000_000, 990_000, 2_000_000, 1_980_000, 3_000_000, 2_970_000)
	for _, sigma := range []uint64{0, 1, 11264, 1<<27 - 1} {
		for _, obs := range []uint64{0, 500, 999, 1000} {
			for _, ttl := range []uint64{0, 6, 180, 1<<16 - 1} {
				for _, spread := range []uint64{0, 1, 3000, 1_999_999, 2_000_000, 1 << 40} {
					wa, _ := decodeMark(word(obs, sigma, ttl, wad, false, false, 0))
					wh, _ := decodeMark(word(1000, 100, 1<<16-1, wad, false, false, 0))
					if wa == nil || wh == nil {
						continue
					}
					wh.base = true
					g := &guard{tokens: []common.Address{a, hub}, legs: []*markLeg{wa, wh}, rooms: make([]*uint256.Int, 2), now: 1000, window: 8}
					got, cut := g.refine(0, 1, nil, spread, raw)
					if len(got) > 0 {
						for w := uint64(0); w <= 8; w++ {
							num, den, ok := cut.factor(stales(g), g.now, w)
							require.True(t, ok, "a refined pair quotes at every age inside the window")
							require.True(t, num > 0 && num < den, "sigma=%d obs=%d ttl=%d spread=%d w=%d", sigma, obs, ttl, spread, w)
						}
					}
					for k, p := range got {
						var r *Point
						for i := range raw {
							if raw[i].In.Eq(&p.In) {
								r = &raw[i]
							}
						}
						require.NotNil(t, r)
						require.False(t, p.Out.Gt(&r.Out), "sigma=%d obs=%d ttl=%d spread=%d k=%d", sigma, obs, ttl, spread, k)
					}
				}
			}
		}
	}
}

// A halted output leg keeps its ladder (the tracker zeroes its reserve instead); a halted input
// or interior leg still empties it.
func TestRefineKeepsLadderOfClosedOutputLeg(t *testing.T) {
	a, b, hub := common.HexToAddress("0x01"), common.HexToAddress("0x02"), common.HexToAddress("0x03")
	live, _ := decodeMark(word(1000, 100, 60, wad, false, false, 0))
	halted, _ := decodeMark(word(1000, 100, 60, wad, true, false, 0))
	hubLeg, _ := decodeMark(word(1000, 100, 60, wad, false, false, 0))
	hubLeg.base = true
	mk := func(la, lb *markLeg) *guard {
		return &guard{
			tokens: []common.Address{a, b, hub}, legs: []*markLeg{la, lb, hubLeg},
			assets: []*rpcAssetResult{bookOf(hub, 1), bookOf(hub, 2), bookOf(common.Address{}, 0)}, base: hub,
			rooms: make([]*uint256.Int, 3), now: 1000, window: 10,
		}
	}
	ladder := pts(1_000_000, 900_000, 2_000_000, 1_800_000)
	hops := []common.Address{a, hub, b}
	g := mk(live, halted)
	require.True(t, g.legClosed(1))
	require.Len(t, g.refinePts(0, 1, hops, 3000, ladder), 2, "closed output leg keeps the ladder")
	require.Nil(t, mk(halted, live).refinePts(0, 1, hops, 3000, ladder), "closed input leg refuses the ladder")
	require.Nil(t, mk(live, nil).refinePts(1, 0, nil, 3000, ladder), "unreadable input leg refuses the ladder")
}

// A pair-level refusal (the route rule: A and its parent C price off one oracle lane) drops only A -> B.
// B stays a live output for C -> B, so its reserve is not zeroed (legClosed is per token).
func TestRefinePairRefusalKeepsOtherPairsToSameOutput(t *testing.T) {
	a, c, b, hub := common.HexToAddress("0x01"), common.HexToAddress("0x02"), common.HexToAddress("0x04"), common.HexToAddress("0x03")
	leg := func() *markLeg { l, _ := decodeMark(word(1000, 100, 60, wad, false, false, 0)); return l }
	hl := leg()
	hl.base = true
	g := &guard{
		tokens: []common.Address{a, c, b, hub}, base: hub,
		legs: []*markLeg{leg(), leg(), leg(), hl},
		// a sits under c and shares its lane, so the ascent a -> c -> hub collides; c -> hub is clean.
		assets: []*rpcAssetResult{bookOf(c, 5), bookOf(hub, 5), bookOf(hub, 6), bookOf(common.Address{}, 0)},
		rooms:  make([]*uint256.Int, 4), now: 1000, window: 10,
	}
	ladder := pts(1_000_000, 900_000, 2_000_000, 1_800_000)
	require.Nil(t, g.refinePts(0, 2, []common.Address{a, c, hub, b}, 3000, ladder), "A->B refused: same lane")
	require.Len(t, g.refinePts(1, 2, []common.Address{c, hub, b}, 3000, ladder), 2, "C->B still quotable")
	require.False(t, g.legClosed(2), "B itself is open, so its reserve stays")
}

// The float root in termAt must equal the integer floor root: the pool's premium steps at every
// perfect square, so one off at a square edge moves the haircut by a whole premium step.
func TestTermAtRootIsIntegerFloor(t *testing.T) {
	l := StaleLeg{Obs: 0, Sigma: 10_000, Grace: 0}
	for n := uint64(1); n < 200_000; n++ {
		for _, x := range []uint64{n*n - 1, n * n, n*n + 1} {
			want := staleZ * l.Sigma * new(uint256.Int).Sqrt(uint256.NewInt(x)).Uint64() / bps
			require.Equal(t, want, l.termAt(x), "excess %d", x)
		}
	}
}
