package dexv2

import (
	"math/big"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/fluid/dex-v2/abis"
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
