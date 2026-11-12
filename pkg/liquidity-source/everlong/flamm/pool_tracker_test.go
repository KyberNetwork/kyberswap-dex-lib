package everlongflamm

import (
	"bytes"
	"context"
	"math"
	"math/big"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	factoryA = "0x00000000000000000000000000000000000000f1"
	factoryB = "0x00000000000000000000000000000000000000f2"
	poolX    = "0x00000000000000000000000000000000000000a1"
	hookA    = "0x00000000000000000000000000000000000000b1"
	hookB    = "0x00000000000000000000000000000000000000b2" // the same family as hookA, other immutables
	pairHk   = "0x00000000000000000000000000000000000000c1"
	stranger = "0x00000000000000000000000000000000000000d1"
	routerA  = "0x00000000000000000000000000000000000000e9"
	feedA    = "0x00000000000000000000000000000000000000fa"
	testBlk  = 1_000
)

func hookCode(imm byte) []byte { return []byte{0x60, 0x80, imm, imm, 0x60, 0x40, 0x52} }

var pairHookCode = []byte{0x61, 0x01, 0x02, 0x03}

func testFamilies() []HookFamily {
	return []HookFamily{
		{Name: "everlong-hook", CodeHash: testHookFamily, Immutables: [][2]int{{2, 2}}, Shape: shapeEverlong},
		{Name: "pair-hook", CodeHash: testPairFamily, Shape: shapeEverlong},
	}
}

// testWorld is one three-token pool of factory A: cbBTC (the pool asset), USDC and USDT, its swap hook hookA, whose
// leverage row is the protocol's like every listed hook's here, and one live Morpho Blue venue lending USDC against
// cbBTC at routerA. cbBTC sells for USDC at 100,000 up to 50 cbBTC; USDC
// buys cbBTC up to a 1,000,000 USDC ceiling, the rest of a larger input returned; USDT sells for USDC up to 500,000.
// No other direction trades. Every fill is the curve's (curveLabel), and USDT/USDC is no hook pair, so it is not
// routed under the everlong shape.
func testWorld() *world {
	w := newWorld()
	w.hook(common.HexToAddress(hookA), hookCode(0xaa))
	w.hook(common.HexToAddress(hookB), hookCode(0xbb))
	w.hook(common.HexToAddress(pairHk), pairHookCode)
	w.code[common.HexToAddress(stranger)] = []byte{0x60, 0x00}
	w.routers[common.HexToAddress(routerA)] = true
	btcUsd, usdBtc, usdtUsd := oracle(1_000, 1, 5, 1_000_000_000), oracle(1, 1_000, 5, 100_000_000_000),
		oracle(1, 1, 1, 100_000_000_000)
	w.add(common.HexToAddress(poolX), &fakePool{
		factory: common.HexToAddress(factoryA), listed: true, version: 1,
		tokens:   []tokenConfig{tokenCfg(cbBTC, 8), tokenCfg(usdc, 6), tokenCfg(usdt, 6)},
		tradable: bits(16*0+1, 16*1+0, 16*2+1),
		hook:     common.HexToAddress(hookA), pairMask: new(big.Int), laneConfig: new(big.Int),
		feed: common.HexToAddress(feedA), router: common.HexToAddress(routerA), venues: []venueView{venue(0, 0, 1, 0)},
		quote: func(from common.Address, i, j int, a *big.Int) (*big.Int, *big.Int, bool) {
			if from != defaultTaker {
				return nil, nil, false // the hook prices its caller: the test's taker only
			}
			switch {
			case i == 0 && j == 1 && a.Cmp(big.NewInt(5_000_000_000)) <= 0:
				return btcUsd(from, i, j, a)
			case i == 1 && j == 0:
				return usdBtc(from, i, j, minBig(a, big.NewInt(1_000_000_000_000)))
			case i == 2 && j == 1 && a.Cmp(big.NewInt(500_000_000_000)) <= 0:
				return usdtUsd(from, i, j, a)
			}
			return nil, nil, false
		},
	})
	return w
}

func testConfig() *Config {
	return &Config{DexID: DexType, ChainID: valueobject.ChainIDBase, Factories: []string{factoryA},
		PriceFeeds: []string{feedA}, HookFamilies: testFamilies()}
}

func listed(t *testing.T, w *world, cfg *Config) (*node, []entity.Pool) {
	t.Helper()
	n := newNode(t, testBlk, w.respond, w.codeAt)
	n.storage = w.storageAt
	pools, _, err := NewPoolsListUpdater(cfg, n.client()).GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	return n, pools
}

func track(t *testing.T, n *node, cfg *Config, p entity.Pool) (entity.Pool, error) {
	t.Helper()
	return NewPoolTracker(cfg, n.client()).GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
}

func extraOf(t *testing.T, p entity.Pool) Extra {
	t.Helper()
	var e Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &e))
	return e
}

func directionsOf(e Extra) [][2]uint8 {
	return lo.Map(e.Ladders, func(l Ladder, _ int) [2]uint8 { return [2]uint8{l.In, l.Out} })
}

func TestTrackerLaddersEveryRoutedDirectionAtOneBlockFromTheTaker(t *testing.T) {
	w, cfg := testWorld(), testConfig()
	n, pools := listed(t, w, cfg)
	require.Len(t, pools, 1)
	before := len(n.calls())
	p, err := track(t, n, cfg, pools[0])
	require.NoError(t, err)

	e := extraOf(t, p)
	require.ElementsMatch(t, [][2]uint8{{0, 1}, {1, 0}}, directionsOf(e))
	require.Equal(t, uint64(testBlk), p.BlockNumber)
	require.Equal(t, uint64(1), e.HaircutBps)
	fp := w.pools[common.HexToAddress(poolX)]
	for _, l := range e.Ladders {
		require.Equal(t, testCells(testHookFamily, l.In == 0, 0), l.Cells, "the cells of the family and the kind")
		require.GreaterOrEqual(t, len(l.Points), 10, "direction %d -> %d", l.In, l.Out)
		for _, pt := range l.Points { // every point is the pool's own quote
			used, out, ok := fp.quote(defaultTaker, int(l.In), int(l.Out), pt[0].ToBig())
			require.True(t, ok)
			require.Equal(t, used, pt[0].ToBig())
			require.Equal(t, out, pt[1].ToBig())
		}
	}
	capped := lo.Filter(e.Ladders, func(l Ladder, _ int) bool { return l.Capped })
	require.Len(t, capped, 1, "the USDC -> cbBTC ceiling")
	require.Equal(t, "1000000000000", capped[0].Points[len(capped[0].Points)-1][0].Dec())

	// the room of each token: the most a direction pays of it
	top := func(i, j uint8) string {
		l, _ := lo.Find(e.Ladders, func(l Ladder) bool { return l.In == i && l.Out == j })
		return l.Points[len(l.Points)-1][1].Dec()
	}
	require.Equal(t, entity.PoolReserves{top(1, 0), top(0, 1), "0"}, p.Reserves)

	// one multicall at the head, everything after it pinned to its block: one sampler call, to the taker with the
	// sampler's code there, its own gas and no sender, at the pinned block, whose every probe comes from the taker
	block := hexutil.EncodeUint64(testBlk)
	var head, pinned int
	for _, c := range n.calls()[before:] {
		if c.tag == "latest" {
			require.Equal(t, multicall3, c.from, "the head's reads are one multicall's")
			require.Zero(t, pinned, "the head is read first")
			head++
			continue
		}
		require.Equal(t, block, c.tag, "a read off the refresh's block")
		pinned++
	}
	require.Positive(t, head)
	asked := n.asked(t, before)
	require.Len(t, asked, 1)
	a := asked[0]
	require.Equal(t, defaultTaker, a.to)
	require.Equal(t, common.Address{}, a.from, "no sender")
	require.Equal(t, defaultSampleGas, a.gas)
	require.Equal(t, block, a.tag, "the block parameter is the pin")
	require.Equal(t, common.HexToAddress(poolX), a.pool)
	require.Equal(t, [5]uint64{defaultProbeGas, samplerReserveGas, samplerFlipShift, samplerEdgeProbes,
		samplerGapPpm}, limitsOf(a.limits))
	require.Len(t, a.asks, 2)
	for _, ask := range a.asks {
		require.Equal(t, common.HexToAddress(hookA), ask.Hook)
		require.Equal(t, int64(maxProbesPerDirection), ask.MaxProbes.Int64())
		require.Len(t, ask.Seeds, firstRefreshSeeds)
		require.Equal(t, leverageRow()[0], ask.TierA, "the row's spreads, so the request is read on a tier label")
		require.Equal(t, leverageRow()[1], ask.TierB)
	}
	var probes int
	for _, c := range n.seen()[before:] {
		if m, _, ok := decode(poolABI, c.data); ok && m.Name == "previewSwap" {
			require.True(t, c.inner, "every probe inside the sampler's call")
			require.Equal(t, defaultTaker, c.from)
			require.Equal(t, block, c.tag)
			probes++
		}
	}
	require.Positive(t, probes)
	require.LessOrEqual(t, probes, 2*(maxProbesPerDirection+1), "each walk's budget and its probe above")
}

