package ambient

import (
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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	testLTRPCURL             = "https://ethereum.publicnode.com"
	testLTMulticallAddress   = "0xcA11bde05977b3631167028862bE2a173976CA11"
	testLTSwapDex            = "0xaaaaaaaaa24eeeb8d57d431224f73832bc34f688"
	testLTNativeTokenAddress = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	testLTMulticallContract  = "0x5ba1e12693dc8f9c48aad8770482f4739beed696"
	testLTIndexerBaseURL     = "https://ambindexer.net"
	testLTIndexerChainID     = "0x1"

	testLTUSDC = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
)

func newTestConfig() *Config {
	return &Config{
		DexId:   DexType,
		ChainId: valueobject.ChainIDEthereum,
		HTTPConfig: HTTPConfig{
			BaseURL: testLTIndexerBaseURL,
			Timeout: durationjson.Duration{Duration: 10 * time.Second},
		},
		IndexerChainId: testLTIndexerChainID,
		PoolIdx:        big.NewInt(420),
		SwapDex:        testLTSwapDex,
		Multicall3:     testLTMulticallContract,
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

// TestPoolLister hits the live Ambient indexer and expects ≥1 pool entity to
// be emitted per active pair under the configured poolIdx.
func TestPoolLister(t *testing.T) {
	t.Parallel()
	test.SkipCI(t)

	cfg := newTestConfig()
	updater := NewPoolListUpdater(cfg, newTestRPCClient())

	pools, _, err := updater.GetNewPools(t.Context(), nil)
	require.NoError(t, err)
	require.Greater(t, len(pools), 0, "expect at least one per-pair pool from indexer")

	for _, p := range pools {
		require.Len(t, p.Tokens, 2, "per-pair pool has exactly 2 tokens")
		require.Len(t, p.Reserves, 2)
		require.NotEmpty(t, p.StaticExtra)

		var se StaticExtra
		require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &se))
		require.Equal(t, cfg.PoolIdx.Uint64(), se.PoolIdx)
	}
	t.Logf("emitted %d per-pair pools", len(pools))
}

// TestPoolTracker exercises cold-load and refresh using one hardcoded pair.
// Set ETHEREUM_RPC_URL to a non-rate-limited RPC for reliable results.
func TestPoolTracker(t *testing.T) {
	t.Parallel()
	test.SkipCI(t)

	cfg := newTestConfig()
	rpcClient := newTestRPCClient()

	baseHex, quoteHex := normalizePair(valueobject.AddrZero.Hex(), testLTUSDC)
	base := common.HexToAddress(baseHex)
	quote := common.HexToAddress(quoteHex)
	poolHash := EncodePoolHash(base, quote, cfg.PoolIdx.Uint64())
	nativeAddr := common.HexToAddress(valueobject.WrappedNativeMap[cfg.ChainId])

	staticExtraBytes, err := json.Marshal(StaticExtra{
		NativeToken: nativeAddr.Hex(),
		PoolIdx:     cfg.PoolIdx.Uint64(),
		SwapDex:     cfg.SwapDex,
		Base:        baseHex,
		Quote:       quoteHex,
	})
	require.NoError(t, err)

	p := entity.Pool{
		Address:     poolHash.Hex(),
		Exchange:    DexType,
		Type:        DexType,
		StaticExtra: string(staticExtraBytes),
		Tokens: []*entity.PoolToken{
			{Address: nativeAddr.Hex(), Swappable: true},
			{Address: testLTUSDC, Swappable: true},
		},
		Reserves: []string{"0", "0"},
	}

	fmt.Println("Testing pool tracker with pool address", p.Address)

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
	require.NotNil(t, trackedExtra.State)
	require.NotNil(t, trackedExtra.State.Curve.PriceRoot)
	require.Equal(t, 1, trackedExtra.State.Curve.PriceRoot.Sign())
	t.Logf("tracked ticks=%d sqrtPrice=%s",
		len(trackedExtra.State.ActiveTicks), trackedExtra.State.Curve.PriceRoot)

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
	require.NotNil(t, refreshedExtra.State)
}
