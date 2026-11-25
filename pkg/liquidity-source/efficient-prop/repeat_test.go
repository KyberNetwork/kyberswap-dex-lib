package efficientprop

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// applied is the simulator after the fill of amountIn it quoted res for.
func applied(sim *PoolSimulator, in string, amountIn *big.Int, res *pool.CalcAmountOutResult) {
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: in, Amount: amountIn},
		TokenAmountOut: *res.TokenAmountOut,
		SwapInfo:       res.SwapInfo,
	})
}

// laterSize is a size for the route's next fill of a side whose ladder is points, used of its input taken: up to a
// point of the ladder exactly, one unit to either side of that, small, or anywhere in what is left.
func laterSize(rng *rand.Rand, points []Point, used *big.Int) *big.Int {
	first, last := points[0][0].ToBig(), points[len(points)-1][0].ToBig()
	left := new(big.Int).Sub(last, used)
	if left.Sign() <= 0 {
		return big.NewInt(1)
	}
	switch rng.Intn(6) {
	case 0, 1:
		to := points[rng.Intn(len(points))][0].ToBig()
		if a := new(big.Int).Sub(to, used); a.Sign() > 0 {
			return a.Add(a, big.NewInt(int64(rng.Intn(3)-1)))
		}
	case 2:
		return new(big.Int).Add(first, new(big.Int).Rand(rng, new(big.Int).Add(first, big.NewInt(1))))
	case 3:
		return left
	}
	return new(big.Int).Add(new(big.Int).Rand(rng, left), big.NewInt(1))
}

// laterSlack is the most a later fill's quote may be under the door on a ladder one block sampled with a point on
// every rung boundary, in base units of the output: 5, plus the price step to a neighbouring span times the part of
// the starting span on that side, the lesser of the two sides (curve.quoteAfter). The side is b as the ladder sampled
// it, used the input the route's fills took. With i the first point at or past used and s[k] the price of the span
// ending at point k:
//
//	right: (x[i] - used) * (s[i] - s[i+1] + 3/dx[i+1]), or (x[i] - used) * s[i] where i is the last point
//	left:  (used - x[i-1]) * (s[i-1] - s[i] + 2/dx[i-1]), or (x[0] - used) * s[0] where i is the first
func laterSlack(b *fakeBook, points []Point, used *big.Int) *big.Rat {
	m := len(points)
	x := func(k int) *big.Rat {
		if k < 0 {
			return new(big.Rat)
		}
		return new(big.Rat).SetInt(points[k][0].ToBig())
	}
	paid := func(k int) *big.Rat {
		if k < 0 {
			return new(big.Rat)
		}
		_, n := b.walk(points[k][0].ToBig())
		return new(big.Rat).SetFrac(n, b.scale())
	}
	dx := func(k int) *big.Rat { return new(big.Rat).Sub(x(k), x(k-1)) }
	slope := func(k int) *big.Rat {
		s := new(big.Rat).Sub(paid(k), paid(k-1))
		return s.Quo(s, dx(k))
	}
	at := new(big.Rat).SetInt(used)
	i := 0
	for i < m-1 && x(i).Cmp(at) < 0 {
		i++
	}
	ahead := new(big.Rat).Sub(x(i), at)
	right := new(big.Rat).Mul(ahead, slope(i))
	if i+1 < m {
		step := new(big.Rat).Sub(slope(i), slope(i+1))
		step.Add(step, new(big.Rat).Quo(big.NewRat(3, 1), dx(i+1)))
		right.Mul(ahead, step)
	}
	left := new(big.Rat).Set(right)
	if i > 0 {
		step := new(big.Rat).Sub(slope(i-1), slope(i))
		step.Add(step, new(big.Rat).Quo(big.NewRat(2, 1), dx(i-1)))
		left.Mul(new(big.Rat).Sub(at, x(i-1)), step)
	}
	if left.Cmp(right) < 0 {
		right = left
	}
	return right.Add(right, big.NewRat(5, 1))
}

