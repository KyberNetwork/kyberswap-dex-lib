package nemoprop

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var ctx = context.Background()

func refresh(t *testing.T, tracker *PoolTracker, p entity.Pool) (entity.Pool, Extra) {
	t.Helper()
	p, err := tracker.GetNewPoolState(ctx, p, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	return p, extra
}

// eventuallyBlock refreshes p until it carries the snapshot of blockNumber.
func eventuallyBlock(t *testing.T, tracker *PoolTracker, p entity.Pool, blockNumber uint64) (entity.Pool, Extra) {
	t.Helper()
	var extra Extra
	require.Eventually(t, func() bool {
		p, extra = refresh(t, tracker, p)
		return p.BlockNumber == blockNumber
	}, 5*time.Second, 2*time.Millisecond, "snapshot for block %d never reached the pool", blockNumber)
	return p, extra
}

// The tracker writes the latest snapshot's ladders, truncated at
// deliverable inventory and buffered, and its inventory into pool state.
func TestPoolTracker_WritesSnapshotIntoPoolState(t *testing.T) {
	t.Parallel()
	f := newFakeFeed(t, "integrator-key")
	cfg := f.config()
	cfg.Buffer, cfg.DecayBps, cfg.Feed.FreshMs, cfg.Feed.MaxAgeMs = 9000, 10, 1000, 0
	tracker := NewPoolTracker(cfg, newFakeChain().rpcClient(t))

	// No snapshot yet: nothing to quote.
	p, extra := refresh(t, tracker, testPool(t))
	assert.Equal(t, Extra{FreshMs: 1000, MaxAgeMs: 5000, DecayBps: 10}, extra)
	assert.Equal(t, entity.PoolReserves{"0", "0"}, p.Reserves)

	f.waitConns(1)
	frame := snapshotFrame(1, 2, 0.5)
	frame.Tokens[hexAddr(testUSDC)] = feedToken{Balance: "700", Allowance: "500"}
	// 5000 WETH balance but only 4000 allowed: the 2-rate ladder caps at 4000 out.
	frame.Tokens[hexAddr(testWETH)] = feedToken{Balance: "5000", Allowance: "4000"}
	f.push(frame)
	p, extra = eventuallyBlock(t, tracker, p, 1001)
	assert.Equal(t, entity.PoolReserves{"500", "4000"}, p.Reserves, "deliverable = min(balance, allowance)")
	assert.InDelta(t, time.Now().UnixMilli(), extra.PricedAtMs, 5000)
	assert.Equal(t, []ladder.Point{{1000, 1800}, {2000, 3600}}, extra.Ladders[0], "truncated, then buffered")
	assert.NotContains(t, p.Extra, "integrator-key", "the auth token never enters pool state")

	sim := newSim(t, p)
	assert.Equal(t, int64(1800), quote1000(sim))
	_, err := quote(sim, 0, big.NewInt(2001))
	assert.ErrorIs(t, err, ladder.ErrAmountInTooLarge)

	// A live snapshot without the market (e.g. dropped by a pricing upgrade)
	// leaves nothing quotable or deliverable through the pool.
	frame = snapshotFrame(2, 2, 0.5)
	delete(frame.Markets, hexAddr(testWETH))
	f.push(frame)
	p, extra = eventuallyBlock(t, tracker, p, 1002)
	assert.Equal(t, entity.PoolReserves{"0", "0"}, p.Reserves)
	assert.Empty(t, extra.Ladders[0])
	assert.Zero(t, extra.PricedAtMs)
}

// A direction with no quote delivers none of its output token.
func TestPoolTracker_ZeroesUnquotedReserves(t *testing.T) {
	t.Parallel()
	f := newFakeFeed(t, "key")
	tracker := NewPoolTracker(f.config(), newFakeChain().rpcClient(t))
	f.waitConns(1)
	frame := snapshotFrame(1, 2, 0.5)
	frame.Markets[hexAddr(testWETH)] = feedMarket{Ladders: [2][]ladder.Point{{{1000, 2000}}, nil}}
	f.push(frame)
	p, _ := eventuallyBlock(t, tracker, testPool(t), 1001)
	assert.Equal(t, entity.PoolReserves{"0", "1000000000"}, p.Reserves, "base is unobtainable")
}

// Withdrawals and invalid frames stop quoting at the next refresh; a
// disconnect keeps the snapshot until the maximum age.
func TestPoolTracker_StopsWithoutLiveSnapshot(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		maxAgeMs int64
		break_   func(f *fakeFeed)
		keeps    bool
	}{
		"disconnected":         {break_: func(f *fakeFeed) { f.dropAll() }, keeps: true},
		"disconnected max age": {maxAgeMs: 300, break_: func(f *fakeFeed) { f.dropAll() }},
		"withdrawn": {break_: func(f *fakeFeed) {
			f.push(feedMessage{V: feedProtocolVersion, Type: feedMsgUnavailable, Reason: "maintenance"})
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
			cfg := f.config()
			if tc.maxAgeMs != 0 {
				cfg.Feed.FreshMs, cfg.Feed.MaxAgeMs = tc.maxAgeMs, tc.maxAgeMs
			}
			tracker := NewPoolTracker(cfg, newFakeChain().rpcClient(t))
			f.waitConns(1)
			f.push(snapshotFrame(1, 2, 0.5))
			p, _ := eventuallyBlock(t, tracker, testPool(t), 1001)
			tc.break_(f)
			if tc.keeps {
				time.Sleep(50 * time.Millisecond)
				p, extra := refresh(t, tracker, p)
				assert.NotEmpty(t, extra.Ladders[0])
				assert.Equal(t, reserves1e9, p.Reserves)
				return
			}
			require.Eventually(t, func() bool {
				var extra Extra
				p, extra = refresh(t, tracker, p)
				return len(extra.Ladders[0]) == 0
			}, 5*time.Second, 2*time.Millisecond)
			assert.Equal(t, entity.PoolReserves{"0", "0"}, p.Reserves)
		})
	}
}

