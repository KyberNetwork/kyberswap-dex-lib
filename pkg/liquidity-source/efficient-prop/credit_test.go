package efficientprop

import (
	"bytes"
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/KyberNetwork/msgpack/v5"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
)

// testCredit is listing 1's configured gas: a custody fill and a route fill, the base (WETH) in and the quote in.
var testCredit = CreditListing{Listing: 1, CustodyBaseIn: 171_000, CustodyQuoteIn: 166_000, RouteBaseIn: 610_501,
	RouteQuoteIn: 187_000}

// planSelector is the tip-taking quoteFrom's.
var planSelector = hexutil.Encode(frontABI.Methods["quoteFrom"].ID)

// creditChain is the tracked listing on Ethereum with credit mode 2, its WETH-in side routed past 2 WETH (custody
// fills up to there) and its pool holding 7,000 USDC, under what the side's top pays.
func creditChain() *fakeChain {
	chain := ethChain()
	l := chain.listings[1]
	l.view.CreditMode = 2
	l.route[0] = &fakeRoute{custodyTo: 2e18}
	chain.balances[[2]common.Address{testUSDC, testPoolA}] = big.NewInt(7_000e6)
	return chain
}

// creditTrack refreshes listing 1 of chain under a /state answer for block 1235 (carry: its word carries), its
// front's config naming the credit listings given, and answers the state, the node and the target block's second.
// setCredit names cfg's front's credit listings, and drops the listing gas of each: a credit listing's figures are its
// credit entry's alone.
func setCredit(cfg *Config, credit ...CreditListing) {
	f := &cfg.Fronts[0]
	f.CreditListings = credit
	var kept []ListingGas
	for _, g := range f.ListingGas {
		if f.creditListing(g.Listing) == nil {
			kept = append(kept, g)
		}
	}
	f.ListingGas = kept
}

func creditTrack(t *testing.T, chain *fakeChain, carry bool, credit ...CreditListing) (entity.Pool, *node, uint64) {
	t.Helper()
	n := builderNode(t, chain, 1234)
	st := newStateServer(t)
	slot := slotAfter(n.time)
	st.set(slot, 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, carry)})
	cfg := ethConfig(st.URL)
	setCredit(cfg, credit...)
	p := ethListed(t, n, cfg)
	return ethTrack(t, n, cfg, p), n, slotTime(slot)
}

// A listing with a credit mode is quoted only where its front's config names it with its gas. Each sampled size is
// classed by the front's plan, asked beside the door's quote in the target block (PLAN_ONLY, the door the taker): the
// ladder pays the door's answer at every point, custody points first, then routed ones, and the step between is
// marked. The ladders ride in the state's credit data alone, so a simulator that predates it quotes nothing, and the
// pool's reserves are its balances: a route pays past them.
func TestCreditListingIsQuotedWithEachSizesClass(t *testing.T) {
	chain := creditChain()
	got, n, target := creditTrack(t, chain, false, testCredit)
	e := extraOf(t, got)
	require.NotNil(t, e.Credit)
	assert.Equal(t, uint8(2), e.CreditMode)
	assert.Empty(t, e.Ladders[0], "no ladder where a simulator that predates credit data reads one")
	assert.Empty(t, e.Ladders[1])
	l := chain.listings[1]
	bid := e.Credit.Ladders[0]
	require.NotEmpty(t, bid)
	from := e.Credit.From[0]
	require.Greater(t, from, 0, "custody points first")
	require.Less(t, from, len(bid), "then routed points")
	for k, pt := range bid {
		want, err := chain.quoteAt(l, 0, pt[0].ToBig(), target, 1235)
		require.NoError(t, err)
		assert.Equal(t, want, pt[1].ToBig(), "the door's answer at %s", pt[0])
		assert.Equal(t, k >= from, pt[0].ToBig().Cmp(big.NewInt(2e18)) > 0, "the plan's class at %s", pt[0])
	}
	require.NotEmpty(t, e.Credit.Ladders[1])
	assert.Equal(t, len(e.Credit.Ladders[1]), e.Credit.From[1], "the side with no route has no routed point")
	assert.Equal(t, [2][2]int64{{171_000, 610_501}, {166_000, 187_000}}, e.Credit.Gas, "WETH, the base, is token 0")
	top := bid[len(bid)-1][1].ToBig()
	assert.Greater(t, top.Cmp(big.NewInt(7_000e6)), 0, "the route pays past the pool's balance")
	assert.Equal(t, entity.PoolReserves{"9000000000000000000", "7000000000"}, got.Reserves, "the pool's balances")

	// every plan read ran in the target block, at PLAN_ONLY and no tip, the door the taker
	plans := 0
	door := common.HexToAddress(got.Address)
	for _, c := range n.calls() {
		if len(c.data) < 4 || hexutil.Encode(c.data[:4]) != planSelector {
			continue
		}
		plans++
		assert.Equal(t, chain.front, c.to)
		assert.Equal(t, uint64(1235), c.number)
		args, err := frontABI.Methods["quoteFrom"].Inputs.Unpack(c.data[4:])
		require.NoError(t, err)
		assert.Equal(t, planOnly, args[4].(*big.Int))
		assert.Equal(t, door, args[5].(common.Address))
		assert.Equal(t, int64(1), args[3].(*big.Int).Int64())
	}
	assert.Equal(t, len(quotesAfter(n, 0))-countQuotesAtHead(n), plans, "one plan read beside each quote")

	// the simulator charges each class its gas, and quotes nothing across the step
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: got})
	require.NoError(t, err)
	weth, usdc := hexutil.Encode(testWETH[:]), hexutil.Encode(testUSDC[:])
	for k, pt := range bid {
		res, err := calc(sim, weth, usdc, pt[0].ToBig(), nil)
		require.NoError(t, err)
		assert.Equal(t, pt[1].ToBig(), res.TokenAmountOut.Amount)
		want := int64(171_000)
		if k >= from {
			want = 610_501
		}
		assert.Equal(t, want, res.Gas, "the class of %s", pt[0])
	}
	between := new(big.Int).Add(bid[from-1][0].ToBig(), big.NewInt(1))
	_, err = calc(sim, weth, usdc, between, nil)
	assert.ErrorIs(t, err, ErrCreditStep)
	res, err := calc(sim, usdc, weth, e.Credit.Ladders[1][0][0].ToBig(), nil)
	require.NoError(t, err)
	assert.Equal(t, int64(166_000), res.Gas, "the quote in, inside the custody")
	limits := sim.CalculateLimit()
	assert.Equal(t, bid[from-1][1].ToBig(), limits[sim.limitKey(1)], "the shared limit is the custody's last point")
	assert.Equal(t, poolOpen, limits[sim.poolKey()])

	// a simulator that reads only the ladders a state held before credit data quotes nothing
	var old struct {
		Ladders [2][]Point `json:"l"`
	}
	require.NoError(t, json.Unmarshal([]byte(got.Extra), &old))
	assert.Empty(t, old.Ladders[0])
	assert.Empty(t, old.Ladders[1])
}