// A direction takes two and three fills a route, each later one priced from the ladder shifted by the input the fills
// before it took, rounded up to the side's board unit as the front's fill memory counts it. Against a side filled in
// sequence as the front fills it, both directions: the simulator never pays more than the door pays that fill, never
// quotes one the door refuses, and its shift is the door's fill memory to the unit. On a ladder with a point on every
// rung boundary it is under the door by no more than laterSlack.
func TestSimulatorPricesLaterFillsFromTheRemainingDepth(t *testing.T) {
	rng := rand.New(rand.NewSource(31))
	var quoted [4][2]int
	var worst [2]*big.Int
	var worstPpm, sumPpm [2]float64
	var bounded [2]int
	for trial := range 1600 {
		b := randomBook(rng, trial%2 == 0)
		edged := trial%8 < 6
		points := refreshed(b, edged)
		if len(points) < 2 {
			continue
		}
		sim, in, out := bookSimOf(t, b, points, repeatOwn)
		dir := sim.GetTokenIndex(in)
		door := b.at(0)
		start := door.memory()
		for n := 1; n <= 3; n++ {
			used := new(big.Int).Mul(new(big.Int).SetUint64(door.memory()-start), b.unit())
			require.Equal(t, used.String(), sim.used[dir].Dec(), "trial %d: the shift is the door's fill memory", trial)
			var a *big.Int
			var res *pool.CalcAmountOutResult
			for range 6 {
				size := laterSize(rng, points, used)
				if n == 1 && rng.Intn(3) > 0 {
					// a first fill that leaves room
					size = new(big.Int).Add(points[0][0].ToBig(), new(big.Int).Rand(rng, points[len(points)/2][0].ToBig()))
				}
				if r, err := calc(sim, in, out, size, nil); err == nil {
					a, res = size, r
					break
				}
			}
			if a == nil {
				break
			}
			want, err := door.quote(a)
			require.NoError(t, err, "trial %d fill %d: the simulator quotes %s where the door refuses", trial, n, a)
			short := new(big.Int).Sub(want, res.TokenAmountOut.Amount)
			require.GreaterOrEqual(t, short.Sign(), 0, "trial %d fill %d of %s after %s: the simulator pays %s, the door %s",
				trial, n, a, used, res.TokenAmountOut.Amount, want)
			quoted[n][dir]++
			if n > 1 && edged {
				slack := laterSlack(b, points, used)
				require.LessOrEqual(t, new(big.Rat).SetInt(short).Cmp(slack), 0,
					"trial %d fill %d of %s after %s: %s under the door, more than %s", trial, n, a, used, short,
					slack.FloatString(2))
				ppm, _ := new(big.Rat).SetFrac(new(big.Int).Mul(short, big.NewInt(bookPPM)), want).Float64()
				bounded[dir]++
				sumPpm[dir] += ppm
				worstPpm[dir] = max(worstPpm[dir], ppm)
				if worst[dir] == nil || short.Cmp(worst[dir]) > 0 {
					worst[dir] = short
				}
			}
			_, err = door.fill(a)
			require.NoError(t, err)
			applied(sim, in, a, res)
		}
	}
	for dir, side := range []string{"base in", "quote in"} {
		assert.Greater(t, quoted[2][dir], 300, "%s: second fills", side)
		assert.Greater(t, quoted[3][dir], 100, "%s: third fills", side)
		t.Logf("%s: %d first, %d second, %d third fills quoted; later fills on ladders with the boundaries: %d, worst "+
			"shortfall %s units (%.1f ppm of the output), mean %.2f ppm", side, quoted[1][dir], quoted[2][dir],
			quoted[3][dir], bounded[dir], worst[dir], worstPpm[dir], sumPpm[dir]/float64(max(bounded[dir], 1)))
	}
}