// limitsOf is the sampler's settings as numbers: the cap, the reserve, the flip shift, the edge probes, the gap.
func limitsOf(l sampleLimits) [5]uint64 {
	return [5]uint64{l.ProbeGas.Uint64(), l.ReserveGas.Uint64(), l.FlipShift.Uint64(), l.EdgeProbes.Uint64(),
		l.GapPpm.Uint64()}
}

func TestTrackerAsksForARefreshWellInsideTheStaleAge(t *testing.T) {
	require.Equal(t, int64(1_010), (&PoolTracker{}).NextUpdateAt(entity.Pool{Timestamp: 1_000}))
	require.Less(t, time.Duration(refreshSeconds)*time.Second, MaxAge/2, "two refreshes may be lost")

	w, cfg := testWorld(), testConfig()
	n, pools := listed(t, w, cfg)
	p, err := track(t, n, cfg, pools[0])
	require.NoError(t, err)
	require.Equal(t, p.Timestamp+10, NewPoolTracker(cfg, n.client()).NextUpdateAt(p))

	// a pool left dark asks too: nothing else tells a gate's read has changed
	w.pools[common.HexToAddress(poolX)].paused = true
	p, err = track(t, n, cfg, p)
	require.NoError(t, err)
	require.Empty(t, extraOf(t, p).Ladders)
	require.Equal(t, p.Timestamp+10, NewPoolTracker(cfg, n.client()).NextUpdateAt(p))
}

// pool-service saves a refresh only where the state changed, so a quiet pool's stamp is what moves its Timestamp. A pool
// with nothing routed has nothing to keep alive: it repeats, and costs no save.
func TestTrackerStampsARefreshThatRoutes(t *testing.T) {
	w, cfg := testWorld(), testConfig()
	n, pools := listed(t, w, cfg)
	p, err := track(t, n, cfg, pools[0])
	require.NoError(t, err)
	require.NotEmpty(t, extraOf(t, p).Ladders)
	require.Equal(t, p.Timestamp, extraOf(t, p).LiveAt)

	for label, shut := range map[string]func(*fakePool){
		"every pair closed": func(wp *fakePool) { wp.tradable = new(big.Int) },
		"paused":            func(wp *fakePool) { wp.paused = true },
	} {
		w, cfg := testWorld(), testConfig()
		n, pools := listed(t, w, cfg)
		shut(w.pools[common.HexToAddress(poolX)])
		first, err := track(t, n, cfg, pools[0])
		require.NoError(t, err, label)
		require.Empty(t, extraOf(t, first).Ladders, label)
		require.Zero(t, extraOf(t, first).LiveAt, label)
		again, err := track(t, n, cfg, first)
		require.NoError(t, err, label)
		require.Equal(t, first.Extra, again.Extra, label)
		require.Equal(t, first.Reserves, again.Reserves, label)
	}
}

// A refresh after a pool's first seeds each direction from its last ladder: one size an eighth of the last first
// point (none under the input's floor), then the rungs from that point to eight times the last top.
func TestTrackerNextRefreshSizesFromTheLastLadder(t *testing.T) {
	w, cfg := testWorld(), testConfig()
	n, pools := listed(t, w, cfg)
	first, err := track(t, n, cfg, pools[0])
	require.NoError(t, err)
	before := len(n.calls())
	second, err := track(t, n, cfg, first)
	require.NoError(t, err)
	last := map[[2]uint8]Ladder{}
	for _, l := range extraOf(t, first).Ladders {
		last[[2]uint8{l.In, l.Out}] = l
	}
	fp := w.pools[common.HexToAddress(poolX)]
	asked := n.asked(t, before)
	require.Len(t, asked, 1)
	require.Len(t, asked[0].asks, 2)
	var below int
	for _, a := range asked[0].asks {
		d := [2]uint8{uint8(fp.index(a.TokenIn)), uint8(fp.index(a.TokenOut))}
		l, ok := last[d]
		require.True(t, ok)
		x0, top := l.Points[0][0].ToBig(), l.Points[len(l.Points)-1][0].ToBig()
		seeds := a.Seeds
		if under := maxBig(new(big.Int).Rsh(x0, 3), pow10(int(fp.tokens[d[0]].Decimals)-3)); under.Cmp(x0) < 0 {
			require.Equal(t, under, seeds[0], "direction %v: one size an eighth of the last first point", d)
			seeds, below = seeds[1:], below+1
		}
		require.Equal(t, x0, seeds[0], "direction %v: from the last first point", d)
		require.Len(t, seeds, defaultRungs)
		require.Equal(t, maxBig(new(big.Int).Lsh(top, 3), new(big.Int).Lsh(x0, 1)), seeds[len(seeds)-1],
			"direction %v: to eight times the last top", d)
	}
	require.Positive(t, below, "a first point over the floor")
	require.ElementsMatch(t, directionsOf(extraOf(t, first)), directionsOf(extraOf(t, second)))
}