// One connection per proxy per process, shared by every tracker; a tracker
// built with a rotated token reconnects it.
func TestPoolTracker_OneConnectionPerProxyAndRotation(t *testing.T) {
	t.Parallel()
	f := newFakeFeed(t, "key-1")
	trackers := []*PoolTracker{NewPoolTracker(f.config(), newFakeChain().rpcClient(t)), NewPoolTracker(f.config(), newFakeChain().rpcClient(t))}
	f.waitConns(1)
	f.push(snapshotFrame(1, 2, 0.5))
	eventuallyBlock(t, trackers[0], testPool(t), 1001)
	eventuallyBlock(t, trackers[1], btcPool(t), 1001)
	assert.Equal(t, int32(1), f.accepted.Load())
	f.mu.Lock()
	assert.Equal(t, []feedSubscribe{{V: 1, Type: "subscribe", ChainID: 8453, Proxy: hexAddr(testProxy)}}, f.subs)
	f.mu.Unlock()

	f.setToken("key-2")
	NewPoolTracker(f.config(), newFakeChain().rpcClient(t))
	f.waitConns(2)
	f.mu.Lock()
	assert.Equal(t, "Bearer key-2", f.authSeen[len(f.authSeen)-1])
	f.mu.Unlock()
	f.push(snapshotFrame(1, 3, 0.5))
	require.Eventually(t, func() bool {
		_, extra := refresh(t, trackers[0], testPool(t))
		return len(extra.Ladders[0]) > 0 && extra.Ladders[0][0].AmountOut() == 3000
	}, 5*time.Second, 2*time.Millisecond)
	assert.Zero(t, f.rejected.Load())
}

func TestPoolTracker_NoFeedConfigured(t *testing.T) {
	t.Parallel()
	tracker := NewPoolTracker(&Config{ChainID: 8453, Address: hexAddr(testProxy)}, nil)
	p, extra := refresh(t, tracker, testPool(t))
	assert.Empty(t, extra.Ladders[0])
	assert.Equal(t, entity.PoolReserves{"0", "0"}, p.Reserves)
}