// The same on a ladder that keeps the least of several blocks, each paying less than the one before as the front's
// discount grows with the quote's age: the tracker finds the ladder steady, and in every one of the blocks the door,
// filled in sequence, pays each fill at least what the simulator quotes.
func TestSimulatorPricesLaterFillsOnALadderOfSeveralBlocks(t *testing.T) {
	rng := rand.New(rand.NewSource(37))
	var quoted [4]int
	var kinds [3]int
	var later int
	var worstPpm, sumPpm float64
	for trial := range 1200 {
		b := randomBook(rng, trial%2 == 0)
		blocks := []*fakeBook{b, b.at(1 + uint64(rng.Intn(400)))}
		if rng.Intn(2) == 0 {
			blocks = append(blocks, b.at(uint64(rng.Intn(200))))
		}
		sizes := bookGrid(b, bookLadder(b, bookGrid(b, nil, true)), trial%4 != 3)
		// each block's answers, and per size the least of them, none where a block refused
		samples := make([][]*big.Int, len(blocks))
		kept, ok := make([]*big.Int, len(sizes)), make([]bool, len(sizes))
		for k, block := range blocks {
			samples[k] = make([]*big.Int, len(sizes))
			for i, size := range sizes {
				if out, err := block.quote(size); err == nil {
					samples[k][i] = out
				}
			}
		}
		for i := range sizes {
			ok[i] = true
			for _, sample := range samples {
				switch {
				case sample[i] == nil:
					kept[i], ok[i] = nil, false
				case ok[i] && (kept[i] == nil || sample[i].Cmp(kept[i]) < 0):
					kept[i] = sample[i]
				}
			}
		}
		points, _ := collectClassedRun(sizes, kept, ok, nil)
		if len(points) < 2 {
			continue
		}
		kind := repeatKind(sizes, points, samples)
		require.NotZero(t, kind, "trial %d: one discount on every rung keeps the ladder steady", trial)
		kinds[kind]++
		sim, in, out := bookSimOf(t, b, points, kind)
		doors := make([]*fakeBook, len(blocks))
		for k, block := range blocks {
			doors[k] = block.at(0)
		}
		start := doors[0].memory()
		for n := 1; n <= 3; n++ {
			used := new(big.Int).Mul(new(big.Int).SetUint64(doors[0].memory()-start), b.unit())
			var a *big.Int
			var res *pool.CalcAmountOutResult
			for range 6 {
				size := laterSize(rng, points, used)
				if n == 1 && rng.Intn(3) > 0 {
					size = new(big.Int).Add(points[0][0].ToBig(), new(big.Int).Rand(rng, points[len(points)/2][0].ToBig()))
				}
				if r, err := calc(sim, in, out, size, nil); err == nil {
					a, res = size, r
					break
				}
			}
			if a == nil {
				break
			}
			var least *big.Int
			for k, door := range doors {
				want, err := door.fill(a)
				require.NoError(t, err, "trial %d fill %d: the simulator quotes %s where block %d refuses", trial, n, a, k)
				require.LessOrEqual(t, res.TokenAmountOut.Amount.Cmp(want), 0,
					"trial %d fill %d of %s after %s: the simulator pays %s, block %d pays %s", trial, n, a, used,
					res.TokenAmountOut.Amount, k, want)
				if least == nil || want.Cmp(least) < 0 {
					least = want
				}
			}
			quoted[n]++
			if n > 1 {
				short := new(big.Int).Sub(least, res.TokenAmountOut.Amount)
				ppm, _ := new(big.Rat).SetFrac(short.Mul(short, big.NewInt(bookPPM)), least).Float64()
				later++
				sumPpm += ppm
				worstPpm = max(worstPpm, ppm)
			}
			applied(sim, in, a, res)
		}
	}
	assert.Greater(t, quoted[2], 400, "second fills")
	assert.Greater(t, quoted[3], 150, "third fills")
	assert.Greater(t, kinds[repeatSteady], 10*kinds[repeatOwn], "most ladders are of blocks that answer differently")
	t.Logf("%d steady ladders, %d of blocks that answer alike; %d first, %d second, %d third fills quoted; later "+
		"fills under the block that pays least by %.1f ppm at worst, %.2f ppm in the mean", kinds[repeatSteady],
		kinds[repeatOwn], quoted[1], quoted[2], quoted[3], worstPpm, sumPpm/float64(max(later, 1)))
}

