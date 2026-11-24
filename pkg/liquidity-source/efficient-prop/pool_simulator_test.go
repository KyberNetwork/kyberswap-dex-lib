package efficientprop

import (
	"bytes"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/KyberNetwork/msgpack/v5"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
)

const (
	simDoor    = "0x5555555555555555555555555555555555555555"
	simFront   = "0x0000000000403aeb66691063cc2fd42e71d6acd6"
	simCustody = "0x7000000000000000000000000000000000000a11"
	simWETH    = "0x4200000000000000000000000000000000000006"
	simUSDC    = "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"
)

func pt(in, out int64) Point {
	return Point{uint256.NewInt(uint64(in)), uint256.NewInt(uint64(out))}
}

// simLadders: selling WETH at 3,000 USDC (1e18 -> 3000e6), 0.1% worse each further ETH; buying at 3,003.
func simLadders() [2][]Point {
	return [2][]Point{
		{pt(1e17, 300e6), pt(1e18, 3000e6), pt(2e18, 5997e6), pt(3e18, 8991e6)},
		{pt(300e6, 99_900_099_900_099_900), pt(3000e6, 999_000_999_000_999_000)},
	}
}

func simPool(t *testing.T, ladders [2][]Point, ts int64) entity.Pool {
	t.Helper()
	extra, err := json.Marshal(Extra{Ladders: ladders, TipCapWei: 1e8, ValidUntil: uint64(ts) + 20})
	require.NoError(t, err)
	staticExtra, err := json.Marshal(StaticExtra{Front: simFront, Listing: 1, Custody: simCustody, Door: simDoor,
		Mode: modeFront})
	require.NoError(t, err)
	return entity.Pool{
		Address: simDoor, Exchange: DexType, Type: DexType, BlockNumber: 77, Timestamp: ts,
		Tokens: []*entity.PoolToken{
			{Address: simWETH, Decimals: 18, Swappable: true},
			{Address: simUSDC, Decimals: 6, Swappable: true},
		},
		Reserves:    entity.PoolReserves{"100000000000000000000", "1000000000000"},
		Extra:       string(extra),
		StaticExtra: string(staticExtra),
	}
}

func mustSim(t *testing.T, ladders [2][]Point) *PoolSimulator {
	t.Helper()
	s, err := NewPoolSimulator(pool.FactoryParams{EntityPool: simPool(t, ladders, time.Now().Unix()),
		Opts: pool.FactoryOpts{StaleCheck: true}})
	require.NoError(t, err)
	return s
}

func calc(sim pool.IPoolSimulator, tokenIn, tokenOut string, amountIn *big.Int, limit pool.SwapLimit) (
	*pool.CalcAmountOutResult, error) {
	return sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn},
		TokenOut:      tokenOut,
		Limit:         limit,
	})
}

func bi(s string) *big.Int {
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic(s)
	}
	return v
}

func TestSimulatorIsExactAtSampledPoints(t *testing.T) {
	sim := mustSim(t, simLadders())
	for _, p := range simLadders()[0] {
		res, err := calc(sim, simWETH, simUSDC, p[0].ToBig(), nil)
		require.NoError(t, err)
		assert.Equal(t, p[1].ToBig(), res.TokenAmountOut.Amount)
		assert.Equal(t, int64(gasWordFrontBaseIn), res.Gas, "the base in, through the front")
	}
	res, err := calc(sim, simUSDC, simWETH, big.NewInt(3000e6), nil)
	require.NoError(t, err)
	assert.Equal(t, "999000999000999000", res.TokenAmountOut.Amount.String())
	assert.Equal(t, int64(gasWordFrontQuoteIn), res.Gas, "the quote in, through the front")
}

// withStatic is p with its static extra changed by edit.
func withStatic(t *testing.T, p entity.Pool, edit func(*StaticExtra)) entity.Pool {
	t.Helper()
	var se StaticExtra
	require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &se))
	edit(&se)
	b, err := json.Marshal(se)
	require.NoError(t, err)
	p.StaticExtra = string(b)
	return p
}