// The simulator of a tracked pool against the pool itself, on fresh state for every size: 2,000 sizes log-spaced
// over each ladder, a unit either side of each point, 300 across each flat span and 300 around each change of route.
// Each quote is at most what the pool pays for the input it uses, or the pool reverts there. On a curve alone every
// span is a chord, and its quote stays within 3 bp of the pool's.
func TestSimulatorOfTheTrackedPoolNeverQuotesMoreThanThePool(t *testing.T) {
	type switchAt = map[uint8][]int64 // by input token: the sizes where the route changes
	curveOnly := func() (*world, *fakePool) { w := testWorld(); return w, w.pools[common.HexToAddress(poolX)] }
	for name, tc := range map[string]struct {
		world    func() (*world, *fakePool)
		decimals []uint8
		switches switchAt
	}{
		"a curve":                  {world: curveOnly, decimals: []uint8{8, 6, 6}},
		"leverage under the curve": {world: tierWorld, decimals: []uint8{8, 6}, switches: switchAt{0: {tierSaleTop}, 1: {tierBuyTop}}},
		"leverage inside the curve": {world: func() (*world, *fakePool) { return tierWorldFrom(1_000_000, 1_000_000_000) },
			decimals: []uint8{8, 6}, switches: switchAt{0: {1_000_000, tierSaleTop}, 1: {1_000_000_000, tierBuyTop}}},
		"leverage opening just over a deep curve": {world: lateLeverWorld, decimals: []uint8{8, 6},
			switches: switchAt{0: {lateLeverFrom, lateLeverTop}, 1: {tierBuyTop}}},
	} {
		t.Run(name, func(t *testing.T) {
			w, fp := tc.world()
			cfg := testConfig()
			n, pools := listed(t, w, cfg)
			p, err := track(t, n, cfg, pools[0])
			require.NoError(t, err)
			for k := range p.Tokens {
				p.Tokens[k].Decimals = tc.decimals[k]
			}
			s, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
			require.NoError(t, err)
			ladders := extraOf(t, p).Ladders
			require.Len(t, ladders, 2)
			for _, l := range ladders {
				first, top := l.Points[0][0].ToBig(), l.Points[len(l.Points)-1][0].ToBig()
				lf, _ := new(big.Float).SetInt(first).Float64()
				hf, _ := new(big.Float).SetInt(top).Float64()
				var logSpaced, sizes []*big.Int
				for k := range 2_000 {
					a, _ := big.NewFloat(lf * math.Pow(hf/lf, float64(k)/1_999)).Int(nil)
					logSpaced = append(logSpaced, a)
				}
				for k, pt := range l.Points {
					x := pt[0].ToBig()
					sizes = append(sizes, new(big.Int).Sub(x, big.NewInt(1)), x, new(big.Int).Add(x, big.NewInt(1)))
					if k > 0 && l.Flat>>k&1 == 1 {
						x0 := l.Points[k-1][0].ToBig()
						for i := range int64(300) {
							a := new(big.Int).Sub(x, x0)
							sizes = append(sizes, a.Mul(a, big.NewInt(i)).Quo(a, big.NewInt(299)).Add(a, x0))
						}
					}
				}
				for _, at := range tc.switches[l.In] {
					for i := range int64(300) {
						sizes = append(sizes, big.NewInt(at-150+i))
					}
				}
				var worst float64
				var quoted, reverted int
				for k, a := range slices.Concat(logSpaced, sizes) {
					if a.Cmp(first) < 0 {
						continue
					}
					res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
						TokenAmountIn: pool.TokenAmount{Token: p.Tokens[l.In].Address, Amount: a},
						TokenOut:      p.Tokens[l.Out].Address})
					if a.Cmp(top) > 0 && !l.Capped {
						require.ErrorIs(t, err, ladder.ErrAmountInTooLarge)
						continue
					}
					require.NoError(t, err, "direction %d -> %d at %s", l.In, l.Out, a)
					used := new(big.Int).Sub(a, res.RemainingTokenAmountIn.Amount)
					quoted++
					_, chain, ok := fp.quote(defaultTaker, int(l.In), int(l.Out), used)
					if !ok {
						reverted++ // the pool reverts there: a failed route, never an over-quote
						continue
					}
					require.LessOrEqual(t, res.TokenAmountOut.Amount.Cmp(chain), 0,
						"direction %d -> %d: an over-quote at %s", l.In, l.Out, a)
					if k < len(logSpaced) && l.Flat == 0 {
						// the shortfall past the two units the floors of a chord's ends may cost
						short := new(big.Int).Sub(chain, res.TokenAmountOut.Amount)
						short.Sub(short, big.NewInt(2))
						bps, _ := new(big.Float).Quo(new(big.Float).SetInt(short), new(big.Float).SetInt(chain)).Float64()
						worst = max(worst, bps*10_000)
					}
				}
				t.Logf("direction %d -> %d: %d points, flat spans %b, %d sizes quoted, %d where the pool reverts", l.In,
					l.Out, len(l.Points), l.Flat, quoted, reverted)
				require.Greater(t, quoted, 2_000, "the sweep quotes")
				if tc.switches == nil {
					t.Logf("direction %d -> %d: at most %.2f bp under the pool", l.In, l.Out, worst)
					require.Zero(t, l.Flat, "a curve's ladder: all chords")
					require.Less(t, worst, 3.0, "direction %d -> %d: at most 3 bp under the pool", l.In, l.Out)
				}
				for _, dust := range []*big.Int{big.NewInt(1), new(big.Int).Rsh(first, 1), new(big.Int).Sub(first, big.NewInt(1))} {
					_, err := s.CalcAmountOut(pool.CalcAmountOutParams{
						TokenAmountIn: pool.TokenAmount{Token: p.Tokens[l.In].Address, Amount: dust},
						TokenOut:      p.Tokens[l.Out].Address})
					require.ErrorIs(t, err, ladder.ErrNoQuote, "direction %d -> %d at %s", l.In, l.Out, dust)
				}
			}
		})
	}
}

func TestTrackerGatesFailClosed(t *testing.T) {
	for name, tc := range map[string]struct {
		edit func(w *world, fp *fakePool, cfg *Config)
		want [][2]uint8 // the directions laddered, each with the cells of the family and the venue's kind
		err  error
	}{
		"all routed, the hook pair only": {want: [][2]uint8{{0, 1}, {1, 0}}},
		"a Factory not configured": {edit: func(_ *world, _ *fakePool, cfg *Config) {
			cfg.Factories = []string{factoryB}
		}},
		"no pool of its Factory":     {edit: func(_ *world, fp *fakePool, _ *Config) { fp.listed = false }},
		"paused":                     {edit: func(_ *world, fp *fakePool, _ *Config) { fp.paused = true }},
		"an implementation unlisted": {edit: func(_ *world, fp *fakePool, _ *Config) { fp.version = 2 }},
		"an implementation listed": {edit: func(_ *world, fp *fakePool, cfg *Config) {
			fp.version, cfg.CodeVersions = 2, []uint32{1, 2}
		}, want: [][2]uint8{{0, 1}, {1, 0}}},
		"a feed the list does not name": {edit: func(_ *world, fp *fakePool, _ *Config) {
			fp.feed = common.HexToAddress(stranger)
		}},
		"no feed listed": {edit: func(_ *world, _ *fakePool, cfg *Config) { cfg.PriceFeeds = nil }},
		"any hook behind a feed the list does not name": {edit: func(_ *world, fp *fakePool, cfg *Config) {
			fp.feed, cfg.AnyHook = common.HexToAddress(stranger), true
		}},
		"a feed the list names, in another case": {edit: func(_ *world, fp *fakePool, cfg *Config) {
			fp.feed, cfg.PriceFeeds = common.HexToAddress(stranger), []string{"0x" + strings.ToUpper(stranger[2:])}
		}, want: [][2]uint8{{0, 1}, {1, 0}}},
		"a closed pair": {edit: func(_ *world, fp *fakePool, _ *Config) {
			fp.tradable = bits(16*0+1, 16*2+1)
		}, want: [][2]uint8{{0, 1}}},
		"another deployment of the family": {edit: func(_ *world, fp *fakePool, _ *Config) {
			fp.hook = common.HexToAddress(hookB)
		}, want: [][2]uint8{{0, 1}, {1, 0}}},
		"a family that states no shape": {edit: func(_ *world, _ *fakePool, cfg *Config) {
			cfg.HookFamilies[0].Shape = ""
		}},
		"a shape this source does not settle": {edit: func(_ *world, _ *fakePool, cfg *Config) {
			cfg.HookFamilies[0].Shape = "concave"
		}},
		"a hook of no family": {edit: func(_ *world, fp *fakePool, _ *Config) {
			fp.hook = common.HexToAddress(stranger)
		}},
		"the protocol-listed families only": {edit: func(_ *world, _ *fakePool, cfg *Config) {
			cfg.HookFamilies = nil
		}},
		"any hook: one of no family has no cells": {edit: func(_ *world, fp *fakePool, cfg *Config) {
			fp.hook, cfg.HookFamilies, cfg.AnyHook = common.HexToAddress(stranger), nil, true
		}},
		"any hook, one with a leverage row": {edit: func(w *world, fp *fakePool, cfg *Config) {
			w.hook(common.HexToAddress(stranger), []byte{0x60, 0x00})
			fp.hook, cfg.HookFamilies, cfg.AnyHook = common.HexToAddress(stranger), nil, true
		}},
		"no swap hook": {edit: func(_ *world, fp *fakePool, _ *Config) { fp.hook = common.Address{} }},
		"a front's pool": {edit: func(_ *world, fp *fakePool, _ *Config) {
			fp.laneConfig = new(big.Int).Lsh(big.NewInt(1), laneVersionShift)
		}},
		"a front's pool, whatever the knobs say": {edit: func(w *world, fp *fakePool, cfg *Config) {
			fp.laneConfig = new(big.Int).Lsh(big.NewInt(1), laneVersionShift)
			w.fronts[fp.hook], cfg.SkipFrontPriced, cfg.AnyHook = true, lo.ToPtr(false), true
		}},
		"a lane on behind a front the Factory delisted": {edit: func(_ *world, fp *fakePool, cfg *Config) {
			fp.laneConfig = new(big.Int).Lsh(big.NewInt(1), laneVersionShift)
			cfg.SkipFrontPriced = lo.ToPtr(false)
		}},
		"a listed front, the lane not open yet": {edit: func(w *world, fp *fakePool, _ *Config) {
			w.fronts[fp.hook] = true
		}},
		"a listed front, the lane not open yet, whatever the knobs say": {edit: func(w *world, fp *fakePool, cfg *Config) {
			w.fronts[fp.hook], cfg.SkipFrontPriced, cfg.AnyHook = true, lo.ToPtr(false), true
		}},
		"a token the list does not hold": {edit: func(_ *world, fp *fakePool, _ *Config) {
			fp.tokens[2].Token = common.HexToAddress(stranger)
		}, err: ErrInvalidToken},
	} {
		t.Run(name, func(t *testing.T) {
			w, cfg := testWorld(), testConfig()
			n, pools := listed(t, w, cfg)
			if tc.edit != nil {
				tc.edit(w, w.pools[common.HexToAddress(poolX)], cfg)
			}
			p, err := track(t, n, cfg, pools[0])
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			e := extraOf(t, p)
			require.ElementsMatch(t, tc.want, directionsOf(e))
			for _, l := range e.Ladders {
				require.Equal(t, testCells(testHookFamily, l.In == 0, 0), l.Cells)
			}
			if len(tc.want) == 0 {
				require.Equal(t, entity.PoolReserves{"0", "0", "0"}, p.Reserves)
			}
		})
	}
}

