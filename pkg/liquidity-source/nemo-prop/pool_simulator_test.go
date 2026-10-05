package nemoprop

import (
	"bytes"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/KyberNetwork/msgpack/v5"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func quote(sim pool.IPoolSimulator, dir int, amountIn *big.Int) (*big.Int, error) {
	res, err := quoteResult(sim, dir, amountIn, nil)
	if err != nil {
		return nil, err
	}
	return res.TokenAmountOut.Amount, nil
}

func quoteResult(sim pool.IPoolSimulator, dir int, amountIn *big.Int,
	limit pool.SwapLimit) (*pool.CalcAmountOutResult, error) {
	tokens := sim.GetTokens()
	return sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokens[dir], Amount: amountIn},
		TokenOut:      tokens[1-dir],
		Limit:         limit,
	})
}

func quote1000(sim pool.IPoolSimulator) int64 {
	out, err := quote(sim, 0, big.NewInt(1000))
	if err != nil {
		return -1
	}
	return out.Int64()
}

func eventuallyQuotes(t *testing.T, sim pool.IPoolSimulator, want int64) {
	t.Helper()
	require.Eventually(t, func() bool { return quote1000(sim) == want }, 5*time.Second, 2*time.Millisecond,
		"quote never became %d (last %d)", want, quote1000(sim))
}

func feedPool(t *testing.T, feed FeedSettings) entity.Pool {
	return testPool(t, feed, time.Now().Unix())
}

func newSim(t *testing.T, p entity.Pool) *PoolSimulator {
	t.Helper()
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
	require.NoError(t, err)
	return sim
}

// 1. A feed push changes the same simulator instance's next quote.
func TestFeed_PushChangesSameSimulatorQuote(t *testing.T) {
	t.Parallel()
	f := newFakeFeed(t, "key")
	sim := newSim(t, feedPool(t, f.settings()))
	_, err := quote(sim, 0, big.NewInt(1000))
	assert.ErrorIs(t, err, ErrNoFeed, "nothing to quote before the first snapshot")

	f.waitConns(1)
	f.push(snapshotFrame(1, 2, 0.5))
	eventuallyQuotes(t, sim, 2000)
	f.push(snapshotFrame(2, 3, 0.5))
	eventuallyQuotes(t, sim, 3000)

	res, err := quoteResult(sim, 0, big.NewInt(1000), nil)
	require.NoError(t, err)
	assert.Equal(t, uint64(2), res.SwapInfo.(SwapInfo).Seq)
	assert.Equal(t, pool.MetaInfo{ApprovalAddress: hexAddr(testProxy), BlockNumber: 1002}, sim.GetMetaInfo("", ""))
}

// 2. A msgpack-decoded simulator in a process without a tracker connects
// and quotes from the live feed using only pool state.
func TestFeed_DecodedSimulatorConnectsFromPoolState(t *testing.T) {
	t.Parallel()
	f := newFakeFeed(t, "key")
	sim := newSim(t, feedPool(t, f.settings()))
	f.waitConns(1)

	require.NoError(t, msgpack.RegisterConcreteType(&PoolSimulator{}))
	var buf bytes.Buffer
	en := msgpack.NewEncoder(&buf)
	en.IncludeUnexported(true)
	en.SetForceAsArray(true)
	require.NoError(t, en.Encode(map[string]pool.IPoolSimulator{"p": sim}))

	// Simulate a fresh router process: no client for this proxy exists.
	v, ok := feeds.LoadAndDelete(feedKey(f.settings(), hexAddr(testProxy)))
	require.True(t, ok)
	v.(*feedClient).close()
	f.dropAll()

	de := msgpack.NewDecoder(&buf)
	de.IncludeUnexported(true)
	de.SetForceAsArray(true)
	var decoded map[string]pool.IPoolSimulator
	require.NoError(t, de.Decode(&decoded))
	dsim := decoded["p"]

	// Decoding alone connects (AfterMsgpackUnmarshal), before any quote.
	f.waitConns(2)
	_, err := quote(dsim, 0, big.NewInt(1000))
	require.ErrorIs(t, err, ErrNoFeed, "no snapshot yet")
	f.push(snapshotFrame(1, 2, 0.5))
	eventuallyQuotes(t, dsim, 2000)
}

