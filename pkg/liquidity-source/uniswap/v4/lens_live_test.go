package uniswapv4

import (
	"math/big"
	"net/http"
	"os"
	"sync/atomic"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// countingTransport counts the HTTP requests a client makes.
type countingTransport struct{ n atomic.Int64 }

func (c *countingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	c.n.Add(1)
	return http.DefaultTransport.RoundTrip(req)
}

// livePool is Ethereum's ETH/USDC 0.05% Uniswap V4 pool, tick spacing 10, no hook.
func livePool() entity.Pool {
	return entity.Pool{
		Address:  "0x21c67e77068de97969ba93d4aab21826d33ca12bb9f565d8496e8fda8a82ca27",
		Exchange: "uniswap-v4",
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Decimals: 18, Swappable: true},
			{Address: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", Decimals: 6, Swappable: true},
		},
		StaticExtra: `{"0x0":[true,false],"fee":500,"tS":10,"hooks":"0x0000000000000000000000000000000000000000"}`,
	}
}

// TestLiveLens: on Ethereum, a bootstrap through the lens holds every tick StateView reports for
// those indices at the same block, a gas cap that forces paging reads the same set, a stored set
// missing a tick gets it back, and a pool without state gets it whole. Run with:
//
//	UNISWAP_V4_RPC_URL=https://... go test ./pkg/liquidity-source/uniswap/v4/ -run LiveLens -v
func TestLiveLens(t *testing.T) {
	rpcURL := os.Getenv("UNISWAP_V4_RPC_URL")
	if rpcURL == "" || os.Getenv("CI") != "" {
		t.Skip("set UNISWAP_V4_RPC_URL to an Ethereum endpoint to run this")
	}
	ctx := t.Context()
	counter := &countingTransport{}
	rc, err := rpc.DialOptions(ctx, rpcURL, rpc.WithHTTPClient(&http.Client{Transport: counter}))
	require.NoError(t, err)
	client := ethrpc.NewWithClient(ethclient.NewClient(rc)).
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	cfg := &Config{
		ChainID:          valueobject.ChainIDEthereum,
		DexID:            "uniswap-v4",
		StateViewAddress: "0x7ffe42c4a5deea5b0fec41c94c136cf115597227",
		UseLens:          true,
	}
	tracker := NewPoolTracker(cfg, client, nil)

	p, err := tracker.BootstrapPoolState(ctx, livePool(), poolpkg.GetNewPoolStateParams{})
	require.NoError(t, err)
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	require.NotEmpty(t, extra.Ticks)
	require.NotZero(t, p.BlockNumber)

	// Each tick against StateView's own answer for it, at the lens's block.
	viaRPC, err := tracker.queryRPCTicksByIndexes(ctx, p.Address,
		lo.Map(extra.Ticks, func(tk Tick, _ int) int { return tk.Index }), p.BlockNumber, nil)
	require.NoError(t, err)
	byIndex := lo.SliceToMap(viaRPC, func(tk tickspkg.Tick) (int, tickspkg.Tick) { return tk.TickIdx, tk })
	for _, tk := range extra.Ticks {
		got, ok := byIndex[tk.Index]
		require.True(t, ok, "tick %d", tk.Index)
		assert.Zero(t, tk.LiquidityGross.Cmp(got.LiquidityGross), "gross of tick %d", tk.Index)
		assert.Zero(t, tk.LiquidityNet.Cmp(got.LiquidityNet), "net of tick %d", tk.Index)
	}

	block := new(big.Int).SetUint64(p.BlockNumber)
	before := counter.n.Load()
	whole, err := tracker.snapshot(ctx, p, block)
	require.NoError(t, err)
	wholeCalls := counter.n.Load() - before
	cfg.LensGas = 3_000_000
	before = counter.n.Load()
	paged, err := tracker.snapshot(ctx, p, block)
	require.NoError(t, err)
	assert.Greater(t, counter.n.Load()-before, wholeCalls, "the cap did not force paging")
	assert.Equal(t, whole.ticks, paged.ticks, "paging changed the set")

	// A stored set missing a tick, as logs can leave it, is replaced by the full one.
	tbp, err := tickspkg.NewTicksBasedPool(p)
	require.NoError(t, err)
	tbp.BlockNumber = p.BlockNumber
	dropped := extra.Ticks[len(extra.Ticks)/2].Index
	delete(tbp.Ticks, dropped)
	require.NoError(t, tracker.replaceTicksFromLens(ctx, p, &tbp))
	assert.Contains(t, tbp.Ticks, dropped, "the missing tick was not restored")
	assert.Len(t, tbp.Ticks, len(extra.Ticks))

	// A pool created from its Initialize log has no state: its first update reads it whole at the
	// batch's newest block, whatever logs the batch carries.
	head := p.BlockNumber
	fresh, err := tracker.GetNewPoolState(ctx, livePool(), poolpkg.GetNewPoolStateParams{
		Logs:         []ethtypes.Log{{}},
		BlockHeaders: map[uint64]entity.BlockHeader{head - 1: {}, head: {}},
	})
	require.NoError(t, err)
	assert.True(t, tickspkg.HasState(fresh))
	assert.Equal(t, head, fresh.BlockNumber)
	var freshExtra Extra
	require.NoError(t, json.Unmarshal([]byte(fresh.Extra), &freshExtra))
	assert.Equal(t, head, freshExtra.LensBlock, "logs the snapshot holds would be applied again")

	t.Logf("%d ticks at block %d", len(whole.ticks), whole.block)
}