// Each direction is charged its own figure, by the pool's fill mode, by whether its input is the pair's base or its
// quote token, by whether the listing is on its pool's hook pair or past it, and by whether its fill reads a price
// source and whether that source is guarded: a print-reading listing adds the measured print difference of its input
// to the WORD figure of its class; a guarded one is charged its own measured figure on the hook pair and, past it, the
// WORD figure plus the guarded difference.
func TestSimulatorChargesEachDirectionsGas(t *testing.T) {
	// the least first fill of each class on the counted clock, less its refund, the recipient holding the output
	assert.Equal(t, 101_664, gasWordFrontBaseIn)
	assert.Equal(t, 93_570, gasWordFrontQuoteIn)
	assert.Equal(t, 114_072, gasWordDoorBaseIn)
	assert.Equal(t, 109_146, gasWordDoorQuoteIn)
	assert.Equal(t, 129_338, gasWordFrontBaseInPast)
	assert.Equal(t, 129_007, gasWordFrontQuoteInPast)
	assert.Equal(t, 145_654, gasWordDoorBaseInPast)
	assert.Equal(t, 144_523, gasWordDoorQuoteInPast)
	assert.Equal(t, 12_562, gasPrintBaseIn)
	assert.Equal(t, 12_806, gasPrintQuoteIn)
	assert.Equal(t, 120_205, gasGuardedFrontBaseIn)
	assert.Equal(t, 112_108, gasGuardedFrontQuoteIn)
	assert.Equal(t, 132_360, gasGuardedDoorBaseIn)
	assert.Equal(t, 127_678, gasGuardedDoorQuoteIn)
	// base in, quote in, by mode, print, guarded and past
	want := map[[4]bool][2]int{
		// {door, print, guarded, past}
		{false, false, false, false}: {101_664, 93_570},
		{false, true, false, false}:  {114_226, 106_376}, // measured on the PRINT listing too
		{false, false, false, true}:  {129_338, 129_007},
		{false, true, false, true}:   {141_900, 141_813}, // a sum
		{false, true, true, false}:   {120_205, 112_108}, // measured on the guarded PRINT listing
		{false, true, true, true}:    {147_626, 147_292}, // a sum
		{true, false, false, false}:  {114_072, 109_146},
		{true, true, false, false}:   {126_634, 121_952}, // measured on the PRINT listing too
		{true, false, false, true}:   {145_654, 144_523},
		{true, true, false, true}:    {158_216, 157_329}, // a sum
		{true, true, true, false}:    {132_360, 127_678}, // measured on the guarded PRINT listing
		{true, true, true, true}:     {163_689, 162_802}, // a sum
	}
	for k, w := range want {
		for _, base1 := range []bool{false, true} {
			p := simPool(t, simLadders(), time.Now().Unix())
			var e Extra
			require.NoError(t, json.Unmarshal([]byte(p.Extra), &e))
			e.BaseToken1, e.Print, e.Guarded, e.Past = base1, k[1], k[2], k[3]
			b, err := json.Marshal(e)
			require.NoError(t, err)
			p.Extra = string(b)
			mode := modeFront
			if k[0] {
				mode = modeDoor
			}
			p = withStatic(t, p, func(se *StaticExtra) { se.Mode = mode })
			sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
			require.NoError(t, err)
			token0, token1 := int64(w[0]), int64(w[1])
			if base1 {
				token0, token1 = token1, token0
			}
			res, err := calc(sim, simWETH, simUSDC, big.NewInt(1e18), nil)
			require.NoError(t, err)
			assert.Equal(t, token0, res.Gas, "token 0 in: door %v, print %v, guarded %v, past %v, base token 1 %v", k[0],
				k[1], k[2], k[3], base1)
			res, err = calc(sim, simUSDC, simWETH, big.NewInt(3000e6), nil)
			require.NoError(t, err)
			assert.Equal(t, token1, res.Gas, "token 1 in: door %v, print %v, guarded %v, past %v, base token 1 %v", k[0],
				k[1], k[2], k[3], base1)
		}
	}
}