// 3. A clone keeps its snapshot after newer pushes; the base moves on.
func TestFeed_CloneKeepsSnapshot(t *testing.T) {
	t.Parallel()
	f := newFakeFeed(t, "key")
	sim := newSim(t, feedPool(t, f.settings()))
	f.waitConns(1)
	f.push(snapshotFrame(1, 2, 0.5))
	eventuallyQuotes(t, sim, 2000)

	clone := sim.CloneState()
	f.push(snapshotFrame(2, 3, 0.5))
	eventuallyQuotes(t, sim, 3000)
	assert.Equal(t, int64(2000), quote1000(clone))

	// Pinned state doesn't age out mid-route either.
	f.push(feedMessage{V: feedProtocolVersion, Type: feedMsgUnavailable})
	eventuallyQuotes(t, sim, -1)
	assert.Equal(t, int64(2000), quote1000(clone))
}

// 4. UpdateBalance applies to the quoted snapshot after a newer one arrives.
func TestFeed_UpdateBalanceUsesQuotedSnapshot(t *testing.T) {
	t.Parallel()
	f := newFakeFeed(t, "key")
	sim := newSim(t, feedPool(t, f.settings()))
	f.waitConns(1)
	f.push(snapshotFrame(1, 2, 0.5))
	eventuallyQuotes(t, sim, 2000)

	amountIn := big.NewInt(1000)
	res, err := quoteResult(sim, 0, amountIn, nil)
	require.NoError(t, err)

	f.push(snapshotFrame(2, 3, 0.5))
	eventuallyQuotes(t, sim, 3000)

	clone := sim.CloneState() // pinned to snapshot 2
	clone.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: sim.GetTokens()[0], Amount: amountIn},
		TokenAmountOut: *res.TokenAmountOut,
		SwapInfo:       res.SwapInfo,
	})
	// Now pinned to snapshot 1 with 1000 consumed: the next 1000 is priced
	// on snapshot 1's ladder from 1000 to 2000.
	next, err := quoteResult(clone, 0, amountIn, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(2000), next.TokenAmountOut.Amount.Int64())
	assert.Equal(t, uint64(1), next.SwapInfo.(SwapInfo).Seq)
	_, err = quote(clone, 0, big.NewInt(2001))
	assert.ErrorIs(t, err, ladder.ErrAmountInTooLarge, "1000 of snapshot 1's 3000 already consumed")
	assert.Equal(t, int64(3000), quote1000(sim))
}