// countQuotesAtHead is the door quotes the node answered outside the target block (none on a builder chain's refresh).
func countQuotesAtHead(n *node) int {
	k := 0
	for _, c := range quotesAfter(n, 0) {
		if c.number != 1235 && c.number != 1236 {
			k++
		}
	}
	return k
}

// Without its front's config naming it, a listing with a credit mode is not quoted, nor its plan asked.
func TestCreditListingNotNamedIsNotQuoted(t *testing.T) {
	chain := creditChain()
	got, n, _ := creditTrack(t, chain, false)
	e := extraOf(t, got)
	assert.Empty(t, e.Ladders[0])
	assert.Empty(t, e.Ladders[1])
	assert.Nil(t, e.Credit)
	for _, c := range n.calls() {
		assert.NotEqual(t, planSelector, hexutil.Encode(c.data[:min(4, len(c.data))]))
	}
	_, err := NewPoolSimulator(pool.FactoryParams{EntityPool: got})
	require.NoError(t, err)

	// another listing named is not this one
	other := testCredit
	other.Listing = 2
	got, _, _ = creditTrack(t, chain, false, other)
	assert.Empty(t, extraOf(t, got).Ladders[0])
}

// A route that pays less than the custody's last point (its pool's fee) ends the ladder there: no routed point is
// quoted, and the side is charged the custody's gas alone. A routed size that pays more than that point despite the
// fee is a point, and the step below it is still not quoted: no chord crosses a fee.
func TestCreditRouteThatPaysLessEndsTheLadder(t *testing.T) {
	chain := creditChain()
	chain.listings[1].route[0].feePpm = 700_000
	got, _, _ := creditTrack(t, chain, false, testCredit)
	e := extraOf(t, got)
	require.NotEmpty(t, e.Credit.Ladders[0])
	assert.Equal(t, len(e.Credit.Ladders[0]), e.Credit.From[0])
	for _, pt := range e.Credit.Ladders[0] {
		assert.LessOrEqual(t, pt[0].ToBig().Cmp(big.NewInt(2e18)), 0)
	}
	assert.Equal(t, "7000000000", got.Reserves[1], "the pool's balance")

	chain.listings[1].route[0].feePpm = 500_000
	got, _, target := creditTrack(t, chain, false, testCredit)
	e = extraOf(t, got)
	bid, from := e.Credit.Ladders[0], e.Credit.From[0]
	require.Less(t, from, len(bid), "the top pays more than the custody's last point")
	require.Positive(t, from)
	for _, pt := range bid[from:] {
		want, err := chain.quoteAt(chain.listings[1], 0, pt[0].ToBig(), target, 1235)
		require.NoError(t, err)
		assert.Equal(t, want, pt[1].ToBig(), "the route's own answer, its fee taken")
	}
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: got})
	require.NoError(t, err)
	mid := new(big.Int).Add(bid[from-1][0].ToBig(), bid[from][0].ToBig())
	_, err = calc(sim, hexutil.Encode(testWETH[:]), hexutil.Encode(testUSDC[:]), mid.Rsh(mid, 1), nil)
	assert.ErrorIs(t, err, ErrCreditStep)
}