func TestConfigReadsTheVenueKinds(t *testing.T) {
	var cfg Config
	require.NoError(t, json.Unmarshal([]byte(`{"dexID":"everlong-flamm","chainID":56,"factories":["0xf1"],`+
		`"venueKinds":[0,2]}`), &cfg))
	require.Equal(t, []uint8{0, 2}, cfg.VenueKinds)
	require.Equal(t, []uint8{2}, cfg.venueKinds(), "narrowed to the chain's own")
	require.Equal(t, chainKinds[valueobject.ChainIDBSC], (&Config{ChainID: valueobject.ChainIDBSC}).venueKinds())
	require.Empty(t, (&Config{}).venueKinds(), "no chain: no kind")
}

// AnyHook adds the builds the gas table measured to a list that names others, and nothing else.
func TestAnyHookAddsTheMeasuredBuildsOnly(t *testing.T) {
	cfg := testConfig()
	require.Equal(t, testFamilies(), cfg.families())
	cfg.AnyHook = true
	require.Equal(t, slices.Concat(testFamilies(), gasFamilies), cfg.families())
	cfg.HookFamilies = nil
	require.Equal(t, defaultHookFamilies, cfg.families(), "the defaults hold them already")
}

func TestTrackerPairHookPricesTheNamedPairs(t *testing.T) {
	w, cfg := testWorld(), testConfig()
	n, pools := listed(t, w, cfg)
	fp := w.pools[common.HexToAddress(poolX)]
	fp.pairHook, fp.pairMask = common.HexToAddress(pairHk), bits(pairBit(0, 1))
	p, err := track(t, n, cfg, pools[0])
	require.NoError(t, err)
	cells := map[[2]uint8][]Cell{}
	for _, l := range extraOf(t, p).Ladders {
		cells[[2]uint8{l.In, l.Out}] = l.Cells
	}
	require.Equal(t, map[[2]uint8][]Cell{{0, 1}: testCells(testPairFamily, true, 0),
		{1, 0}: testCells(testPairFamily, false, 0)}, cells, "the swap hook's USDT/USDC is no hook pair")

	// the pair hook's pairs stay routed on a front's pool: the front prices only the swap hook's
	fp.laneConfig = new(big.Int).Lsh(big.NewInt(1), laneVersionShift)
	p, err = track(t, n, cfg, pools[0])
	require.NoError(t, err)
	require.ElementsMatch(t, [][2]uint8{{0, 1}, {1, 0}}, directionsOf(extraOf(t, p)))

	// a pair hook of no family leaves its pairs unrouted
	w.code[common.HexToAddress(pairHk)] = []byte{0xfe}
	p, err = NewPoolTracker(cfg, n.client()).GetNewPoolState(context.Background(), pools[0],
		pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	require.Empty(t, extraOf(t, p).Ladders)
}

func TestTrackerRespectsTheNotionalCap(t *testing.T) {
	w, cfg := testWorld(), testConfig()
	fp := w.pools[common.HexToAddress(poolX)]
	fp.tokens[0].MaxSwapNotional = big.NewInt(300_000_000) // 3 cbBTC
	n, pools := listed(t, w, cfg)
	p, err := track(t, n, cfg, pools[0])
	require.NoError(t, err)
	asked := n.asked(t, 0)
	require.Len(t, asked, 1)
	sale, ok := lo.Find(asked[0].asks, func(a sampleAsk) bool { return a.TokenIn == common.HexToAddress(cbBTC) })
	require.True(t, ok)
	for _, a := range sale.Seeds {
		require.LessOrEqual(t, a.Cmp(fp.tokens[0].MaxSwapNotional), 0, "no size past the cap")
	}
	require.Equal(t, fp.tokens[0].MaxSwapNotional, sale.Seeds[len(sale.Seeds)-1], "the cap is the last size")
	for _, c := range n.seen() {
		if m, args, ok := decode(poolABI, c.data); ok && m.Name == "previewSwap" &&
			args[0].(common.Address) == common.HexToAddress(cbBTC) {
			require.LessOrEqual(t, args[2].(*big.Int).Cmp(fp.tokens[0].MaxSwapNotional), 0)
		}
	}
	l, _ := lo.Find(extraOf(t, p).Ladders, func(l Ladder) bool { return l.In == 0 })
	require.Equal(t, "300000000", l.Points[len(l.Points)-1][0].Dec(), "the ladder reaches the cap")
}

func TestTrackerFailsOnAReadThatFails(t *testing.T) {
	w, cfg := testWorld(), testConfig()
	n, pools := listed(t, w, cfg)
	n.setDown(true)
	_, err := track(t, n, cfg, pools[0])
	require.Error(t, err, "a refresh whose reads fail keeps the last state, aged out by MaxAge")
	n.setDown(false)

	w.pools[common.HexToAddress(poolX)].laneConfig = nil // lane() reverts: a gate read lost
	_, err = track(t, n, cfg, pools[0])
	require.ErrorIs(t, err, ErrRead)

	w.pools[common.HexToAddress(poolX)].laneConfig = new(big.Int)
	w.pools[common.HexToAddress(poolX)].noFeed = true // the feed unread is no feed listed
	_, err = track(t, n, cfg, pools[0])
	require.ErrorIs(t, err, ErrRead)
}

// On the plain path a revert is the pool's word on a size, and a probe the node ran out of gas, or that reverted
// ReadStarved() or GasShort(), starved: past a price either only ends the ladder. A node that fails a probe any
// other way says nothing of the pool, so the refresh fails, where taking it for a refusal would cut the ladder or
// leave the pool dark with a fresh timestamp.
func TestTrackerTakesOnlyARevertForARefusal(t *testing.T) {
	const limit = 1_000_000_000 // 10 cbBTC: the probes past it fail
	for name, tc := range map[string]struct {
		fail map[string]any
		err  bool
	}{
		"a revert":                   {fail: map[string]any{"code": 3, "message": "execution reverted"}},
		"a revert with its reason":   {fail: map[string]any{"code": 3, "message": "execution reverted: custom error 0x1234"}},
		"a client's own revert code": {fail: map[string]any{"code": -32000, "message": "execution reverted"}},
		"a block the node lacks":     {fail: map[string]any{"code": -32000, "message": "header not found"}, err: true},
		"a node's timeout":           {fail: map[string]any{"code": -32603, "message": "request timed out"}, err: true},
		"out of gas":                 {fail: map[string]any{"code": -32000, "message": "out of gas"}},
		"ReadStarved()": {fail: map[string]any{"code": 3, "message": "execution reverted",
			"data": hexutil.Encode(starvedReverts[0])}},
	} {
		t.Run(name, func(t *testing.T) {
			w, cfg := testWorld(), testConfig()
			cfg.DisableSampler = true // its faults are per preview
			n, pools := listed(t, w, cfg)
			n.callError = func(to common.Address, data []byte) map[string]any {
				m, args, ok := decode(poolABI, data)
				if ok && m.Name == "previewSwap" && args[0].(common.Address) == common.HexToAddress(cbBTC) &&
					args[2].(*big.Int).Cmp(big.NewInt(limit)) > 0 {
					return tc.fail
				}
				return nil
			}
			p, err := track(t, n, cfg, pools[0])
			if tc.err {
				require.ErrorIs(t, err, ErrRead)
				return
			}
			require.NoError(t, err)
			l, ok := lo.Find(extraOf(t, p).Ladders, func(l Ladder) bool { return l.In == 0 })
			require.True(t, ok)
			require.LessOrEqual(t, l.Points[len(l.Points)-1][0].ToBig().Cmp(big.NewInt(limit)), 0, "the ladder ends there")
			require.Len(t, extraOf(t, p).Ladders, 2)
		})
	}
}

// An answer that cannot be a fill is malformed, and settle stops at it: a direction whose first answer is one
// publishes empty, with no error, and the next refresh asks again.
func TestTrackerLeavesEmptyADirectionWhoseAnswerItCannotRead(t *testing.T) {
	for name, disable := range map[string]bool{"the sampler": false, "the plain path": true} {
		t.Run(name, func(t *testing.T) {
			w, cfg := testWorld(), testConfig()
			cfg.DisableSampler = disable
			n, pools := listed(t, w, cfg)
			n.respond = func(from, to common.Address, data []byte) (bool, []byte) {
				if m, _, ok := decode(poolABI, data); ok && m.Name == "previewSwap" {
					return true, nil
				}
				return w.respond(from, to, data)
			}
			p, err := track(t, n, cfg, pools[0])
			require.NoError(t, err)
			require.Empty(t, extraOf(t, p).Ladders)
		})
	}
}

func TestTrackerNeverProbesAPoolBehindAFeedTheListDoesNotName(t *testing.T) {
	w, cfg := testWorld(), testConfig()
	n, pools := listed(t, w, cfg)
	w.pools[common.HexToAddress(poolX)].feed = common.HexToAddress(stranger)
	before := len(n.calls())
	p, err := track(t, n, cfg, pools[0])
	require.NoError(t, err)
	require.Empty(t, extraOf(t, p).Ladders)
	require.Equal(t, uint64(testBlk), p.BlockNumber)
	for _, c := range n.seen()[before:] {
		require.NotEqual(t, common.HexToAddress(stranger), c.to, "nothing of the feed is called")
		require.False(t, c.sampler, "a pool behind a stranger's feed is never sampled")
		m, _, ok := decode(poolABI, c.data)
		require.False(t, ok && m.Name == "previewSwap", "a pool behind a stranger's feed is never probed")
	}
}

// A pricer the Factory lists is a front, and so is a swap hook on a pool whose cheap lane is on: this build routes
// neither, whatever SkipFrontPriced and AnyHook say and whatever code the front runs or proxies to, and reads no
// ERC-1967 slot to tell. A front's family stays in the config, unused.
func TestTrackerRefusesAListedFrontWhateverTheKnobsSay(t *testing.T) {
	front, impl := common.HexToAddress("0xe1"), common.HexToAddress("0xe2")
	frontCode := []byte{0x60, 0x01, 0x60, 0x03}
	for name, tc := range map[string]struct {
		lane, listed bool
		code, impl   []byte // the front's code and, where it is a proxy, the code its slot names
	}{
		"a proxy, listed, the lane on, a front's build behind it":  {lane: true, listed: true, code: fakeProxyCode(), impl: frontCode},
		"a proxy, listed, the lane not open yet":                   {listed: true, code: fakeProxyCode(), impl: frontCode},
		"a proxy, listed, a listed hook's build behind it":         {listed: true, code: fakeProxyCode(), impl: hookCode(0xaa)},
		"a front's build, listed, the lane not open yet":           {listed: true, code: frontCode},
		"a listed hook's build, listed":                            {listed: true, code: hookCode(0xcc)},
		"a listed hook's build, the lane on, the Factory delisted": {lane: true, code: hookCode(0xcc)},
	} {
		for knobs, set := range map[string]func(*Config){
			"defaults":        func(*Config) {},
			"front pairs on":  func(cfg *Config) { cfg.SkipFrontPriced = lo.ToPtr(false) },
			"any hook, too":   func(cfg *Config) { cfg.SkipFrontPriced, cfg.AnyHook = lo.ToPtr(false), true },
			"front pairs off": func(cfg *Config) { cfg.SkipFrontPriced = lo.ToPtr(true) },
		} {
			t.Run(name+", "+knobs, func(t *testing.T) {
				w, cfg := testWorld(), testConfig()
				w.hook(front, tc.code)
				w.code[impl], w.impl[front] = tc.impl, impl
				cfg.HookFamilies = append(cfg.HookFamilies, HookFamily{Name: "front",
					CodeHash: crypto.Keccak256Hash(frontCode).Hex(), Front: true})
				set(cfg)
				n, pools := listed(t, w, cfg)
				fp := w.pools[common.HexToAddress(poolX)]
				fp.hook, w.fronts[front] = front, tc.listed
				if tc.lane {
					fp.laneConfig = new(big.Int).Lsh(big.NewInt(1), laneVersionShift)
				}
				before := len(n.calls())
				p, err := track(t, n, cfg, pools[0])
				require.NoError(t, err)
				require.Empty(t, extraOf(t, p).Ladders)
				require.Empty(t, slotReads(n, before), "no slot is read")
				for _, c := range n.seen()[before:] {
					m, _, ok := decode(poolABI, c.data)
					require.False(t, ok && m.Name == "previewSwap", "a front's pairs are never probed")
					require.False(t, c.sampler, "nor sampled")
				}
			})
		}
	}

	// the control: the same hook, unlisted on a pool whose lane is shut, is routed
	w, cfg := testWorld(), testConfig()
	w.hook(front, hookCode(0xcc))
	n, pools := listed(t, w, cfg)
	w.pools[common.HexToAddress(poolX)].hook = front
	p, err := track(t, n, cfg, pools[0])
	require.NoError(t, err)
	require.ElementsMatch(t, [][2]uint8{{0, 1}, {1, 0}}, directionsOf(extraOf(t, p)))
}

// fakeProxyCode is a contract's code with the bytes a proxy's reads its implementation with: PUSH32 slot, SLOAD, then
// a DELEGATECALL. Any contract can carry them, a proxy or not.
func fakeProxyCode() []byte {
	return append(append([]byte{0x60, 0x80, 0x7f}, erc1967Slot[:]...), 0x54, 0xf4)
}

func slotReads(n *node, before int) []sent {
	return lo.Filter(n.calls()[before:], func(c sent, _ int) bool {
		return c.from == (common.Address{}) &&
			c.data != nil && common.BytesToHash(c.data) == erc1967Slot
	})
}

// Any pool creator can install a hook whose code holds a proxy's bytes and whose ERC-1967 slot names a listed build.
// Every hook stands on its own code, and one the Factory lists is a front, not routed.
func TestTrackerJudgesAHookTheFactoryDoesNotListByItsOwnCode(t *testing.T) {
	fake := common.HexToAddress("0xe1")
	for name, tc := range map[string]struct {
		edit func(w *world, fp *fakePool, cfg *Config)
	}{
		"as the swap hook": {edit: func(_ *world, fp *fakePool, _ *Config) { fp.hook = fake }},
		"as the pair hook": {edit: func(_ *world, fp *fakePool, _ *Config) {
			fp.pairHook, fp.pairMask = fake, bits(pairBit(0, 1))
		}},
		"as both": {edit: func(_ *world, fp *fakePool, _ *Config) {
			fp.hook, fp.pairHook, fp.pairMask = fake, fake, bits(pairBit(0, 1))
		}},
		"front pairs on, the Factory silent": {edit: func(_ *world, fp *fakePool, cfg *Config) {
			fp.hook, cfg.SkipFrontPriced = fake, lo.ToPtr(false)
		}},
		"the same contract once the Factory lists it, whatever the knobs say": {
			edit: func(w *world, fp *fakePool, cfg *Config) {
				fp.hook, w.fronts[fake], cfg.SkipFrontPriced, cfg.AnyHook = fake, true, lo.ToPtr(false), true
			},
		},
		"the same contract once the Factory lists it, front pairs skipped": {
			edit: func(w *world, fp *fakePool, _ *Config) { fp.hook, w.fronts[fake] = fake, true },
		},
	} {
		t.Run(name, func(t *testing.T) {
			w, cfg := testWorld(), testConfig()
			w.code[fake], w.impl[fake] = fakeProxyCode(), common.HexToAddress(hookA) // hookA: of a listed family
			n, pools := listed(t, w, cfg)
			tc.edit(w, w.pools[common.HexToAddress(poolX)], cfg)
			before := len(n.calls())
			p, err := track(t, n, cfg, pools[0])
			require.NoError(t, err)
			require.Empty(t, extraOf(t, p).Ladders, "the hook pair is the fake's: USDT/USDC is no hook pair")
			require.Empty(t, slotReads(n, before), "no slot is read")
		})
	}
}

// The same with the protocol-listed builds and a real hook's code, the slot naming a deployed EverlongHook.
func TestTrackerRefusesAFakeProxyOfTheProtocolListedBuild(t *testing.T) {
	fx := loadFixture(t, "testdata/base-fork.json.gz")
	w, cfg := testWorld(), testConfig()
	cfg.HookFamilies = fx.Config.HookFamilies // the deployed build, and the front, as that deploy listed them
	fake, real := common.HexToAddress("0xe1"), common.HexToAddress("0xe2")
	w.code[fake], w.impl[fake] = fakeProxyCode(), real
	w.hook(real, common.FromHex(lo.Values(fx.Code)[0]))
	n, pools := listed(t, w, cfg)
	fp := w.pools[common.HexToAddress(poolX)]

	fp.hook = real // the control: the build itself passes
	p, err := track(t, n, cfg, pools[0])
	require.NoError(t, err)
	require.Len(t, extraOf(t, p).Ladders, 2)

	fp.hook = fake
	p, err = track(t, n, cfg, pools[0])
	require.NoError(t, err)
	require.Empty(t, extraOf(t, p).Ladders)
}

// A stranger's deployment of a front's build is no family's: unlisted it is not routed, and listed it is a front,
// refused whatever the knobs say.
func TestTrackerNeverRoutesAFrontsBuildListedOrNot(t *testing.T) {
	copyOf := common.HexToAddress("0xe1") // a stranger's deployment of a front's build, not a proxy
	frontCode := []byte{0x60, 0x01, 0x60, 0x03}
	w, cfg := testWorld(), testConfig()
	w.hook(copyOf, frontCode)
	cfg.HookFamilies = append(cfg.HookFamilies, HookFamily{Name: "front",
		CodeHash: crypto.Keccak256Hash(frontCode).Hex(), Front: true})
	cfg.SkipFrontPriced = lo.ToPtr(false)
	n, pools := listed(t, w, cfg)
	w.pools[common.HexToAddress(poolX)].hook = copyOf
	p, err := track(t, n, cfg, pools[0])
	require.NoError(t, err)
	require.Empty(t, extraOf(t, p).Ladders)

	w.fronts[copyOf] = true
	for _, anyHook := range []bool{false, true} {
		cfg.AnyHook = anyHook
		p, err = track(t, n, cfg, pools[0])
		require.NoError(t, err)
		require.Empty(t, extraOf(t, p).Ladders, "any hook %v", anyHook)
	}
}

// A listed front is refused wherever it prices, as the pair hook too, whatever the knobs say.
func TestTrackerRefusesAListedFrontAsThePairHook(t *testing.T) {
	w, cfg := testWorld(), testConfig()
	n, pools := listed(t, w, cfg)
	fp := w.pools[common.HexToAddress(poolX)]
	fp.pairHook, fp.pairMask = common.HexToAddress(pairHk), bits(pairBit(0, 1))
	p, err := track(t, n, cfg, pools[0])
	require.NoError(t, err)
	require.ElementsMatch(t, [][2]uint8{{0, 1}, {1, 0}}, directionsOf(extraOf(t, p)), "the control: unlisted")

	w.fronts[common.HexToAddress(pairHk)] = true
	for name, set := range map[string]func(){
		"defaults":       func() {},
		"front pairs on": func() { cfg.SkipFrontPriced = lo.ToPtr(false) },
		"any hook, too":  func() { cfg.AnyHook = true },
	} {
		set()
		p, err = track(t, n, cfg, pools[0])
		require.NoError(t, err)
		require.Empty(t, extraOf(t, p).Ladders, name)
	}
}

// firstLever is a direction's first point past the leverage route of tierWorld: the first the curve priced.
func firstLever(l Ladder) int {
	top := lo.Ternary(l.In == 0, big.NewInt(tierSaleTop), big.NewInt(tierBuyTop))
	return slices.IndexFunc(l.Points, func(pt Point) bool { return pt[0].ToBig().Cmp(top) > 0 })
}

// A hook whose leverage route beats its curve on small fills. Each direction reads the pool's request once, at its
// first tier-labelled price over dust, from the taker at the block, and a buy reads the hook's levState on the
// request's words. The span across the change of route is flat, its left end lowered to the ray of its right end,
// and the simulator takes one fill a direction. The same on the plain path.
func TestTrackerSettlesALeverageRouteUnderItsCurve(t *testing.T) {
	for name, disable := range map[string]bool{"the sampler": false, "the plain path": true} {
		t.Run(name, func(t *testing.T) { settlesALeverageRouteUnderItsCurve(t, disable) })
	}
}

func settlesALeverageRouteUnderItsCurve(t *testing.T, disable bool) {
	w, fp := tierWorld()
	cfg := testConfig()
	cfg.DisableSampler = disable
	n, pools := listed(t, w, cfg)
	before := len(n.seen())
	p, err := track(t, n, cfg, pools[0])
	require.NoError(t, err)
	e := extraOf(t, p)
	require.ElementsMatch(t, [][2]uint8{{0, 1}, {1, 0}}, directionsOf(e))

	block, hook := hexutil.EncodeUint64(testBlk), common.HexToAddress(hookA)
	requests, first := map[[2]int]*big.Int{}, map[[2]int]*big.Int{}
	var levStates [][]byte
	for _, c := range n.seen()[before:] {
		if c.to == hook && len(c.data) > 4 {
			require.True(t, bytes.HasPrefix(c.data, levStateSelector), "no hook read but levState: no curve fee here")
			require.Equal(t, defaultTaker, c.from)
			require.Equal(t, block, c.tag)
			levStates = append(levStates, c.data)
			continue
		}
		m, args, ok := decode(poolABI, c.data)
		if !ok || m.Name != "pairRequest" && m.Name != "previewSwap" {
			continue
		}
		d := [2]int{fp.index(args[0].(common.Address)), fp.index(args[1].(common.Address))}
		switch a := args[2].(*big.Int); m.Name {
		case "pairRequest":
			require.Equal(t, defaultTaker, c.from)
			require.Equal(t, block, c.tag)
			require.NotContains(t, requests, d, "one request a direction")
			requests[d] = a
		case "previewSwap":
			_, out, ok := fp.quote(defaultTaker, d[0], d[1], a)
			if ok && out.Cmp(big.NewInt(dustOut)) >= 0 && fp.fee(d[0], d[1], a) != curveLabel &&
				(first[d] == nil || a.Cmp(first[d]) < 0) {
				first[d] = a
			}
		}
	}
	require.Len(t, requests, 2)
	require.Equal(t, first, requests, "each at the direction's first tier-labelled price over dust")
	require.Len(t, levStates, 1, "on the buy only")
	m := poolABI.Methods["pairRequest"]
	_, words := pack(&m, fp.request(defaultTaker, 1, 0, requests[[2]int{1, 0}]))
	require.Equal(t, slices.Concat(levStateSelector, words), levStates[0], "the request's 26 words, as answered")

	for _, l := range e.Ladders {
		k := firstLever(l)
		require.Positive(t, k, "direction %d -> %d", l.In, l.Out)
		require.Equal(t, uint64(1)<<k, l.Flat, "direction %d -> %d: flat across the change of route only", l.In, l.Out)
		ray := new(big.Int).Mul(l.Points[k][1].ToBig(), l.Points[k-1][0].ToBig())
		ray.Quo(ray, l.Points[k][0].ToBig())
		require.Equal(t, ray, l.Points[k-1][1].ToBig(), "the leverage top lowered to the ray of the curve's point")
		for _, pt := range slices.Concat(l.Points[:k-1], l.Points[k:]) { // every other point as the pool paid
			_, out, _ := fp.quote(defaultTaker, int(l.In), int(l.Out), pt[0].ToBig())
			require.Equal(t, out, pt[1].ToBig())
		}
	}
	l, _ := lo.Find(e.Ladders, func(l Ladder) bool { return l.In == 1 })
	require.True(t, l.Capped, "the curve's clip at the USDC ceiling")

	for k := range p.Tokens {
		p.Tokens[k].Decimals = []uint8{8, 6}[k]
	}
	s, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
	require.NoError(t, err)
	sale := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: cbBTC, Amount: big.NewInt(1_000_000)},
		TokenOut: usdc}
	res, err := s.CalcAmountOut(sale)
	require.NoError(t, err)
	s.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: sale.TokenAmountIn, TokenAmountOut: *res.TokenAmountOut,
		Fee: *res.Fee, SwapInfo: res.SwapInfo})
	_, err = s.CalcAmountOut(sale)
	require.ErrorIs(t, err, ladder.ErrNoQuote, "a direction with a flat span takes one fill")
	_, err = s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: usdc, Amount: big.NewInt(1_000_000_000)}, TokenOut: cbBTC})
	require.NoError(t, err, "the other direction's first fill")
}