// 5. A route's consumption never changes what other quotes see, and
// inventory consumed in one market lowers what other markets can deliver.
func TestFeed_RouteConsumptionIsolatedAndSharedInventory(t *testing.T) {
	t.Parallel()
	f := newFakeFeed(t, "key")
	sim := newSim(t, feedPool(t, f.settings()))

	btcPool := feedPool(t, f.settings())
	btcPool.Address = poolAddress(testProxy, testUSDC, testCBBTC)
	btcPool.Tokens[1] = &entity.PoolToken{Address: hexAddr(testCBBTC), Decimals: 8, Swappable: true}
	btc := newSim(t, btcPool)

	frame := snapshotFrame(1, 2, 0.5)
	frame.Tokens[hexAddr(testUSDC)] = feedToken{Balance: "2000", Allowance: "1500"}
	frame.Tokens[hexAddr(testCBBTC)] = feedToken{Balance: "1000000000", Allowance: "1000000000"}
	frame.Markets[hexAddr(testCBBTC)] = feedMarket{Ladders: linearLadders(2, 0.5)}
	f.waitConns(1)
	f.push(frame)
	eventuallyQuotes(t, sim, 2000)

	params := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: hexAddr(testWETH), Amount: big.NewInt(2000)},
		TokenOut:      hexAddr(testUSDC),
	}
	testutil.TestCloneState(t, sim, params, nil)

	limits := map[string]*big.Int{}
	for _, s := range []*PoolSimulator{sim, btc} {
		for token, l := range s.CalculateLimit() {
			limits[token] = l
		}
	}
	assert.Equal(t, big.NewInt(1500), limits[hexAddr(testUSDC)], "deliverable = min(balance, allowance)")
	inventory := swaplimit.NewInventory(DexType, limits)

	// Sell WETH for 1000 USDC through one market...
	res, err := quoteResult(sim, 1, big.NewInt(2000), inventory)
	require.NoError(t, err)
	require.Equal(t, int64(1000), res.TokenAmountOut.Amount.Int64())
	route := sim.CloneState()
	route.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn: params.TokenAmountIn, TokenAmountOut: *res.TokenAmountOut,
		SwapInfo: res.SwapInfo, SwapLimit: inventory,
	})

	// ...which leaves 500 deliverable USDC for the other market.
	_, err = quoteResult(btc, 1, big.NewInt(2000), inventory)
	assert.ErrorIs(t, err, pool.ErrNotEnoughInventory)
	res, err = quoteResult(btc, 1, big.NewInt(1000), inventory)
	require.NoError(t, err)
	assert.Equal(t, int64(500), res.TokenAmountOut.Amount.Int64())

	// The base simulators and a fresh clone are untouched.
	out, err := quote(sim, 1, big.NewInt(2000))
	require.NoError(t, err)
	assert.Equal(t, int64(1000), out.Int64())
	out, err = quote(sim.CloneState(), 1, big.NewInt(2000))
	require.NoError(t, err)
	assert.Equal(t, int64(1000), out.Int64())
}

// 6. Without a live, valid, fresh snapshot the adapter doesn't quote.
func TestFeed_StopsQuotingWithoutLiveFeed(t *testing.T) {
	t.Parallel()

	t.Run("unconfigured", func(t *testing.T) {
		t.Parallel()
		sim := newSim(t, feedPool(t, FeedSettings{}))
		_, err := quote(sim, 0, big.NewInt(1000))
		assert.ErrorIs(t, err, ErrNoFeed)
		assert.Equal(t, map[string]*big.Int{hexAddr(testUSDC): {}, hexAddr(testWETH): {}}, sim.CalculateLimit())
	})

	cases := map[string]struct {
		settings func(FeedSettings) FeedSettings
		break_   func(f *fakeFeed)
	}{
		"disconnected without decay": {
			settings: func(s FeedSettings) FeedSettings { s.FreshMs = 400; return s },
			break_:   func(f *fakeFeed) { f.dropAll() },
		},
		"withdrawn": {break_: func(f *fakeFeed) {
			f.push(feedMessage{V: feedProtocolVersion, Type: feedMsgUnavailable, Reason: "maintenance"})
		}},
		"stale without decay": {settings: func(s FeedSettings) FeedSettings { s.FreshMs = 400; return s }},
		"past max age with decay": {settings: func(s FeedSettings) FeedSettings {
			s.FreshMs, s.DecayBps, s.MaxAgeMs = 400, 10, 800
			return s
		}},
		"invalid json":    {break_: func(f *fakeFeed) { f.send([]byte(`{"v":1,"type":`)) }},
		"invalid version": {break_: func(f *fakeFeed) { f.send([]byte(`{"v":2,"type":"snapshot","seq":9}`)) }},
		"invalid ladder": {break_: func(f *fakeFeed) {
			m := snapshotFrame(9, 2, 0.5)
			m.Markets[hexAddr(testWETH)] = feedMarket{Ladders: [2][]ladder.Point{{{2000, 2}, {1000, 3}}, nil}}
			f.push(m)
		}},
		"missing inventory": {break_: func(f *fakeFeed) {
			m := snapshotFrame(9, 2, 0.5)
			delete(m.Tokens, hexAddr(testUSDC))
			f.push(m)
		}},
		"market not covered": {break_: func(f *fakeFeed) {
			m := snapshotFrame(9, 2, 0.5)
			delete(m.Markets, hexAddr(testWETH))
			f.push(m)
		}},
		"other proxy": {break_: func(f *fakeFeed) {
			m := snapshotFrame(9, 2, 0.5)
			m.Proxy = hexAddr(testOther)
			f.push(m)
		}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			f := newFakeFeed(t, "key")
			settings := f.settings()
			// Long enough that only the case's own break stops quoting.
			settings.FreshMs, settings.MaxAgeMs = 60_000, 60_000
			if tc.settings != nil {
				settings = tc.settings(settings)
			}
			sim := newSim(t, feedPool(t, settings))
			f.waitConns(1)
			f.push(snapshotFrame(1, 2, 0.5))
			eventuallyQuotes(t, sim, 2000)
			if tc.break_ != nil {
				tc.break_(f)
			}
			eventuallyQuotes(t, sim, -1)
			_, err := quote(sim, 0, big.NewInt(1000))
			assert.ErrorIs(t, err, ErrNoFeed)
		})
	}
}