// A size whose plan answer is neither the route's tag nor the door's own output is no point, and a side whose plan is
// not read has no ladder.
func TestCreditPlanThatDisagreesIsNoPoint(t *testing.T) {
	chain := creditChain()
	chain.listings[1].route[0].planWrong = true
	got, _, _ := creditTrack(t, chain, false, testCredit)
	e := extraOf(t, got)
	require.NotEmpty(t, e.Credit.Ladders[0], "the routed sizes still are points")
	assert.Zero(t, e.Credit.From[0], "every point routed: no custody size agreed")
	for _, pt := range e.Credit.Ladders[0] {
		assert.Positive(t, pt[0].ToBig().Cmp(big.NewInt(2e18)))
	}
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: got})
	require.NoError(t, err)
	res, err := calc(sim, hexutil.Encode(testWETH[:]), hexutil.Encode(testUSDC[:]), e.Credit.Ladders[0][0][0].ToBig(),
		nil)
	require.NoError(t, err)
	assert.Equal(t, int64(610_501), res.Gas, "the route's gas from the first point")

	chain.listings[1].route[0].planWrong, chain.listings[1].route[0].planReverts = false, true
	got, _, _ = creditTrack(t, chain, false, testCredit)
	e = extraOf(t, got)
	assert.Empty(t, e.Credit.Ladders[0])
	assert.NotEmpty(t, e.Credit.Ladders[1], "the other side's plan is read")
}

// Where the word carries, each size's class must be the same in the target block and the next: a size the front fills
// inside custody in one and routes in the other is no point, which ends the run there.
func TestCreditClassThatFlipsInTheCarriedBlockIsNoPoint(t *testing.T) {
	chain := creditChain()
	chain.listings[1].view.Scopes = 2 | 8
	r := chain.listings[1].route[0]
	r.laterAt, r.laterTo = 1236, 1e18
	got, n, _ := creditTrack(t, chain, true, testCredit)
	carried := 0
	for _, c := range n.calls() {
		if c.number == 1236 && len(c.data) >= 4 && hexutil.Encode(c.data[:4]) == planSelector {
			carried++
		}
	}
	require.Positive(t, carried, "the plan is asked in the carried block too")
	e := extraOf(t, got)
	require.NotEmpty(t, e.Credit.Ladders[0])
	for _, pt := range e.Credit.Ladders[0] {
		assert.LessOrEqual(t, pt[0].ToBig().Cmp(big.NewInt(1e18)), 0, "no size whose class flips, nor past it")
	}
	assert.Equal(t, len(e.Credit.Ladders[0]), e.Credit.From[0])

	// a route from the custody's cut in both blocks keeps its routed sizes
	r.laterAt = 0
	got, _, _ = creditTrack(t, chain, true, testCredit)
	e = extraOf(t, got)
	assert.Less(t, e.Credit.From[0], len(e.Credit.Ladders[0]))
}

// A credit listing is named once, with both custody figures measured, on a builder chain and on a chain read at the
// head (Base); a route figure is measured or 0, a side whose route was never measured.
func TestCreditListingsConfig(t *testing.T) {
	eth := func(cl ...CreditListing) Config {
		c := *ethConfig("http://state")
		setCredit(&c, cl...)
		return c
	}
	good := eth(testCredit)
	require.NoError(t, good.validate())
	assert.Equal(t, &good.Fronts[0].CreditListings[0], good.Fronts[0].creditListing(1))
	assert.Nil(t, good.Fronts[0].creditListing(2))

	noRoute := testCredit
	noRoute.RouteBaseIn, noRoute.RouteQuoteIn = 0, 0
	unmeasured := eth(noRoute)
	require.NoError(t, unmeasured.validate(), "a route never measured")
	// a route's gas limit, where stated, covers its side's charged figure: the dearer venue's fill, never under it
	limited := testCredit
	limited.RouteLimitBaseIn = 1_400_000
	withLimit := eth(limited)
	require.NoError(t, withLimit.validate(), "a route limit above the charged figure")
	lowLimit, noFigure, hugeLimit := limited, noRoute, limited
	lowLimit.RouteLimitBaseIn, noFigure.RouteLimitBaseIn = testCredit.RouteBaseIn-1, 1_400_000
	hugeLimit.RouteLimitBaseIn = maxCreditGasLimit + 1
	zero, huge, negative := testCredit, testCredit, testCredit
	zero.CustodyBaseIn, huge.CustodyQuoteIn, negative.RouteQuoteIn = 0, maxCreditGas+1, -1
	noID := testCredit
	noID.Listing = 0
	wide := testCredit
	wide.Listing = 1 << 24
	base := *testConfig()
	base.Fronts[0].CreditListings = []CreditListing{testCredit}
	require.NoError(t, base.validate(), "on Base")
	baseZero := *testConfig()
	baseZero.Fronts[0].CreditListings = []CreditListing{zero}
	for name, bad := range map[string]Config{
		"a figure missing":             eth(zero),
		"a figure too high":            eth(huge),
		"a negative figure":            eth(negative),
		"a limit under its figure":     eth(lowLimit),
		"a limit with no route figure": eth(noFigure),
		"a limit past the bound":       eth(hugeLimit),
		"no listing":                   eth(noID),
		"past 24 bits":                 eth(wide),
		"named twice":                  eth(testCredit, testCredit),
		"a figure missing on Base":     baseZero,
	} {
		assert.ErrorIs(t, bad.validate(), ErrInvalidConfig, name)
	}
}

