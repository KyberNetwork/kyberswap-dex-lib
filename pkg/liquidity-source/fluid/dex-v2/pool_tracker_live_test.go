package dexv2

import (
	"math/big"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/fluid/dex-v2/abis"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// liveConfig is the polygon/fluid-dex-v2 exchange as production runs it.
var liveConfig = Config{
	DexID:     "fluid-dex-v2",
	ChainID:   valueobject.ChainIDPolygon,
	Dex:       "0x7822b3944b1a68b231a6e7f55b57967f28bb369e",
	Resolver:  "0x1E45589D501AcED82013c2838552122f943B33Ac",
	Liquidity: "0x52aa899454998be5b000ad077a46bbe360f4e497",
}

// retryOn429 waits out a provider's rate limit, which a shared key keeps hitting.
type retryOn429 struct{}

func (retryOn429) RoundTrip(req *http.Request) (*http.Response, error) {
	for attempt := 1; ; attempt++ {
		if req.GetBody != nil {
			req.Body, _ = req.GetBody()
		}
		resp, err := http.DefaultTransport.RoundTrip(req)
		if err != nil || resp.StatusCode != http.StatusTooManyRequests || attempt == 20 {
			return resp, err
		}
		_ = resp.Body.Close()
		time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
	}
}

// liveTracker is a tracker on Polygon, and every permissioned pool built as the pool factory builds
// it from its LogInitialize, from the Resolver's list. It skips without FLUID_DEX_V2_RPC_URL:
//
//	FLUID_DEX_V2_RPC_URL=https://... go test ./pkg/liquidity-source/fluid/dex-v2/ -run Live -v
func liveTracker(t *testing.T, cfg Config) (*PoolTracker, []entity.Pool) {
	t.Helper()
	rpcURL := os.Getenv("FLUID_DEX_V2_RPC_URL")
	if rpcURL == "" || os.Getenv("CI") != "" {
		t.Skip("set FLUID_DEX_V2_RPC_URL to a Polygon endpoint to run this")
	}
	rc, err := rpc.DialOptions(t.Context(), rpcURL, rpc.WithHTTPClient(&http.Client{Transport: retryOn429{}}))
	require.NoError(t, err)
	client := ethrpc.NewWithClient(ethclient.NewClient(rc)).
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	tracker := NewPoolTracker(&cfg, client, nil)

	var configs []struct {
		DexType    *big.Int
		DexKey     DexKey
		DexId      [32]byte
		DexAssetId [32]byte
	}
	_, err = client.NewRequest().SetContext(t.Context()).AddCall(&ethrpc.Call{
		ABI:    abis.ResolverABI,
		Target: cfg.Resolver,
		Method: "getAllPermissionedDexes",
	}, []any{&configs}).Call()
	require.NoError(t, err)
	require.NotEmpty(t, configs)

	factory := NewPoolFactory(&cfg)
	pools := make([]entity.Pool, 0, len(configs))
	for _, c := range configs {
		p, err := factory.newPool(&abis.FluidDexV2LogInitialize{
			DexType: c.DexType,
			DexId:   c.DexId,
			DexKey: abis.DexKey{
				Token0:      c.DexKey.Token0,
				Token1:      c.DexKey.Token1,
				Fee:         c.DexKey.Fee,
				TickSpacing: c.DexKey.TickSpacing,
				Controller:  c.DexKey.Controller,
			},
		}, 0)
		require.NoError(t, err)
		pools = append(pools, *p)
	}
	return tracker, pools
}

// TestLiveUpdateRefreshesFeesAndReserves: on Polygon, a pool written from its read before the swap
// at block 93210176 holds, after one update, the fee bits and reserves of the chain now.
func TestLiveUpdateRefreshesFeesAndReserves(t *testing.T) {
	ctx := t.Context()
	tracker, pools := liveTracker(t, liveConfig)
	for _, p := range pools {
		old, err := tracker.FetchRPCData(ctx, &p, 93210175)
		require.NoError(t, err)
		old.Ticks = testTicks()
		p, err := withState(p, old)
		require.NoError(t, err)
		p.BlockNumber = 93210175

		p, err = tracker.GetNewPoolState(ctx, p, poolpkg.GetNewPoolStateParams{Logs: triggerLog(93210176)})
		require.NoError(t, err)
		now, err := tracker.FetchRPCData(ctx, &p, 0)
		require.NoError(t, err)

		got := mustExtra(t, p.Extra)
		assert.Zero(t, got.DexVariables2.Cmp(now.DexVariables2), p.Address)
		assert.Zero(t, got.TokenReserves.Cmp(now.TokenReserves), p.Address)
		t.Logf("%s: reserves %v, at 93210175 %v", p.Address, p.Reserves, extractReserves(t, old))
	}
}

func extractReserves(t *testing.T, extra Extra) entity.PoolReserves {
	t.Helper()
	p, err := withState(entity.Pool{Address: testPoolAddress}, extra)
	require.NoError(t, err)
	return p.Reserves
}

// TestLiveLens: on Polygon, for every permissioned pool, a bootstrap through the lens holds every
// tick the DEX's storage reports for those indices at the same block and the state the tracker
// reads there; a gas cap that forces paging reads the same set; a stored set missing a tick gets it
// back; a pool the factory built gets its state whole from its first update, which a replay of the
// same logs leaves as it is; and the simulator quotes the result.
func TestLiveLens(t *testing.T) {
	ctx := t.Context()
	cfg := liveConfig
	cfg.UseLens = true
	// Room for a shared key's rate limits, which the transport waits out.
	cfg.LensGuard.TimeoutSecs = 300
	tracker, pools := liveTracker(t, cfg)
	// Each step that goes through the guard gets a tracker of its own, as the guard lets a pool's
	// reads through once a minute.
	guarded := func() *PoolTracker { return NewPoolTracker(&cfg, tracker.ethrpcClient, nil) }
	capped := *tracker
	capped.config = &Config{Dex: cfg.Dex, LensGas: 3_000_000}

	logs, err := tracker.ethrpcClient.GetETHClient().FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: big.NewInt(93210176), ToBlock: big.NewInt(93210176),
		Addresses: []common.Address{common.HexToAddress(cfg.Dex)},
	})
	require.NoError(t, err)
	factory := NewPoolFactory(&cfg)

	for _, built := range pools {
		t.Run(built.Address, func(t *testing.T) {
			p, err := tracker.BootstrapPoolState(ctx, built, poolpkg.GetNewPoolStateParams{})
			require.NoError(t, err)
			extra := mustExtra(t, p.Extra)
			require.NotEmpty(t, extra.Ticks)
			require.Equal(t, extra.LensBlock, p.BlockNumber, "the pool is at the lens's block")

			// Each tick against the DEX's storage for it, at the lens's block.
			indexes := lo.Map(extra.Ticks, func(tk Tick, _ int) int { return tk.Index })
			stored, err := tracker.readTicks(ctx, p.Address, indexes, extra.LensBlock)
			require.NoError(t, err)
			for i, tk := range extra.Ticks {
				assert.Zero(t, tk.LiquidityGross.Cmp(stored[i].LiquidityGross), "gross of %d", tk.Index)
				assert.Zero(t, tk.LiquidityNet.Cmp(stored[i].LiquidityNet), "net of %d", tk.Index)
			}

			// Pages under a gas cap put back the same set.
			paged, err := capped.snapshot(ctx, p.Address, new(big.Int).SetUint64(extra.LensBlock))
			require.NoError(t, err)
			assert.Equal(t, extra.Ticks, paged.ticks)
			assert.Equal(t, extra.LensBlock, paged.block)

			// A set missing a tick gets it back.
			pool, err := tickspkg.NewTicksBasedPool(p)
			require.NoError(t, err)
			pool.BlockNumber = p.BlockNumber
			delete(pool.Ticks, extra.Ticks[len(extra.Ticks)/2].Index)
			require.NoError(t, guarded().refetchAllTicks(ctx, &pool))
			assert.Len(t, pool.Ticks, len(extra.Ticks))
			assert.True(t, pool.HasValidTicks())

			// A pool the factory built, given the logs the caller would route to it or the empty
			// one an interval update sends.
			var batch []ethtypes.Log
			for _, l := range logs {
				if addrs, _ := factory.DecodePoolAddressesFromFactoryLog(ctx, l); lo.Contains(addrs, built.Address) {
					batch = append(batch, l)
				}
			}
			routed := len(batch)
			if routed == 0 {
				batch = triggerLog(p.BlockNumber)
			}
			require.False(t, tickspkg.HasState(built))
			stateless := guarded()
			fresh, err := stateless.GetNewPoolState(ctx, built, poolpkg.GetNewPoolStateParams{Logs: batch})
			require.NoError(t, err)
			require.True(t, tickspkg.HasState(fresh), "the pool got its state")
			freshExtra := mustExtra(t, fresh.Extra)
			assert.Equal(t, extra.Ticks, freshExtra.Ticks)
			assert.GreaterOrEqual(t, freshExtra.LensBlock, uint64(93210176))

			again, err := stateless.GetNewPoolState(ctx, fresh, poolpkg.GetNewPoolStateParams{Logs: batch})
			require.NoError(t, err)
			assert.Equal(t, freshExtra.Ticks, mustExtra(t, again.Extra).Ticks, "a replay changes no tick")

			sim, err := NewPoolSimulator(p, valueobject.ChainIDPolygon)
			require.NoError(t, err)
			t.Logf("%d ticks, spacing %d, lens block %d, %d logs of block 93210176 routed to it, pages under "+
				"3M gas give %d ticks, reserves %v, simulator %s", len(extra.Ticks), mustStatic(t, p).TickSpacing,
				extra.LensBlock, routed, len(paged.ticks), p.Reserves, sim.GetAddress())
		})
	}
}

func mustStatic(t *testing.T, p entity.Pool) StaticExtra {
	t.Helper()
	var s StaticExtra
	require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &s))
	return s
}