// A pool whose static extra names a fill mode the adapter does not take is no pool.
func TestSimulatorRefusesAnUnknownFillMode(t *testing.T) {
	p := withStatic(t, simPool(t, simLadders(), time.Now().Unix()), func(se *StaticExtra) { se.Mode = 2 })
	_, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
	assert.ErrorIs(t, err, ErrInvalidState)
}

// Between two points the chord, floored; under the first point and past the last nothing.
func TestSimulatorChordAndEdges(t *testing.T) {
	sim := mustSim(t, simLadders())

	res, err := calc(sim, simWETH, simUSDC, bi("1500000000000000000"), nil)
	require.NoError(t, err)
	assert.Equal(t, "4498500000", res.TokenAmountOut.Amount.String(), "(3000e6 + 5997e6) / 2")

	res, err = calc(sim, simWETH, simUSDC, bi("1000000000000000001"), nil)
	require.NoError(t, err)
	assert.Equal(t, "3000000000", res.TokenAmountOut.Amount.String(), "floored")

	_, err = calc(sim, simWETH, simUSDC, big.NewInt(1e17-1), nil)
	assert.ErrorIs(t, err, ErrBelowFirstPoint)
	_, err = calc(sim, simWETH, simUSDC, bi("3000000000000000001"), nil)
	assert.ErrorIs(t, err, ladder.ErrAmountInTooLarge)
	_, err = calc(sim, simWETH, simUSDC, big.NewInt(0), nil)
	assert.ErrorIs(t, err, ladder.ErrZeroAmountIn)
	_, err = calc(sim, simWETH, "0x9999999999999999999999999999999999999999", big.NewInt(1e18), nil)
	assert.ErrorIs(t, err, ladder.ErrInvalidToken)
}

// A span steeper than the one before it (more than the floors explain) is not concave there: it quotes the point
// below it, as does the span before it, whose chord the steeper one could not vouch for.
func TestSimulatorQuotesThePointBelowAcrossAConvexSpan(t *testing.T) {
	ladders := [2][]Point{{pt(100, 100), pt(200, 190), pt(300, 300), pt(400, 390)}, nil}
	sim := mustSim(t, ladders)
	c := sim.curves[0]
	assert.Equal(t, []bool{false, false, false, true}, c.chord)

	for amount, want := range map[int64]int64{150: 100, 250: 190, 350: 345} {
		res, err := calc(sim, simWETH, simUSDC, big.NewInt(amount), nil)
		require.NoError(t, err)
		assert.Equal(t, want, res.TokenAmountOut.Amount.Int64(), "amount %d", amount)
	}
	_, err := calc(sim, simUSDC, simWETH, big.NewInt(100), nil)
	assert.ErrorIs(t, err, ladder.ErrNoQuote, "an empty side quotes nothing")
}

// The front's walk floors a concave output, so the simulator, sampled from it, never pays more than it at any
// size: checked at random sizes against fake listings of random concave levels.
func TestSimulatorNeverQuotesAboveTheDoor(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	checked := 0
	for trial := range 200 {
		side := randomSide(rng)
		var points []Point
		for _, x := range sampleSizes(rng, side) {
			out, err := side.quote(big.NewInt(x))
			if err != nil {
				continue
			}
			points = append(points, Point{uint256.NewInt(uint64(x)), uint256.MustFromBig(out)})
		}
		// a fake side refuses under its minimum and past its last level only, so its passing sizes are one run
		sim := mustSim(t, [2][]Point{points, nil})
		for range 200 {
			x := rng.Int63n(side.rungs[len(side.rungs)-1].upTo*2) + 1
			res, err := calc(sim, simWETH, simUSDC, big.NewInt(x), nil)
			if err != nil {
				continue
			}
			want, err := side.quote(big.NewInt(x))
			require.NoError(t, err, "trial %d: the simulator quotes %d where the door refuses", trial, x)
			require.LessOrEqual(t, res.TokenAmountOut.Amount.Cmp(want), 0,
				"trial %d: at %d the simulator pays %s, the door %s", trial, x, res.TokenAmountOut.Amount, want)
			checked++
		}
	}
	assert.Greater(t, checked, 10_000, "most random sizes fall inside the sampled range")
}