// creditPool is a builder state of the simulator's ladders with credit data: the WETH-in side routed from point
// from, each class's gas.
func creditPool(t *testing.T, from int, edit func(*Extra)) entity.Pool {
	t.Helper()
	p := simPool(t, simLadders(), time.Now().Unix())
	var e Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &e))
	e.Builder, e.CreditMode = true, 2
	e.Credit = &Credit{Ladders: e.Ladders, From: [2]int{from, len(e.Ladders[1])},
		Gas: [2][2]int64{{171_000, 610_501}, {166_000, 187_000}}}
	e.Ladders = [2][]Point{}
	if edit != nil {
		edit(&e)
	}
	b, err := json.Marshal(e)
	require.NoError(t, err)
	p.Extra = string(b)
	return withStatic(t, p, func(se *StaticExtra) { se.Mode = modeDoor })
}

// The simulator takes a credit state, from a builder front or one read at the head, with each custody figure set, a
// route figure on every side that holds a routed point, each step inside its ladder, and no ladder of a listing with a
// credit mode without one. A fill in either direction ends both: a route moves the pool's debt, which both sides'
// quotes read.
func TestSimulatorCreditState(t *testing.T) {
	_, err := NewPoolSimulator(pool.FactoryParams{EntityPool: creditPool(t, 2, func(e *Extra) {
		e.Builder, e.ValidUntilMs, e.AllowanceMs, e.Allowance = false, 0, 0, 1
	})})
	require.NoError(t, err, "a front read at the head")
	for name, edit := range map[string]func(*Extra){
		"no credit mode":           func(e *Extra) { e.CreditMode = 0 },
		"a custody figure missing": func(e *Extra) { e.Credit.Gas[1][0] = 0 },
		"a routed side's figure 0": func(e *Extra) { e.Credit.Gas[0][1] = 0 },
		"a negative route figure":  func(e *Extra) { e.Credit.Gas[1][1] = -1 },
		"a step past the end":      func(e *Extra) { e.Credit.From[0] = len(e.Credit.Ladders[0]) + 1 },
		"ladders in both":          func(e *Extra) { e.Ladders = e.Credit.Ladders },
		"a negative step":          func(e *Extra) { e.Credit.From[1] = -1 },
		"no credit data":           func(e *Extra) { e.Ladders, e.Credit = e.Credit.Ladders, nil },
	} {
		_, err := NewPoolSimulator(pool.FactoryParams{EntityPool: creditPool(t, 2, edit)})
		assert.ErrorIs(t, err, ErrInvalidState, name)
	}

	// a side that holds no routed point needs no route figure
	_, err = NewPoolSimulator(pool.FactoryParams{EntityPool: creditPool(t, 2,
		func(e *Extra) { e.Credit.Gas[1][1] = 0 })})
	require.NoError(t, err)

	// routed from the first point: every size the route's gas
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: creditPool(t, 0, nil)})
	require.NoError(t, err)
	for _, amount := range []int64{1e17, 5e17, 1e18, 3e18} {
		res, err := calc(sim, simWETH, simUSDC, big.NewInt(amount), nil)
		require.NoError(t, err)
		assert.Equal(t, int64(610_501), res.Gas)
	}

	// routed from point 2 (2 WETH): a chord inside the custody, nothing between 1 and 2 WETH, the route's gas from 2
	sim, err = NewPoolSimulator(pool.FactoryParams{EntityPool: creditPool(t, 2, nil)})
	require.NoError(t, err)
	res, err := calc(sim, simWETH, simUSDC, big.NewInt(5e17), nil)
	require.NoError(t, err)
	assert.Equal(t, int64(171_000), res.Gas)
	_, err = calc(sim, simWETH, simUSDC, big.NewInt(15e17), nil)
	assert.ErrorIs(t, err, ErrCreditStep)
	res, err = calc(sim, simWETH, simUSDC, big.NewInt(2e18), nil)
	require.NoError(t, err)
	assert.Equal(t, int64(610_501), res.Gas)
	res, err = calc(sim, simWETH, simUSDC, big.NewInt(25e17), nil)
	require.NoError(t, err)
	assert.Equal(t, int64(610_501), res.Gas)

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: simWETH, Amount: big.NewInt(2e18)},
		TokenAmountOut: pool.TokenAmount{Token: simUSDC, Amount: res.TokenAmountOut.Amount},
	})
	_, err = calc(sim, simUSDC, simWETH, big.NewInt(300e6), nil)
	assert.ErrorIs(t, err, ErrSecondFill, "the other side after a route")

	// the credit state survives the simulator's serialization
	sim, err = NewPoolSimulator(pool.FactoryParams{EntityPool: creditPool(t, 2, nil)})
	require.NoError(t, err)
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
	for _, amount := range []int64{5e17, 2e18, 25e17} {
		want, err := calc(sim, simWETH, simUSDC, big.NewInt(amount), nil)
		require.NoError(t, err)
		got, err := calc(&decoded, simWETH, simUSDC, big.NewInt(amount), nil)
		require.NoError(t, err)
		assert.Equal(t, want.Gas, got.Gas)
		assert.Equal(t, want.TokenAmountOut.Amount, got.TokenAmountOut.Amount)
	}
	_, err = calc(&decoded, simWETH, simUSDC, big.NewInt(15e17), nil)
	assert.ErrorIs(t, err, ErrCreditStep)
}

