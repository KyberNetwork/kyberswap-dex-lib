package ambient

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

type emptyPoolDatastore struct{}

func (emptyPoolDatastore) Get(_ context.Context, _ string) (entity.Pool, error) {
	return entity.Pool{}, fmt.Errorf("pool not found")
}

const (
	testLTRPCURL                 = "https://mainnet.gateway.tenderly.co/6td6HqHO4vq7x66oUgqUKX"
	testLTMulticallAddress       = "0xcA11bde05977b3631167028862bE2a173976CA11"
	testLTSwapDexContractAddress = "0xaaaaaaaaa24eeeb8d57d431224f73832bc34f688"
	testLTNativeTokenAddress     = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	testLTMulticallContract      = "0x5ba1e12693dc8f9c48aad8770482f4739beed696"
	testLTSubgraphAPI            = "https://api.studio.thegraph.com/query/47610/croc-mainnet/version/latest"

	testLTUSDC = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
	testLTUSDT = "0xdac17f958d2ee523a2206206994597c13d831ec7"
	testLTWBTC = "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599"
)

func newTestConfig() *Config {
	return &Config{
		DexID:                    DexType,
		SubgraphAPI:              testLTSubgraphAPI,
		SubgraphTimeout:          durationjson.Duration{Duration: 10 * time.Second},
		PoolIdx:                  big.NewInt(420),
		NativeTokenAddress:       testLTNativeTokenAddress,
		SwapDexContractAddress:   testLTSwapDexContractAddress,
		MulticallContractAddress: testLTMulticallContract,
	}
}

func rpcURL() string {
	if v := os.Getenv("ETHEREUM_RPC_URL"); v != "" {
		return v
	}
	return testLTRPCURL
}

func newTestRPCClient() *ethrpc.Client {
	return ethrpc.New(rpcURL()).
		SetMulticallContract(common.HexToAddress(testLTMulticallAddress))
}

func TestPoolLister(t *testing.T) {
	t.Parallel()
	test.SkipCI(t)

	cfg := newTestConfig()
	graphqlClient := graphqlpkg.NewClient(cfg.SubgraphAPI)

	updater, err := NewPoolsListUpdater(cfg, emptyPoolDatastore{}, graphqlClient)
	require.NoError(t, err)

	pools, metaBytes, err := updater.GetNewPools(t.Context(), nil)
	require.NoError(t, err)
	require.Len(t, pools, 1, "ambient singleton: expect exactly one pool entity")

	p := pools[0]
	t.Logf("pool address=%s tokens=%d reserves=%d", p.Address, len(p.Tokens), len(p.Reserves))
	require.Greater(t, len(p.Tokens), 0)

	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	t.Logf("token pairs discovered: %d", len(extra.TokenPairs))
	require.Greater(t, len(extra.TokenPairs), 0)

	for pair, info := range extra.TokenPairs {
		require.NotNil(t, info.PoolIdx, "PoolIdx for %s", pair)
		require.Nil(t, info.State, "State should be nil before tracking for %s", pair)
	}

	pools2, _, err := updater.GetNewPools(t.Context(), metaBytes)
	require.NoError(t, err)
	require.Empty(t, pools2, "second call should return no new pools")
}

// TestPoolTracker exercises cold-load and refresh using hardcoded pairs.
// Set ETHEREUM_RPC_URL to a non-rate-limited RPC for reliable results.
func TestPoolTracker(t *testing.T) {
	t.Parallel()
	test.SkipCI(t)

	cfg := newTestConfig()
	rpcClient := newTestRPCClient()

	pairs := map[TokenPair]*TokenPairInfo{
		{Base: NativeTokenPlaceholderAddress, Quote: common.HexToAddress(testLTUSDC)}: {PoolIdx: big.NewInt(420)},
		{Base: NativeTokenPlaceholderAddress, Quote: common.HexToAddress(testLTUSDT)}: {PoolIdx: big.NewInt(420)},
		{Base: NativeTokenPlaceholderAddress, Quote: common.HexToAddress(testLTWBTC)}: {PoolIdx: big.NewInt(420)},
	}

	extraBytes, err := json.Marshal(Extra{TokenPairs: pairs})
	require.NoError(t, err)

	staticExtraBytes, err := json.Marshal(StaticExtra{
		NativeTokenAddress: common.HexToAddress(cfg.NativeTokenAddress),
		PoolIdx:            cfg.PoolIdx.Uint64(),
		SwapDex:            common.HexToAddress(cfg.SwapDexContractAddress),
	})
	require.NoError(t, err)

	nativeAddr := common.HexToAddress(cfg.NativeTokenAddress)
	p := entity.Pool{
		Address:     cfg.SwapDexContractAddress,
		Exchange:    DexType,
		Type:        DexType,
		Extra:       string(extraBytes),
		StaticExtra: string(staticExtraBytes),
		Tokens: []*entity.PoolToken{
			{Address: nativeAddr.Hex(), Swappable: true},
			{Address: testLTUSDC, Swappable: true},
			{Address: testLTUSDT, Swappable: true},
			{Address: testLTWBTC, Swappable: true},
		},
		Reserves: []string{"0", "0", "0", "0"},
	}

	tracker, err := NewPoolTracker(cfg, rpcClient)
	require.NoError(t, err)

	// --- Cold load ---
	start := time.Now()
	tracked, err := tracker.GetNewPoolState(t.Context(), p, pool.GetNewPoolStateParams{})
	coldElapsed := time.Since(start)
	require.NoError(t, err)
	t.Logf("cold load took %s", coldElapsed)

	require.Greater(t, tracked.BlockNumber, uint64(0))
	require.Greater(t, tracked.Timestamp, int64(0))

	var trackedExtra Extra
	require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &trackedExtra))

	trackedCount := 0
	for pair, info := range trackedExtra.TokenPairs {
		require.NotNil(t, info, "pair info for %s", pair)
		if info.State != nil {
			trackedCount++
			require.NotNil(t, info.State.Curve.PriceRoot, "PriceRoot for %s", pair)
			require.Equal(t, 1, info.State.Curve.PriceRoot.Sign(), "PriceRoot positive for %s", pair)
			t.Logf("pair %s: ticks=%d sqrtPrice=%s",
				pair, len(info.State.ActiveTicks), info.State.Curve.PriceRoot)
		}
	}
	t.Logf("tracked %d / %d pairs", trackedCount, len(trackedExtra.TokenPairs))
	require.Equal(t, len(pairs), trackedCount, "all pairs must be tracked")

	// --- Verify simulator can be built from tracked state ---
	sim, err := NewPoolSimulator(tracked)
	require.NoError(t, err)
	require.NotNil(t, sim)
	t.Logf("simulator built: tokens=%d", len(sim.GetTokens()))

	// --- Refresh ---
	time.Sleep(10 * time.Second)
	start = time.Now()
	refreshed, err := tracker.GetNewPoolState(t.Context(), tracked, pool.GetNewPoolStateParams{})
	refreshElapsed := time.Since(start)
	if err != nil {
		t.Logf("refresh failed (likely RPC rate limit, set ETHEREUM_RPC_URL): %v", err)
		t.SkipNow()
	}
	t.Logf("refresh took %s (cold: %s)", refreshElapsed, coldElapsed)

	var refreshedExtra Extra
	require.NoError(t, json.Unmarshal([]byte(refreshed.Extra), &refreshedExtra))
	require.Equal(t, len(trackedExtra.TokenPairs), len(refreshedExtra.TokenPairs))
}