// A tier-labelled point stands only where the refresh can vouch for it: the request read, the row's gate passed at
// the live band, a pool of two tokens, and on a buy a top bid above zero. Otherwise the ladder is cut before its
// first tier-labelled point and keeps the curve's points under it.
func TestTrackerCutsTheTierPointsItCannotVouchFor(t *testing.T) {
	const saleFrom, buyFrom = 1_000_000, 1_000_000_000 // the curve pays under them
	for name, tc := range map[string]struct {
		edit      func(w *world, fp *fakePool, h *fakeHook)
		sale, buy bool // whether each direction keeps its tier points
	}{
		"the control":                  {sale: true, buy: true},
		"the request reverts":          {edit: func(_ *world, fp *fakePool, _ *fakeHook) { fp.noRequest = true }},
		"the request has no USD words": {edit: func(_ *world, fp *fakePool, _ *fakeHook) { fp.usd = nil }},
		"a buy's top bid reads zero": {edit: func(_ *world, _ *fakePool, h *fakeHook) { h.bid = new(big.Int) },
			sale: true},
		"levState reverts":       {edit: func(_ *world, _ *fakePool, h *fakeHook) { h.noLev = true }, sale: true},
		"levState answers short": {edit: func(_ *world, _ *fakePool, h *fakeHook) { h.levWords = 7 }, sale: true},
		"a buy of a book that holds no pool asset": {edit: func(_ *world, fp *fakePool, _ *fakeHook) {
			fp.legs[0] = [4]*big.Int{new(big.Int), new(big.Int), new(big.Int), new(big.Int)}
		}, sale: true},
		"a row with no lever-down cap": {edit: func(_ *world, _ *fakePool, h *fakeHook) { h.row[3] = new(big.Int) }},
		"a live band too tight for the row's caps": {edit: func(_ *world, fp *fakePool, _ *fakeHook) {
			fp.tokens[1].SwapPriceBandWad = 20_000_000_000_000_000 // 2 %: the lever-down's cap leaves no room
		}},
		"a third token": {edit: func(_ *world, fp *fakePool, _ *fakeHook) {
			fp.tokens = append(fp.tokens, tokenCfg(usdt, 6))
			fp.legs = append(fp.legs, fp.legs[1])
			fp.usd = append(fp.usd, wad)
		}},
	} {
		t.Run(name, func(t *testing.T) {
			w, fp := tierWorldFrom(saleFrom, buyFrom)
			if tc.edit != nil {
				tc.edit(w, fp, w.hooks[common.HexToAddress(hookA)])
			}
			cfg := testConfig()
			n, pools := listed(t, w, cfg)
			p, err := track(t, n, cfg, pools[0])
			require.NoError(t, err)
			e := extraOf(t, p)
			require.ElementsMatch(t, [][2]uint8{{0, 1}, {1, 0}}, directionsOf(e))
			for _, l := range e.Ladders {
				from := lo.Ternary[int64](l.In == 0, saleFrom, buyFrom)
				keeps := lo.Ternary(l.In == 0, tc.sale, tc.buy)
				top := l.Points[len(l.Points)-1][0].ToBig()
				if !keeps {
					require.Negative(t, top.Cmp(big.NewInt(from)), "direction %d -> %d: cut under the leverage", l.In,
						l.Out)
					require.False(t, l.Capped)
					require.Zero(t, l.Flat, "the curve's points only")
					continue
				}
				require.Positive(t, firstLever(l), "direction %d -> %d: the tier points and the curve past them", l.In,
					l.Out)
			}
		})
	}
}