// collectClassedRun keeps custody points, then routed points, ending at a custody point after a routed one.
func TestCollectClassedRun(t *testing.T) {
	sizes := []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3), big.NewInt(4), big.NewInt(5)}
	outs := []*big.Int{big.NewInt(10), big.NewInt(20), big.NewInt(30), big.NewInt(40), big.NewInt(50)}
	ok := []bool{true, true, true, true, true}
	run, from := collectClassedRun(sizes, outs, ok, []bool{false, false, true, false, true})
	assert.Len(t, run, 3, "a custody point after a routed one ends the run")
	assert.Equal(t, 2, from)
	run, from = collectClassedRun(sizes, outs, ok, nil)
	assert.Len(t, run, 5)
	assert.Equal(t, 5, from)
	run, from = collectClassedRun(sizes, outs, ok, []bool{true, true, true, true, true})
	assert.Len(t, run, 5)
	assert.Zero(t, from)
	falling := []*big.Int{big.NewInt(10), big.NewInt(20), big.NewInt(15), big.NewInt(40), big.NewInt(50)}
	run, from = collectClassedRun(sizes, falling, ok, []bool{false, false, true, true, true})
	assert.Len(t, run, 2, "a route that pays less than the custody ends the run")
	assert.Equal(t, 2, from)

	// classify: tagged and nonzero routes; untagged and equal is custody; anything else no point
	tag := func(v int64) *big.Int { return new(big.Int).Or(big.NewInt(v), routedTagTest) }
	classed, routed := classify([2][]*big.Int{{big.NewInt(10), big.NewInt(20), big.NewInt(30), big.NewInt(40), nil}},
		[2][]*big.Int{{big.NewInt(10), tag(25), big.NewInt(31), tag(0), big.NewInt(50)}})
	assert.Equal(t, []*big.Int{big.NewInt(10), big.NewInt(20), nil, nil, nil}, classed[0])
	assert.Equal(t, []bool{false, true, false, false, false}, routed[0])
}

// sharedLimit is the route's shared limits of sims, merged as route finding merges them.
func sharedLimit(sims ...*PoolSimulator) *swaplimit.Inventory {
	limits := map[string]*big.Int{}
	for _, sim := range sims {
		pool.MergeSwapLimits(limits, sim)
	}
	return swaplimit.NewInventory(DexType, limits)
}

// fillSim applies a quoted fill of sim to its state and the shared limit, as route finding does.
func fillSim(t *testing.T, sim *PoolSimulator, tokenIn, tokenOut string, amountIn *big.Int, inv pool.SwapLimit) {
	t.Helper()
	res, err := calc(sim, tokenIn, tokenOut, amountIn, inv)
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn},
		TokenAmountOut: *res.TokenAmountOut, SwapInfo: res.SwapInfo, SwapLimit: inv})
}