// What a ladder proves of a later fill: one sample's own ladder everything, several samples a steady ladder alone,
// and a sample that pays less over a stretch than the ladder rises there, or misses a point, nothing.
func TestRepeatKind(t *testing.T) {
	n := func(v ...int64) []*big.Int {
		out := make([]*big.Int, len(v))
		for i, x := range v {
			if x >= 0 {
				out[i] = big.NewInt(x)
			}
		}
		return out
	}
	sizes := n(5, 10, 20, 40, 80)
	run := []Point{pt(10, 100), pt(20, 190), pt(40, 350)}
	assert.Equal(t, uint8(repeatOwn), repeatKind(sizes, run, [][]*big.Int{n(-1, 100, 190, 350, -1)}))
	assert.Equal(t, uint8(repeatOwn), repeatKind(sizes, run, [][]*big.Int{n(-1, 100, 190, 350, -1),
		n(50, 100, 190, 350, 600)}), "several samples, each the ladder itself")
	assert.Equal(t, uint8(repeatSteady), repeatKind(sizes, run, [][]*big.Int{n(-1, 100, 191, 350, -1)}),
		"a sample above the ladder at a point is not its own")
	assert.Equal(t, uint8(repeatSteady), repeatKind(sizes, run, [][]*big.Int{n(50, 100, 195, 360, 600),
		n(-1, 104, 193, 353, -1)}), "an excess that falls by a unit is the floors'")
	assert.Zero(t, repeatKind(sizes, run, [][]*big.Int{n(50, 100, 195, 360, 600), n(-1, 104, 192, 352, -1)}),
		"a sample that pays 2 less than the ladder's rise between two points")
	assert.Zero(t, repeatKind(sizes, run, [][]*big.Int{n(50, 100, 195, 360, 600), n(-1, 100, 189, 350, -1)}),
		"a sample under the ladder")
	assert.Zero(t, repeatKind(sizes, run, [][]*big.Int{n(50, 100, 195, 360, 600), n(-1, 100, -1, 350, -1)}),
		"a sample with no answer at a point")
	assert.Zero(t, repeatKind(sizes, run, [][]*big.Int{n(100, 190, 350)}), "a sample of other sizes")
	assert.Zero(t, repeatKind(n(5, 10, 30, 40, 80), run, [][]*big.Int{n(-1, 100, 190, 350, -1)}),
		"a ladder that is no run of the sizes")
	assert.Zero(t, repeatKind(sizes, nil, [][]*big.Int{n(-1, 100, 190, 350, -1)}))
	assert.Zero(t, repeatKind(sizes, run, nil))
}

// A state that says nothing of a later fill keeps the one-fill rule, a direction at a time; a credit listing keeps it
// whatever its state says; and a clone filled further shares nothing with the simulator it was cloned from.
func TestSimulatorLaterFillsAreByDirectionAndByClone(t *testing.T) {
	build := func(r *Repeat) (*PoolSimulator, error) {
		p := simPool(t, simLadders(), 0)
		var e Extra
		require.NoError(t, json.Unmarshal([]byte(p.Extra), &e))
		e.Repeat = r
		extra, err := json.Marshal(e)
		require.NoError(t, err)
		p.Extra = string(extra)
		return NewPoolSimulator(pool.FactoryParams{EntityPool: p})
	}
	_, err := build(&Repeat{Kind: [2]uint8{3, 0}})
	assert.ErrorIs(t, err, ErrInvalidState, "a kind this does not know")
	_, err = build(&Repeat{Unit: [2]uint8{17, 0}, Kind: [2]uint8{1, 0}})
	assert.ErrorIs(t, err, ErrInvalidState, "a board unit the front never sets")

	// selling WETH takes further fills, in units of 1e13; buying it one
	sim, err := build(&Repeat{Unit: [2]uint8{13, 0}, Kind: [2]uint8{repeatOwn, 0}})
	require.NoError(t, err)
	half := big.NewInt(5e17 + 1)
	first, err := calc(sim, simWETH, simUSDC, half, nil)
	require.NoError(t, err)
	filled := sim.CloneState().(*PoolSimulator)
	applied(filled, simWETH, half, first)
	assert.Equal(t, "500010000000000000", filled.used[0].Dec(), "the input, rounded up to the board unit")
	assert.True(t, sim.used[0].IsZero(), "the original keeps its state")

	second, err := calc(filled, simWETH, simUSDC, big.NewInt(1e18), nil)
	require.NoError(t, err)
	// the ladder pays 4,498.5 USDC for 1.5 WETH and 1,500 for the first half: no more than the difference
	assert.LessOrEqual(t, second.TokenAmountOut.Amount.Int64(), int64(2998.5e6))
	assert.Greater(t, second.TokenAmountOut.Amount.Int64(), int64(2990e6))
	_, err = calc(filled, simWETH, simUSDC, big.NewInt(26e17), nil)
	assert.Error(t, err, "past the ladder's last size once shifted")
	_, err = calc(filled, simWETH, simUSDC, big.NewInt(9e16), nil)
	assert.ErrorIs(t, err, ErrBelowFirstPoint, "the minimum fill binds each fill")

	// a clone of the filled simulator, filled again: neither the filled one nor the original moves
	twice := filled.CloneState().(*PoolSimulator)
	applied(twice, simWETH, big.NewInt(1e18), second)
	assert.Equal(t, "1500010000000000000", twice.used[0].Dec())
	assert.Equal(t, "500010000000000000", filled.used[0].Dec())
	again, err := calc(filled, simWETH, simUSDC, big.NewInt(1e18), nil)
	require.NoError(t, err)
	assert.Equal(t, second.TokenAmountOut.Amount, again.TokenAmountOut.Amount, "the filled one quotes as before")
	third, err := calc(twice, simWETH, simUSDC, big.NewInt(1e18), nil)
	require.NoError(t, err)
	assert.Less(t, third.TokenAmountOut.Amount.Cmp(second.TokenAmountOut.Amount), 0, "further up the rungs")
	fresh, err := calc(sim, simWETH, simUSDC, half, nil)
	require.NoError(t, err)
	assert.Equal(t, first.TokenAmountOut.Amount, fresh.TokenAmountOut.Amount)
	assert.Equal(t, "100000000000000000000", sim.Info.Reserves[0].String())
	assert.Equal(t, "101500000000000000001", twice.Info.Reserves[0].String())

	// the other direction, of which the state says nothing: one fill
	buy, err := calc(filled, simUSDC, simWETH, big.NewInt(300e6), nil)
	require.NoError(t, err)
	applied(filled, simUSDC, big.NewInt(300e6), buy)
	_, err = calc(filled, simUSDC, simWETH, big.NewInt(300e6), nil)
	assert.ErrorIs(t, err, ErrSecondFill)
	assert.True(t, filled.used[1].IsZero())
}