// Discovery is keyed by asset, so a pricing upgrade that reorders or
// adds markets keeps existing pool addresses.
func TestPoolsListUpdater_GetNewPools(t *testing.T) {
	t.Parallel()
	chain := newFakeChain()
	cfg := &Config{DexID: "nemo-prop-base", Address: hexAddr(testProxy)}
	u := NewPoolsListUpdater(cfg, chain.rpcClient(t))

	pools, metadata, err := u.GetNewPools(ctx, nil)
	require.NoError(t, err)
	require.Len(t, pools, 2)
	for i, asset := range []common.Address{testWETH, testCBBTC} {
		assert.Equal(t, poolAddress(testProxy, testUSDC, asset), pools[i].Address)
		assert.Equal(t, "nemo-prop-base", pools[i].Exchange)
		assert.Equal(t, DexType, pools[i].Type)
		assert.Equal(t, hexAddr(testUSDC), pools[i].Tokens[0].Address)
		assert.Equal(t, hexAddr(asset), pools[i].Tokens[1].Address)
		assert.Equal(t, entity.PoolReserves{"0", "0"}, pools[i].Reserves)
		assert.JSONEq(t, `{"a":"`+hexAddr(testProxy)+`"}`, pools[i].StaticExtra)
	}

	pools, metadata, err = u.GetNewPools(ctx, metadata)
	require.NoError(t, err)
	assert.Empty(t, pools)

	// A pricing upgrade reorders the table and adds a market: only the new
	// market yields a pool, and existing pools keep their addresses.
	added := common.HexToAddress("0x00000000000000000000000000000000000a5e7a")
	chain.mu.Lock()
	chain.markets = []common.Address{testCBBTC, added, testWETH}
	chain.units[added] = big.NewInt(1e18)
	chain.mu.Unlock()
	pools, _, err = u.GetNewPools(ctx, metadata)
	require.NoError(t, err)
	require.Len(t, pools, 1)
	assert.Equal(t, poolAddress(testProxy, testUSDC, added), pools[0].Address)
}

func TestDecodeMarkets(t *testing.T) {
	t.Parallel()
	chain := newFakeChain()
	assets, err := decodeMarkets(testUSDC, chain.marketsBytes())
	require.NoError(t, err)
	assert.Equal(t, []common.Address{testWETH, testCBBTC}, assets)

	_, err = decodeMarkets(testUSDC, chain.marketsBytes()[:40])
	assert.ErrorIs(t, err, ErrInvalidMarkets)
	_, err = decodeMarkets(testUSDC, make([]byte, 32))
	assert.ErrorIs(t, err, ErrInvalidMarkets)
}

// Swaps execute against Nemo's on-chain anchors. Once those are older than
// the maximum age, a live feed must not keep the pool quoting; a pricing
// upgrade switches the anchors read to the new pricing contract.
func TestPoolTracker_StopsOnStaleOnChainAnchors(t *testing.T) {
	t.Parallel()
	f := newFakeFeed(t, "key")
	chain := newFakeChain()
	cfg := f.config()
	tracker := NewPoolTracker(cfg, chain.rpcClient(t))
	f.waitConns(1)
	f.push(snapshotFrame(1, 2, 0.5))
	p, extra := eventuallyBlock(t, tracker, testPool(t), 1001)
	require.NotEmpty(t, extra.Ladders[0], "fresh anchors quote")

	maxAge := time.Duration(cfg.maxAgeMs()) * time.Millisecond
	chain.setAnchorsAge(testPricing, maxAge+time.Second)
	p, extra = refresh(t, tracker, p)
	assert.Empty(t, extra.Ladders[0])
	assert.Zero(t, extra.PricedAtMs)
	assert.Equal(t, entity.PoolReserves{"0", "0"}, p.Reserves)

	// Upgrade to a pricing contract with fresh anchors: the refresh that sees
	// the upgrade stays empty, the next quotes again.
	upgraded := common.HexToAddress("0x00000000000000000000000000000000000e0e04")
	chain.mu.Lock()
	chain.pricing, chain.anchorsAge[upgraded] = upgraded, 0
	chain.mu.Unlock()
	_, extra = refresh(t, tracker, p)
	assert.Empty(t, extra.Ladders[0])
	p, extra = refresh(t, tracker, p)
	assert.NotEmpty(t, extra.Ladders[0])
	assert.Equal(t, reserves1e9, p.Reserves)
}

// An RPC failure reading the anchors fails the refresh, so the pool keeps its
// previous state and ages out in the simulator rather than quoting unchecked.
func TestPoolTracker_AnchorsReadFailure(t *testing.T) {
	t.Parallel()
	f := newFakeFeed(t, "key")
	chain := newFakeChain()
	chain.pricing = testOther // no pricing contract there: lastUpdatedTimestampMs reverts
	tracker := NewPoolTracker(f.config(), chain.rpcClient(t))
	f.waitConns(1)
	f.push(snapshotFrame(1, 2, 0.5))
	require.Eventually(t, func() bool {
		_, err := tracker.GetNewPoolState(ctx, testPool(t), pool.GetNewPoolStateParams{})
		return err != nil
	}, 5*time.Second, 2*time.Millisecond)
}