// decayed is the expected decayed output for an undecayed ladder output.
func decayed(out int64, decay uint64) int64 {
	return new(big.Int).Quo(new(big.Int).Mul(big.NewInt(out), big.NewInt(decayScale)),
		new(big.Int).Add(big.NewInt(decayScale), new(big.Int).SetUint64(decay))).Int64()
}

// 7. Past the fresh window quotes decay per second like on-chain, including
// after a disconnect, instead of stopping.
func TestFeed_DecaysAfterFreshWindow(t *testing.T) {
	t.Parallel()
	f := newFakeFeed(t, "key")
	settings := f.settings()
	settings.FreshMs, settings.DecayBps = 200, 5000 // 50%/s, to see it quickly
	sim := newSim(t, feedPool(t, settings))
	f.waitConns(1)
	f.push(snapshotFrame(1, 2, 0.5))
	// Any quote: on a slow machine the first one may already be decayed.
	require.Eventually(t, func() bool {
		_, err := quote(sim, 0, big.NewInt(1000))
		return err == nil
	}, 5*time.Second, 2*time.Millisecond)
	f.dropAll()

	require.Eventually(t, func() bool {
		res, err := quoteResult(sim, 0, big.NewInt(1000), nil)
		return err == nil && res.SwapInfo.(SwapInfo).Decay > 500*5000
	}, 5*time.Second, 5*time.Millisecond, "still quoting, decayed, after the disconnect")
	res, err := quoteResult(sim, 0, big.NewInt(1000), nil)
	require.NoError(t, err)
	decay := res.SwapInfo.(SwapInfo).Decay
	assert.Equal(t, decayed(2000, decay), res.TokenAmountOut.Amount.Int64())
	assert.Less(t, res.TokenAmountOut.Amount.Int64(), int64(1600), "more than 0.5s of 50%/s: below 2000/1.25")

	// A route pins the decay with the snapshot. The ladder consumes the
	// undecayed 2000 of its 6000; the inventory limit consumes what is paid.
	limits := sim.CalculateLimit()
	wethBefore := limits[hexAddr(testWETH)]
	inventory := swaplimit.NewInventory(DexType, limits)
	clone := sim.CloneState()
	res, err = quoteResult(clone, 0, big.NewInt(1000), inventory)
	require.NoError(t, err)
	paid := res.TokenAmountOut.Amount
	clone.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: hexAddr(testUSDC), Amount: big.NewInt(1000)},
		TokenAmountOut: *res.TokenAmountOut, SwapInfo: res.SwapInfo, SwapLimit: inventory,
	})
	pinnedDecay := res.SwapInfo.(SwapInfo).Decay
	time.Sleep(20 * time.Millisecond)
	next, err := quoteResult(clone, 0, big.NewInt(1000), nil)
	require.NoError(t, err)
	assert.Equal(t, pinnedDecay, next.SwapInfo.(SwapInfo).Decay)
	assert.Equal(t, decayed(2000, pinnedDecay), next.TokenAmountOut.Amount.Int64())
	_, err = quote(clone, 0, big.NewInt(2001))
	assert.ErrorIs(t, err, ladder.ErrAmountInTooLarge)
	assert.Equal(t, new(big.Int).Sub(wethBefore, paid), inventory.GetLimit(hexAddr(testWETH)))

	// After the reconnect, a new snapshot takes over from the decayed one.
	f.waitConns(2)
	f.push(snapshotFrame(2, 3, 0.5))
	require.Eventually(t, func() bool {
		res, err := quoteResult(sim, 0, big.NewInt(1000), nil)
		return err == nil && res.SwapInfo.(SwapInfo).Seq == 2
	}, 5*time.Second, 2*time.Millisecond)
}