// The tracker says what a later fill is priced by where it reads the front's words by their layout: each direction's
// board unit from the pair word, and that the ladder is steady over the blocks it sampled. It says nothing under an
// implementation the config does not list, of a front that does not answer the storage read, or of words of another
// layout; and the simulator built from the state prices a second fill no higher than the rungs past the first.
func TestTrackerSaysWhatALaterFillIsPricedBy(t *testing.T) {
	chain, _ := rungChain()
	n := chain.node(t, 1234)
	got := track(t, n, listed(t, n))
	e := extraOf(t, got)
	require.NotNil(t, e.Repeat)
	assert.Equal(t, [2]uint8{13, 4}, e.Repeat.Unit)
	for dir := range 2 {
		assert.NotZero(t, e.Repeat.Kind[dir], "direction %d", dir)
	}

	for name, set := range map[string]func(*fakeChain){
		"an implementation not listed": func(c *fakeChain) {
			c.impl = common.HexToAddress("0x6000000000000000000000000000000000000002")
		},
		"no storage read": func(c *fakeChain) { c.noSloads = true },
		"depth words of another version": func(c *fakeChain) {
			for _, k := range []int{0, 2} {
				d := c.listings[1].view.Depth[k]
				d.Xor(d, big.NewInt(3))
			}
		},
	} {
		other, _ := rungChain()
		set(other)
		m := other.node(t, 1234)
		assert.Nil(t, extraOf(t, track(t, m, listed(t, m))).Repeat, name)
	}

	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: got, Opts: pool.FactoryOpts{StaleCheck: true}})
	require.NoError(t, err)
	tokens := [2]string{hexutil.Encode(testWETH[:]), hexutil.Encode(testUSDC[:])}
	for dir, sizes := range [2][2]int64{{6e17, 15e17}, {2_000e6, 4_000e6}} {
		side := chain.listings[1].sides[dir]
		a, b := big.NewInt(sizes[0]), big.NewInt(sizes[1])
		first, err := calc(sim, tokens[dir], tokens[1-dir], a, nil)
		require.NoError(t, err)
		applied(sim, tokens[dir], a, first)
		second, err := calc(sim, tokens[dir], tokens[1-dir], b, nil)
		require.NoError(t, err, "direction %d", dir)
		// the rungs past the first fill: the walk to both less the walk to the first
		both, err := side.quote(new(big.Int).Add(a, b))
		require.NoError(t, err)
		one, err := side.quote(a)
		require.NoError(t, err)
		rest := new(big.Int).Sub(both, one)
		assert.LessOrEqual(t, second.TokenAmountOut.Amount.Cmp(rest), 0, "direction %d", dir)
		short := new(big.Int).Sub(rest, second.TokenAmountOut.Amount)
		assert.LessOrEqual(t, new(big.Int).Mul(short, big.NewInt(10_000)).Cmp(rest), 0,
			"direction %d: more than 1 bp under the rungs past the first fill", dir)
		assert.Equal(t, uint8(repeatOwn), e.Repeat.Kind[dir], "direction %d: no level moves over the window", dir)
	}
}