func randomSide(rng *rand.Rand) fakeSide {
	n := 1 + rng.Intn(6)
	var rungs []fakeRung
	upTo := int64(0)
	num := int64(1_000_000 + rng.Intn(5_000_000))
	for range n {
		upTo += 1 + rng.Int63n(1e9)
		rungs = append(rungs, fakeRung{upTo: upTo, num: num, den: 1_000_000 + rng.Int63n(1000)})
		num -= rng.Int63n(num / 10)
	}
	return fakeSide{rungs: rungs, minOut: rng.Int63n(1_000_000)}
}

func sampleSizes(rng *rand.Rand, s fakeSide) []int64 {
	top := s.rungs[len(s.rungs)-1].upTo
	var xs []int64
	x := int64(1 + rng.Intn(1000))
	for x < top*3 {
		xs = append(xs, x)
		x = x*2 + rng.Int63n(x+1)
	}
	return xs
}

func TestSimulatorTakesOneFillADirection(t *testing.T) {
	sim := mustSim(t, simLadders())
	res, err := calc(sim, simWETH, simUSDC, big.NewInt(1e18), nil)
	require.NoError(t, err)
	cloned := sim.CloneState().(*PoolSimulator)
	cloned.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: simWETH, Amount: big.NewInt(1e18)},
		TokenAmountOut: *res.TokenAmountOut,
	})

	_, err = calc(cloned, simWETH, simUSDC, big.NewInt(1e18), nil)
	assert.ErrorIs(t, err, ErrSecondFill)
	_, err = calc(cloned, simUSDC, simWETH, big.NewInt(3000e6), nil)
	assert.NoError(t, err, "the other side keeps its own book")
	_, err = calc(sim, simWETH, simUSDC, big.NewInt(1e18), nil)
	assert.NoError(t, err, "the original keeps its state")

	assert.Equal(t, "101000000000000000000", cloned.Info.Reserves[0].String())
	assert.Equal(t, "997000000000", cloned.Info.Reserves[1].String())
	assert.Equal(t, "100000000000000000000", sim.Info.Reserves[0].String())
}

// The payout comes from the listing's pool: no quote above its custody, and listings of one pool share the limit.
func TestSimulatorCustodyAndSharedLimit(t *testing.T) {
	p := simPool(t, simLadders(), time.Now().Unix())
	p.Reserves = entity.PoolReserves{"100000000000000000000", "4000000000"}
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
	require.NoError(t, err)
	_, err = calc(sim, simWETH, simUSDC, big.NewInt(2e18), nil)
	assert.ErrorIs(t, err, ladder.ErrInsufficientLiquidity)

	limits := sim.CalculateLimit()
	assert.Equal(t, map[string]*big.Int{
		simCustody + ":" + simWETH: bi("999000999000999000"), // the ladder's top, under the custody
		simCustody + ":" + simUSDC: big.NewInt(4000e6),       // the custody, under the ladder's top
		simCustody + ":route":      poolOpen,                 // no fill of the pool yet
	}, limits)

	inv := swaplimit.NewInventory(DexType, limits)
	res, err := calc(sim, simWETH, simUSDC, big.NewInt(1e18), inv)
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: simWETH, Amount: big.NewInt(1e18)},
		TokenAmountOut: *res.TokenAmountOut,
		SwapLimit:      inv,
	})
	assert.Equal(t, big.NewInt(1000e6), inv.GetLimit(simCustody+":"+simUSDC))

	// a second door on the same pool sees the custody the first spent
	other := simPool(t, simLadders(), time.Now().Unix())
	other.Address = "0x6666666666666666666666666666666666666666"
	sibling, err := NewPoolSimulator(pool.FactoryParams{EntityPool: other})
	require.NoError(t, err)
	_, err = calc(sibling, simWETH, simUSDC, big.NewInt(1e18), inv)
	assert.ErrorIs(t, err, pool.ErrNotEnoughInventory)
	_, err = calc(sibling, simWETH, simUSDC, big.NewInt(1e17), inv)
	assert.NoError(t, err)
}