// The probe above: where the priced probe of highest ask over dust is tier-labelled, one previewSwap at
// used + (used >> 10) + 1. Priced, it keeps the tier top; refused, the tier points within 1/1024 of the top go. A walk
// of one tier label is flat on the plain path, which does not read the curve's fee; the sampler reads it, and as it
// is not the label the spans are chords.
func TestTrackerProbesAboveATierTop(t *testing.T) {
	for name, disable := range map[string]bool{"the sampler": false, "the plain path": true} {
		t.Run(name, func(t *testing.T) { probesAboveATierTop(t, disable) })
	}
}

func probesAboveATierTop(t *testing.T, disable bool) {
	const top = tierSaleTop
	above := big.NewInt(top + top>>aboveShift + 1)
	for name, curveAbove := range map[string]bool{"the curve above": true, "a refusal above": false} {
		t.Run(name, func(t *testing.T) {
			w, fp := tierWorld()
			fp.tokens[0].MaxSwapNotional = big.NewInt(top) // the walk ends on the tier top
			if !curveAbove {
				quote := fp.quote
				fp.quote = func(from common.Address, i, j int, a *big.Int) (*big.Int, *big.Int, bool) {
					if i == 0 && a.Cmp(big.NewInt(top)) > 0 {
						return nil, nil, false
					}
					return quote(from, i, j, a)
				}
			}
			cfg := testConfig()
			cfg.DisableSampler = disable
			n, pools := listed(t, w, cfg)
			before := len(n.seen())
			p, err := track(t, n, cfg, pools[0])
			require.NoError(t, err)
			var past []*big.Int // the sales' asks past the cap
			for _, c := range n.seen()[before:] {
				if m, args, ok := decode(poolABI, c.data); ok && m.Name == "previewSwap" &&
					args[0].(common.Address) == common.HexToAddress(cbBTC) && args[2].(*big.Int).Cmp(big.NewInt(top)) > 0 {
					past = append(past, args[2].(*big.Int))
				}
			}
			require.Equal(t, []*big.Int{above}, past, "one probe above, nothing else past the cap")

			l, ok := lo.Find(extraOf(t, p).Ladders, func(l Ladder) bool { return l.In == 0 })
			require.True(t, ok)
			last := l.Points[len(l.Points)-1][0].ToBig()
			if curveAbove {
				require.Equal(t, above, last, "the probe above is the ladder's top")
				require.Equal(t, big.NewInt(top), l.Points[len(l.Points)-2][0].ToBig(), "the tier top stands")
				require.Equal(t, uint64(1)<<(len(l.Points)-1), l.Flat)
				return
			}
			require.LessOrEqual(t, new(big.Int).Add(last, new(big.Int).Rsh(last, aboveShift)).Cmp(big.NewInt(top)), 0,
				"no tier point within 1/1024 of the last priced size")
			if disable {
				require.Equal(t, uint64(1)<<len(l.Points)-2, l.Flat, "one tier label, the curve's fee unread: all flat")
			} else {
				require.Zero(t, l.Flat, "one tier label, the curve's fee read and not the label: chords")
			}
			require.False(t, l.Capped)
		})
	}
}

