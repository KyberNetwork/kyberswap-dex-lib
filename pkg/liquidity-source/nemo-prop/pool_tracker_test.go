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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var ctx = context.Background()

// 10. The tracker mirrors the feed's deliverable inventory, zeroed once a
// live snapshot no longer covers the market.
func TestPoolTracker_WritesFeedSettingsAndMirrorsInventory(t *testing.T) {
	t.Parallel()
	f := newFakeFeed(t, "integrator-key")
	cfg := &Config{
		DexID: DexType, ChainID: 8453, Address: hexAddr(testProxy), Buffer: 9990, DecayBps: 10,
		Feed: FeedConfig{URL: f.url(), AuthToken: "integrator-key", FreshMs: 10_000, MaxAgeMs: 30_000},
	}
	tracker := NewPoolTracker(cfg, nil)
	p := testPool(t, FeedSettings{}, 0)
	p.Extra, p.Reserves = "{}", entity.PoolReserves{"0", "0"}

	// No snapshot yet: settings already travel in Extra, reserves untouched.
	p, err := tracker.GetNewPoolState(ctx, p, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	assert.Equal(t, FeedSettings{URL: f.url(), AuthToken: "integrator-key", ChainID: 8453, FreshMs: 10_000,
		MaxAgeMs: 30_000, Buffer: 9990, DecayBps: 10}, extra.Feed)
	assert.Equal(t, entity.PoolReserves{"0", "0"}, p.Reserves)
	assert.InDelta(t, time.Now().Unix(), p.Timestamp, 2)

	// The tracker's own connection is the one simulators share.
	f.waitConns(1)
	frame := snapshotFrame(1, 2, 0.5)
	frame.Tokens[hexAddr(testUSDC)] = feedToken{Balance: "700", Allowance: "500"}
	f.push(frame)
	require.Eventually(t, func() bool {
		next, err := tracker.GetNewPoolState(ctx, p, pool.GetNewPoolStateParams{})
		if err != nil || next.BlockNumber != 1001 {
			return false
		}
		p = next
		return true
	}, 5*time.Second, 5*time.Millisecond)
	assert.Equal(t, entity.PoolReserves{"500", "1000000000"}, p.Reserves, "deliverable = min(balance, allowance)")

	sim := newSim(t, p)
	eventuallyQuotes(t, sim, 1998) // rate 2, buffered by 9990
	assert.Equal(t, int32(1), f.accepted.Load())

	// A live snapshot without the market (e.g. dropped by a pricing upgrade)
	// leaves nothing deliverable through the pool.
	frame = snapshotFrame(2, 2, 0.5)
	delete(frame.Markets, hexAddr(testWETH))
	f.push(frame)
	require.Eventually(t, func() bool {
		next, err := tracker.GetNewPoolState(ctx, p, pool.GetNewPoolStateParams{})
		if err != nil || next.BlockNumber != 1002 {
			return false
		}
		p = next
		return true
	}, 5*time.Second, 5*time.Millisecond)
	assert.Equal(t, entity.PoolReserves{"0", "0"}, p.Reserves)
}

// 11. Discovery is keyed by asset, so a pricing upgrade that reorders or
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