// Route finding refuses a ladder older than maxAge; the indexer, with StaleCheck off, keeps pricing it.
func TestSimulatorStaleness(t *testing.T) {
	old := time.Now().Add(-maxAge - time.Second).Unix()
	_, err := NewPoolSimulator(pool.FactoryParams{EntityPool: simPool(t, simLadders(), old),
		Opts: pool.FactoryOpts{StaleCheck: true}})
	assert.ErrorIs(t, err, ladder.ErrStale)

	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: simPool(t, simLadders(), old)})
	require.NoError(t, err)
	_, err = calc(sim, simWETH, simUSDC, big.NewInt(1e18), nil)
	assert.NoError(t, err)
}

func TestSimulatorRefusesAMalformedLadder(t *testing.T) {
	for name, l := range map[string][]Point{
		"input not rising": {pt(100, 10), pt(100, 11)},
		"output falling":   {pt(100, 10), pt(200, 9)},
		"zero output":      {pt(100, 0)},
		"missing amount":   {{uint256.NewInt(1), nil}},
		"too wide":         {{new(uint256.Int).Lsh(uint256.NewInt(1), 130), uint256.NewInt(1)}},
	} {
		_, err := NewPoolSimulator(pool.FactoryParams{EntityPool: simPool(t, [2][]Point{l, nil}, time.Now().Unix())})
		assert.ErrorIs(t, err, ErrInvalidState, name)
	}
}

// The meta names what the adapter's data is built from, abi.encode(door, front, listing, mode), and what the adapter
// may approve: the door in the door's mode, the front in the front's.
func TestGetMetaInfo(t *testing.T) {
	sim := mustSim(t, simLadders())
	assert.Equal(t, PoolMeta{BlockNumber: 77, Door: simDoor, Front: simFront, Listing: 1, Mode: modeFront,
		ApprovalAddress: simFront, TipCapWei: 1e8}, sim.GetMetaInfo(simWETH, simUSDC))

	p := withStatic(t, simPool(t, simLadders(), time.Now().Unix()), func(se *StaticExtra) { se.Mode = modeDoor })
	door, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
	require.NoError(t, err)
	assert.Equal(t, PoolMeta{BlockNumber: 77, Door: simDoor, Front: simFront, Listing: 1, Mode: modeDoor,
		ApprovalAddress: simDoor, TipCapWei: 1e8}, door.GetMetaInfo(simWETH, simUSDC))

	// the mode is always written, the door's 0 included, so the consumer never reads a missing field as a mode
	b, err := json.Marshal(door.GetMetaInfo(simWETH, simUSDC))
	require.NoError(t, err)
	assert.Contains(t, string(b), `"mode":0`)
	assert.Contains(t, string(b), `"listing":1`)
}

// The simulator is serialized field by field, unexported ones included, as pool-service ships it.
func TestSimulatorMsgpackRoundTrip(t *testing.T) {
	sim := mustSim(t, simLadders())
	var buf bytes.Buffer
	en := msgpack.NewEncoder(&buf)
	en.IncludeUnexported(true)
	en.SetForceAsArray(true)
	require.NoError(t, en.Encode(sim))

	var decoded PoolSimulator
	de := msgpack.NewDecoder(&buf)
	de.IncludeUnexported(true)
	de.SetForceAsArray(true)
	require.NoError(t, de.Decode(&decoded))
	for _, x := range []string{"100000000000000000", "1500000000000000000", "3000000000000000000"} {
		want, err := calc(sim, simWETH, simUSDC, bi(x), nil)
		require.NoError(t, err)
		got, err := calc(&decoded, simWETH, simUSDC, bi(x), nil)
		require.NoError(t, err)
		assert.Equal(t, want.TokenAmountOut.Amount, got.TokenAmountOut.Amount)
	}
}