// A credit route is quoted only on a pool no fill of the route has touched, past the pool's balance, and once taken
// closes the pool to every listing of it in that route: the route borrows, after which the pool's cheap fills of the
// borrowed token are refused and its mode-2 bids route. A credit listing's custody fills share the pool's limit with
// its other listings, as any listing's do: the custody's last point, never the route's reach.
func TestCreditRouteClosesItsPoolInTheRoute(t *testing.T) {
	// listing A routes from 2 WETH on a pool holding 7,000 USDC; listing B is a cheap listing of the same pool
	newA := func() *PoolSimulator {
		p := creditPool(t, 2, nil)
		p.Reserves = entity.PoolReserves{"100000000000000000000", "7000000000"}
		sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
		require.NoError(t, err)
		return sim
	}
	newB := func() *PoolSimulator {
		p := creditPool(t, 2, func(e *Extra) { e.CreditMode, e.Credit, e.Print = 0, nil, true; e.Ladders = simLadders() })
		p.Address = "0x6666666666666666666666666666666666666666"
		p.Reserves = entity.PoolReserves{"100000000000000000000", "7000000000"}
		sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
		require.NoError(t, err)
		return sim
	}
	a, b := newA(), newB()
	inv := sharedLimit(a, b)
	assert.Equal(t, big.NewInt(7_000e6), inv.GetLimit(a.limitKey(1)), "the custody, never the route's reach")
	assert.Equal(t, poolOpen, inv.GetLimit(a.poolKey()))

	// a route past the balance, on an untouched pool
	res, err := calc(a, simWETH, simUSDC, big.NewInt(3e18), inv)
	require.NoError(t, err)
	assert.Equal(t, big.NewInt(8991e6), res.TokenAmountOut.Amount, "past the 7,000 the pool holds")
	assert.Equal(t, SwapInfo{Routed: true}, res.SwapInfo)

	// B fills first: A's route is not quoted after it, and A's custody fills share what B left
	fillSim(t, b, simWETH, simUSDC, big.NewInt(2e18), inv)
	_, err = calc(a, simWETH, simUSDC, big.NewInt(3e18), inv)
	assert.ErrorIs(t, err, pool.ErrNotEnoughInventory, "no route on a pool the route touched")
	_, err = calc(a, simWETH, simUSDC, big.NewInt(1e18), inv)
	assert.ErrorIs(t, err, pool.ErrNotEnoughInventory, "1,003 USDC left of the custody")
	res, err = calc(a, simWETH, simUSDC, big.NewInt(1e17), inv)
	require.NoError(t, err)
	assert.Equal(t, int64(171_000), res.Gas)

	// A routes first: neither listing of the pool is quoted after it
	a, b = newA(), newB()
	inv = sharedLimit(a, b)
	fillSim(t, a, simWETH, simUSDC, big.NewInt(3e18), inv)
	assert.Zero(t, inv.GetLimit(a.poolKey()).Sign(), "the route closed the pool")
	_, err = calc(b, simWETH, simUSDC, big.NewInt(1e17), inv)
	assert.ErrorIs(t, err, pool.ErrNotEnoughInventory)
	_, err = calc(b, simUSDC, simWETH, big.NewInt(300e6), inv)
	assert.ErrorIs(t, err, pool.ErrNotEnoughInventory)

	// a fill whose class the route finder did not hand back is classed by its size
	a, b = newA(), newB()
	inv = sharedLimit(a, b)
	res, err = calc(a, simWETH, simUSDC, big.NewInt(3e18), inv)
	require.NoError(t, err)
	a.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: pool.TokenAmount{Token: simWETH, Amount: big.NewInt(3e18)},
		TokenAmountOut: *res.TokenAmountOut, SwapLimit: inv})
	assert.Zero(t, inv.GetLimit(a.poolKey()).Sign())

	// a limit without the pool's route key quotes no route, and the custody fills as before
	a = newA()
	bare := swaplimit.NewInventory(DexType, map[string]*big.Int{a.limitKey(0): bi("100000000000000000000"),
		a.limitKey(1): big.NewInt(7_000e6)})
	_, err = calc(a, simWETH, simUSDC, big.NewInt(3e18), bare)
	assert.ErrorIs(t, err, pool.ErrNotEnoughInventory)
	_, err = calc(a, simWETH, simUSDC, big.NewInt(1e18), bare)
	assert.NoError(t, err)
}

// Credit mode 3 routes every fill the pool's switches let it: a side whose every size routes is quoted from its first
// point at the route's gas, and proves no custody to the shared limit.
func TestCreditEverySizeRouted(t *testing.T) {
	chain := creditChain()
	chain.listings[1].view.CreditMode = 3
	chain.listings[1].route[0].custodyTo = 0
	got, _, _ := creditTrack(t, chain, false, testCredit)
	e := extraOf(t, got)
	require.NotEmpty(t, e.Credit.Ladders[0])
	assert.Zero(t, e.Credit.From[0])
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: got})
	require.NoError(t, err)
	for _, pt := range e.Credit.Ladders[0] {
		res, err := calc(sim, hexutil.Encode(testWETH[:]), hexutil.Encode(testUSDC[:]), pt[0].ToBig(), nil)
		require.NoError(t, err)
		assert.Equal(t, int64(610_501), res.Gas)
	}
	assert.Zero(t, sim.CalculateLimit()[sim.limitKey(1)].Sign(), "no custody point pays USDC")
}