// A listed family's params() is read once a tracker, its row fixed at the hook's construction; a read that fails
// fails the refresh, as no row would skip the room test. A hook of no family, under AnyHook, is asked twice before it
// counts as having no row, and only a decoded answer is kept.
func TestTrackerReadsAHooksRowOnce(t *testing.T) {
	type hookCase struct {
		reverts, words int
		zeroRow        bool
	}
	for name, tc := range map[string]struct {
		hook   hookCase
		unlist bool
		err    bool
		asked  []int // the params() calls after each of two refreshes
		dirs   [][2]uint8
	}{
		"a listed hook":                    {asked: []int{1, 1}, dirs: [][2]uint8{{0, 1}, {1, 0}}},
		"a listed hook, one revert":        {hook: hookCase{reverts: 1}, asked: []int{2, 2}, dirs: [][2]uint8{{0, 1}, {1, 0}}},
		"a listed hook that reverts":       {hook: hookCase{reverts: -1}, err: true},
		"a listed hook that answers short": {hook: hookCase{words: 34}, err: true},
		// a hook of no family has no measured fill, so no cells: its pairs are not routed and its row is never read
		"a hook of no family, no row":       {unlist: true, hook: hookCase{zeroRow: true}, asked: []int{0, 0}},
		"a hook of no family that reverts":  {unlist: true, hook: hookCase{reverts: -1}, asked: []int{0, 0}},
		"a hook of no family, one revert":   {unlist: true, hook: hookCase{reverts: 1, zeroRow: true}, asked: []int{0, 0}},
		"a hook of no family with a row":    {unlist: true, asked: []int{0, 0}},
		"a hook of no family answers short": {unlist: true, hook: hookCase{words: 34}, asked: []int{0, 0}},
	} {
		t.Run(name, func(t *testing.T) {
			w, cfg := testWorld(), testConfig()
			h := w.hooks[common.HexToAddress(hookA)]
			h.reverts, h.words = tc.hook.reverts, tc.hook.words
			if tc.hook.zeroRow {
				h.row = [8]*big.Int{}
			}
			if tc.unlist {
				cfg.HookFamilies, cfg.AnyHook = cfg.HookFamilies[1:], true
			}
			n, pools := listed(t, w, cfg)
			tracker := NewPoolTracker(cfg, n.client())
			for k := range 2 {
				p, err := tracker.GetNewPoolState(context.Background(), pools[0], pool.GetNewPoolStateParams{})
				if tc.err {
					require.ErrorIs(t, err, ErrRead)
					return
				}
				require.NoError(t, err)
				require.Equal(t, tc.asked[k], h.asked, "params() calls after refresh %d", k+1)
				e := extraOf(t, p)
				require.ElementsMatch(t, tc.dirs, directionsOf(e))
				for _, l := range e.Ladders {
					require.Zero(t, l.Flat, "a curve's ladder: all chords")
				}
			}
		})
	}
}