// 6/7. A snapshot quotes undecayed through the fresh window, decayed after
// it, and not at all past the maximum age, timed from receipt.
func TestFeed_QuotableWindows(t *testing.T) {
	t.Parallel()
	msg := snapshotFrame(1, 2, 0.5)
	now := time.Now()
	for name, tc := range map[string]struct {
		settings FeedSettings
		age      time.Duration
		ok       bool
		decay    uint64
	}{
		"fresh":                       {FeedSettings{DecayBps: 10}, 999 * time.Millisecond, true, 0},
		"decayed":                     {FeedSettings{DecayBps: 10}, 3 * time.Second, true, 10 * 2000},
		"default max age":             {FeedSettings{DecayBps: 10}, defaultFeedMaxAge, false, 0},
		"stale without decay":         {FeedSettings{}, 2 * time.Second, false, 0},
		"max age covers fresh window": {FeedSettings{FreshMs: 8000}, 7 * time.Second, true, 0},
		"configured max age":          {FeedSettings{DecayBps: 10, MaxAgeMs: 2000}, 2 * time.Second, false, 0},
	} {
		snap, err := buildFeedSnapshot(&msg, 0, now.Add(-tc.age))
		require.NoError(t, err)
		c := &feedClient{}
		c.settings.Store(&versionedSettings{FeedSettings: tc.settings})
		c.snapshot.Store(snap)
		got, decay := c.quotable(now)
		assert.Equal(t, tc.ok, got != nil, name)
		assert.Equal(t, tc.decay, decay, name)
	}
}

// 8. One connection per proxy per process; rotated settings are adopted.
func TestFeed_OneConnectionPerProxyAndRotation(t *testing.T) {
	t.Parallel()
	f := newFakeFeed(t, "key-1")
	old := f.settings()
	tracker := NewPoolTracker(&Config{ChainID: 8453, Address: hexAddr(testProxy),
		Feed: FeedConfig{URL: old.URL, AuthToken: old.AuthToken}}, nil)
	_, err := tracker.GetNewPoolState(ctx, feedPool(t, FeedSettings{}), pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	sims := make([]*PoolSimulator, 3)
	for i := range sims {
		sims[i] = newSim(t, feedPool(t, old))
	}
	f.waitConns(1)
	f.push(snapshotFrame(1, 2, 0.5))
	for _, sim := range sims {
		eventuallyQuotes(t, sim, 2000)
	}
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int32(1), f.accepted.Load(), "tracker and simulators share one connection")
	f.mu.Lock()
	assert.Equal(t, []feedSubscribe{{V: 1, Type: "subscribe", ChainID: 8453, Proxy: hexAddr(testProxy)}}, f.subs)
	f.mu.Unlock()

	// The key rotates server-side; fresher pool state carries the new one.
	f.setToken("key-2")
	rotated := old
	rotated.AuthToken = "key-2"
	p := feedPool(t, rotated)
	p.Timestamp = time.Now().Unix() + 10
	newSim(t, p)
	f.waitConns(2)
	f.mu.Lock()
	assert.Equal(t, "Bearer key-2", f.authSeen[len(f.authSeen)-1])
	f.mu.Unlock()

	// Older pool state with the old key doesn't roll the settings back.
	quote1000(sims[0])
	newSim(t, feedPool(t, old))
	f.push(snapshotFrame(1, 4, 0.5))
	eventuallyQuotes(t, sims[0], 4000)
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int32(2), f.accepted.Load())
	assert.Zero(t, f.rejected.Load())
}