// The front's plan holds a route to the output token's cap where the door's preview does not: past it the plan answers
// 0 while the door routes, and the size is no point, which ends the run there.
func TestCreditPlanHeldToTheOutCapIsNoPoint(t *testing.T) {
	chain := creditChain()
	got, _, _ := creditTrack(t, chain, false, testCredit)
	e := extraOf(t, got)
	bid, from := e.Credit.Ladders[0], e.Credit.From[0]
	require.Less(t, from, len(bid), "the top routes")
	capOut := new(big.Int).Sub(bid[len(bid)-1][1].ToBig(), big.NewInt(1))
	chain.listings[1].route[0].planCapOut = capOut

	got, _, target := creditTrack(t, chain, false, testCredit)
	e = extraOf(t, got)
	capped := e.Credit.Ladders[0]
	require.Len(t, capped, len(bid)-1, "the top, past the plan's cap, is no point")
	assert.Equal(t, min(from, len(capped)), e.Credit.From[0])
	for k, pt := range capped {
		assert.Equal(t, bid[k], pt)
		assert.LessOrEqual(t, pt[1].ToBig().Cmp(capOut), 0)
	}
	// the door still routes the top: the plan, not the door, refused it
	top, err := chain.quoteAt(chain.listings[1], 0, bid[len(bid)-1][0].ToBig(), target, 1235)
	require.NoError(t, err)
	assert.Positive(t, top.Sign())
}

// Where the carried block refuses every size of a side, the side keeps the target block's ladder and classes, valid
// to the target block's second, as any listing's does.
func TestCreditCarriedBlockThatRefusesKeepsTheTargetBlock(t *testing.T) {
	chain := creditChain()
	chain.listings[1].view.Scopes = 2 | 8
	n := builderNode(t, chain, 1234)
	slot := slotAfter(n.time)
	chain.listings[1].shrinkAt, chain.listings[1].shrinkTo = slotTime(slot)+12, 0
	st := newStateServer(t)
	st.set(slot, 1235, map[common.Address]*big.Int{testCellA: deskSlot0(1235, true)})
	cfg := ethConfig(st.URL)
	setCredit(cfg, testCredit)
	got := ethTrack(t, n, cfg, ethListed(t, n, cfg))
	e := extraOf(t, got)
	bid := e.Credit.Ladders[0]
	require.NotEmpty(t, bid)
	from := e.Credit.From[0]
	assert.Positive(t, from)
	assert.Less(t, from, len(bid))
	assert.Equal(t, slotTime(slot)-1, e.ValidUntil, "the target block alone")
	for k, pt := range bid {
		want, err := chain.quoteAt(chain.listings[1], 0, pt[0].ToBig(), slotTime(slot), 1235)
		require.NoError(t, err)
		assert.Equal(t, want, pt[1].ToBig())
		assert.Equal(t, k >= from, pt[0].ToBig().Cmp(big.NewInt(2e18)) > 0)
	}
}

// A side whose route has no measured figure (its config's route figure 0) is quoted inside its custody alone: its
// routed sizes are dropped from the ladder, so no route is quoted at a gas nobody measured, and the other side
// stands as it was.
func TestCreditRouteWithNoFigureIsNotQuoted(t *testing.T) {
	noRoute := testCredit
	noRoute.RouteBaseIn = 0
	got, _, _ := creditTrack(t, creditChain(), false, noRoute)
	e := extraOf(t, got)
	require.NotNil(t, e.Credit)
	bid := e.Credit.Ladders[0]
	require.NotEmpty(t, bid, "the custody points stand")
	assert.Equal(t, len(bid), e.Credit.From[0], "no routed point")
	for _, pt := range bid {
		assert.LessOrEqual(t, pt[0].ToBig().Cmp(big.NewInt(2e18)), 0, "inside the custody: %s", pt[0])
	}
	assert.Equal(t, [2][2]int64{{171_000, 0}, {166_000, 187_000}}, e.Credit.Gas)
	require.NotEmpty(t, e.Credit.Ladders[1])

	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: got})
	require.NoError(t, err)
	weth, usdc := hexutil.Encode(testWETH[:]), hexutil.Encode(testUSDC[:])
	res, err := calc(sim, weth, usdc, bid[0][0].ToBig(), nil)
	require.NoError(t, err)
	assert.Equal(t, int64(171_000), res.Gas)
	_, err = calc(sim, weth, usdc, big.NewInt(3e18), nil)
	assert.Error(t, err, "a size the route alone fills")

	// with the figure, the same refresh quotes the routed sizes
	with, _, _ := creditTrack(t, creditChain(), false, testCredit)
	assert.Greater(t, len(extraOf(t, with).Credit.Ladders[0]), len(bid))
}