// levState takes the hook's PairRequest, 26 static words, the same words pairRequest answers.
func TestTrackerPinsTheLevStateSelector(t *testing.T) {
	sig := "levState((address,address,address,uint8,uint8,uint16,uint16,uint16,uint64,uint64,uint64" +
		strings.Repeat(",uint256", 15) + "))"
	require.Equal(t, crypto.Keccak256([]byte(sig))[:4], levStateSelector)
	require.Len(t, poolABI.Methods["pairRequest"].Outputs, 1)
	require.Equal(t, requestWords, poolABI.Methods["pairRequest"].Outputs[0].Type.Size)
	require.Equal(t, paramsWords, hookABI.Methods["params"].Outputs[0].Type.Size)
}

// A price feed that is not an address, or is the zero address, refuses the config: read as an address, it would
// otherwise be the zero address, which a pool's unset feed also reads as.
func TestConfigRefusesAMalformedOrZeroPriceFeed(t *testing.T) {
	const feed = "0x00000000000000000000000000000000000000fe"
	var ok Config
	require.NoError(t, json.Unmarshal([]byte(`{"dexID":"everlong-flamm","chainID":56,"priceFeeds":["`+feed+`"]}`), &ok))
	require.True(t, ok.hasFeed(common.HexToAddress(feed)))
	for name, bad := range map[string]string{
		"not hex":   "0xnothex",
		"too short": "0x1234",
		"empty":     "",
		"zero":      "0x0000000000000000000000000000000000000000",
	} {
		var cfg Config
		require.Error(t, json.Unmarshal([]byte(`{"priceFeeds":["`+feed+`","`+bad+`"]}`), &cfg), name)
	}
}