// 9. Feed ladders are truncated at deliverable inventory and buffered.
func TestFeed_LaddersTruncatedAndBuffered(t *testing.T) {
	t.Parallel()
	f := newFakeFeed(t, "key")
	settings := f.settings()
	settings.Buffer = 9000
	sim := newSim(t, feedPool(t, settings))

	frame := snapshotFrame(1, 2, 0.5)
	// 5000 WETH balance but only 4000 allowed: the 2-rate ladder caps at 4000 out.
	frame.Tokens[hexAddr(testWETH)] = feedToken{Balance: "5000", Allowance: "4000"}
	f.waitConns(1)
	f.push(frame)
	eventuallyQuotes(t, sim, 1800)

	out, err := quote(sim, 0, big.NewInt(2000))
	require.NoError(t, err)
	assert.Equal(t, int64(3600), out.Int64(), "buffered cap")
	_, err = quote(sim, 0, big.NewInt(2001))
	assert.ErrorIs(t, err, ladder.ErrAmountInTooLarge)
	assert.Equal(t, map[string]*big.Int{hexAddr(testUSDC): big.NewInt(1e9), hexAddr(testWETH): big.NewInt(4000)},
		sim.CalculateLimit())
	assert.Equal(t, []*big.Int{big.NewInt(1e9), big.NewInt(4000)}, sim.GetReserves())
}

// 12. Auth, sequence ordering and protocol version are enforced.
func TestFeed_RejectsWrongToken(t *testing.T) {
	t.Parallel()
	f := newFakeFeed(t, "right")
	settings := f.settings()
	settings.AuthToken = "wrong"
	sim := newSim(t, feedPool(t, settings))
	require.Eventually(t, func() bool { return f.rejected.Load() >= 1 }, 5*time.Second, 5*time.Millisecond)
	assert.Zero(t, f.accepted.Load())
	assert.Equal(t, int64(-1), quote1000(sim))
	f.mu.Lock()
	assert.Equal(t, "Bearer wrong", f.authSeen[0], "token only in the handshake header")
	f.mu.Unlock()
}

func TestFeed_SequenceOrdering(t *testing.T) {
	t.Parallel()
	f := newFakeFeed(t, "key")
	sim := newSim(t, feedPool(t, f.settings()))
	f.waitConns(1)
	f.push(snapshotFrame(5, 2, 0.5))
	eventuallyQuotes(t, sim, 2000)

	// Older and repeated frames are ignored; frames apply in order, so once
	// seq 6's direction-1 change shows, seq 3 and the duplicate 5 are done.
	f.push(snapshotFrame(3, 3, 0.5))
	f.push(snapshotFrame(5, 3, 0.5))
	f.push(snapshotFrame(6, 2, 0.25))
	require.Eventually(t, func() bool {
		out, err := quote(sim, 1, big.NewInt(1000))
		return err == nil && out.Int64() == 250
	}, 5*time.Second, 2*time.Millisecond)
	assert.Equal(t, int64(2000), quote1000(sim))

	// Sequence numbers are per connection.
	f.dropAll()
	f.waitConns(2)
	f.push(snapshotFrame(1, 4, 0.5))
	eventuallyQuotes(t, sim, 4000)
}