// A pair that may borrow at more than one lending venue: the front's plan tells a route from a custody fill and nothing
// of the venue, so a route is charged the cheaper venue's fill (the configured route figure, never above a route's
// gas) and carries the dearer venue's gas limit beside it, where the config states one. A custody fill carries none.
func TestCreditRouteIsChargedTheCheaperVenueAndCarriesTheDearersLimit(t *testing.T) {
	cl := testCredit
	cl.RouteLimitBaseIn = 1_400_000
	c := creditOf(&cl, false, [2]int{2, 2}, simLadders())
	require.NotNil(t, c.Limit)
	assert.Equal(t, [2]int64{1_400_000, 0}, *c.Limit)
	assert.Equal(t, [2]int64{0, 1_400_000}, *creditOf(&cl, true, [2]int{2, 2}, simLadders()).Limit, "by the input")
	assert.Nil(t, creditOf(&testCredit, false, [2]int{2, 2}, simLadders()).Limit, "none stated")

	limit := [2]int64{1_400_000, 0}
	p := creditPool(t, 2, func(e *Extra) { e.Credit.Limit = &limit })
	p.Reserves = entity.PoolReserves{"100000000000000000000", "7000000000"}
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
	require.NoError(t, err)
	res, err := calc(sim, simWETH, simUSDC, big.NewInt(3e18), nil)
	require.NoError(t, err)
	assert.Equal(t, int64(610_501), res.Gas, "the charge is the configured route figure")
	assert.Equal(t, SwapInfo{Routed: true, GasLimit: 1_400_000}, res.SwapInfo)
	res, err = calc(sim, simWETH, simUSDC, big.NewInt(1e18), nil)
	require.NoError(t, err)
	assert.Equal(t, SwapInfo{}, res.SwapInfo, "a custody fill")

	low := [2]int64{610_500, 0}
	_, err = NewPoolSimulator(pool.FactoryParams{EntityPool: creditPool(t, 2, func(e *Extra) { e.Credit.Limit = &low })})
	assert.ErrorIs(t, err, ErrInvalidState, "a limit under the gas it covers")
}

// On a front read at the head (Base, a storage build) a listing with a credit mode is quoted as on a builder front
// where its front's config names it: each sampled size classed by the front's plan beside the door's quote, at the
// head and at every moment the window samples, the ladders in the state's credit data alone at the configured gas.
func TestCreditListingOnAFrontReadAtTheHead(t *testing.T) {
	chain := trackedChain()
	l := chain.listings[1]
	l.view.CreditMode = 2
	l.route[0] = &fakeRoute{custodyTo: 2e18}
	chain.balances[[2]common.Address{testUSDC, testPoolA}] = big.NewInt(7_000e6)
	n := chain.node(t, 900)
	p := listed(t, n)

	got := track(t, n, p)
	assert.Nil(t, extraOf(t, got).Credit, "not named: not quoted")
	assert.Empty(t, extraOf(t, got).Ladders[0])

	cfg := testConfig()
	cfg.Fronts[0].CreditListings = []CreditListing{testCredit}
	tr, err := NewPoolTracker(cfg, n.client())
	require.NoError(t, err)
	before := len(n.calls())
	got, err = tr.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	e := extraOf(t, got)
	assert.False(t, e.Builder)
	require.NotNil(t, e.Credit)
	assert.Empty(t, e.Ladders[0], "the ladders ride in the credit data alone")
	bid := e.Credit.Ladders[0]
	require.NotEmpty(t, bid)
	from := e.Credit.From[0]
	require.Greater(t, from, 0, "custody points first")
	require.Less(t, from, len(bid), "then routed points")
	for k, pt := range bid {
		assert.Equal(t, k >= from, pt[0].ToBig().Cmp(big.NewInt(2e18)) > 0, "the plan's class at %s", pt[0])
	}
	assert.Equal(t, [2][2]int64{{171_000, 610_501}, {166_000, 187_000}}, e.Credit.Gas)
	assert.NotZero(t, e.ValidUntil)
	plans := 0
	for _, c := range n.calls()[before:] {
		if len(c.data) >= 4 && hexutil.Encode(c.data[:4]) == planSelector {
			plans++
			assert.Equal(t, chain.front, c.to)
		}
	}
	assert.Positive(t, plans, "each size's class is asked of the front's plan")

	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: got})
	require.NoError(t, err)
	res, err := calc(sim, hexutil.Encode(testWETH[:]), hexutil.Encode(testUSDC[:]), bid[len(bid)-1][0].ToBig(), nil)
	require.NoError(t, err)
	assert.Equal(t, int64(610_501), res.Gas, "a routed size is charged the route's gas")
}