// Route finding quotes nothing past the last second the sampled word and depth certainly stand; the indexer, with
// StaleCheck off, keeps pricing the ladder.
func TestSimulatorValidUntil(t *testing.T) {
	now := time.Now().Unix()
	p := simPool(t, simLadders(), now)
	var e Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &e))
	e.ValidUntil = uint64(now) - 1
	b, err := json.Marshal(e)
	require.NoError(t, err)
	p.Extra = string(b)

	_, err = NewPoolSimulator(pool.FactoryParams{EntityPool: p, Opts: pool.FactoryOpts{StaleCheck: true}})
	assert.ErrorIs(t, err, ladder.ErrStale, "built after the word's end")

	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
	require.NoError(t, err)
	_, err = calc(sim, simWETH, simUSDC, big.NewInt(1e18), nil)
	assert.NoError(t, err, "no StaleCheck")

	// built in time, quoted after the word's end
	sim = mustSim(t, simLadders())
	assert.True(t, sim.validAt(sim.validUntil))
	assert.False(t, sim.validAt(sim.validUntil+1))
	sim.validUntil = uint64(now) - 1
	_, err = calc(sim, simWETH, simUSDC, big.NewInt(1e18), nil)
	assert.ErrorIs(t, err, ladder.ErrStale)

	// a listing with no ladder has no word to bound it, and quotes nothing either way
	_, err = NewPoolSimulator(pool.FactoryParams{EntityPool: simPool(t, [2][]Point{}, now),
		Opts: pool.FactoryOpts{StaleCheck: true}})
	assert.NoError(t, err)
}

// A fill lands some seconds after its route is built: route finding quotes only where the fill, landing within the
// state's allowance, still finds the word standing.
func TestSimulatorInclusionAllowance(t *testing.T) {
	now := uint64(time.Now().Unix())
	build := func(validUntil, allowance uint64, staleCheck bool) (*PoolSimulator, error) {
		p := simPool(t, simLadders(), int64(now))
		var e Extra
		require.NoError(t, json.Unmarshal([]byte(p.Extra), &e))
		e.ValidUntil, e.Allowance = validUntil, allowance
		b, err := json.Marshal(e)
		require.NoError(t, err)
		p.Extra = string(b)
		return NewPoolSimulator(pool.FactoryParams{EntityPool: p, Opts: pool.FactoryOpts{StaleCheck: staleCheck}})
	}
	_, err := build(now+2, 3, true)
	assert.ErrorIs(t, err, ladder.ErrStale, "a fill landing in 3 s finds the word gone")

	sim, err := build(now+5, 3, true)
	require.NoError(t, err)
	_, err = calc(sim, simWETH, simUSDC, big.NewInt(1e18), nil)
	assert.NoError(t, err)
	assert.True(t, sim.validAt(now+2))
	assert.False(t, sim.validAt(now+3), "lands at now+6, past the word")

	sim, err = build(now+2, 3, false)
	require.NoError(t, err, "no StaleCheck")
	_, err = calc(sim, simWETH, simUSDC, big.NewInt(1e18), nil)
	assert.NoError(t, err)
}