// 12. Frames of unknown types are skipped, even when their fields clash
// with a snapshot's, so the server can add message types without
// withdrawing quotes from routers on older versions of this client.
func TestFeed_IgnoresUnknownMessageTypes(t *testing.T) {
	t.Parallel()
	c := &feedClient{chainID: 8453, proxy: hexAddr(testProxy)}
	c.settings.Store(&versionedSettings{})
	frame, err := json.Marshal(snapshotFrame(1, 2, 0.5))
	require.NoError(t, err)
	require.True(t, c.handle(frame))
	snap := c.snapshot.Load()
	require.NotNil(t, snap)

	for _, frame := range []string{
		`{"v":1,"type":"heartbeat"}`,
		`{"v":1,"type":"stats","seq":"many","tokens":[1,2,3]}`,
	} {
		assert.False(t, c.handle([]byte(frame)), frame)
		assert.Same(t, snap, c.snapshot.Load(), frame)
	}

	// Malformed frames and other protocol versions still withdraw it.
	assert.False(t, c.handle([]byte(`{"v":2,"type":"heartbeat"}`)))
	assert.Nil(t, c.snapshot.Load())
}

// A buffer change from fresher pool state reshapes the current snapshot in
// place, and never brings back one that was withdrawn.
func TestFeed_BufferChangeReshapesSnapshot(t *testing.T) {
	t.Parallel()
	c := &feedClient{chainID: 8453, proxy: hexAddr(testProxy), reconnect: make(chan struct{}, 1)}
	c.settings.Store(&versionedSettings{version: 1})
	frame, err := json.Marshal(snapshotFrame(1, 2, 0.5))
	require.NoError(t, err)
	require.True(t, c.handle(frame))
	quoteSnapshot := func() int64 {
		state := c.snapshot.Load().market(hexAddr(testUSDC), hexAddr(testWETH))
		res, err := state.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: hexAddr(testUSDC), Amount: big.NewInt(1000)},
			TokenOut:      hexAddr(testWETH),
		})
		require.NoError(t, err)
		return res.TokenAmountOut.Amount.Int64()
	}
	assert.Equal(t, int64(2000), quoteSnapshot())

	c.offer(FeedSettings{Buffer: 9000}, 2)
	assert.Equal(t, int64(1800), quoteSnapshot())

	c.clear()
	c.offer(FeedSettings{Buffer: 8000}, 3)
	assert.Nil(t, c.snapshot.Load())
}

// 13. Quotes, clones and updates race with pushes (run with -race).
func TestFeed_ConcurrentQuotesDuringPushes(t *testing.T) {
	t.Parallel()
	f := newFakeFeed(t, "key")
	sim := newSim(t, feedPool(t, f.settings()))
	f.waitConns(1)

	var wg sync.WaitGroup
	stop := make(chan struct{})
	for range 8 {
		wg.Go(func() {
			for {
				select {
				case <-stop:
					return
				default:
				}
				res, err := quoteResult(sim, 0, big.NewInt(1000), nil)
				if err != nil {
					continue
				}
				clone := sim.CloneState()
				clone.UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn:  pool.TokenAmount{Token: sim.GetTokens()[0], Amount: big.NewInt(1000)},
					TokenAmountOut: *res.TokenAmountOut,
					SwapInfo:       res.SwapInfo,
				})
				_, _ = quote(clone, 0, big.NewInt(1000))
				_ = sim.CalculateLimit()
				_ = sim.GetMetaInfo("", "")
			}
		})
	}
	for seq := uint64(1); seq <= 50; seq++ {
		f.push(snapshotFrame(seq, float64(1+seq%3), 0.5))
		time.Sleep(time.Millisecond)
	}
	eventuallyQuotes(t, sim, 3000) // seq 50: rate 1+50%3 = 3
	close(stop)
	wg.Wait()
}

// Reading the latest snapshot on the quote path doesn't allocate.
func TestFeed_QuotePathLookupDoesNotAllocate(t *testing.T) {
	f := newFakeFeed(t, "key")
	sim := newSim(t, feedPool(t, f.settings()))
	f.waitConns(1)
	f.push(snapshotFrame(1, 2, 0.5))
	eventuallyQuotes(t, sim, 2000)

	allocs := testing.AllocsPerRun(1000, func() {
		if _, _, err := sim.current(time.Now()); err != nil {
			panic(err)
		}
	})
	assert.Zero(t, allocs)
}