// The pool's room for a token, which its listings share, may be under its balance (an outflow bucket, a cap, a fee
// owed): the shared limit is no more than what a ladder's last point proves the pool pays out, the larger of its
// listings' (max merge), so a route split across two listings of one pool never takes more than one could fill.
func TestSimulatorLimitIsWhatTheLadderProves(t *testing.T) {
	sim := mustSim(t, simLadders())
	assert.Equal(t, map[string]*big.Int{
		simCustody + ":" + simWETH: bi("999000999000999000"), // buying WETH: the ladder's top output
		simCustody + ":" + simUSDC: big.NewInt(8991e6),       // selling WETH
		simCustody + ":route":      poolOpen,
	}, sim.CalculateLimit())

	// a direction with no ladder proves nothing
	sim = mustSim(t, [2][]Point{simLadders()[0], nil})
	assert.Zero(t, sim.CalculateLimit()[simCustody+":"+simWETH].Sign())

	// two listings of one pool: the route takes no more of USDC than the larger ladder's top in all
	small := mustSim(t, [2][]Point{{pt(1e17, 300e6), pt(1e18, 3000e6)}, nil})
	large := mustSim(t, simLadders())
	limits := small.CalculateLimit()
	for k, v := range large.CalculateLimit() {
		if limits[k] == nil || v.Cmp(limits[k]) > 0 {
			limits[k] = v
		}
	}
	inv := swaplimit.NewInventory(DexType, limits)
	res, err := calc(large, simWETH, simUSDC, big.NewInt(3e18), inv)
	require.NoError(t, err)
	large.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: simWETH, Amount: big.NewInt(3e18)},
		TokenAmountOut: *res.TokenAmountOut,
		SwapLimit:      inv,
	})
	_, err = calc(small, simWETH, simUSDC, big.NewInt(1e17), inv)
	assert.ErrorIs(t, err, pool.ErrNotEnoughInventory, "the larger ladder's top is spent")
}

// A limit with no entry for the pool's token is no room: nothing is quoted.
func TestSimulatorRefusesAMissingLimit(t *testing.T) {
	sim := mustSim(t, simLadders())
	inv := swaplimit.NewInventory(DexType, map[string]*big.Int{simCustody + ":" + simWETH: bi("1000000000000000000")})
	_, err := calc(sim, simWETH, simUSDC, big.NewInt(1e18), inv)
	assert.ErrorIs(t, err, pool.ErrNotEnoughInventory)
}

// Two listings of one pool that holds none of a token share its first write: in a route, the first fill with that
// token in is charged the write from zero, and a later fill of the pool's other listing with the same token in is not.
// With no shared limit, or on another pool, each is charged its own.
func TestFirstWriteOfAnInputIsChargedOnceAcrossAPoolsListings(t *testing.T) {
	listing := func(address, custody string) *PoolSimulator {
		p := simPool(t, simLadders(), time.Now().Unix())
		var e Extra
		require.NoError(t, json.Unmarshal([]byte(p.Extra), &e))
		e.NoneHeld = 1 // the pool holds none of token 0
		b, err := json.Marshal(e)
		require.NoError(t, err)
		p.Extra, p.Address = string(b), address
		if custody != "" {
			p = withStatic(t, p, func(se *StaticExtra) { se.Custody = custody })
		}
		sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
		require.NoError(t, err)
		return sim
	}
	size := big.NewInt(1e17)
	gasOf := func(sim *PoolSimulator, limit pool.SwapLimit) int64 {
		res, err := calc(sim, simWETH, simUSDC, size, limit)
		require.NoError(t, err)
		return res.Gas
	}
	a := listing("0x7777777777777777777777777777777777777777", "")
	b := listing("0x8888888888888888888888888888888888888888", "")
	other := listing("0x9999999999999999999999999999999999999999", "0x00000000000000000000000000000000000000cc")
	plain := gasOf(a, nil) - gasFirstOfInput
	require.Positive(t, plain)

	inv := sharedLimit(a, b, other)
	assert.Equal(t, big.NewInt(1), inv.GetLimit(a.firstKey(0)), "the first write is open")
	assert.Nil(t, inv.GetLimit(a.firstKey(1)), "no key for a token the pool holds")
	assert.Equal(t, plain+gasFirstOfInput, gasOf(a, inv))
	assert.Equal(t, plain+gasFirstOfInput, gasOf(b, inv), "no fill yet: either listing's would be the first")
	fillSim(t, a, simWETH, simUSDC, size, inv)
	assert.Equal(t, plain, gasOf(b, inv), "the pool's other listing: listing a's fill paid the pool the token")
	assert.Equal(t, plain+gasFirstOfInput, gasOf(other, inv), "another pool's first write is its own")
	assert.Equal(t, plain+gasFirstOfInput, gasOf(b, nil), "with no shared limit a listing knows its own fills alone")
}
